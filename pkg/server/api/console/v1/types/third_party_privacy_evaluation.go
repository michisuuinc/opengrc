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

// ThirdPartyPrivacyEvaluation is the outsourcing evaluation of a
// third-party risk assessment. The evaluator and approver are resolved from
// their IDs, since reading them needs the member list permission.
type ThirdPartyPrivacyEvaluation struct {
	RiskAssessmentID gid.GID                                 `json:"-"`
	EvaluatorID      *gid.GID                                `json:"-"`
	ApproverID       *gid.GID                                `json:"-"`
	Kind             *coredata.ThirdPartyEvaluationKind      `json:"kind,omitempty"`
	AdHocReason      string                                  `json:"adHocReason"`
	EntrustedWork    string                                  `json:"entrustedWork"`
	PersonalData     string                                  `json:"personalData"`
	StorageCountries string                                  `json:"storageCountries"`
	Evaluator        *Profile                                `json:"evaluator,omitempty"`
	EvaluatedOn      *string                                 `json:"evaluatedOn,omitempty"`
	Answers          []*ThirdPartyEvaluationAnswer           `json:"answers"`
	Judgement        *coredata.ThirdPartyEvaluationJudgement `json:"judgement,omitempty"`
	Conditions       string                                  `json:"conditions"`
	Approver         *Profile                                `json:"approver,omitempty"`
	ApprovedOn       *string                                 `json:"approvedOn,omitempty"`
	UpdatedAt        *time.Time                              `json:"updatedAt,omitempty"`
}
