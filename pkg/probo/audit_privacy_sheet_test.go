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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/validator"
	"go.probo.inc/probo/pkg/xlsxexport"
)

func auditReportTestFinding(referenceID string, kind coredata.FindingKind) *coredata.Finding {
	description := referenceID + " の内容"

	return &coredata.Finding{
		ID:          gid.New(gid.NewTenantID(), coredata.FindingEntityType),
		ReferenceID: referenceID,
		Kind:        kind,
		Description: &description,
		Status:      coredata.FindingStatusOpen,
	}
}

func auditReportTestCategory(c coredata.FindingPrivacyCategory) *coredata.FindingPrivacyCategory {
	return &c
}

func TestFindingPrivacyCategoryLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		kind     coredata.FindingKind
		category *coredata.FindingPrivacyCategory
		want     string
	}{
		{"major from kind", coredata.FindingKindMajorNonconformity, nil, "不適合（重大）"},
		{"minor from kind", coredata.FindingKindMinorNonconformity, nil, "不適合"},
		{"observation from kind", coredata.FindingKindObservation, nil, "観察事項"},
		{"exception has none", coredata.FindingKindException, nil, ""},
		{"chosen improvement", coredata.FindingKindObservation, auditReportTestCategory(coredata.FindingPrivacyCategoryImprovementOpportunity), "改善の機会"},
		{"chosen over exception", coredata.FindingKindException, auditReportTestCategory(coredata.FindingPrivacyCategoryObservation), "観察事項"},
		{"chosen nonconformity keeps major", coredata.FindingKindMajorNonconformity, auditReportTestCategory(coredata.FindingPrivacyCategoryNonconformity), "不適合（重大）"},
		{"major reclassified", coredata.FindingKindMajorNonconformity, auditReportTestCategory(coredata.FindingPrivacyCategoryObservation), "観察事項"},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Parallel()

				finding := auditReportTestFinding("FND-1", tt.kind)
				details := &coredata.FindingPrivacyDetails{Category: tt.category}

				assert.Equal(t, tt.want, findingPrivacyCategoryLabel(finding, details))
			},
		)
	}
}

func auditReportTestData() auditReportData {
	name := "2026 年度 内部監査"
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	due := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	rootCause := "手順の周知不足"

	major := auditReportTestFinding("FND-1", coredata.FindingKindMajorNonconformity)
	major.DueDate = &due
	major.RootCause = &rootCause
	observation := auditReportTestFinding("FND-2", coredata.FindingKindObservation)
	observation.Status = coredata.FindingStatusClosed
	exception := auditReportTestFinding("FND-3", coredata.FindingKindException)

	return auditReportData{
		OrganizationName: "サンプル株式会社",
		Audit: &coredata.Audit{
			Name:           &name,
			AuditStartDate: &start,
			AuditEndDate:   &end,
		},
		FrameworkName: "JIS Q 15001",
		Details: &coredata.AuditPrivacyDetails{
			Scope:              "J.1〜J.11",
			Criteria:           "JIS Q 15001:2023、当社の規程",
			AuditeeDepartments: []string{"営業部", "管理部"},
			ExternalAuditors:   []string{"外部 太郎"},
			OverallOpinion:     "概ね適合している。",
		},
		AuditorNames: []string{"監査 花子"},
		Findings: []auditReportFinding{
			{Finding: major, OwnerName: "担当 一郎", Details: &coredata.FindingPrivacyDetails{Basis: "J.7"}},
			{Finding: observation},
			{Finding: exception},
		},
	}
}

func TestBuildAuditReportSheets_Summary(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 15, 9, 0, 0, 0, ledgerTimeZone)
	sheets := buildAuditReportSheets(auditReportTestData(), now)

	require.Len(t, sheets, 2)
	summary := sheets[0]
	assert.Equal(t, "監査概要", summary.Name)
	assert.Equal(t, "内部監査報告書", summary.Title)
	assert.Equal(t, "サンプル株式会社　出力日：2026-10-15", summary.Subtitle)

	values := make(map[string]string, len(summary.Rows))
	for _, row := range summary.Rows {
		values[row[0]] = row[1]
	}

	assert.Equal(t, "2026 年度 内部監査", values["監査の名前"])
	assert.Equal(t, "2026-09-01〜2026-09-03", values["監査日"])
	assert.Equal(t, "J.1〜J.11", values["監査の範囲"])
	assert.Equal(t, "監査 花子、外部 太郎（外部）", values["監査員"])
	assert.Equal(t, "営業部、管理部", values["被監査部門"])
	assert.Equal(t, "計 3 件（不適合（重大） 1 件、不適合 0 件、観察事項 1 件、改善の機会 0 件、その他 1 件）", values["指摘事項の件数"])
	assert.Equal(t, "概ね適合している。", values["総合的な所見"])

	_, err := xlsxexport.BuildWorkbook(sheets...)
	require.NoError(t, err)
}

func TestBuildAuditReportSheets_Findings(t *testing.T) {
	t.Parallel()

	sheets := buildAuditReportSheets(auditReportTestData(), time.Now())
	rows := sheets[1].Rows

	require.Len(t, rows, 3)
	assert.Equal(t, []string{"1", "FND-1", "不適合（重大）", "FND-1 の内容", "J.7", "手順の周知不足", "", "担当 一郎", "2026-10-31", "未完了", ""}, rows[0])
	assert.Equal(t, "観察事項", rows[1][2])
	assert.Equal(t, "完了", rows[1][9])
	assert.Equal(t, "", rows[2][2])
}

func TestBuildAuditReportSheets_Empty(t *testing.T) {
	t.Parallel()

	data := auditReportData{OrganizationName: "サンプル株式会社", Audit: &coredata.Audit{}}
	sheets := buildAuditReportSheets(data, time.Now())

	values := make(map[string]string)
	for _, row := range sheets[0].Rows {
		values[row[0]] = row[1]
	}
	assert.Equal(t, "", values["監査日"])
	assert.Equal(t, "", values["監査員"])
	assert.Equal(t, "計 0 件（不適合（重大） 0 件、不適合 0 件、観察事項 0 件、改善の機会 0 件）", values["指摘事項の件数"])

	require.Len(t, sheets[1].Rows, 1)
	assert.Equal(t, "（指摘事項なし）", sheets[1].Rows[0][2])

	_, err := xlsxexport.BuildWorkbook(sheets...)
	require.NoError(t, err)
}

func TestFormatDateRange(t *testing.T) {
	t.Parallel()

	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	later := day.AddDate(0, 0, 2)

	assert.Equal(t, "", formatDateRange(nil, nil))
	assert.Equal(t, "2026-09-01", formatDateRange(&day, nil))
	assert.Equal(t, "2026-09-01", formatDateRange(&day, &day))
	assert.Equal(t, "〜2026-09-03", formatDateRange(nil, &later))
	assert.Equal(t, "2026-09-01〜2026-09-03", formatDateRange(&day, &later))
}

func TestFindIndependenceConflicts(t *testing.T) {
	t.Parallel()

	sales := " 営業部 "
	admin := "Admin"
	auditors := coredata.MembershipProfiles{
		{FullName: "A", Department: &sales},
		{FullName: "B", Department: &admin},
		{FullName: "C"},
	}

	conflicts := findIndependenceConflicts(auditors, []string{"営業部", "admin"})

	require.Len(t, conflicts, 2)
	assert.Equal(t, "A", conflicts[0].Auditor.FullName)
	assert.Equal(t, "営業部", conflicts[0].Department)
	assert.Equal(t, "B", conflicts[1].Auditor.FullName)
	assert.Equal(t, "admin", conflicts[1].Department)

	assert.Empty(t, findIndependenceConflicts(auditors, []string{"開発部"}))
}

func TestUpdateAuditPrivacyDetailsRequest_Validate(t *testing.T) {
	t.Parallel()

	valid := func() UpdateAuditPrivacyDetailsRequest {
		return UpdateAuditPrivacyDetailsRequest{
			AuditID:            gid.New(gid.NewTenantID(), coredata.AuditEntityType),
			AuditeeDepartments: []string{"営業部"},
			AuditorIDs:         []gid.GID{gid.New(gid.NewTenantID(), coredata.MembershipProfileEntityType)},
		}
	}

	r := valid()
	assert.NoError(t, r.Validate())

	r = valid()
	r.AuditeeDepartments = []string{"営業部", "営業部"}
	assert.Error(t, r.Validate())

	r = valid()
	r.AuditeeDepartments = []string{""}
	assert.Error(t, r.Validate())

	r = valid()
	r.ExternalAuditors = make([]string, auditPrivacyMaxExternalAuditors+1)
	for i := range r.ExternalAuditors {
		r.ExternalAuditors[i] = string(rune('a' + i))
	}
	assert.Error(t, r.Validate())

	r = valid()
	r.AuditorIDs = []gid.GID{gid.New(gid.NewTenantID(), coredata.RiskEntityType)}
	assert.Error(t, r.Validate())
}

func TestMaxItems(t *testing.T) {
	t.Parallel()

	ids := []gid.GID{
		gid.New(gid.NewTenantID(), coredata.RiskEntityType),
		gid.New(gid.NewTenantID(), coredata.RiskEntityType),
	}

	v := validatorForTest(&ids, maxItems(1))
	assert.Error(t, v)

	v = validatorForTest(&ids, maxItems(2))
	assert.NoError(t, v)

	var none *[]gid.GID
	assert.NoError(t, validatorForTest(none, maxItems(0)))
}

func validatorForTest(value any, check validator.ValidatorFunc) error {
	v := validator.New()
	v.Check(value, "items", check)

	return v.Error()
}
