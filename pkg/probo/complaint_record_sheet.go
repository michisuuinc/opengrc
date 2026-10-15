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
	"slices"
	"strconv"
	"strings"
	"time"

	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/xlsxexport"
)

// The complaint and consultation record follows section 12 of the rights
// request procedure template (docs/ja/templates/14-rights-request-
// procedure.md): receipt, the department concerned, the first response,
// the course and result, and the finding opened for a cause in the
// company's own handling.

const (
	complaintRecordTitle            = "苦情・相談の記録"
	complaintRecordSheetName        = "苦情・相談の記録"
	complaintDepartmentSheetName    = "部門別の件数"
	complaintRecordNoComplaints     = "（苦情・相談なし）"
	complaintRecordNoDepartment     = "（部門未記入）"
	complaintRecordFirstResponseDue = "一次回答が未記入"
)

var (
	complaintRecordColumns = []xlsxexport.Column{
		{Header: "No.", Width: 5},
		{Header: "受付日", Width: 11},
		{Header: "受付方法", Width: 10},
		{Header: "種類", Width: 6},
		{Header: "申出者", Width: 14},
		{Header: "内容", Width: 30},
		{Header: "対象の部門", Width: 12},
		{Header: "窓口担当者", Width: 12},
		{Header: "一次回答日", Width: 11},
		{Header: "一次回答までの日数", Width: 10},
		{Header: "状態", Width: 8},
		{Header: "対応の経過と結果", Width: 30},
		{Header: "回答日", Width: 11},
		{Header: "承認者", Width: 12},
		{Header: "指摘事項", Width: 10},
		{Header: "確認事項", Width: 20},
	}
	complaintDepartmentColumns = []xlsxexport.Column{
		{Header: "対象の部門", Width: 20},
		{Header: "苦情", Width: 8},
		{Header: "相談", Width: 8},
		{Header: "計", Width: 8},
		{Header: "未回答", Width: 8},
		{Header: "指摘事項に登録", Width: 14},
	}
)

// buildComplaintRecordSheets returns the record of each complaint and
// consultation, and the counts per department for the management review.
func buildComplaintRecordSheets(data rightsRequestRecordData, now time.Time) []xlsxexport.Sheet {
	subtitle := data.OrganizationName +
		"　受付日：" + rightsRequestPeriodLabel(data.From, data.To) +
		"　出力日：" + now.Format(ledgerDateLayout)

	return []xlsxexport.Sheet{
		{
			Name:     complaintRecordSheetName,
			Title:    complaintRecordTitle,
			Subtitle: subtitle,
			Columns:  complaintRecordColumns,
			Rows:     complaintRecordRows(data.Records, now),
		},
		{
			Name:     complaintDepartmentSheetName,
			Title:    complaintRecordTitle + "　" + complaintDepartmentSheetName,
			Subtitle: subtitle,
			Columns:  complaintDepartmentColumns,
			Rows:     complaintDepartmentRows(data.Records),
		},
	}
}

func complaintRecordRows(records []rightsRequestRecord, now time.Time) [][]string {
	if len(records) == 0 {
		return [][]string{padRow([]string{"", "", "", "", "", complaintRecordNoComplaints}, len(complaintRecordColumns))}
	}

	rows := make([][]string, 0, len(records))
	for i, r := range records {
		d := r.Details
		if d == nil {
			d = &coredata.RightsRequestPrivacyDetails{}
		}

		receivedOn := r.ReceivedOn
		firstResponseDays := ""
		if d.FirstRespondedOn != nil {
			firstResponseDays = strconv.Itoa(max(daysBetween(receivedOn, *d.FirstRespondedOn), 0)) + rightsRequestRecordDaysUnit
		}

		rows = append(rows, []string{
			strconv.Itoa(i + 1),
			formatLedgerDate(&receivedOn),
			labelOf(rightsRequestChannelLabels, RightsRequestEffectiveReceiptChannel(r.Request, r.Details)),
			rightsRequestCategoryLabels[RightsRequestEffectiveCategory(r.Request, r.Details)],
			derefOrBlank(r.Request.DataSubject),
			derefOrBlank(r.Request.Details),
			d.TargetDepartment,
			r.HandlerName,
			formatLedgerDate(d.FirstRespondedOn),
			firstResponseDays,
			rightsRequestStateLabels[r.Request.RequestState],
			derefOrBlank(r.Request.ActionTaken),
			formatLedgerDate(d.AnsweredOn),
			r.ApproverName,
			r.FindingReference,
			strings.Join(complaintRemarks(r, now), rightsRequestRecordSeparator),
		})
	}

	return rows
}

// complaintRemarks is what an auditor would ask about a complaint: the
// remarks of any request, without identity verification, which complaints
// do not need, plus a missing first response.
func complaintRemarks(r rightsRequestRecord, now time.Time) []string {
	remarks := rightsRequestRemarks(r, now)

	if r.Details != nil && r.Details.FirstRespondedOn == nil {
		remarks = append(remarks, complaintRecordFirstResponseDue)
	}

	return remarks
}

type complaintDepartmentCount struct {
	complaints    int
	consultations int
	notAnswered   int
	withFinding   int
}

// complaintDepartmentRows counts the complaints and consultations per
// department in name order, departments left blank last, then the total.
func complaintDepartmentRows(records []rightsRequestRecord) [][]string {
	counts := map[string]*complaintDepartmentCount{}
	total := &complaintDepartmentCount{}

	for _, r := range records {
		department := complaintRecordNoDepartment
		if r.Details != nil && r.Details.TargetDepartment != "" {
			department = r.Details.TargetDepartment
		}

		if counts[department] == nil {
			counts[department] = &complaintDepartmentCount{}
		}

		for _, c := range []*complaintDepartmentCount{counts[department], total} {
			if RightsRequestEffectiveCategory(r.Request, r.Details) == coredata.RightsRequestPrivacyCategoryConsultation {
				c.consultations++
			} else {
				c.complaints++
			}

			if r.Details == nil || r.Details.AnsweredOn == nil {
				c.notAnswered++
			}

			if r.Details != nil && r.Details.FindingID != nil {
				c.withFinding++
			}
		}
	}

	departments := make([]string, 0, len(counts))
	for department := range counts {
		if department != complaintRecordNoDepartment {
			departments = append(departments, department)
		}
	}
	slices.Sort(departments)
	if counts[complaintRecordNoDepartment] != nil {
		departments = append(departments, complaintRecordNoDepartment)
	}

	row := func(label string, c *complaintDepartmentCount) []string {
		return []string{
			label,
			strconv.Itoa(c.complaints),
			strconv.Itoa(c.consultations),
			strconv.Itoa(c.complaints + c.consultations),
			strconv.Itoa(c.notAnswered),
			strconv.Itoa(c.withFinding),
		}
	}

	rows := make([][]string, 0, len(departments)+1)
	for _, department := range departments {
		rows = append(rows, row(department, counts[department]))
	}

	return append(rows, row(rightsRequestRecordTotal, total))
}
