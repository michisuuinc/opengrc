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
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/xlsxexport"
)

const (
	auditReportSummarySheetName  = "監査概要"
	auditReportFindingsSheetName = "指摘事項一覧"
	auditReportTitle             = "内部監査報告書"
	auditReportNoFindings        = "（指摘事項なし）"
	auditReportExternalSuffix    = "（外部）"
	auditReportListSeparator     = "、"
)

type (
	auditReportFinding struct {
		Finding   *coredata.Finding
		Details   *coredata.FindingPrivacyDetails
		OwnerName string
	}

	auditReportData struct {
		OrganizationName string
		Audit            *coredata.Audit
		FrameworkName    string
		Details          *coredata.AuditPrivacyDetails
		AuditorNames     []string
		Findings         []auditReportFinding
	}
)

var (
	findingPrivacyCategoryLabels = map[coredata.FindingPrivacyCategory]string{
		coredata.FindingPrivacyCategoryNonconformity:          "不適合",
		coredata.FindingPrivacyCategoryObservation:            "観察事項",
		coredata.FindingPrivacyCategoryImprovementOpportunity: "改善の機会",
	}
	findingStatusLabels = map[coredata.FindingStatus]string{
		coredata.FindingStatusOpen:          "未完了",
		coredata.FindingStatusInProgress:    "対応中",
		coredata.FindingStatusClosed:        "完了",
		coredata.FindingStatusRiskAccepted:  "リスク受容",
		coredata.FindingStatusMitigated:     "低減済み",
		coredata.FindingStatusFalsePositive: "誤検知",
	}
	auditReportSummaryColumns = []xlsxexport.Column{
		{Header: "項目", Width: 16},
		{Header: "内容", Width: 80},
	}
	auditReportFindingColumns = []xlsxexport.Column{
		{Header: "No.", Width: 5},
		{Header: "参照 ID", Width: 10},
		{Header: "分類", Width: 12},
		{Header: "内容", Width: 36},
		{Header: "根拠", Width: 18},
		{Header: "原因", Width: 24},
		{Header: "是正処置", Width: 28},
		{Header: "担当者", Width: 14},
		{Header: "期限", Width: 11},
		{Header: "状態", Width: 10},
		{Header: "有効性の確認", Width: 24},
	}
)

// findingPrivacyCategoryLabel names the category of the finding for the
// report: the chosen one, or the one its upstream kind falls into. A major
// nonconformity is marked as such (重大な不適合).
func findingPrivacyCategoryLabel(
	finding *coredata.Finding,
	details *coredata.FindingPrivacyDetails,
) string {
	category := coredata.FindingPrivacyCategoryForKind(finding.Kind)
	if details != nil && details.Category != nil {
		category = details.Category
	}

	if category == nil {
		return ""
	}

	label := findingPrivacyCategoryLabels[*category]
	if *category == coredata.FindingPrivacyCategoryNonconformity &&
		finding.Kind == coredata.FindingKindMajorNonconformity {
		label += "（重大）"
	}

	return label
}

// buildAuditReportSheets returns the summary sheet and the finding list of
// the internal audit report.
func buildAuditReportSheets(data auditReportData, now time.Time) []xlsxexport.Sheet {
	subtitle := data.OrganizationName + "　出力日：" + now.Format(ledgerDateLayout)

	return []xlsxexport.Sheet{
		{
			Name:     auditReportSummarySheetName,
			Title:    auditReportTitle,
			Subtitle: subtitle,
			Columns:  auditReportSummaryColumns,
			Rows:     auditReportSummaryRows(data),
		},
		{
			Name:     auditReportFindingsSheetName,
			Title:    auditReportTitle + "　" + auditReportFindingsSheetName,
			Subtitle: subtitle,
			Columns:  auditReportFindingColumns,
			Rows:     auditReportFindingRows(data.Findings),
		},
	}
}

func auditReportSummaryRows(data auditReportData) [][]string {
	details := data.Details
	if details == nil {
		details = &coredata.AuditPrivacyDetails{}
	}

	auditors := make([]string, 0, len(data.AuditorNames)+len(details.ExternalAuditors))
	auditors = append(auditors, data.AuditorNames...)
	for _, name := range details.ExternalAuditors {
		auditors = append(auditors, name+auditReportExternalSuffix)
	}

	return [][]string{
		{"監査の名前", derefOrBlank(data.Audit.Name)},
		{"フレームワーク", data.FrameworkName},
		{"監査日", formatDateRange(data.Audit.AuditStartDate, data.Audit.AuditEndDate)},
		{"監査の範囲", details.Scope},
		{"監査の基準", details.Criteria},
		{"監査員", strings.Join(auditors, auditReportListSeparator)},
		{"被監査部門", strings.Join(details.AuditeeDepartments, auditReportListSeparator)},
		{"指摘事項の件数", auditReportFindingCounts(data.Findings)},
		{"総合的な所見", details.OverallOpinion},
	}
}

// auditReportFindingCounts counts the findings by category, in the order
// of the internal audit procedure. Findings with no category are counted
// as その他.
func auditReportFindingCounts(findings []auditReportFinding) string {
	counts := make(map[string]int)
	for _, f := range findings {
		counts[findingPrivacyCategoryLabel(f.Finding, f.Details)]++
	}

	labels := []string{"不適合（重大）", "不適合", "観察事項", "改善の機会"}
	parts := make([]string, 0, len(labels)+1)
	for _, label := range labels {
		parts = append(parts, fmt.Sprintf("%s %d 件", label, counts[label]))
	}
	if n := counts[""]; n > 0 {
		parts = append(parts, fmt.Sprintf("その他 %d 件", n))
	}

	return fmt.Sprintf("計 %d 件（%s）", len(findings), strings.Join(parts, auditReportListSeparator))
}

func auditReportFindingRows(findings []auditReportFinding) [][]string {
	if len(findings) == 0 {
		return [][]string{padRow([]string{"", "", auditReportNoFindings}, len(auditReportFindingColumns))}
	}

	rows := make([][]string, 0, len(findings))
	for i, f := range findings {
		basis := ""
		if f.Details != nil {
			basis = f.Details.Basis
		}

		rows = append(rows, []string{
			strconv.Itoa(i + 1),
			f.Finding.ReferenceID,
			findingPrivacyCategoryLabel(f.Finding, f.Details),
			derefOrBlank(f.Finding.Description),
			basis,
			derefOrBlank(f.Finding.RootCause),
			derefOrBlank(f.Finding.CorrectiveAction),
			f.OwnerName,
			formatLedgerDate(f.Finding.DueDate),
			findingStatusLabels[f.Finding.Status],
			derefOrBlank(f.Finding.EffectivenessCheck),
		})
	}

	return rows
}

func formatDateRange(start *time.Time, end *time.Time) string {
	switch {
	case start == nil && end == nil:
		return ""
	case end == nil || (start != nil && start.Equal(*end)):
		return formatLedgerDate(start)
	case start == nil:
		return "〜" + formatLedgerDate(end)
	}

	return formatLedgerDate(start) + "〜" + formatLedgerDate(end)
}
