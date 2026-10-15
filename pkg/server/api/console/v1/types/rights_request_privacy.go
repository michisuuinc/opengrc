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
package types

import (
	"time"

	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
)

// RightsRequestPrivacyDetails is the record of a request from an
// individual. The handler and approver are resolved from their IDs, since
// reading them needs the member list permission.
type RightsRequestPrivacyDetails struct {
	RightsRequestID         gid.GID                                   `json:"-"`
	HandlerID               *gid.GID                                  `json:"-"`
	ApproverID              *gid.GID                                  `json:"-"`
	FindingID               *gid.GID                                  `json:"-"`
	Category                *coredata.RightsRequestPrivacyCategory    `json:"category,omitempty"`
	EffectiveCategory       coredata.RightsRequestPrivacyCategory     `json:"effectiveCategory"`
	ReceivedOn              *string                                   `json:"receivedOn,omitempty"`
	EffectiveReceivedOn     string                                    `json:"effectiveReceivedOn"`
	ReceiptChannel          *coredata.RightsRequestReceiptChannel     `json:"receiptChannel,omitempty"`
	EffectiveReceiptChannel *coredata.RightsRequestReceiptChannel     `json:"effectiveReceiptChannel,omitempty"`
	ByAgent                 bool                                      `json:"byAgent"`
	TargetDepartment        string                                    `json:"targetDepartment"`
	Handler                 *Profile                                  `json:"handler,omitempty"`
	VerificationMethod      *coredata.RightsRequestVerificationMethod `json:"verificationMethod,omitempty"`
	VerifiedOn              *string                                   `json:"verifiedOn,omitempty"`
	VerificationNote        string                                    `json:"verificationNote"`
	SupplementRequestedOn   *string                                   `json:"supplementRequestedOn,omitempty"`
	SupplementReceivedOn    *string                                   `json:"supplementReceivedOn,omitempty"`
	Decision                *coredata.RightsRequestDecision           `json:"decision,omitempty"`
	DecisionReason          string                                    `json:"decisionReason"`
	AnswerMethod            *coredata.RightsRequestAnswerMethod       `json:"answerMethod,omitempty"`
	AnsweredOn              *string                                   `json:"answeredOn,omitempty"`
	Approver                *Profile                                  `json:"approver,omitempty"`
	FirstRespondedOn        *string                                   `json:"firstRespondedOn,omitempty"`
	Finding                 *Finding                                  `json:"finding,omitempty"`
	HandlingDays            *int                                      `json:"handlingDays,omitempty"`
	UpdatedAt               *time.Time                                `json:"updatedAt,omitempty"`
}
