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
	"strings"
	"time"

	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/validator"
)

// ThirdPartyPrivacyEvaluationService manages the Japanese PrivacyMark
// outsourcing evaluation (委託先評価表): the evaluation checklist of a
// third-party risk assessment, and the Excel export of every third party's
// latest evaluation.
type ThirdPartyPrivacyEvaluationService struct {
	svc *Service
}

type (
	ThirdPartyEvaluationAnswer struct {
		Item   coredata.ThirdPartyEvaluationItem
		Result *coredata.ThirdPartyEvaluationResult
		Note   string
	}

	UpdateThirdPartyPrivacyEvaluationRequest struct {
		RiskAssessmentID gid.GID
		Kind             *coredata.ThirdPartyEvaluationKind
		AdHocReason      string
		EntrustedWork    string
		PersonalData     string
		StorageCountries string
		EvaluatorID      *gid.GID
		EvaluatedOn      *time.Time
		Answers          []ThirdPartyEvaluationAnswer
		Judgement        *coredata.ThirdPartyEvaluationJudgement
		Conditions       string
		ApproverID       *gid.GID
		ApprovedOn       *time.Time
	}

	// ThirdPartyPrivacyEvaluationView is an evaluation with the answer to
	// every checklist item, in the checklist order. Items not answered
	// have no result and an empty note.
	ThirdPartyPrivacyEvaluationView struct {
		Evaluation *coredata.ThirdPartyPrivacyEvaluation
		Answers    []ThirdPartyEvaluationAnswer
	}

	// ErrThirdPartyEvaluationInconsistent is an evaluation whose values
	// contradict the evaluation checklist's rules.
	ErrThirdPartyEvaluationInconsistent struct {
		Reason string
	}
)

func (e *ErrThirdPartyEvaluationInconsistent) Error() string {
	return e.Reason
}

func (r *UpdateThirdPartyPrivacyEvaluationRequest) Validate() error {
	v := validator.New()

	v.Check(r.RiskAssessmentID, "risk_assessment_id", validator.Required(), validator.GID(coredata.ThirdPartyRiskAssessmentEntityType))
	v.Check(r.Kind, "kind", validator.OneOfSlice(coredata.ThirdPartyEvaluationKinds()))
	v.Check(r.Judgement, "judgement", validator.OneOfSlice(coredata.ThirdPartyEvaluationJudgements()))
	v.Check(r.EvaluatorID, "evaluator_id", validator.GID(coredata.MembershipProfileEntityType))
	v.Check(r.ApproverID, "approver_id", validator.GID(coredata.MembershipProfileEntityType))
	checkOptionalTrainingText(v, r.AdHocReason, "ad_hoc_reason", validator.SafeTextNoNewLine(TitleMaxLength))
	checkOptionalTrainingText(v, r.EntrustedWork, "entrusted_work", validator.SafeText(ContentMaxLength))
	checkOptionalTrainingText(v, r.PersonalData, "personal_data", validator.SafeText(ContentMaxLength))
	checkOptionalTrainingText(v, r.StorageCountries, "storage_countries", validator.SafeTextNoNewLine(TitleMaxLength))
	checkOptionalTrainingText(v, r.Conditions, "conditions", validator.SafeText(ContentMaxLength))

	items := make([]coredata.ThirdPartyEvaluationItem, len(r.Answers))
	for i, a := range r.Answers {
		items[i] = a.Item
	}

	v.Check(&items, "answers", validator.NoDuplicates(), maxItems(len(coredata.ThirdPartyEvaluationItems())))
	for i, a := range r.Answers {
		field := fmt.Sprintf("answers[%d]", i)
		v.Check(a.Item, field+".item", validator.Required(), validator.OneOfSlice(coredata.ThirdPartyEvaluationItems()))
		v.Check(a.Result, field+".result", validator.OneOfSlice(coredata.ThirdPartyEvaluationResults()))
		checkOptionalTrainingText(v, a.Note, field+".note", validator.SafeText(TitleMaxLength))
	}

	return v.Error()
}

// checkConsistency applies the rules of the evaluation checklist (35):
// outsourcing is allowed outright only when no item fails, a conditional
// approval states its conditions, and an ad hoc evaluation states why.
func (r *UpdateThirdPartyPrivacyEvaluationRequest) checkConsistency() error {
	if r.Kind != nil && *r.Kind == coredata.ThirdPartyEvaluationKindAdHoc && r.AdHocReason == "" {
		return &ErrThirdPartyEvaluationInconsistent{Reason: "an ad hoc evaluation needs a reason"}
	}

	if r.Judgement != nil {
		switch *r.Judgement {
		case coredata.ThirdPartyEvaluationJudgementApproved:
			for _, a := range r.Answers {
				if a.Result != nil && *a.Result == coredata.ThirdPartyEvaluationResultUnsuitable {
					return &ErrThirdPartyEvaluationInconsistent{
						Reason: "outsourcing cannot be approved without conditions while a checklist item is unsuitable",
					}
				}
			}
		case coredata.ThirdPartyEvaluationJudgementConditionallyApproved:
			if r.Conditions == "" {
				return &ErrThirdPartyEvaluationInconsistent{Reason: "a conditional approval needs its conditions"}
			}
		}
	}

	if r.EvaluatedOn != nil && r.ApprovedOn != nil && r.ApprovedOn.Before(*r.EvaluatedOn) {
		return &ErrThirdPartyEvaluationInconsistent{Reason: "the approval date must not be before the evaluation date"}
	}

	return nil
}

// normalize trims the texts, drops a reason given for an evaluation that is
// not ad hoc and drops answers with neither a result nor a note.
func (r *UpdateThirdPartyPrivacyEvaluationRequest) normalize() *UpdateThirdPartyPrivacyEvaluationRequest {
	normalized := *r
	normalized.AdHocReason = strings.TrimSpace(r.AdHocReason)
	normalized.EntrustedWork = strings.TrimSpace(r.EntrustedWork)
	normalized.PersonalData = strings.TrimSpace(r.PersonalData)
	normalized.StorageCountries = strings.TrimSpace(r.StorageCountries)
	normalized.Conditions = strings.TrimSpace(r.Conditions)

	if r.Kind == nil || *r.Kind != coredata.ThirdPartyEvaluationKindAdHoc {
		normalized.AdHocReason = ""
	}

	if r.EvaluatedOn != nil {
		evaluatedOn := trainingDateOnly(*r.EvaluatedOn)
		normalized.EvaluatedOn = &evaluatedOn
	}

	if r.ApprovedOn != nil {
		approvedOn := trainingDateOnly(*r.ApprovedOn)
		normalized.ApprovedOn = &approvedOn
	}

	normalized.Answers = make([]ThirdPartyEvaluationAnswer, 0, len(r.Answers))
	for _, a := range r.Answers {
		a.Note = strings.TrimSpace(a.Note)
		if a.Result == nil && a.Note == "" {
			continue
		}
		normalized.Answers = append(normalized.Answers, a)
	}

	return &normalized
}

// Get returns the evaluation of a risk assessment. An assessment with no
// evaluation yet gets an empty one, not an error.
func (s ThirdPartyPrivacyEvaluationService) Get(
	ctx context.Context,
	scope coredata.Scoper,
	riskAssessmentID gid.GID,
) (*ThirdPartyPrivacyEvaluationView, error) {
	var view *ThirdPartyPrivacyEvaluationView

	err := s.svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			evaluation := &coredata.ThirdPartyPrivacyEvaluation{}
			err := evaluation.LoadByRiskAssessmentID(ctx, conn, scope, riskAssessmentID)
			switch {
			case errors.Is(err, coredata.ErrResourceNotFound):
				evaluation = &coredata.ThirdPartyPrivacyEvaluation{RiskAssessmentID: riskAssessmentID}
			case err != nil:
				return fmt.Errorf("cannot load third party privacy evaluation: %w", err)
			}

			var items coredata.ThirdPartyPrivacyEvaluationItems
			if err := items.LoadByRiskAssessmentID(ctx, conn, scope, riskAssessmentID); err != nil {
				return fmt.Errorf("cannot load third party privacy evaluation items: %w", err)
			}

			view = &ThirdPartyPrivacyEvaluationView{
				Evaluation: evaluation,
				Answers:    checklistAnswers(items),
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return view, nil
}

// checklistAnswers returns the answer to every checklist item, in the
// checklist order.
func checklistAnswers(items coredata.ThirdPartyPrivacyEvaluationItems) []ThirdPartyEvaluationAnswer {
	byItem := make(map[coredata.ThirdPartyEvaluationItem]*coredata.ThirdPartyPrivacyEvaluationItem, len(items))
	for _, i := range items {
		byItem[i.Item] = i
	}

	all := coredata.ThirdPartyEvaluationItems()
	answers := make([]ThirdPartyEvaluationAnswer, len(all))
	for i, item := range all {
		answers[i] = ThirdPartyEvaluationAnswer{Item: item}
		if stored, ok := byItem[item]; ok {
			answers[i].Result = stored.Result
			answers[i].Note = stored.Note
		}
	}

	return answers
}

// GetProfile returns a member profile, or nil when the ID is nil or the
// profile is gone.
func (s ThirdPartyPrivacyEvaluationService) GetProfile(
	ctx context.Context,
	scope coredata.Scoper,
	profileID *gid.GID,
) (*coredata.MembershipProfile, error) {
	return getOptionalProfile(ctx, s.svc, scope, profileID)
}

// getOptionalProfile returns a member profile, or nil when the ID is nil or
// the profile is gone.
func getOptionalProfile(
	ctx context.Context,
	svc *Service,
	scope coredata.Scoper,
	profileID *gid.GID,
) (*coredata.MembershipProfile, error) {
	if profileID == nil {
		return nil, nil
	}

	var profiles coredata.MembershipProfiles

	err := svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			return profiles.LoadByIDs(ctx, conn, scope, []gid.GID{*profileID})
		},
	)
	if err != nil && !errors.Is(err, coredata.ErrResourceNotFound) {
		return nil, fmt.Errorf("cannot load membership profile: %w", err)
	}

	if len(profiles) == 0 {
		return nil, nil
	}

	return profiles[0], nil
}

// Update replaces the evaluation of the risk assessment and its checklist
// answers.
func (s ThirdPartyPrivacyEvaluationService) Update(
	ctx context.Context,
	scope coredata.Scoper,
	request *UpdateThirdPartyPrivacyEvaluationRequest,
) error {
	req := request.normalize()

	if err := req.Validate(); err != nil {
		return err
	}

	if err := req.checkConsistency(); err != nil {
		return err
	}

	return s.svc.pg.WithTx(
		ctx,
		func(ctx context.Context, tx pg.Tx) error {
			evaluation := &coredata.ThirdPartyPrivacyEvaluation{}

			organizationID, err := evaluation.LockRiskAssessment(ctx, tx, scope, req.RiskAssessmentID)
			if err != nil {
				return fmt.Errorf("cannot lock third party risk assessment: %w", err)
			}

			now := time.Now()

			createdAt := now
			err = evaluation.LoadByRiskAssessmentID(ctx, tx, scope, req.RiskAssessmentID)
			switch {
			case err == nil:
				createdAt = evaluation.CreatedAt
			case !errors.Is(err, coredata.ErrResourceNotFound):
				return fmt.Errorf("cannot load third party privacy evaluation: %w", err)
			}

			updated := &coredata.ThirdPartyPrivacyEvaluation{
				RiskAssessmentID:   req.RiskAssessmentID,
				OrganizationID:     organizationID,
				Kind:               req.Kind,
				AdHocReason:        req.AdHocReason,
				EntrustedWork:      req.EntrustedWork,
				PersonalData:       req.PersonalData,
				StorageCountries:   req.StorageCountries,
				EvaluatorProfileID: req.EvaluatorID,
				EvaluatedOn:        req.EvaluatedOn,
				Judgement:          req.Judgement,
				Conditions:         req.Conditions,
				ApproverProfileID:  req.ApproverID,
				ApprovedOn:         req.ApprovedOn,
				CreatedAt:          createdAt,
				UpdatedAt:          now,
			}

			if err := updated.Upsert(ctx, tx, scope); err != nil {
				return fmt.Errorf("cannot upsert third party privacy evaluation: %w", err)
			}

			answers := make(coredata.ThirdPartyPrivacyEvaluationItems, len(req.Answers))
			for i, a := range req.Answers {
				answers[i] = &coredata.ThirdPartyPrivacyEvaluationItem{
					Item:   a.Item,
					Result: a.Result,
					Note:   a.Note,
				}
			}

			if err := answers.Replace(ctx, tx, scope, req.RiskAssessmentID, organizationID, now); err != nil {
				return fmt.Errorf("cannot replace third party privacy evaluation items: %w", err)
			}

			return nil
		},
	)
}
