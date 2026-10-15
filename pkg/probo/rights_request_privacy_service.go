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

// RightsRequestPrivacyService manages the Japanese PrivacyMark record of
// requests from individuals (本人からの請求の記録): the receipt,
// verification, decision and answer of each request, and the Excel export
// of the records with a summary for the management review.
type RightsRequestPrivacyService struct {
	svc *Service
}

type (
	UpdateRightsRequestPrivacyDetailsRequest struct {
		RightsRequestID       gid.GID
		Category              *coredata.RightsRequestPrivacyCategory
		ReceivedOn            *time.Time
		ReceiptChannel        *coredata.RightsRequestReceiptChannel
		ByAgent               bool
		TargetDepartment      string
		HandlerID             *gid.GID
		VerificationMethod    *coredata.RightsRequestVerificationMethod
		VerifiedOn            *time.Time
		VerificationNote      string
		SupplementRequestedOn *time.Time
		SupplementReceivedOn  *time.Time
		Decision              *coredata.RightsRequestDecision
		DecisionReason        string
		AnswerMethod          *coredata.RightsRequestAnswerMethod
		AnsweredOn            *time.Time
		ApproverID            *gid.GID
		FirstRespondedOn      *time.Time
		FindingID             *gid.GID
	}

	// ErrRightsRequestRecordInconsistent is a record whose values
	// contradict the rights request procedure or each other.
	ErrRightsRequestRecordInconsistent struct {
		Reason string
	}
)

func (e *ErrRightsRequestRecordInconsistent) Error() string {
	return e.Reason
}

func (r *UpdateRightsRequestPrivacyDetailsRequest) Validate() error {
	v := validator.New()

	v.Check(r.RightsRequestID, "rights_request_id", validator.Required(), validator.GID(coredata.RightsRequestEntityType))
	v.Check(r.Category, "category", validator.OneOfSlice(coredata.RightsRequestPrivacyCategories()))
	v.Check(r.ReceiptChannel, "receipt_channel", validator.OneOfSlice(coredata.RightsRequestReceiptChannels()))
	v.Check(r.VerificationMethod, "verification_method", validator.OneOfSlice(coredata.RightsRequestVerificationMethods()))
	v.Check(r.Decision, "decision", validator.OneOfSlice(coredata.RightsRequestDecisions()))
	v.Check(r.AnswerMethod, "answer_method", validator.OneOfSlice(coredata.RightsRequestAnswerMethods()))
	v.Check(r.HandlerID, "handler_id", validator.GID(coredata.MembershipProfileEntityType))
	v.Check(r.ApproverID, "approver_id", validator.GID(coredata.MembershipProfileEntityType))
	v.Check(r.FindingID, "finding_id", validator.GID(coredata.FindingEntityType))
	checkOptionalTrainingText(v, r.TargetDepartment, "target_department", validator.SafeTextNoNewLine(NameMaxLength))
	checkOptionalTrainingText(v, r.VerificationNote, "verification_note", validator.SafeText(TitleMaxLength))
	checkOptionalTrainingText(v, r.DecisionReason, "decision_reason", validator.SafeText(ContentMaxLength))

	return v.Error()
}

// checkConsistency applies the rights request procedure (14): a request
// not answered as requested states why, and no step happens before the
// request was received or a document arrives before it was asked for.
func (r *UpdateRightsRequestPrivacyDetailsRequest) checkConsistency() error {
	if r.Decision != nil && *r.Decision != coredata.RightsRequestDecisionFulfilled && r.DecisionReason == "" {
		return &ErrRightsRequestRecordInconsistent{Reason: "a request not answered as requested needs the reason"}
	}

	if r.SupplementReceivedOn != nil {
		if r.SupplementRequestedOn == nil {
			return &ErrRightsRequestRecordInconsistent{Reason: "missing documents cannot be received before they were asked for"}
		}

		if r.SupplementReceivedOn.Before(*r.SupplementRequestedOn) {
			return &ErrRightsRequestRecordInconsistent{Reason: "missing documents cannot be received before they were asked for"}
		}
	}

	if r.ReceivedOn != nil {
		for _, date := range []*time.Time{r.VerifiedOn, r.SupplementRequestedOn, r.AnsweredOn, r.FirstRespondedOn} {
			if date != nil && date.Before(*r.ReceivedOn) {
				return &ErrRightsRequestRecordInconsistent{Reason: "verification, requests for documents, the first response and the answer cannot come before the request was received"}
			}
		}
	}

	return nil
}

// normalize trims the texts, drops the reason of a request answered as
// requested and keeps only the date of each date.
func (r *UpdateRightsRequestPrivacyDetailsRequest) normalize() *UpdateRightsRequestPrivacyDetailsRequest {
	normalized := *r
	normalized.TargetDepartment = strings.TrimSpace(r.TargetDepartment)
	normalized.VerificationNote = strings.TrimSpace(r.VerificationNote)
	normalized.DecisionReason = strings.TrimSpace(r.DecisionReason)

	if r.Decision == nil || *r.Decision == coredata.RightsRequestDecisionFulfilled {
		normalized.DecisionReason = ""
	}

	for _, date := range []**time.Time{
		&normalized.ReceivedOn,
		&normalized.VerifiedOn,
		&normalized.SupplementRequestedOn,
		&normalized.SupplementReceivedOn,
		&normalized.AnsweredOn,
		&normalized.FirstRespondedOn,
	} {
		if *date != nil {
			value := trainingDateOnly(**date)
			*date = &value
		}
	}

	return &normalized
}

// GetDetails returns the record items of a request. A request with no
// items yet gets an empty value, not an error.
func (s RightsRequestPrivacyService) GetDetails(
	ctx context.Context,
	scope coredata.Scoper,
	rightsRequestID gid.GID,
) (*coredata.RightsRequestPrivacyDetails, error) {
	details := &coredata.RightsRequestPrivacyDetails{}

	err := s.svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			return details.LoadByRightsRequestID(ctx, conn, scope, rightsRequestID)
		},
	)
	if errors.Is(err, coredata.ErrResourceNotFound) {
		return &coredata.RightsRequestPrivacyDetails{RightsRequestID: rightsRequestID}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("cannot load rights request privacy details: %w", err)
	}

	return details, nil
}

// GetProfile returns a member profile, or nil when the ID is nil or the
// profile is gone.
func (s RightsRequestPrivacyService) GetProfile(
	ctx context.Context,
	scope coredata.Scoper,
	profileID *gid.GID,
) (*coredata.MembershipProfile, error) {
	return getOptionalProfile(ctx, s.svc, scope, profileID)
}

// UpdateDetails replaces the record items of the request.
func (s RightsRequestPrivacyService) UpdateDetails(
	ctx context.Context,
	scope coredata.Scoper,
	request *UpdateRightsRequestPrivacyDetailsRequest,
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
			details := &coredata.RightsRequestPrivacyDetails{}

			organizationID, err := details.LockRightsRequest(ctx, tx, scope, req.RightsRequestID)
			if err != nil {
				return fmt.Errorf("cannot lock rights request: %w", err)
			}

			now := time.Now()

			createdAt := now
			err = details.LoadByRightsRequestID(ctx, tx, scope, req.RightsRequestID)
			switch {
			case err == nil:
				createdAt = details.CreatedAt
			case !errors.Is(err, coredata.ErrResourceNotFound):
				return fmt.Errorf("cannot load rights request privacy details: %w", err)
			}

			updated := &coredata.RightsRequestPrivacyDetails{
				RightsRequestID:       req.RightsRequestID,
				OrganizationID:        organizationID,
				Category:              req.Category,
				ReceivedOn:            req.ReceivedOn,
				ReceiptChannel:        req.ReceiptChannel,
				ByAgent:               req.ByAgent,
				TargetDepartment:      req.TargetDepartment,
				HandlerProfileID:      req.HandlerID,
				VerificationMethod:    req.VerificationMethod,
				VerifiedOn:            req.VerifiedOn,
				VerificationNote:      req.VerificationNote,
				SupplementRequestedOn: req.SupplementRequestedOn,
				SupplementReceivedOn:  req.SupplementReceivedOn,
				Decision:              req.Decision,
				DecisionReason:        req.DecisionReason,
				AnswerMethod:          req.AnswerMethod,
				AnsweredOn:            req.AnsweredOn,
				ApproverProfileID:     req.ApproverID,
				FirstRespondedOn:      req.FirstRespondedOn,
				FindingID:             req.FindingID,
				CreatedAt:             createdAt,
				UpdatedAt:             now,
			}

			if err := updated.Upsert(ctx, tx, scope); err != nil {
				return fmt.Errorf("cannot upsert rights request privacy details: %w", err)
			}

			return nil
		},
	)
}

// RightsRequestEffectiveCategory is the chosen category, or the one the
// upstream type falls into.
func RightsRequestEffectiveCategory(
	request *coredata.RightsRequest,
	details *coredata.RightsRequestPrivacyDetails,
) coredata.RightsRequestPrivacyCategory {
	if details != nil && details.Category != nil {
		return *details.Category
	}

	return coredata.RightsRequestPrivacyCategoryForType(request.RequestType)
}

// RightsRequestEffectiveReceivedOn is the receipt date entered, or else the
// day (in Japan) the request was registered.
func RightsRequestEffectiveReceivedOn(
	request *coredata.RightsRequest,
	details *coredata.RightsRequestPrivacyDetails,
) time.Time {
	if details != nil && details.ReceivedOn != nil {
		return *details.ReceivedOn
	}

	return trainingDateOnly(request.CreatedAt.In(ledgerTimeZone))
}

// RightsRequestEffectiveReceiptChannel is the channel entered, or the web
// form for a request sent through the compliance portal.
func RightsRequestEffectiveReceiptChannel(
	request *coredata.RightsRequest,
	details *coredata.RightsRequestPrivacyDetails,
) *coredata.RightsRequestReceiptChannel {
	if details != nil && details.ReceiptChannel != nil {
		return details.ReceiptChannel
	}

	if request.CompliancePortalID != nil {
		channel := coredata.RightsRequestReceiptChannelWebForm
		return &channel
	}

	return nil
}

// RightsRequestHandlingDays counts the days from receipt to the answer,
// without the wait for missing documents (procedure 14, section 7). It is
// nil until the request is answered.
func RightsRequestHandlingDays(receivedOn time.Time, details *coredata.RightsRequestPrivacyDetails) *int {
	if details == nil || details.AnsweredOn == nil {
		return nil
	}

	days := daysBetween(receivedOn, *details.AnsweredOn)

	if details.SupplementRequestedOn != nil {
		resumedOn := *details.AnsweredOn
		if details.SupplementReceivedOn != nil && details.SupplementReceivedOn.Before(resumedOn) {
			resumedOn = *details.SupplementReceivedOn
		}

		if wait := daysBetween(*details.SupplementRequestedOn, resumedOn); wait > 0 {
			days -= wait
		}
	}

	if days < 0 {
		days = 0
	}

	return &days
}

func daysBetween(from time.Time, to time.Time) int {
	return int(trainingDateOnly(to).Sub(trainingDateOnly(from)).Hours() / 24)
}
