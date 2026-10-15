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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/docgen"
	"go.probo.inc/probo/pkg/gid"
)

func soaTestRow(language string, applicable bool) docgen.SOARow {
	return docgen.SOARow{
		FrameworkName:        "JIS Q 15001",
		ControlSection:       "A.1",
		ControlName:          "個人情報保護方針",
		Applicability:        docgen.ApplicabilityLabelIn(language, applicable),
		Justification:        "-",
		MaturityLevel:        docgen.MaturityLabelIn(language, coredata.ControlMaturityLevelDefined),
		NotImplJustification: "-",
		Regulatory:           docgen.BoolLabelIn(language, true),
		Contractual:          docgen.BoolLabelIn(language, false),
		BestPractice:         docgen.BoolLabelIn(language, true),
		RiskAssessment:       docgen.BoolLabelIn(language, false),
	}
}

func TestBuildStatementOfApplicabilityDocumentJapanese(t *testing.T) {
	t.Parallel()

	out, err := BuildStatementOfApplicabilityDocument(docgen.StatementOfApplicabilityData{
		Language: "ja",
		Rows:     []docgen.SOARow{soaTestRow("ja", true), soaTestRow("ja", false)},
	})
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(out)))

	for _, want := range []string{"1. 目的", "2. 管理策", "3. 用語の説明", "適用除外の根拠", "成熟度", "適用除外", "3 - 定義されている", "はい", "いいえ"} {
		assert.Contains(t, out, want)
	}
	for _, unwanted := range []string{"Purpose", "Justification", "Definitions", "Yes", "Defined"} {
		assert.NotContains(t, out, unwanted)
	}
}

func TestBuildStatementOfApplicabilityDocumentEnglishUnchanged(t *testing.T) {
	t.Parallel()

	out, err := BuildStatementOfApplicabilityDocument(docgen.StatementOfApplicabilityData{
		Rows: []docgen.SOARow{soaTestRow("en", true)},
	})
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(out)))

	for _, want := range []string{"1. Purpose", "Justification for non-applicability", "3 - Defined", "Yes"} {
		assert.Contains(t, out, want)
	}
}

func TestStatementOfApplicabilityLanguage(t *testing.T) {
	t.Parallel()

	tenant := gid.NewTenantID()
	frameworkID := gid.New(tenant, coredata.FrameworkEntityType)
	controlID := gid.New(tenant, coredata.ControlEntityType)
	statements := []*coredata.ApplicabilityStatement{{ControlID: controlID}}
	frameworks := map[gid.GID]*coredata.Framework{frameworkID: {ID: frameworkID, Name: "JIS Q 15001"}}

	japanese := map[gid.GID]*coredata.Control{controlID: {ID: controlID, FrameworkID: frameworkID, Name: "個人情報保護方針の策定"}}
	english := map[gid.GID]*coredata.Control{controlID: {ID: controlID, FrameworkID: frameworkID, Name: "Information security policy"}}

	assert.Equal(t, "ja", statementOfApplicabilityLanguage("SoA 2026", statements, japanese, frameworks))
	assert.Equal(t, "en", statementOfApplicabilityLanguage("SoA 2026", statements, english, frameworks))
	assert.Equal(t, "ja", statementOfApplicabilityLanguage("Pマークの適用宣言書", nil, nil, nil))
}
