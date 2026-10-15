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

// The enums of the Japanese PrivacyMark record of requests from individuals
// (本人からの請求の記録).

// RightsRequestPrivacyCategory is the kind of a request under the Japanese
// Act on the Protection of Personal Information, as the PrivacyMark rights
// request procedure lists them, plus complaints and consultations (苦情・相談).
type RightsRequestPrivacyCategory string

const (
	RightsRequestPrivacyCategoryPurposeNotification        RightsRequestPrivacyCategory = "PURPOSE_NOTIFICATION"
	RightsRequestPrivacyCategoryDisclosure                 RightsRequestPrivacyCategory = "DISCLOSURE"
	RightsRequestPrivacyCategoryCorrection                 RightsRequestPrivacyCategory = "CORRECTION"
	RightsRequestPrivacyCategorySuspension                 RightsRequestPrivacyCategory = "SUSPENSION"
	RightsRequestPrivacyCategoryThirdPartyProvisionStop    RightsRequestPrivacyCategory = "THIRD_PARTY_PROVISION_STOP"
	RightsRequestPrivacyCategoryThirdPartyRecordDisclosure RightsRequestPrivacyCategory = "THIRD_PARTY_RECORD_DISCLOSURE"
	RightsRequestPrivacyCategoryComplaint                  RightsRequestPrivacyCategory = "COMPLAINT"
	RightsRequestPrivacyCategoryConsultation               RightsRequestPrivacyCategory = "CONSULTATION"
)

var (
	_ fmt.Stringer             = RightsRequestPrivacyCategory("")
	_ encoding.TextMarshaler   = RightsRequestPrivacyCategory("")
	_ encoding.TextUnmarshaler = (*RightsRequestPrivacyCategory)(nil)
)

func RightsRequestPrivacyCategories() []RightsRequestPrivacyCategory {
	return []RightsRequestPrivacyCategory{
		RightsRequestPrivacyCategoryPurposeNotification,
		RightsRequestPrivacyCategoryDisclosure,
		RightsRequestPrivacyCategoryCorrection,
		RightsRequestPrivacyCategorySuspension,
		RightsRequestPrivacyCategoryThirdPartyProvisionStop,
		RightsRequestPrivacyCategoryThirdPartyRecordDisclosure,
		RightsRequestPrivacyCategoryComplaint,
		RightsRequestPrivacyCategoryConsultation,
	}
}

func (v RightsRequestPrivacyCategory) IsValid() bool {
	for _, valid := range RightsRequestPrivacyCategories() {
		if v == valid {
			return true
		}
	}

	return false
}

func (v RightsRequestPrivacyCategory) String() string {
	return string(v)
}

func (v RightsRequestPrivacyCategory) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *RightsRequestPrivacyCategory) UnmarshalText(text []byte) error {
	val := RightsRequestPrivacyCategory(text)
	if !val.IsValid() {
		return fmt.Errorf("invalid RightsRequestPrivacyCategory value: %q", string(text))
	}

	*v = val

	return nil
}

// RightsRequestReceiptChannel is how a request was received.
type RightsRequestReceiptChannel string

const (
	RightsRequestReceiptChannelPostal   RightsRequestReceiptChannel = "POSTAL"
	RightsRequestReceiptChannelEmail    RightsRequestReceiptChannel = "EMAIL"
	RightsRequestReceiptChannelWebForm  RightsRequestReceiptChannel = "WEB_FORM"
	RightsRequestReceiptChannelPhone    RightsRequestReceiptChannel = "PHONE"
	RightsRequestReceiptChannelInPerson RightsRequestReceiptChannel = "IN_PERSON"
	RightsRequestReceiptChannelOther    RightsRequestReceiptChannel = "OTHER"
)

var (
	_ fmt.Stringer             = RightsRequestReceiptChannel("")
	_ encoding.TextMarshaler   = RightsRequestReceiptChannel("")
	_ encoding.TextUnmarshaler = (*RightsRequestReceiptChannel)(nil)
)

func RightsRequestReceiptChannels() []RightsRequestReceiptChannel {
	return []RightsRequestReceiptChannel{
		RightsRequestReceiptChannelPostal,
		RightsRequestReceiptChannelEmail,
		RightsRequestReceiptChannelWebForm,
		RightsRequestReceiptChannelPhone,
		RightsRequestReceiptChannelInPerson,
		RightsRequestReceiptChannelOther,
	}
}

func (v RightsRequestReceiptChannel) IsValid() bool {
	for _, valid := range RightsRequestReceiptChannels() {
		if v == valid {
			return true
		}
	}

	return false
}

func (v RightsRequestReceiptChannel) String() string {
	return string(v)
}

func (v RightsRequestReceiptChannel) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *RightsRequestReceiptChannel) UnmarshalText(text []byte) error {
	val := RightsRequestReceiptChannel(text)
	if !val.IsValid() {
		return fmt.Errorf("invalid RightsRequestReceiptChannel value: %q", string(text))
	}

	*v = val

	return nil
}

// RightsRequestVerificationMethod is how the requester was verified to be the
// individual: identity documents, the individual's account or registered
// information, another way, or no verification (for complaints and
// consultations).
type RightsRequestVerificationMethod string

const (
	RightsRequestVerificationMethodIDDocument  RightsRequestVerificationMethod = "ID_DOCUMENT"
	RightsRequestVerificationMethodAccount     RightsRequestVerificationMethod = "ACCOUNT"
	RightsRequestVerificationMethodOther       RightsRequestVerificationMethod = "OTHER"
	RightsRequestVerificationMethodNotRequired RightsRequestVerificationMethod = "NOT_REQUIRED"
)

var (
	_ fmt.Stringer             = RightsRequestVerificationMethod("")
	_ encoding.TextMarshaler   = RightsRequestVerificationMethod("")
	_ encoding.TextUnmarshaler = (*RightsRequestVerificationMethod)(nil)
)

func RightsRequestVerificationMethods() []RightsRequestVerificationMethod {
	return []RightsRequestVerificationMethod{
		RightsRequestVerificationMethodIDDocument,
		RightsRequestVerificationMethodAccount,
		RightsRequestVerificationMethodOther,
		RightsRequestVerificationMethodNotRequired,
	}
}

func (v RightsRequestVerificationMethod) IsValid() bool {
	for _, valid := range RightsRequestVerificationMethods() {
		if v == valid {
			return true
		}
	}

	return false
}

func (v RightsRequestVerificationMethod) String() string {
	return string(v)
}

func (v RightsRequestVerificationMethod) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *RightsRequestVerificationMethod) UnmarshalText(text []byte) error {
	val := RightsRequestVerificationMethod(text)
	if !val.IsValid() {
		return fmt.Errorf("invalid RightsRequestVerificationMethod value: %q", string(text))
	}

	*v = val

	return nil
}

// RightsRequestDecision is how a request was answered: as requested, partly
// (some information not disclosed or some action not taken) or not at all.
type RightsRequestDecision string

const (
	RightsRequestDecisionFulfilled          RightsRequestDecision = "FULFILLED"
	RightsRequestDecisionPartiallyFulfilled RightsRequestDecision = "PARTIALLY_FULFILLED"
	RightsRequestDecisionNotFulfilled       RightsRequestDecision = "NOT_FULFILLED"
)

var (
	_ fmt.Stringer             = RightsRequestDecision("")
	_ encoding.TextMarshaler   = RightsRequestDecision("")
	_ encoding.TextUnmarshaler = (*RightsRequestDecision)(nil)
)

func RightsRequestDecisions() []RightsRequestDecision {
	return []RightsRequestDecision{
		RightsRequestDecisionFulfilled,
		RightsRequestDecisionPartiallyFulfilled,
		RightsRequestDecisionNotFulfilled,
	}
}

func (v RightsRequestDecision) IsValid() bool {
	for _, valid := range RightsRequestDecisions() {
		if v == valid {
			return true
		}
	}

	return false
}

func (v RightsRequestDecision) String() string {
	return string(v)
}

func (v RightsRequestDecision) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *RightsRequestDecision) UnmarshalText(text []byte) error {
	val := RightsRequestDecision(text)
	if !val.IsValid() {
		return fmt.Errorf("invalid RightsRequestDecision value: %q", string(text))
	}

	*v = val

	return nil
}

// RightsRequestAnswerMethod is how the answer was sent.
type RightsRequestAnswerMethod string

const (
	RightsRequestAnswerMethodPostal           RightsRequestAnswerMethod = "POSTAL"
	RightsRequestAnswerMethodEmail            RightsRequestAnswerMethod = "EMAIL"
	RightsRequestAnswerMethodElectronicRecord RightsRequestAnswerMethod = "ELECTRONIC_RECORD"
	RightsRequestAnswerMethodPhone            RightsRequestAnswerMethod = "PHONE"
	RightsRequestAnswerMethodOther            RightsRequestAnswerMethod = "OTHER"
)

var (
	_ fmt.Stringer             = RightsRequestAnswerMethod("")
	_ encoding.TextMarshaler   = RightsRequestAnswerMethod("")
	_ encoding.TextUnmarshaler = (*RightsRequestAnswerMethod)(nil)
)

func RightsRequestAnswerMethods() []RightsRequestAnswerMethod {
	return []RightsRequestAnswerMethod{
		RightsRequestAnswerMethodPostal,
		RightsRequestAnswerMethodEmail,
		RightsRequestAnswerMethodElectronicRecord,
		RightsRequestAnswerMethodPhone,
		RightsRequestAnswerMethodOther,
	}
}

func (v RightsRequestAnswerMethod) IsValid() bool {
	for _, valid := range RightsRequestAnswerMethods() {
		if v == valid {
			return true
		}
	}

	return false
}

func (v RightsRequestAnswerMethod) String() string {
	return string(v)
}

func (v RightsRequestAnswerMethod) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *RightsRequestAnswerMethod) UnmarshalText(text []byte) error {
	val := RightsRequestAnswerMethod(text)
	if !val.IsValid() {
		return fmt.Errorf("invalid RightsRequestAnswerMethod value: %q", string(text))
	}

	*v = val

	return nil
}

// RightsRequestPrivacyCategoryForType returns the category a request of the
// given upstream type falls into when no category was chosen. Erasure and
// objection fall into suspension and erasure (利用停止・消去), the closest
// Japanese request.
func RightsRequestPrivacyCategoryForType(t RightsRequestType) RightsRequestPrivacyCategory {
	switch t {
	case RightsRequestTypeRectification:
		return RightsRequestPrivacyCategoryCorrection
	case RightsRequestTypeDeletion, RightsRequestTypeObjection:
		return RightsRequestPrivacyCategorySuspension
	case RightsRequestTypeComplaint:
		return RightsRequestPrivacyCategoryComplaint
	default:
		return RightsRequestPrivacyCategoryDisclosure
	}
}
