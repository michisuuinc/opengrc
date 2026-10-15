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
package coredata

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/jackc/pgx/v5"
	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/gid"
)

type (
	// ThirdPartyPrivacyEvaluation holds the Japanese PrivacyMark outsourcing
	// evaluation (委託先評価表) of a third-party risk assessment: what is
	// outsourced, who evaluated and approved it, and the overall judgement.
	// An assessment has at most one row. The checklist answers are stored
	// separately in ThirdPartyPrivacyEvaluationItems.
	ThirdPartyPrivacyEvaluation struct {
		RiskAssessmentID   gid.GID                        `db:"risk_assessment_id"`
		OrganizationID     gid.GID                        `db:"organization_id"`
		Kind               *ThirdPartyEvaluationKind      `db:"kind"`
		AdHocReason        string                         `db:"ad_hoc_reason"`
		EntrustedWork      string                         `db:"entrusted_work"`
		PersonalData       string                         `db:"personal_data"`
		StorageCountries   string                         `db:"storage_countries"`
		EvaluatorProfileID *gid.GID                       `db:"evaluator_profile_id"`
		EvaluatedOn        *time.Time                     `db:"evaluated_on"`
		Judgement          *ThirdPartyEvaluationJudgement `db:"judgement"`
		Conditions         string                         `db:"conditions"`
		ApproverProfileID  *gid.GID                       `db:"approver_profile_id"`
		ApprovedOn         *time.Time                     `db:"approved_on"`
		CreatedAt          time.Time                      `db:"created_at"`
		UpdatedAt          time.Time                      `db:"updated_at"`
	}

	ThirdPartyPrivacyEvaluations []*ThirdPartyPrivacyEvaluation
)

const thirdPartyPrivacyEvaluationColumns = `
	risk_assessment_id,
	organization_id,
	kind,
	ad_hoc_reason,
	entrusted_work,
	personal_data,
	storage_countries,
	evaluator_profile_id,
	evaluated_on,
	judgement,
	conditions,
	approver_profile_id,
	approved_on,
	created_at,
	updated_at
`

func (e *ThirdPartyPrivacyEvaluation) LoadByRiskAssessmentID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	riskAssessmentID gid.GID,
) error {
	q := `
SELECT
` + thirdPartyPrivacyEvaluationColumns + `
FROM
	third_party_privacy_evaluations
WHERE
	%s
	AND risk_assessment_id = @risk_assessment_id
LIMIT 1;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"risk_assessment_id": riskAssessmentID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query third party privacy evaluation: %w", err)
	}

	evaluation, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[ThirdPartyPrivacyEvaluation])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect third party privacy evaluation: %w", err)
	}

	*e = evaluation

	return nil
}

// LoadByOrganizationID loads the evaluations of every assessment of the
// organization.
func (l *ThirdPartyPrivacyEvaluations) LoadByOrganizationID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	organizationID gid.GID,
) error {
	q := `
SELECT
` + thirdPartyPrivacyEvaluationColumns + `
FROM
	third_party_privacy_evaluations
WHERE
	%s
	AND organization_id = @organization_id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"organization_id": organizationID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query third party privacy evaluations: %w", err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[ThirdPartyPrivacyEvaluation])
	if err != nil {
		return fmt.Errorf("cannot collect third party privacy evaluations: %w", err)
	}

	*l = list

	return nil
}

// LockRiskAssessment locks the parent assessment row for the rest of the
// transaction and returns its organization. Concurrent updates of the same
// evaluation then run one after the other, including the first insert.
func (e *ThirdPartyPrivacyEvaluation) LockRiskAssessment(
	ctx context.Context,
	tx pg.Tx,
	scope Scoper,
	riskAssessmentID gid.GID,
) (gid.GID, error) {
	q := `
SELECT
	organization_id
FROM
	third_party_risk_assessments
WHERE
	%s
	AND id = @risk_assessment_id
FOR UPDATE;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"risk_assessment_id": riskAssessmentID}
	maps.Copy(args, scope.SQLArguments())

	var organizationID gid.GID
	if err := tx.QueryRow(ctx, q, args).Scan(&organizationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gid.GID{}, ErrResourceNotFound
		}

		return gid.GID{}, fmt.Errorf("cannot lock third party risk assessment: %w", err)
	}

	return organizationID, nil
}

// Upsert inserts the row or replaces its values. The receiver is synced with
// the stored row, so CreatedAt keeps its first value. The evaluator and
// approver must belong to the organization, or ErrResourceNotFound is
// returned.
func (e *ThirdPartyPrivacyEvaluation) Upsert(
	ctx context.Context,
	tx pg.Tx,
	scope Scoper,
) error {
	profileIDs := make([]gid.GID, 0, 2)
	for _, id := range []*gid.GID{e.EvaluatorProfileID, e.ApproverProfileID} {
		if id != nil && (len(profileIDs) == 0 || profileIDs[0] != *id) {
			profileIDs = append(profileIDs, *id)
		}
	}

	if err := checkProfilesInOrganization(ctx, tx, scope, e.OrganizationID, gidStrings(profileIDs)); err != nil {
		return err
	}

	q := `
INSERT INTO third_party_privacy_evaluations (
	risk_assessment_id,
	tenant_id,
	organization_id,
	kind,
	ad_hoc_reason,
	entrusted_work,
	personal_data,
	storage_countries,
	evaluator_profile_id,
	evaluated_on,
	judgement,
	conditions,
	approver_profile_id,
	approved_on,
	created_at,
	updated_at
) VALUES (
	@risk_assessment_id,
	@tenant_id,
	@organization_id,
	@kind,
	@ad_hoc_reason,
	@entrusted_work,
	@personal_data,
	@storage_countries,
	@evaluator_profile_id,
	@evaluated_on,
	@judgement,
	@conditions,
	@approver_profile_id,
	@approved_on,
	@created_at,
	@updated_at
)
ON CONFLICT (risk_assessment_id) DO UPDATE
SET
	kind = EXCLUDED.kind,
	ad_hoc_reason = EXCLUDED.ad_hoc_reason,
	entrusted_work = EXCLUDED.entrusted_work,
	personal_data = EXCLUDED.personal_data,
	storage_countries = EXCLUDED.storage_countries,
	evaluator_profile_id = EXCLUDED.evaluator_profile_id,
	evaluated_on = EXCLUDED.evaluated_on,
	judgement = EXCLUDED.judgement,
	conditions = EXCLUDED.conditions,
	approver_profile_id = EXCLUDED.approver_profile_id,
	approved_on = EXCLUDED.approved_on,
	updated_at = EXCLUDED.updated_at
WHERE
	third_party_privacy_evaluations.tenant_id = EXCLUDED.tenant_id
RETURNING
` + thirdPartyPrivacyEvaluationColumns

	args := pgx.StrictNamedArgs{
		"risk_assessment_id":   e.RiskAssessmentID,
		"tenant_id":            scope.GetTenantID(),
		"organization_id":      e.OrganizationID,
		"kind":                 e.Kind,
		"ad_hoc_reason":        e.AdHocReason,
		"entrusted_work":       e.EntrustedWork,
		"personal_data":        e.PersonalData,
		"storage_countries":    e.StorageCountries,
		"evaluator_profile_id": e.EvaluatorProfileID,
		"evaluated_on":         e.EvaluatedOn,
		"judgement":            e.Judgement,
		"conditions":           e.Conditions,
		"approver_profile_id":  e.ApproverProfileID,
		"approved_on":          e.ApprovedOn,
		"created_at":           e.CreatedAt,
		"updated_at":           e.UpdatedAt,
	}

	rows, err := tx.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot upsert third party privacy evaluation: %w", err)
	}

	evaluation, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[ThirdPartyPrivacyEvaluation])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect upserted third party privacy evaluation: %w", err)
	}

	*e = evaluation

	return nil
}
