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

// FindingPrivacyCategory classifies a finding the way the Japanese
// PrivacyMark internal audit procedure does (不適合, 観察事項, 改善の機会).
type FindingPrivacyCategory string

const (
	FindingPrivacyCategoryNonconformity          FindingPrivacyCategory = "NONCONFORMITY"
	FindingPrivacyCategoryObservation            FindingPrivacyCategory = "OBSERVATION"
	FindingPrivacyCategoryImprovementOpportunity FindingPrivacyCategory = "IMPROVEMENT_OPPORTUNITY"
)

var (
	_ fmt.Stringer             = FindingPrivacyCategory("")
	_ encoding.TextMarshaler   = FindingPrivacyCategory("")
	_ encoding.TextUnmarshaler = (*FindingPrivacyCategory)(nil)
)

func FindingPrivacyCategories() []FindingPrivacyCategory {
	return []FindingPrivacyCategory{
		FindingPrivacyCategoryNonconformity,
		FindingPrivacyCategoryObservation,
		FindingPrivacyCategoryImprovementOpportunity,
	}
}

// FindingPrivacyCategoryForKind returns the category a finding of the given
// upstream kind falls into when no category was chosen. An exception has
// none.
func FindingPrivacyCategoryForKind(kind FindingKind) *FindingPrivacyCategory {
	var category FindingPrivacyCategory

	switch kind {
	case FindingKindMajorNonconformity, FindingKindMinorNonconformity:
		category = FindingPrivacyCategoryNonconformity
	case FindingKindObservation:
		category = FindingPrivacyCategoryObservation
	default:
		return nil
	}

	return &category
}

func (v FindingPrivacyCategory) IsValid() bool {
	switch v {
	case
		FindingPrivacyCategoryNonconformity,
		FindingPrivacyCategoryObservation,
		FindingPrivacyCategoryImprovementOpportunity:
		return true
	}

	return false
}

func (v FindingPrivacyCategory) String() string {
	return string(v)
}

func (v FindingPrivacyCategory) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *FindingPrivacyCategory) UnmarshalText(text []byte) error {
	val := FindingPrivacyCategory(text)
	if !val.IsValid() {
		return fmt.Errorf("invalid FindingPrivacyCategory value: %q", string(text))
	}

	*v = val

	return nil
}
