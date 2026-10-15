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
	"strconv"
	"strings"
	"time"

	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/xlsxexport"
)

// The register is a Japanese audit form, so its labels are Japanese only.
// The column order follows the register items listed in the PMS rules
// template (docs/ja/templates/10-pms-rules.md, article 14).

const (
	ledgerSheetName  = "個人情報管理台帳"
	ledgerBlank      = ""
	ledgerDateLayout = "2006-01-02"
	ledgerYes        = "あり"
	ledgerNo         = "なし"
)

type ledgerData struct {
	OrganizationName string
	Activities       []*coredata.ProcessingActivity
	Details          map[gid.GID]*coredata.ProcessingActivityLedgerDetails
	ThirdPartyNames  map[gid.GID][]string
}

type ledgerColumn struct {
	column xlsxexport.Column
	value  func(a *coredata.ProcessingActivity, d *coredata.ProcessingActivityLedgerDetails, thirdParties []string) string
}

var ledgerColumns = []ledgerColumn{
	{xlsxexport.Column{Header: "業務の名称", Width: 24}, func(a *coredata.ProcessingActivity, _ *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return a.Name
	}},
	{xlsxexport.Column{Header: "担当部門", Width: 14}, func(_ *coredata.ProcessingActivity, d *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return derefOrBlank(d.Department)
	}},
	{xlsxexport.Column{Header: "個人情報の項目", Width: 28}, func(a *coredata.ProcessingActivity, _ *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return derefOrBlank(a.PersonalDataCategory)
	}},
	{xlsxexport.Column{Header: "要配慮個人情報", Width: 10}, func(a *coredata.ProcessingActivity, _ *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return ledgerSpecialDataLabel(a.SpecialOrCriminalData)
	}},
	{xlsxexport.Column{Header: "本人の種類", Width: 16}, func(a *coredata.ProcessingActivity, _ *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return derefOrBlank(a.DataSubjectCategory)
	}},
	{xlsxexport.Column{Header: "おおよその件数", Width: 12}, func(_ *coredata.ProcessingActivity, d *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return derefOrBlank(d.ApproximateRecordCount)
	}},
	{xlsxexport.Column{Header: "利用目的", Width: 30}, func(a *coredata.ProcessingActivity, _ *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return derefOrBlank(a.Purpose)
	}},
	{xlsxexport.Column{Header: "取得の方法", Width: 20}, func(_ *coredata.ProcessingActivity, d *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return derefOrBlank(d.CollectionMethod)
	}},
	{xlsxexport.Column{Header: "保管の場所と媒体", Width: 20}, func(_ *coredata.ProcessingActivity, d *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return derefOrBlank(d.StorageLocation)
	}},
	{xlsxexport.Column{Header: "アクセスできる人の範囲", Width: 20}, func(_ *coredata.ProcessingActivity, d *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return derefOrBlank(d.AccessScope)
	}},
	{xlsxexport.Column{Header: "保存期間", Width: 14}, func(a *coredata.ProcessingActivity, _ *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return derefOrBlank(a.RetentionPeriod)
	}},
	{xlsxexport.Column{Header: "廃棄の方法", Width: 18}, func(_ *coredata.ProcessingActivity, d *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return derefOrBlank(d.DisposalMethod)
	}},
	{xlsxexport.Column{Header: "委託先", Width: 20}, func(_ *coredata.ProcessingActivity, _ *coredata.ProcessingActivityLedgerDetails, thirdParties []string) string {
		if len(thirdParties) == 0 {
			return ledgerNo
		}
		return strings.Join(thirdParties, "\n")
	}},
	{xlsxexport.Column{Header: "第三者提供の相手", Width: 20}, func(a *coredata.ProcessingActivity, _ *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return derefOrBlank(a.Recipients)
	}},
	{xlsxexport.Column{Header: "外国への移転", Width: 16}, func(a *coredata.ProcessingActivity, _ *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return ledgerTransferLabel(a.InternationalTransfers, a.TransferSafeguard)
	}},
	{xlsxexport.Column{Header: "安全管理措置", Width: 28}, func(a *coredata.ProcessingActivity, _ *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return derefOrBlank(a.SecurityMeasures)
	}},
	{xlsxexport.Column{Header: "最終見直し日", Width: 12}, func(a *coredata.ProcessingActivity, _ *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return formatLedgerDate(a.LastReviewDate)
	}},
	{xlsxexport.Column{Header: "次回見直し日", Width: 12}, func(a *coredata.ProcessingActivity, _ *coredata.ProcessingActivityLedgerDetails, _ []string) string {
		return formatLedgerDate(a.NextReviewDate)
	}},
}

func buildLedgerSheet(data ledgerData, now time.Time) xlsxexport.Sheet {
	columns := make([]xlsxexport.Column, 0, len(ledgerColumns)+1)
	columns = append(columns, xlsxexport.Column{Header: "No.", Width: 5})
	for _, c := range ledgerColumns {
		columns = append(columns, c.column)
	}

	rows := make([][]string, len(data.Activities))
	for i, a := range data.Activities {
		d, ok := data.Details[a.ID]
		if !ok {
			d = &coredata.ProcessingActivityLedgerDetails{}
		}

		row := make([]string, 0, len(columns))
		row = append(row, strconv.Itoa(i+1))
		for _, c := range ledgerColumns {
			row = append(row, c.value(a, d, data.ThirdPartyNames[a.ID]))
		}
		rows[i] = row
	}

	return xlsxexport.Sheet{
		Name:     ledgerSheetName,
		Title:    ledgerSheetName,
		Subtitle: data.OrganizationName + "　出力日：" + now.Format(ledgerDateLayout),
		Columns:  columns,
		Rows:     rows,
	}
}

func derefOrBlank(s *string) string {
	if s == nil {
		return ledgerBlank
	}
	return *s
}

func formatLedgerDate(t *time.Time) string {
	if t == nil {
		return ledgerBlank
	}
	return t.Format(ledgerDateLayout)
}

func ledgerSpecialDataLabel(v coredata.ProcessingActivitySpecialOrCriminalDatum) string {
	switch v {
	case coredata.ProcessingActivitySpecialOrCriminalDatumYes:
		return ledgerYes
	case coredata.ProcessingActivitySpecialOrCriminalDatumNo:
		return ledgerNo
	case coredata.ProcessingActivitySpecialOrCriminalDatumPossible:
		return "可能性あり"
	}
	return string(v)
}

var ledgerSafeguardLabels = map[coredata.ProcessingActivityTransferSafeguard]string{
	coredata.ProcessingActivityTransferSafeguardStandardContractualClauses: "標準契約条項",
	coredata.ProcessingActivityTransferSafeguardBindingCorporateRules:      "拘束的企業準則",
	coredata.ProcessingActivityTransferSafeguardAdequacyDecision:           "十分性認定",
	coredata.ProcessingActivityTransferSafeguardDerogations:                "例外規定",
	coredata.ProcessingActivityTransferSafeguardCodesOfConduct:             "行動規範",
	coredata.ProcessingActivityTransferSafeguardCertificationMechanisms:    "認証メカニズム",
}

func ledgerTransferLabel(transfers bool, safeguard *coredata.ProcessingActivityTransferSafeguard) string {
	if !transfers {
		return ledgerNo
	}

	if safeguard == nil {
		return ledgerYes
	}

	label, ok := ledgerSafeguardLabels[*safeguard]
	if !ok {
		label = string(*safeguard)
	}

	return ledgerYes + "（" + label + "）"
}
