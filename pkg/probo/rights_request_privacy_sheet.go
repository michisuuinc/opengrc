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

// The record of requests is a Japanese audit form, so its labels are
// Japanese only. The items follow the rights request procedure template
// (docs/ja/templates/14-rights-request-procedure.md, sections 4, 5, 9, 11
// and 14).

const (
	rightsRequestRecordTitle             = "本人からの請求の記録"
	rightsRequestRecordSheetName         = "請求の記録"
	rightsRequestSummarySheetName        = "集計"
	rightsRequestRecordNoRequests        = "（請求なし）"
	rightsRequestRecordSeparator         = "、"
	rightsRequestRecordAgent             = "代理人"
	rightsRequestRecordAllPeriods        = "すべて"
	rightsRequestRecordTotal             = "合計"
	rightsRequestRecordDaysUnit          = " 日"
	rightsRequestRecordAverageDaysLayout = "%.1f 日"
)

type (
	rightsRequestRecord struct {
		Request *coredata.RightsRequest
		// Details is nil when no record item was entered.
		Details      *coredata.RightsRequestPrivacyDetails
		ReceivedOn   time.Time
		HandlerName  string
		ApproverName string
	}

	rightsRequestRecordData struct {
		OrganizationName string
		// From and To bound the receipt dates exported, when set.
		From    *time.Time
		To      *time.Time
		Records []rightsRequestRecord
	}
)

var (
	rightsRequestCategoryLabels = map[coredata.RightsRequestPrivacyCategory]string{
		coredata.RightsRequestPrivacyCategoryPurposeNotification:        "利用目的の通知",
		coredata.RightsRequestPrivacyCategoryDisclosure:                 "開示",
		coredata.RightsRequestPrivacyCategoryCorrection:                 "訂正・追加・削除",
		coredata.RightsRequestPrivacyCategorySuspension:                 "利用停止・消去",
		coredata.RightsRequestPrivacyCategoryThirdPartyProvisionStop:    "第三者提供の停止",
		coredata.RightsRequestPrivacyCategoryThirdPartyRecordDisclosure: "第三者提供記録の開示",
		coredata.RightsRequestPrivacyCategoryComplaint:                  "苦情",
		coredata.RightsRequestPrivacyCategoryConsultation:               "相談",
	}
	rightsRequestChannelLabels = map[coredata.RightsRequestReceiptChannel]string{
		coredata.RightsRequestReceiptChannelPostal:   "郵送",
		coredata.RightsRequestReceiptChannelEmail:    "メール",
		coredata.RightsRequestReceiptChannelWebForm:  "Web フォーム",
		coredata.RightsRequestReceiptChannelPhone:    "電話",
		coredata.RightsRequestReceiptChannelInPerson: "窓口",
		coredata.RightsRequestReceiptChannelOther:    "その他",
	}
	rightsRequestVerificationLabels = map[coredata.RightsRequestVerificationMethod]string{
		coredata.RightsRequestVerificationMethodIDDocument:  "本人確認書類",
		coredata.RightsRequestVerificationMethodAccount:     "アカウント・登録情報との照合",
		coredata.RightsRequestVerificationMethodOther:       "その他",
		coredata.RightsRequestVerificationMethodNotRequired: "不要",
	}
	rightsRequestDecisionLabels = map[coredata.RightsRequestDecision]string{
		coredata.RightsRequestDecisionFulfilled:          "請求どおり対応",
		coredata.RightsRequestDecisionPartiallyFulfilled: "一部対応（一部不開示等）",
		coredata.RightsRequestDecisionNotFulfilled:       "対応しない（不開示等）",
	}
	rightsRequestAnswerMethodLabels = map[coredata.RightsRequestAnswerMethod]string{
		coredata.RightsRequestAnswerMethodPostal:           "書面（郵送）",
		coredata.RightsRequestAnswerMethodEmail:            "メール",
		coredata.RightsRequestAnswerMethodElectronicRecord: "電磁的記録の提供",
		coredata.RightsRequestAnswerMethodPhone:            "電話・口頭",
		coredata.RightsRequestAnswerMethodOther:            "その他",
	}
	rightsRequestStateLabels = map[coredata.RightsRequestState]string{
		coredata.RightsRequestStateTodo:       "未着手",
		coredata.RightsRequestStateInProgress: "対応中",
		coredata.RightsRequestStateDone:       "完了",
		coredata.RightsRequestStateRejected:   "却下",
	}
	rightsRequestRecordColumns = []xlsxexport.Column{
		{Header: "No.", Width: 5},
		{Header: "受付日", Width: 11},
		{Header: "受付方法", Width: 10},
		{Header: "請求の種類", Width: 14},
		{Header: "請求者", Width: 14},
		{Header: "代理人", Width: 7},
		{Header: "内容", Width: 28},
		{Header: "対象の部門", Width: 12},
		{Header: "窓口担当者", Width: 12},
		{Header: "本人確認の方法", Width: 18},
		{Header: "本人確認日", Width: 11},
		{Header: "補正等の依頼日", Width: 11},
		{Header: "補正等の受領日", Width: 11},
		{Header: "期限", Width: 11},
		{Header: "状態", Width: 8},
		{Header: "対応の内容", Width: 28},
		{Header: "判断", Width: 16},
		{Header: "不開示等の理由", Width: 24},
		{Header: "回答方法", Width: 12},
		{Header: "回答日", Width: 11},
		{Header: "承認者", Width: 12},
		{Header: "対応日数", Width: 8},
		{Header: "確認事項", Width: 20},
	}
	rightsRequestSummaryColumns = []xlsxexport.Column{
		{Header: "請求の種類", Width: 20},
		{Header: "件数", Width: 8},
		{Header: "回答済み", Width: 9},
		{Header: "未回答", Width: 8},
		{Header: "平均対応日数", Width: 12},
		{Header: "最長対応日数", Width: 12},
		{Header: "一部対応", Width: 9},
		{Header: "対応しない", Width: 10},
	}
)

// buildRightsRequestRecordSheets returns the record of each request and a
// summary per kind for the management review.
func buildRightsRequestRecordSheets(data rightsRequestRecordData, now time.Time) []xlsxexport.Sheet {
	subtitle := data.OrganizationName +
		"　受付日：" + rightsRequestPeriodLabel(data.From, data.To) +
		"　出力日：" + now.Format(ledgerDateLayout)

	return []xlsxexport.Sheet{
		{
			Name:     rightsRequestRecordSheetName,
			Title:    rightsRequestRecordTitle,
			Subtitle: subtitle,
			Columns:  rightsRequestRecordColumns,
			Rows:     rightsRequestRecordRows(data.Records, now),
		},
		{
			Name:     rightsRequestSummarySheetName,
			Title:    rightsRequestRecordTitle + "　" + rightsRequestSummarySheetName,
			Subtitle: subtitle,
			Columns:  rightsRequestSummaryColumns,
			Rows:     rightsRequestSummaryRows(data.Records),
		},
	}
}

func rightsRequestPeriodLabel(from *time.Time, to *time.Time) string {
	if from == nil && to == nil {
		return rightsRequestRecordAllPeriods
	}

	return formatLedgerDate(from) + "〜" + formatLedgerDate(to)
}

func rightsRequestRecordRows(records []rightsRequestRecord, now time.Time) [][]string {
	if len(records) == 0 {
		return [][]string{padRow([]string{"", "", "", rightsRequestRecordNoRequests}, len(rightsRequestRecordColumns))}
	}

	rows := make([][]string, 0, len(records))
	for i, r := range records {
		d := r.Details
		if d == nil {
			d = &coredata.RightsRequestPrivacyDetails{}
		}

		agent := ""
		if d.ByAgent {
			agent = rightsRequestRecordAgent
		}

		days := ""
		if n := RightsRequestHandlingDays(r.ReceivedOn, r.Details); n != nil {
			days = strconv.Itoa(*n) + rightsRequestRecordDaysUnit
		}

		var deadline *time.Time
		if r.Request.Deadline != nil {
			value := r.Request.Deadline.In(ledgerTimeZone)
			deadline = &value
		}

		receivedOn := r.ReceivedOn

		rows = append(rows, []string{
			strconv.Itoa(i + 1),
			formatLedgerDate(&receivedOn),
			labelOf(rightsRequestChannelLabels, RightsRequestEffectiveReceiptChannel(r.Request, r.Details)),
			rightsRequestCategoryLabels[RightsRequestEffectiveCategory(r.Request, r.Details)],
			derefOrBlank(r.Request.DataSubject),
			agent,
			derefOrBlank(r.Request.Details),
			d.TargetDepartment,
			r.HandlerName,
			rightsRequestVerificationLabel(d),
			formatLedgerDate(d.VerifiedOn),
			formatLedgerDate(d.SupplementRequestedOn),
			formatLedgerDate(d.SupplementReceivedOn),
			formatLedgerDate(deadline),
			rightsRequestStateLabels[r.Request.RequestState],
			derefOrBlank(r.Request.ActionTaken),
			labelOf(rightsRequestDecisionLabels, d.Decision),
			d.DecisionReason,
			labelOf(rightsRequestAnswerMethodLabels, d.AnswerMethod),
			formatLedgerDate(d.AnsweredOn),
			r.ApproverName,
			days,
			strings.Join(rightsRequestRemarks(r, now), rightsRequestRecordSeparator),
		})
	}

	return rows
}

func labelOf[T comparable](labels map[T]string, value *T) string {
	if value == nil {
		return ""
	}

	return labels[*value]
}

func rightsRequestVerificationLabel(d *coredata.RightsRequestPrivacyDetails) string {
	label := labelOf(rightsRequestVerificationLabels, d.VerificationMethod)
	if d.VerificationNote == "" {
		return label
	}

	if label == "" {
		return d.VerificationNote
	}

	return label + "（" + d.VerificationNote + "）"
}

func isComplaintOrConsultation(category coredata.RightsRequestPrivacyCategory) bool {
	return category == coredata.RightsRequestPrivacyCategoryComplaint ||
		category == coredata.RightsRequestPrivacyCategoryConsultation
}

// rightsRequestRemarks lists what an auditor would ask about: a missed
// deadline, a closed request with no answer date, a request answered
// without verification or approval, and a request with no record items.
func rightsRequestRemarks(r rightsRequestRecord, now time.Time) []string {
	remarks := []string{}
	d := r.Details
	answered := d != nil && d.AnsweredOn != nil

	if !answered && r.Request.Deadline != nil && r.Request.Deadline.Before(now) {
		remarks = append(remarks, "期限を過ぎている")
	}

	if d == nil {
		return append(remarks, "記録の項目が未記入")
	}

	closed := r.Request.RequestState == coredata.RightsRequestStateDone ||
		r.Request.RequestState == coredata.RightsRequestStateRejected
	if closed && !answered {
		remarks = append(remarks, "回答日が未記入")
	}

	if !isComplaintOrConsultation(RightsRequestEffectiveCategory(r.Request, d)) && d.VerificationMethod == nil {
		remarks = append(remarks, "本人確認が未記入")
	}

	if answered && d.ApproverProfileID == nil {
		remarks = append(remarks, "回答の承認者が未記入")
	}

	return remarks
}

type rightsRequestSummary struct {
	count       int
	answered    int
	totalDays   int
	maxDays     int
	partially   int
	notAnswered int
	refused     int
}

func (s *rightsRequestSummary) add(r rightsRequestRecord) {
	s.count++

	if r.Details == nil || r.Details.AnsweredOn == nil {
		s.notAnswered++
	} else {
		s.answered++
	}

	if days := RightsRequestHandlingDays(r.ReceivedOn, r.Details); days != nil {
		s.totalDays += *days
		s.maxDays = max(s.maxDays, *days)
	}

	if r.Details != nil && r.Details.Decision != nil {
		switch *r.Details.Decision {
		case coredata.RightsRequestDecisionPartiallyFulfilled:
			s.partially++
		case coredata.RightsRequestDecisionNotFulfilled:
			s.refused++
		}
	}
}

func (s rightsRequestSummary) row(label string) []string {
	average, longest := "", ""
	if s.answered > 0 {
		average = fmt.Sprintf(rightsRequestRecordAverageDaysLayout, float64(s.totalDays)/float64(s.answered))
		longest = strconv.Itoa(s.maxDays) + rightsRequestRecordDaysUnit
	}

	return []string{
		label,
		strconv.Itoa(s.count),
		strconv.Itoa(s.answered),
		strconv.Itoa(s.notAnswered),
		average,
		longest,
		strconv.Itoa(s.partially),
		strconv.Itoa(s.refused),
	}
}

// rightsRequestSummaryRows counts the requests per kind, in the order of
// the procedure, then the total (procedure 14, section 14).
func rightsRequestSummaryRows(records []rightsRequestRecord) [][]string {
	byCategory := make(map[coredata.RightsRequestPrivacyCategory]*rightsRequestSummary)
	total := &rightsRequestSummary{}

	for _, r := range records {
		category := RightsRequestEffectiveCategory(r.Request, r.Details)
		if byCategory[category] == nil {
			byCategory[category] = &rightsRequestSummary{}
		}
		byCategory[category].add(r)
		total.add(r)
	}

	categories := coredata.RightsRequestPrivacyCategories()
	rows := make([][]string, 0, len(categories)+1)
	for _, category := range categories {
		summary := byCategory[category]
		if summary == nil {
			summary = &rightsRequestSummary{}
		}
		rows = append(rows, summary.row(rightsRequestCategoryLabels[category]))
	}

	return append(rows, total.row(rightsRequestRecordTotal))
}
