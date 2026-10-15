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

package docgen

import (
	"time"

	"go.probo.inc/probo/pkg/coredata"
)

// Labels holds the fixed texts of the document PDF and its signature page
// in one language. The Japanese terms follow docs/ja/glossary.md and the
// console's ja-JP translations.
type Labels struct {
	Classification     string
	Approver           string
	Approvers          string
	Version            string
	Published          string
	DocumentSignatures string
	Signatory          string
	Status             string
	RequestedDate      string
	SignedDate         string
	// Classifications and States map the stored values to their labels.
	// Values missing from a map are printed as stored.
	Classifications map[Classification]string
	States          map[string]string
	// LongDate and ShortDate are time.Format layouts.
	LongDate  string
	ShortDate string
	// Location is the time zone dates are shown in, or nil to keep the
	// time's own zone.
	Location *time.Location
}

var (
	englishLabels = Labels{
		Classification:     "Classification",
		Approver:           "Approver",
		Approvers:          "Approvers",
		Version:            "Version:",
		Published:          "Published",
		DocumentSignatures: "Document Signatures",
		Signatory:          "Signatory",
		Status:             "Status",
		RequestedDate:      "Requested Date",
		SignedDate:         "Signed Date",
		Classifications:    map[Classification]string{},
		States:             map[string]string{},
		LongDate:           "January 2, 2006",
		ShortDate:          "Jan 2, 2006",
	}

	japaneseLabels = Labels{
		Classification:     "機密区分",
		Approver:           "承認者",
		Approvers:          "承認者",
		Version:            "版",
		Published:          "公開日",
		DocumentSignatures: "文書の署名",
		Signatory:          "署名者",
		Status:             "状態",
		RequestedDate:      "依頼日",
		SignedDate:         "署名日",
		Classifications: map[Classification]string{
			ClassificationPublic:       "公開",
			ClassificationInternal:     "社内限",
			ClassificationConfidential: "機密",
			ClassificationSecret:       "極秘",
		},
		States: map[string]string{
			"REQUESTED": "依頼済み",
			"SIGNED":    "署名済み",
		},
		LongDate:  "2006年1月2日",
		ShortDate: "2006/01/02",
		Location:  time.FixedZone("JST", 9*60*60),
	}
)

// LabelsFor returns the fixed texts for a BCP 47 language tag. Japanese
// ("ja", "ja-JP") gets Japanese texts; any other language gets English.
func LabelsFor(language string) Labels {
	if IsJapanese(language) {
		return japaneseLabels
	}

	return englishLabels
}

// IsJapanese reports whether a BCP 47 language tag is Japanese.
func IsJapanese(language string) bool {
	return language == "ja" || len(language) > 3 && language[:3] == "ja-"
}

// label returns the label of a stored value, or the value itself.
func label[K comparable](labels map[K]string, value K, fallback string) string {
	if l, ok := labels[value]; ok {
		return l
	}

	return fallback
}

// FormatDate formats a time with the given layout, in the labels' time
// zone when there is one.
func (l Labels) FormatDate(t time.Time, layout string) string {
	if l.Location != nil {
		t = t.In(l.Location)
	}

	return t.Format(layout)
}

// LongDateOf and ShortDateOf format a date for the document and signature
// tables. They take a time.Time or a *time.Time; nil gives "".
func (l Labels) LongDateOf(t any) string  { return l.formatAny(t, l.LongDate) }
func (l Labels) ShortDateOf(t any) string { return l.formatAny(t, l.ShortDate) }

func (l Labels) formatAny(t any, layout string) string {
	switch v := t.(type) {
	case time.Time:
		return l.FormatDate(v, layout)
	case *time.Time:
		if v == nil {
			return ""
		}

		return l.FormatDate(*v, layout)
	}

	return ""
}

// BoolLabelIn is BoolLabel in the given language.
func BoolLabelIn(language string, v bool) string {
	if !IsJapanese(language) {
		return BoolLabel(v)
	}

	if v {
		return "はい"
	}

	return "いいえ"
}

// ApplicabilityLabelIn labels whether a control applies. English keeps the
// yes/no of BoolLabel; Japanese uses the standard's terms.
func ApplicabilityLabelIn(language string, applicable bool) string {
	if !IsJapanese(language) {
		return BoolLabel(applicable)
	}

	if applicable {
		return "適用"
	}

	return "適用除外"
}

var japaneseMaturityLabels = map[coredata.ControlMaturityLevel]string{
	coredata.ControlMaturityLevelNone:                  "0 - なし",
	coredata.ControlMaturityLevelInitial:               "1 - 初期",
	coredata.ControlMaturityLevelManaged:               "2 - 管理されている",
	coredata.ControlMaturityLevelDefined:               "3 - 定義されている",
	coredata.ControlMaturityLevelQuantitativelyManaged: "4 - 定量的に管理されている",
	coredata.ControlMaturityLevelOptimizing:            "5 - 最適化されている",
}

// MaturityLabelIn is MaturityLabel in the given language. The Japanese
// labels match the console's.
func MaturityLabelIn(language string, l coredata.ControlMaturityLevel) string {
	if !IsJapanese(language) {
		return MaturityLabel(l)
	}

	return label(japaneseMaturityLabels, l, "未設定")
}
