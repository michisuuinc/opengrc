// Copyright (c) 2026 MICHISUU Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package probo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/page"
	"go.probo.inc/probo/pkg/xlsxexport"
)

// Export renders the outsourcing evaluation (委託先評価表) of the
// organization as an .xlsx workbook. It lists the third parties that
// register entries are outsourced to and those with an evaluation, each with
// its latest risk assessment, so a third party left unevaluated shows up.
func (s ThirdPartyPrivacyEvaluationService) Export(
	ctx context.Context,
	scope coredata.Scoper,
	organizationID gid.GID,
) ([]byte, error) {
	var data thirdPartyEvaluationData

	err := s.svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			var err error
			data, err = loadThirdPartyEvaluationData(ctx, conn, scope, organizationID)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	workbook, err := xlsxexport.BuildWorkbook(buildThirdPartyEvaluationSheets(data, time.Now().In(ledgerTimeZone))...)
	if err != nil {
		return nil, fmt.Errorf("cannot build outsourcing evaluation workbook: %w", err)
	}

	return workbook, nil
}

func loadThirdPartyEvaluationData(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	organizationID gid.GID,
) (thirdPartyEvaluationData, error) {
	organization := &coredata.Organization{}
	if err := organization.LoadByID(ctx, conn, scope, organizationID); err != nil {
		return thirdPartyEvaluationData{}, fmt.Errorf("cannot load organization: %w", err)
	}

	thirdParties, err := loadAllThirdParties(ctx, conn, scope, organizationID)
	if err != nil {
		return thirdPartyEvaluationData{}, err
	}

	var activities coredata.ProcessingActivities
	activityNames, err := activities.LoadNamesByThirdParty(ctx, conn, scope, organizationID)
	if err != nil {
		return thirdPartyEvaluationData{}, fmt.Errorf("cannot load processing activity names: %w", err)
	}

	var evaluations coredata.ThirdPartyPrivacyEvaluations
	if err := evaluations.LoadByOrganizationID(ctx, conn, scope, organizationID); err != nil {
		return thirdPartyEvaluationData{}, fmt.Errorf("cannot load third party privacy evaluations: %w", err)
	}

	evaluationByAssessment := make(map[gid.GID]*coredata.ThirdPartyPrivacyEvaluation, len(evaluations))
	for _, e := range evaluations {
		evaluationByAssessment[e.RiskAssessmentID] = e
	}

	var items coredata.ThirdPartyPrivacyEvaluationItems
	if err := items.LoadByOrganizationID(ctx, conn, scope, organizationID); err != nil {
		return thirdPartyEvaluationData{}, fmt.Errorf("cannot load third party privacy evaluation items: %w", err)
	}

	itemsByAssessment := make(map[gid.GID]coredata.ThirdPartyPrivacyEvaluationItems)
	for _, i := range items {
		itemsByAssessment[i.RiskAssessmentID] = append(itemsByAssessment[i.RiskAssessmentID], i)
	}

	latest, evaluated, err := loadLatestAssessments(ctx, conn, scope, thirdParties, evaluationByAssessment)
	if err != nil {
		return thirdPartyEvaluationData{}, err
	}

	names := make(map[gid.GID]string, len(thirdParties))
	for _, tp := range thirdParties {
		names[tp.ID] = tp.Name
	}

	rows := make([]thirdPartyEvaluationRow, 0, len(thirdParties))
	for _, tp := range thirdParties {
		if len(activityNames[tp.ID]) == 0 && !evaluated[tp.ID] {
			continue
		}

		row := thirdPartyEvaluationRow{
			ThirdParty:           tp,
			ProcessingActivities: activityNames[tp.ID],
			Assessment:           latest[tp.ID],
		}

		if tp.ParentThirdPartyID != nil {
			row.ParentName = names[*tp.ParentThirdPartyID]
		}

		if row.Assessment != nil {
			row.Evaluation = evaluationByAssessment[row.Assessment.ID]
			row.Answers = checklistAnswers(itemsByAssessment[row.Assessment.ID])
		}

		rows = append(rows, row)
	}

	if err := fillEvaluationProfileNames(ctx, conn, scope, rows); err != nil {
		return thirdPartyEvaluationData{}, err
	}

	return thirdPartyEvaluationData{
		OrganizationName: organization.Name,
		Rows:             rows,
	}, nil
}

func loadAllThirdParties(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	organizationID gid.GID,
) ([]*coredata.ThirdParty, error) {
	return page.LoadAll(
		ctx,
		page.OrderBy[coredata.ThirdPartyOrderField]{
			Field:     coredata.ThirdPartyOrderFieldName,
			Direction: page.OrderDirectionAsc,
		},
		func(ctx context.Context, cursor *page.Cursor[coredata.ThirdPartyOrderField]) ([]*coredata.ThirdParty, error) {
			var batch coredata.ThirdParties
			if err := batch.LoadByOrganizationID(ctx, conn, scope, organizationID, cursor, nil); err != nil {
				return nil, fmt.Errorf("cannot load third parties: %w", err)
			}

			return batch, nil
		},
	)
}

// loadLatestAssessments returns the latest risk assessment of each third
// party, and which third parties have an evaluation on any assessment.
func loadLatestAssessments(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	thirdParties []*coredata.ThirdParty,
	evaluationByAssessment map[gid.GID]*coredata.ThirdPartyPrivacyEvaluation,
) (map[gid.GID]*coredata.ThirdPartyRiskAssessment, map[gid.GID]bool, error) {
	ids := make([]gid.GID, len(thirdParties))
	for i, tp := range thirdParties {
		ids[i] = tp.ID
	}

	var assessments coredata.ThirdPartyRiskAssessments
	if err := assessments.LoadByThirdPartyIDs(ctx, conn, scope, ids); err != nil {
		return nil, nil, fmt.Errorf("cannot load third party risk assessments: %w", err)
	}

	latest := make(map[gid.GID]*coredata.ThirdPartyRiskAssessment, len(thirdParties))
	evaluated := make(map[gid.GID]bool, len(thirdParties))
	for _, a := range assessments {
		if current, ok := latest[a.ThirdPartyID]; !ok || a.CreatedAt.After(current.CreatedAt) {
			latest[a.ThirdPartyID] = a
		}

		if _, ok := evaluationByAssessment[a.ID]; ok {
			evaluated[a.ThirdPartyID] = true
		}
	}

	return latest, evaluated, nil
}

func fillEvaluationProfileNames(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	rows []thirdPartyEvaluationRow,
) error {
	ids := []gid.GID{}
	seen := map[gid.GID]bool{}
	for _, r := range rows {
		if r.Evaluation == nil {
			continue
		}

		for _, id := range []*gid.GID{r.Evaluation.EvaluatorProfileID, r.Evaluation.ApproverProfileID} {
			if id != nil && !seen[*id] {
				seen[*id] = true
				ids = append(ids, *id)
			}
		}
	}

	if len(ids) == 0 {
		return nil
	}

	var profiles coredata.MembershipProfiles
	if err := profiles.LoadByIDs(ctx, conn, scope, ids); err != nil && !errors.Is(err, coredata.ErrResourceNotFound) {
		return fmt.Errorf("cannot load evaluators: %w", err)
	}

	names := make(map[gid.GID]string, len(profiles))
	for _, p := range profiles {
		names[p.ID] = p.FullName
	}

	for i := range rows {
		e := rows[i].Evaluation
		if e == nil {
			continue
		}

		if e.EvaluatorProfileID != nil {
			rows[i].EvaluatorName = names[*e.EvaluatorProfileID]
		}

		if e.ApproverProfileID != nil {
			rows[i].ApproverName = names[*e.ApproverProfileID]
		}
	}

	return nil
}
