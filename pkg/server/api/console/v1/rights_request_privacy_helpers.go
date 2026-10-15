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
	"time"

	"go.gearno.de/kit/log"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/iam"
	"go.probo.inc/probo/pkg/probo"
	"go.probo.inc/probo/pkg/server/api/console/v1/types"
	"go.probo.inc/probo/pkg/server/gqlutils"
	"go.probo.inc/probo/pkg/validator"
)

func newRightsRequestPrivacyDetails(
	request *coredata.RightsRequest,
	d *coredata.RightsRequestPrivacyDetails,
) *types.RightsRequestPrivacyDetails {
	receivedOn := probo.RightsRequestEffectiveReceivedOn(request, d)

	object := &types.RightsRequestPrivacyDetails{
		RightsRequestID:         request.ID,
		HandlerID:               d.HandlerProfileID,
		ApproverID:              d.ApproverProfileID,
		Category:                d.Category,
		EffectiveCategory:       probo.RightsRequestEffectiveCategory(request, d),
		ReceivedOn:              formatTrainingDate(d.ReceivedOn),
		EffectiveReceivedOn:     receivedOn.Format(time.DateOnly),
		ReceiptChannel:          d.ReceiptChannel,
		EffectiveReceiptChannel: probo.RightsRequestEffectiveReceiptChannel(request, d),
		ByAgent:                 d.ByAgent,
		TargetDepartment:        d.TargetDepartment,
		VerificationMethod:      d.VerificationMethod,
		VerifiedOn:              formatTrainingDate(d.VerifiedOn),
		VerificationNote:        d.VerificationNote,
		SupplementRequestedOn:   formatTrainingDate(d.SupplementRequestedOn),
		SupplementReceivedOn:    formatTrainingDate(d.SupplementReceivedOn),
		Decision:                d.Decision,
		DecisionReason:          d.DecisionReason,
		AnswerMethod:            d.AnswerMethod,
		AnsweredOn:              formatTrainingDate(d.AnsweredOn),
		HandlingDays:            probo.RightsRequestHandlingDays(receivedOn, d),
	}

	if !d.UpdatedAt.IsZero() {
		object.UpdatedAt = &d.UpdatedAt
	}

	return object
}

// rightsRequestRecordProfile resolves the handler or approver of a request.
// Reading it needs the member list besides the request.
func (r *Resolver) rightsRequestRecordProfile(
	ctx context.Context,
	rightsRequestID gid.GID,
	profileID *gid.GID,
) (*types.Profile, error) {
	if _, err := r.authorize(ctx, rightsRequestID, iam.ActionMembershipProfileList); err != nil {
		return nil, err
	}

	scope, err := r.authorize(ctx, rightsRequestID, probo.ActionRightsRequestGet)
	if err != nil {
		return nil, err
	}

	profile, err := r.probo.RightsRequestPrivacy.GetProfile(ctx, scope, profileID)
	if err != nil {
		r.logger.ErrorCtx(ctx, "cannot get rights request record profile", log.Error(err))
		return nil, gqlutils.Internal(ctx)
	}

	if profile == nil {
		return nil, nil
	}

	return types.NewProfile(profile), nil
}

func newUpdateRightsRequestPrivacyDetailsRequest(
	input types.UpdateRightsRequestPrivacyDetailsInput,
) (*probo.UpdateRightsRequestPrivacyDetailsRequest, error) {
	req := &probo.UpdateRightsRequestPrivacyDetailsRequest{
		RightsRequestID:    input.RightsRequestID,
		Category:           input.Category,
		ReceiptChannel:     input.ReceiptChannel,
		ByAgent:            input.ByAgent,
		TargetDepartment:   input.TargetDepartment,
		HandlerID:          input.HandlerID,
		VerificationMethod: input.VerificationMethod,
		VerificationNote:   input.VerificationNote,
		Decision:           input.Decision,
		DecisionReason:     input.DecisionReason,
		AnswerMethod:       input.AnswerMethod,
		ApproverID:         input.ApproverID,
	}

	for _, date := range []struct {
		field  string
		value  *string
		target **time.Time
	}{
		{"receivedOn", input.ReceivedOn, &req.ReceivedOn},
		{"verifiedOn", input.VerifiedOn, &req.VerifiedOn},
		{"supplementRequestedOn", input.SupplementRequestedOn, &req.SupplementRequestedOn},
		{"supplementReceivedOn", input.SupplementReceivedOn, &req.SupplementReceivedOn},
		{"answeredOn", input.AnsweredOn, &req.AnsweredOn},
	} {
		parsed, err := parseOptionalTrainingDate(date.field, date.value)
		if err != nil {
			return nil, err
		}
		*date.target = parsed
	}

	return req, nil
}

// rightsRequestRecordError maps errors of the record API to GraphQL errors.
// msg is logged for unexpected errors.
func (r *Resolver) rightsRequestRecordError(ctx context.Context, err error, msg string) error {
	if errs, ok := errors.AsType[validator.ValidationErrors](err); ok {
		return gqlutils.InvalidValidationErrors(ctx, errs)
	}
	if errDate, ok := errors.AsType[*errInvalidTrainingDate](err); ok {
		return gqlutils.Invalid(ctx, errDate)
	}
	if errInconsistent, ok := errors.AsType[*probo.ErrRightsRequestRecordInconsistent](err); ok {
		return gqlutils.Invalid(ctx, errInconsistent)
	}
	if errPeriod, ok := errors.AsType[*probo.ErrRightsRequestInvalidPeriod](err); ok {
		return gqlutils.Invalid(ctx, errPeriod)
	}
	if errors.Is(err, coredata.ErrResourceNotFound) {
		return gqlutils.NotFoundf(ctx, "request, handler or approver not found")
	}

	r.logger.ErrorCtx(ctx, msg, log.Error(err))

	return gqlutils.Internal(ctx)
}
