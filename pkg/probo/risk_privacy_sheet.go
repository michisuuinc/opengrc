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
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/xlsxexport"
)

// The risk analysis is a Japanese audit form, so its labels are Japanese
// only. It has one row per register entry (processing activity) and risk,
// following the risk assessment article of the PMS rules template
// (docs/ja/templates/10-pms-rules.md, article 16). A register entry with no
// risk gets one row that says so, so missing assessments show up.

const (
	riskPrivacySheetName = "リスク分析表"
	riskPrivacyNoRisk    = "（リスク未登録）"
)

var riskPrivacyStageLabels = map[coredata.PersonalDataLifecycleStage]string{
	coredata.PersonalDataLifecycleStageCollection: "取得・入力",
	coredata.PersonalDataLifecycleStageTransfer:   "移送・送信",
	coredata.PersonalDataLifecycleStageUse:        "利用・加工",
	coredata.PersonalDataLifecycleStageStorage:    "保管・バックアップ",
	coredata.PersonalDataLifecycleStageDisposal:   "消去・廃棄",
}

// The treatment labels follow the PMS rules template, article 17.
var riskPrivacyTreatmentLabels = map[coredata.RiskTreatment]string{
	coredata.RiskTreatmentMitigated:   "低減",
	coredata.RiskTreatmentAccepted:    "受容",
	coredata.RiskTreatmentAvoided:     "回避",
	coredata.RiskTreatmentTransferred: "移転",
}

type riskPrivacyData struct {
	OrganizationName string
	// Activities are in register order (by name).
	Activities []*coredata.ProcessingActivity
	Risks      map[gid.GID]*coredata.Risk
	// RiskIDs lists, per processing activity, the risks linked to it.
	RiskIDs              map[gid.GID][]gid.GID
	Stages               map[gid.GID][]coredata.PersonalDataLifecycleStage
	InternalControlNames map[gid.GID][]string
	OwnerNames           map[gid.GID]string
}

type riskPrivacyColumn struct {
	column xlsxexport.Column
	value  func(data riskPrivacyData, r *coredata.Risk) string
}

var riskPrivacyColumns = []riskPrivacyColumn{
	{xlsxexport.Column{Header: "リスクID", Width: 10}, func(_ riskPrivacyData, r *coredata.Risk) string {
		return r.ReferenceID
	}},
	{xlsxexport.Column{Header: "段階", Width: 14}, func(data riskPrivacyData, r *coredata.Risk) string {
		return riskPrivacyStagesLabel(data.Stages[r.ID])
	}},
	{xlsxexport.Column{Header: "想定されるリスク", Width: 24}, func(_ riskPrivacyData, r *coredata.Risk) string {
		return r.Name
	}},
	{xlsxexport.Column{Header: "リスクの内容", Width: 30}, func(_ riskPrivacyData, r *coredata.Risk) string {
		return derefOrBlank(r.Description)
	}},
	{xlsxexport.Column{Header: "発生可能性", Width: 8}, func(_ riskPrivacyData, r *coredata.Risk) string {
		return formatRiskPrivacyScore(r.InherentLikelihood)
	}},
	{xlsxexport.Column{Header: "影響度", Width: 8}, func(_ riskPrivacyData, r *coredata.Risk) string {
		return formatRiskPrivacyScore(r.InherentImpact)
	}},
	{xlsxexport.Column{Header: "リスク値", Width: 8}, func(_ riskPrivacyData, r *coredata.Risk) string {
		return formatRiskPrivacyScore(r.InherentRiskScore)
	}},
	{xlsxexport.Column{Header: "対応方針", Width: 8}, func(_ riskPrivacyData, r *coredata.Risk) string {
		return riskPrivacyTreatmentLabel(r.Treatment)
	}},
	{xlsxexport.Column{Header: "対策（内部統制）", Width: 30}, func(data riskPrivacyData, r *coredata.Risk) string {
		return strings.Join(data.InternalControlNames[r.ID], "\n")
	}},
	{xlsxexport.Column{Header: "残留リスクの発生可能性", Width: 10}, func(_ riskPrivacyData, r *coredata.Risk) string {
		return formatRiskPrivacyScore(r.ResidualLikelihood)
	}},
	{xlsxexport.Column{Header: "残留リスクの影響度", Width: 10}, func(_ riskPrivacyData, r *coredata.Risk) string {
		return formatRiskPrivacyScore(r.ResidualImpact)
	}},
	{xlsxexport.Column{Header: "残留リスク値", Width: 8}, func(_ riskPrivacyData, r *coredata.Risk) string {
		return formatRiskPrivacyScore(r.ResidualRiskScore)
	}},
	{xlsxexport.Column{Header: "責任者", Width: 12}, func(data riskPrivacyData, r *coredata.Risk) string {
		if r.OwnerID == nil {
			return ledgerBlank
		}
		return data.OwnerNames[*r.OwnerID]
	}},
	{xlsxexport.Column{Header: "備考", Width: 20}, func(_ riskPrivacyData, r *coredata.Risk) string {
		return r.Note
	}},
}

func buildRiskPrivacySheet(data riskPrivacyData, now time.Time) xlsxexport.Sheet {
	columns := []xlsxexport.Column{
		{Header: "No.", Width: 5},
		{Header: "業務の名称", Width: 20},
		{Header: "要配慮個人情報", Width: 10},
	}
	for _, c := range riskPrivacyColumns {
		columns = append(columns, c.column)
	}

	rows := make([][]string, 0, len(data.Activities))
	for _, a := range data.Activities {
		activityCells := []string{a.Name, ledgerSpecialDataLabel(a.SpecialOrCriminalData)}

		risks := data.activityRisks(a.ID)
		if len(risks) == 0 {
			row := append([]string{strconv.Itoa(len(rows) + 1)}, activityCells...)
			row = append(row, ledgerBlank, ledgerBlank, riskPrivacyNoRisk)
			rows = append(rows, padRow(row, len(columns)))
			continue
		}

		for _, r := range risks {
			row := append([]string{strconv.Itoa(len(rows) + 1)}, activityCells...)
			for _, c := range riskPrivacyColumns {
				row = append(row, c.value(data, r))
			}
			rows = append(rows, row)
		}
	}

	return xlsxexport.Sheet{
		Name:     riskPrivacySheetName,
		Title:    riskPrivacySheetName,
		Subtitle: data.OrganizationName + "　出力日：" + now.Format(ledgerDateLayout),
		Columns:  columns,
		Rows:     rows,
	}
}

// activityRisks returns the risks linked to the processing activity, by
// their first lifecycle stage, then by reference ID. Risks with no stage come
// last.
func (data riskPrivacyData) activityRisks(activityID gid.GID) []*coredata.Risk {
	risks := make([]*coredata.Risk, 0, len(data.RiskIDs[activityID]))
	for _, id := range data.RiskIDs[activityID] {
		if r, ok := data.Risks[id]; ok {
			risks = append(risks, r)
		}
	}

	slices.SortStableFunc(risks, func(a, b *coredata.Risk) int {
		return cmp.Or(
			cmp.Compare(firstStageIndex(data.Stages[a.ID]), firstStageIndex(data.Stages[b.ID])),
			cmp.Compare(a.ReferenceID, b.ReferenceID),
		)
	})

	return risks
}

func firstStageIndex(stages []coredata.PersonalDataLifecycleStage) int {
	all := coredata.PersonalDataLifecycleStages()
	for i, s := range all {
		if slices.Contains(stages, s) {
			return i
		}
	}
	return len(all)
}

// riskPrivacyStagesLabel lists the stages in lifecycle order, whatever order
// they were stored in.
func riskPrivacyStagesLabel(stages []coredata.PersonalDataLifecycleStage) string {
	labels := make([]string, 0, len(stages))
	for _, s := range coredata.PersonalDataLifecycleStages() {
		if slices.Contains(stages, s) {
			labels = append(labels, riskPrivacyStageLabels[s])
		}
	}
	return strings.Join(labels, "\n")
}

func riskPrivacyTreatmentLabel(t *coredata.RiskTreatment) string {
	if t == nil {
		return ledgerBlank
	}

	label, ok := riskPrivacyTreatmentLabels[*t]
	if !ok {
		return string(*t)
	}
	return label
}

func formatRiskPrivacyScore(v *int) string {
	if v == nil {
		return ledgerBlank
	}
	return strconv.Itoa(*v)
}

func padRow(row []string, n int) []string {
	for len(row) < n {
		row = append(row, ledgerBlank)
	}
	return row
}
