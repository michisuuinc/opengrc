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
package console_v1

import (
	"context"
	"errors"

	"go.gearno.de/kit/log"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/iam"
	"go.probo.inc/probo/pkg/probo"
	"go.probo.inc/probo/pkg/server/api/console/v1/types"
	"go.probo.inc/probo/pkg/server/gqlutils"
	"go.probo.inc/probo/pkg/validator"
)

func newThirdPartyPrivacyEvaluation(v *probo.ThirdPartyPrivacyEvaluationView) *types.ThirdPartyPrivacyEvaluation {
	e := v.Evaluation

	object := &types.ThirdPartyPrivacyEvaluation{
		RiskAssessmentID: e.RiskAssessmentID,
		EvaluatorID:      e.EvaluatorProfileID,
		ApproverID:       e.ApproverProfileID,
		Kind:             e.Kind,
		AdHocReason:      e.AdHocReason,
		EntrustedWork:    e.EntrustedWork,
		PersonalData:     e.PersonalData,
		StorageCountries: e.StorageCountries,
		EvaluatedOn:      formatTrainingDate(e.EvaluatedOn),
		Answers:          make([]*types.ThirdPartyEvaluationAnswer, len(v.Answers)),
		Judgement:        e.Judgement,
		Conditions:       e.Conditions,
		ApprovedOn:       formatTrainingDate(e.ApprovedOn),
	}

	for i, a := range v.Answers {
		object.Answers[i] = &types.ThirdPartyEvaluationAnswer{
			Item:   a.Item,
			Result: a.Result,
			Note:   a.Note,
		}
	}

	if !e.UpdatedAt.IsZero() {
		object.UpdatedAt = &e.UpdatedAt
	}

	return object
}

// thirdPartyEvaluationProfile resolves the evaluator or approver of an
// evaluation. Reading it needs the member list besides the assessment.
func (r *Resolver) thirdPartyEvaluationProfile(
	ctx context.Context,
	riskAssessmentID gid.GID,
	profileID *gid.GID,
) (*types.Profile, error) {
	if _, err := r.authorize(ctx, riskAssessmentID, iam.ActionMembershipProfileList); err != nil {
		return nil, err
	}

	scope, err := r.authorize(ctx, riskAssessmentID, probo.ActionThirdPartyRiskAssessmentList)
	if err != nil {
		return nil, err
	}

	profile, err := r.probo.ThirdPartyPrivacyEvaluations.GetProfile(ctx, scope, profileID)
	if err != nil {
		r.logger.ErrorCtx(ctx, "cannot get third party evaluation profile", log.Error(err))
		return nil, gqlutils.Internal(ctx)
	}

	if profile == nil {
		return nil, nil
	}

	return types.NewProfile(profile), nil
}

func newUpdateThirdPartyPrivacyEvaluationRequest(
	input types.UpdateThirdPartyPrivacyEvaluationInput,
) (*probo.UpdateThirdPartyPrivacyEvaluationRequest, error) {
	evaluatedOn, err := parseOptionalTrainingDate("evaluatedOn", input.EvaluatedOn)
	if err != nil {
		return nil, err
	}

	approvedOn, err := parseOptionalTrainingDate("approvedOn", input.ApprovedOn)
	if err != nil {
		return nil, err
	}

	answers := make([]probo.ThirdPartyEvaluationAnswer, len(input.Answers))
	for i, a := range input.Answers {
		answers[i] = probo.ThirdPartyEvaluationAnswer{
			Item:   a.Item,
			Result: a.Result,
			Note:   a.Note,
		}
	}

	return &probo.UpdateThirdPartyPrivacyEvaluationRequest{
		RiskAssessmentID: input.RiskAssessmentID,
		Kind:             input.Kind,
		AdHocReason:      input.AdHocReason,
		EntrustedWork:    input.EntrustedWork,
		PersonalData:     input.PersonalData,
		StorageCountries: input.StorageCountries,
		EvaluatorID:      input.EvaluatorID,
		EvaluatedOn:      evaluatedOn,
		Answers:          answers,
		Judgement:        input.Judgement,
		Conditions:       input.Conditions,
		ApproverID:       input.ApproverID,
		ApprovedOn:       approvedOn,
	}, nil
}

// thirdPartyEvaluationError maps errors of the outsourcing evaluation API to
// GraphQL errors. msg is logged for unexpected errors.
func (r *Resolver) thirdPartyEvaluationError(ctx context.Context, err error, msg string) error {
	if errs, ok := errors.AsType[validator.ValidationErrors](err); ok {
		return gqlutils.InvalidValidationErrors(ctx, errs)
	}
	if errDate, ok := errors.AsType[*errInvalidTrainingDate](err); ok {
		return gqlutils.Invalid(ctx, errDate)
	}
	if errInconsistent, ok := errors.AsType[*probo.ErrThirdPartyEvaluationInconsistent](err); ok {
		return gqlutils.Invalid(ctx, errInconsistent)
	}
	if errors.Is(err, coredata.ErrResourceNotFound) {
		return gqlutils.NotFoundf(ctx, "risk assessment, evaluator or approver not found")
	}

	r.logger.ErrorCtx(ctx, msg, log.Error(err))

	return gqlutils.Internal(ctx)
}
