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
	"go.probo.inc/probo/pkg/coredata"
)

func NewAuditPrivacyDetails(d *coredata.AuditPrivacyDetails) *AuditPrivacyDetails {
	object := &AuditPrivacyDetails{
		Scope:              d.Scope,
		Criteria:           d.Criteria,
		AuditeeDepartments: d.AuditeeDepartments,
		ExternalAuditors:   d.ExternalAuditors,
		OverallOpinion:     d.OverallOpinion,
	}

	if object.AuditeeDepartments == nil {
		object.AuditeeDepartments = []string{}
	}

	if object.ExternalAuditors == nil {
		object.ExternalAuditors = []string{}
	}

	if !d.UpdatedAt.IsZero() {
		object.UpdatedAt = &d.UpdatedAt
	}

	return object
}

// NewFindingPrivacyDetails also gives the category that applies: the chosen
// one, or the one the finding kind falls into.
func NewFindingPrivacyDetails(
	d *coredata.FindingPrivacyDetails,
	kind coredata.FindingKind,
) *FindingPrivacyDetails {
	object := &FindingPrivacyDetails{
		Category:          d.Category,
		EffectiveCategory: d.Category,
		Basis:             d.Basis,
	}

	if object.EffectiveCategory == nil {
		object.EffectiveCategory = coredata.FindingPrivacyCategoryForKind(kind)
	}

	if !d.UpdatedAt.IsZero() {
		object.UpdatedAt = &d.UpdatedAt
	}

	return object
}
