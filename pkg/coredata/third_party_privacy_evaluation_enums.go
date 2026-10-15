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
	"encoding"
	"fmt"
)

// The enums of the Japanese PrivacyMark outsourcing evaluation (委託先評価表).

// ThirdPartyEvaluationKind is why a third party was evaluated: before
// outsourcing to it (新規), the yearly review (定期) or an ad hoc review (臨時).
type ThirdPartyEvaluationKind string

const (
	ThirdPartyEvaluationKindNew      ThirdPartyEvaluationKind = "NEW"
	ThirdPartyEvaluationKindPeriodic ThirdPartyEvaluationKind = "PERIODIC"
	ThirdPartyEvaluationKindAdHoc    ThirdPartyEvaluationKind = "AD_HOC"
)

var (
	_ fmt.Stringer             = ThirdPartyEvaluationKind("")
	_ encoding.TextMarshaler   = ThirdPartyEvaluationKind("")
	_ encoding.TextUnmarshaler = (*ThirdPartyEvaluationKind)(nil)
)

func ThirdPartyEvaluationKinds() []ThirdPartyEvaluationKind {
	return []ThirdPartyEvaluationKind{
		ThirdPartyEvaluationKindNew,
		ThirdPartyEvaluationKindPeriodic,
		ThirdPartyEvaluationKindAdHoc,
	}
}

func (v ThirdPartyEvaluationKind) IsValid() bool {
	for _, valid := range ThirdPartyEvaluationKinds() {
		if v == valid {
			return true
		}
	}

	return false
}

func (v ThirdPartyEvaluationKind) String() string {
	return string(v)
}

func (v ThirdPartyEvaluationKind) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *ThirdPartyEvaluationKind) UnmarshalText(text []byte) error {
	val := ThirdPartyEvaluationKind(text)
	if !val.IsValid() {
		return fmt.Errorf("invalid ThirdPartyEvaluationKind value: %q", string(text))
	}

	*v = val

	return nil
}

// ThirdPartyEvaluationJudgement is the overall judgement of an outsourcing
// evaluation: outsourcing allowed (委託可), allowed on conditions (条件付きで委託可) or
// not allowed (委託不可).
type ThirdPartyEvaluationJudgement string

const (
	ThirdPartyEvaluationJudgementApproved              ThirdPartyEvaluationJudgement = "APPROVED"
	ThirdPartyEvaluationJudgementConditionallyApproved ThirdPartyEvaluationJudgement = "CONDITIONALLY_APPROVED"
	ThirdPartyEvaluationJudgementRejected              ThirdPartyEvaluationJudgement = "REJECTED"
)

var (
	_ fmt.Stringer             = ThirdPartyEvaluationJudgement("")
	_ encoding.TextMarshaler   = ThirdPartyEvaluationJudgement("")
	_ encoding.TextUnmarshaler = (*ThirdPartyEvaluationJudgement)(nil)
)

func ThirdPartyEvaluationJudgements() []ThirdPartyEvaluationJudgement {
	return []ThirdPartyEvaluationJudgement{
		ThirdPartyEvaluationJudgementApproved,
		ThirdPartyEvaluationJudgementConditionallyApproved,
		ThirdPartyEvaluationJudgementRejected,
	}
}

func (v ThirdPartyEvaluationJudgement) IsValid() bool {
	for _, valid := range ThirdPartyEvaluationJudgements() {
		if v == valid {
			return true
		}
	}

	return false
}

func (v ThirdPartyEvaluationJudgement) String() string {
	return string(v)
}

func (v ThirdPartyEvaluationJudgement) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *ThirdPartyEvaluationJudgement) UnmarshalText(text []byte) error {
	val := ThirdPartyEvaluationJudgement(text)
	if !val.IsValid() {
		return fmt.Errorf("invalid ThirdPartyEvaluationJudgement value: %q", string(text))
	}

	*v = val

	return nil
}

// ThirdPartyEvaluationResult is the answer to one checklist item: meets the
// criterion (適), does not (否) or does not apply to this outsourcing (該当なし).
type ThirdPartyEvaluationResult string

const (
	ThirdPartyEvaluationResultSuitable      ThirdPartyEvaluationResult = "SUITABLE"
	ThirdPartyEvaluationResultUnsuitable    ThirdPartyEvaluationResult = "UNSUITABLE"
	ThirdPartyEvaluationResultNotApplicable ThirdPartyEvaluationResult = "NOT_APPLICABLE"
)

var (
	_ fmt.Stringer             = ThirdPartyEvaluationResult("")
	_ encoding.TextMarshaler   = ThirdPartyEvaluationResult("")
	_ encoding.TextUnmarshaler = (*ThirdPartyEvaluationResult)(nil)
)

func ThirdPartyEvaluationResults() []ThirdPartyEvaluationResult {
	return []ThirdPartyEvaluationResult{
		ThirdPartyEvaluationResultSuitable,
		ThirdPartyEvaluationResultUnsuitable,
		ThirdPartyEvaluationResultNotApplicable,
	}
}

func (v ThirdPartyEvaluationResult) IsValid() bool {
	for _, valid := range ThirdPartyEvaluationResults() {
		if v == valid {
			return true
		}
	}

	return false
}

func (v ThirdPartyEvaluationResult) String() string {
	return string(v)
}

func (v ThirdPartyEvaluationResult) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *ThirdPartyEvaluationResult) UnmarshalText(text []byte) error {
	val := ThirdPartyEvaluationResult(text)
	if !val.IsValid() {
		return fmt.Errorf("invalid ThirdPartyEvaluationResult value: %q", string(text))
	}

	*v = val

	return nil
}

// ThirdPartyEvaluationItem is one item of the outsourcing evaluation
// checklist (委託先評価チェックリスト), in the checklist order.
type ThirdPartyEvaluationItem string

const (
	ThirdPartyEvaluationItemCertification       ThirdPartyEvaluationItem = "CERTIFICATION"
	ThirdPartyEvaluationItemManagementStructure ThirdPartyEvaluationItem = "MANAGEMENT_STRUCTURE"
	ThirdPartyEvaluationItemRules               ThirdPartyEvaluationItem = "RULES"
	ThirdPartyEvaluationItemEmployeeTraining    ThirdPartyEvaluationItem = "EMPLOYEE_TRAINING"
	ThirdPartyEvaluationItemPhysicalSecurity    ThirdPartyEvaluationItem = "PHYSICAL_SECURITY"
	ThirdPartyEvaluationItemAccessControl       ThirdPartyEvaluationItem = "ACCESS_CONTROL"
	ThirdPartyEvaluationItemTechnicalSecurity   ThirdPartyEvaluationItem = "TECHNICAL_SECURITY"
	ThirdPartyEvaluationItemSubcontracting      ThirdPartyEvaluationItem = "SUBCONTRACTING"
	ThirdPartyEvaluationItemIncidentResponse    ThirdPartyEvaluationItem = "INCIDENT_RESPONSE"
	ThirdPartyEvaluationItemPastIncidents       ThirdPartyEvaluationItem = "PAST_INCIDENTS"
	ThirdPartyEvaluationItemReturnAndDeletion   ThirdPartyEvaluationItem = "RETURN_AND_DELETION"
	ThirdPartyEvaluationItemForeignStorage      ThirdPartyEvaluationItem = "FOREIGN_STORAGE"
	ThirdPartyEvaluationItemContract            ThirdPartyEvaluationItem = "CONTRACT"
)

var (
	_ fmt.Stringer             = ThirdPartyEvaluationItem("")
	_ encoding.TextMarshaler   = ThirdPartyEvaluationItem("")
	_ encoding.TextUnmarshaler = (*ThirdPartyEvaluationItem)(nil)
)

func ThirdPartyEvaluationItems() []ThirdPartyEvaluationItem {
	return []ThirdPartyEvaluationItem{
		ThirdPartyEvaluationItemCertification,
		ThirdPartyEvaluationItemManagementStructure,
		ThirdPartyEvaluationItemRules,
		ThirdPartyEvaluationItemEmployeeTraining,
		ThirdPartyEvaluationItemPhysicalSecurity,
		ThirdPartyEvaluationItemAccessControl,
		ThirdPartyEvaluationItemTechnicalSecurity,
		ThirdPartyEvaluationItemSubcontracting,
		ThirdPartyEvaluationItemIncidentResponse,
		ThirdPartyEvaluationItemPastIncidents,
		ThirdPartyEvaluationItemReturnAndDeletion,
		ThirdPartyEvaluationItemForeignStorage,
		ThirdPartyEvaluationItemContract,
	}
}

func (v ThirdPartyEvaluationItem) IsValid() bool {
	for _, valid := range ThirdPartyEvaluationItems() {
		if v == valid {
			return true
		}
	}

	return false
}

func (v ThirdPartyEvaluationItem) String() string {
	return string(v)
}

func (v ThirdPartyEvaluationItem) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *ThirdPartyEvaluationItem) UnmarshalText(text []byte) error {
	val := ThirdPartyEvaluationItem(text)
	if !val.IsValid() {
		return fmt.Errorf("invalid ThirdPartyEvaluationItem value: %q", string(text))
	}

	*v = val

	return nil
}
