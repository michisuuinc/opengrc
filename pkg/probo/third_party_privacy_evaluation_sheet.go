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

// The outsourcing evaluation is a Japanese audit form, so its labels are
// Japanese only. The items and criteria follow the evaluation checklist
// template (docs/ja/templates/35-form-outsourcing-checklist.md).

const (
	thirdPartyEvaluationTitle          = "委託先評価表"
	thirdPartyEvaluationListSheetName  = "委託先一覧"
	thirdPartyEvaluationItemsSheetName = "評価項目"
	thirdPartyEvaluationNoThirdParties = "（委託先なし）"
	thirdPartyEvaluationNoEvaluations  = "（評価の記入なし）"
	thirdPartyEvaluationNotRegistered  = "（評価未登録）"
	thirdPartyEvaluationNotFilled      = "（評価未記入）"
	thirdPartyEvaluationNotJudged      = "未判定"
	thirdPartyEvaluationNotAnswered    = "未評価"
	thirdPartyEvaluationSeparator      = "、"
)

type (
	thirdPartyEvaluationRow struct {
		ThirdParty           *coredata.ThirdParty
		ParentName           string
		ProcessingActivities []string
		// Assessment is the latest risk assessment, or nil when there is
		// none. Evaluation is its evaluation, or nil when it has none.
		Assessment    *coredata.ThirdPartyRiskAssessment
		Evaluation    *coredata.ThirdPartyPrivacyEvaluation
		Answers       []ThirdPartyEvaluationAnswer
		EvaluatorName string
		ApproverName  string
	}

	thirdPartyEvaluationData struct {
		OrganizationName string
		Rows             []thirdPartyEvaluationRow
	}

	thirdPartyEvaluationItemLabel struct {
		Label     string
		Criterion string
	}
)

var (
	thirdPartyEvaluationItemLabels = map[coredata.ThirdPartyEvaluationItem]thirdPartyEvaluationItemLabel{
		coredata.ThirdPartyEvaluationItemCertification: {
			"認証の取得",
			"プライバシーマーク又は ISMS（ISO 27001）の認証を取得し、有効期限内である。取得していない場合は、No.2〜11 で同等の水準を確認する",
		},
		coredata.ThirdPartyEvaluationItemManagementStructure: {
			"管理体制",
			"個人情報の管理責任者が定められ、責任と権限が明確である",
		},
		coredata.ThirdPartyEvaluationItemRules: {
			"規程の整備",
			"個人情報の取扱いと安全管理の規程があり、従業者に周知されている",
		},
		coredata.ThirdPartyEvaluationItemEmployeeTraining: {
			"従業者の教育",
			"従業者に年 1 回以上の教育を行い、秘密保持の誓約を取っている",
		},
		coredata.ThirdPartyEvaluationItemPhysicalSecurity: {
			"物理的安全管理",
			"個人情報を扱う区域の入退室管理、書類・機器の施錠保管、確実な廃棄を行っている",
		},
		coredata.ThirdPartyEvaluationItemAccessControl: {
			"アクセス制御",
			"個人情報へのアクセスを必要な者に限り、ID を共有せず、多要素認証などで認証を強化している",
		},
		coredata.ThirdPartyEvaluationItemTechnicalSecurity: {
			"技術的安全管理",
			"マルウェア対策、ソフトウェアの更新、通信と保存データの暗号化、ログの取得、バックアップを行っている",
		},
		coredata.ThirdPartyEvaluationItemSubcontracting: {
			"再委託の管理",
			"再委託するときは事前に当社の承認を得る。再委託先にも同等の義務を課し、監督している",
		},
		coredata.ThirdPartyEvaluationItemIncidentResponse: {
			"事故時の対応",
			"漏えい等の際に、当社へ直ちに報告する体制と手順がある",
		},
		coredata.ThirdPartyEvaluationItemPastIncidents: {
			"過去の事故",
			"チェックリストで定める期間内に重大な漏えい等がない。ある場合は、再発防止策が実施されている",
		},
		coredata.ThirdPartyEvaluationItemReturnAndDeletion: {
			"返却・消去",
			"契約終了時に個人情報を返却又は消去し、消去を証明できる",
		},
		coredata.ThirdPartyEvaluationItemForeignStorage: {
			"外国でのデータの保管",
			"データを保管・処理する国が明らかである。外国の場合は、その国の個人情報の保護に関する制度を当社が把握している",
		},
		coredata.ThirdPartyEvaluationItemContract: {
			"契約",
			"『個人情報の取扱いに関する覚書』の内容を含む契約を結んでいる（又は結ぶ予定である）",
		},
	}
	thirdPartyEvaluationResultLabels = map[coredata.ThirdPartyEvaluationResult]string{
		coredata.ThirdPartyEvaluationResultSuitable:      "適",
		coredata.ThirdPartyEvaluationResultUnsuitable:    "否",
		coredata.ThirdPartyEvaluationResultNotApplicable: "該当なし",
	}
	thirdPartyEvaluationKindLabels = map[coredata.ThirdPartyEvaluationKind]string{
		coredata.ThirdPartyEvaluationKindNew:      "新規",
		coredata.ThirdPartyEvaluationKindPeriodic: "定期",
		coredata.ThirdPartyEvaluationKindAdHoc:    "臨時",
	}
	thirdPartyEvaluationJudgementLabels = map[coredata.ThirdPartyEvaluationJudgement]string{
		coredata.ThirdPartyEvaluationJudgementApproved:              "委託可",
		coredata.ThirdPartyEvaluationJudgementConditionallyApproved: "条件付きで委託可",
		coredata.ThirdPartyEvaluationJudgementRejected:              "委託不可",
	}
	thirdPartyEvaluationListColumns = []xlsxexport.Column{
		{Header: "No.", Width: 5},
		{Header: "委託先", Width: 18},
		{Header: "再委託元", Width: 14},
		{Header: "委託する業務", Width: 22},
		{Header: "取り扱う個人情報", Width: 22},
		{Header: "台帳の業務", Width: 18},
		{Header: "評価の種類", Width: 12},
		{Header: "評価日", Width: 11},
		{Header: "評価者", Width: 12},
		{Header: "評価結果", Width: 18},
		{Header: "総合判定", Width: 14},
		{Header: "条件・改善を求める事項", Width: 22},
		{Header: "承認者", Width: 12},
		{Header: "承認日", Width: 11},
		{Header: "次回評価予定", Width: 11},
		{Header: "確認事項", Width: 18},
	}
	thirdPartyEvaluationItemColumns = []xlsxexport.Column{
		{Header: "委託先", Width: 18},
		{Header: "評価日", Width: 11},
		{Header: "No.", Width: 5},
		{Header: "評価項目", Width: 16},
		{Header: "判定基準", Width: 48},
		{Header: "評価", Width: 9},
		{Header: "確認した資料・備考", Width: 32},
	}
)

// buildThirdPartyEvaluationSheets returns the list of third parties with
// their latest evaluation, and the checklist answers of those evaluations.
func buildThirdPartyEvaluationSheets(data thirdPartyEvaluationData, now time.Time) []xlsxexport.Sheet {
	subtitle := data.OrganizationName + "　出力日：" + now.Format(ledgerDateLayout)

	return []xlsxexport.Sheet{
		{
			Name:     thirdPartyEvaluationListSheetName,
			Title:    thirdPartyEvaluationTitle,
			Subtitle: subtitle,
			Columns:  thirdPartyEvaluationListColumns,
			Rows:     thirdPartyEvaluationListRows(data.Rows, now),
		},
		{
			Name:     thirdPartyEvaluationItemsSheetName,
			Title:    thirdPartyEvaluationTitle + "　" + thirdPartyEvaluationItemsSheetName,
			Subtitle: subtitle,
			Columns:  thirdPartyEvaluationItemColumns,
			Rows:     thirdPartyEvaluationItemRows(data.Rows),
		},
	}
}

func thirdPartyEvaluationListRows(rows []thirdPartyEvaluationRow, now time.Time) [][]string {
	if len(rows) == 0 {
		return [][]string{padRow([]string{"", thirdPartyEvaluationNoThirdParties}, len(thirdPartyEvaluationListColumns))}
	}

	result := make([][]string, 0, len(rows))
	for i, r := range rows {
		line := []string{
			strconv.Itoa(i + 1),
			r.ThirdParty.Name,
			r.ParentName,
		}

		e := r.Evaluation
		if e == nil {
			e = &coredata.ThirdPartyPrivacyEvaluation{}
		}

		line = append(line,
			e.EntrustedWork,
			e.PersonalData,
			strings.Join(r.ProcessingActivities, thirdPartyEvaluationSeparator),
			thirdPartyEvaluationKindLabel(e),
			thirdPartyEvaluationDate(r),
			r.EvaluatorName,
			thirdPartyEvaluationResultSummary(r),
			thirdPartyEvaluationJudgementLabel(r),
			e.Conditions,
			r.ApproverName,
			formatLedgerDate(e.ApprovedOn),
			thirdPartyEvaluationNextDate(r),
			strings.Join(thirdPartyEvaluationRemarks(r, now), thirdPartyEvaluationSeparator),
		)

		result = append(result, line)
	}

	return result
}

func thirdPartyEvaluationKindLabel(e *coredata.ThirdPartyPrivacyEvaluation) string {
	if e.Kind == nil {
		return ""
	}

	label := thirdPartyEvaluationKindLabels[*e.Kind]
	if *e.Kind == coredata.ThirdPartyEvaluationKindAdHoc && e.AdHocReason != "" {
		label += "（" + e.AdHocReason + "）"
	}

	return label
}

// thirdPartyEvaluationDate is the evaluation date entered, or else the date
// the risk assessment was recorded.
func thirdPartyEvaluationDate(r thirdPartyEvaluationRow) string {
	if r.Evaluation != nil && r.Evaluation.EvaluatedOn != nil {
		return formatLedgerDate(r.Evaluation.EvaluatedOn)
	}

	if r.Assessment == nil {
		return ""
	}

	createdAt := r.Assessment.CreatedAt.In(ledgerTimeZone)

	return formatLedgerDate(&createdAt)
}

func thirdPartyEvaluationNextDate(r thirdPartyEvaluationRow) string {
	if r.Assessment == nil {
		return ""
	}

	expiresAt := r.Assessment.ExpiresAt.In(ledgerTimeZone)

	return formatLedgerDate(&expiresAt)
}

// thirdPartyEvaluationResultSummary counts the answers, for example
// "適 11・否 1・該当なし 1".
func thirdPartyEvaluationResultSummary(r thirdPartyEvaluationRow) string {
	if r.Evaluation == nil {
		return ""
	}

	counts := make(map[coredata.ThirdPartyEvaluationResult]int)
	notAnswered := 0
	for _, a := range r.Answers {
		if a.Result == nil {
			notAnswered++
			continue
		}
		counts[*a.Result]++
	}

	parts := make([]string, 0, 4)
	for _, result := range coredata.ThirdPartyEvaluationResults() {
		parts = append(parts, fmt.Sprintf("%s %d", thirdPartyEvaluationResultLabels[result], counts[result]))
	}
	if notAnswered > 0 {
		parts = append(parts, fmt.Sprintf("%s %d", thirdPartyEvaluationNotAnswered, notAnswered))
	}

	return strings.Join(parts, "・")
}

func thirdPartyEvaluationJudgementLabel(r thirdPartyEvaluationRow) string {
	switch {
	case r.Assessment == nil:
		return thirdPartyEvaluationNotRegistered
	case r.Evaluation == nil:
		return thirdPartyEvaluationNotFilled
	case r.Evaluation.Judgement == nil:
		return thirdPartyEvaluationNotJudged
	}

	return thirdPartyEvaluationJudgementLabels[*r.Evaluation.Judgement]
}

// thirdPartyEvaluationRemarks lists what an auditor would ask about: a
// missing or expired evaluation, unanswered items and a missing approval.
func thirdPartyEvaluationRemarks(r thirdPartyEvaluationRow, now time.Time) []string {
	remarks := []string{}

	if r.Assessment == nil {
		return append(remarks, "評価がない")
	}

	if r.Assessment.ExpiresAt.Before(now) {
		remarks = append(remarks, "次回評価予定を過ぎている")
	}

	if r.Evaluation == nil {
		return append(remarks, "チェックリストが未記入")
	}

	for _, a := range r.Answers {
		if a.Result == nil {
			remarks = append(remarks, "未評価の項目がある")
			break
		}
	}

	if r.Evaluation.Judgement != nil && r.Evaluation.ApproverProfileID == nil {
		remarks = append(remarks, "承認者が未記入")
	}

	return remarks
}

func thirdPartyEvaluationItemRows(rows []thirdPartyEvaluationRow) [][]string {
	result := [][]string{}
	for _, r := range rows {
		if r.Evaluation == nil {
			continue
		}

		date := thirdPartyEvaluationDate(r)
		for i, a := range r.Answers {
			label := thirdPartyEvaluationItemLabels[a.Item]

			resultLabel := thirdPartyEvaluationNotAnswered
			if a.Result != nil {
				resultLabel = thirdPartyEvaluationResultLabels[*a.Result]
			}

			result = append(result, []string{
				r.ThirdParty.Name,
				date,
				strconv.Itoa(i + 1),
				label.Label,
				label.Criterion,
				resultLabel,
				thirdPartyEvaluationAnswerNote(r.Evaluation, a),
			})
		}
	}

	if len(result) == 0 {
		return [][]string{padRow([]string{thirdPartyEvaluationNoEvaluations}, len(thirdPartyEvaluationItemColumns))}
	}

	return result
}

// thirdPartyEvaluationAnswerNote is the note of the answer. The foreign
// storage item also names the countries where the data is stored, as the
// checklist does.
func thirdPartyEvaluationAnswerNote(e *coredata.ThirdPartyPrivacyEvaluation, a ThirdPartyEvaluationAnswer) string {
	if a.Item != coredata.ThirdPartyEvaluationItemForeignStorage || e.StorageCountries == "" {
		return a.Note
	}

	countries := "保管する国：" + e.StorageCountries
	if a.Note == "" {
		return countries
	}

	return countries + "\n" + a.Note
}
