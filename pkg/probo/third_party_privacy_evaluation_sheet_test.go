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

func evaluationTestPtr[T any](v T) *T {
	return &v
}

func evaluationTestThirdParty(name string) *coredata.ThirdParty {
	return &coredata.ThirdParty{
		ID:   gid.New(gid.NewTenantID(), coredata.ThirdPartyEntityType),
		Name: name,
	}
}

// evaluationTestAnswers answers every item as suitable, except the given
// ones.
func evaluationTestAnswers(overrides map[coredata.ThirdPartyEvaluationItem]*coredata.ThirdPartyEvaluationResult) []ThirdPartyEvaluationAnswer {
	answers := checklistAnswers(nil)
	for i := range answers {
		result, ok := overrides[answers[i].Item]
		if !ok {
			result = evaluationTestPtr(coredata.ThirdPartyEvaluationResultSuitable)
		}
		answers[i].Result = result
	}

	return answers
}

func evaluationTestData(now time.Time) thirdPartyEvaluationData {
	tenantID := gid.NewTenantID()
	evaluator := gid.New(tenantID, coredata.MembershipProfileEntityType)
	approver := gid.New(tenantID, coredata.MembershipProfileEntityType)

	cloud := evaluationTestThirdParty("クラウド会社")
	payroll := evaluationTestThirdParty("給与計算会社")
	sub := evaluationTestThirdParty("データセンター")
	sub.ParentThirdPartyID = &cloud.ID
	shredder := evaluationTestThirdParty("廃棄業者")

	return thirdPartyEvaluationData{
		OrganizationName: "サンプル株式会社",
		Rows: []thirdPartyEvaluationRow{
			{
				ThirdParty:           cloud,
				ProcessingActivities: []string{"顧客管理", "問い合わせ対応"},
				Assessment: &coredata.ThirdPartyRiskAssessment{
					CreatedAt: time.Date(2026, 4, 1, 1, 0, 0, 0, time.UTC),
					ExpiresAt: time.Date(2027, 3, 31, 15, 0, 0, 0, time.UTC),
				},
				Evaluation: &coredata.ThirdPartyPrivacyEvaluation{
					Kind:               evaluationTestPtr(coredata.ThirdPartyEvaluationKindAdHoc),
					AdHocReason:        "再委託先の変更",
					EntrustedWork:      "顧客データの保管",
					PersonalData:       "氏名、メールアドレス（約 1 万件）",
					StorageCountries:   "日本、米国",
					EvaluatorProfileID: &evaluator,
					EvaluatedOn:        evaluationTestPtr(time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)),
					Judgement:          evaluationTestPtr(coredata.ThirdPartyEvaluationJudgementConditionallyApproved),
					Conditions:         "米国の制度を確認する",
					ApproverProfileID:  &approver,
					ApprovedOn:         evaluationTestPtr(time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)),
				},
				Answers: evaluationTestAnswers(map[coredata.ThirdPartyEvaluationItem]*coredata.ThirdPartyEvaluationResult{
					coredata.ThirdPartyEvaluationItemForeignStorage: evaluationTestPtr(coredata.ThirdPartyEvaluationResultUnsuitable),
					coredata.ThirdPartyEvaluationItemCertification:  evaluationTestPtr(coredata.ThirdPartyEvaluationResultNotApplicable),
				}),
				EvaluatorName: "評価 一郎",
				ApproverName:  "管理 花子",
			},
			{
				ThirdParty: sub,
				ParentName: "クラウド会社",
				Assessment: &coredata.ThirdPartyRiskAssessment{
					CreatedAt: time.Date(2025, 4, 1, 1, 0, 0, 0, time.UTC),
					ExpiresAt: now.Add(-24 * time.Hour),
				},
				Evaluation: &coredata.ThirdPartyPrivacyEvaluation{
					Judgement: evaluationTestPtr(coredata.ThirdPartyEvaluationJudgementApproved),
				},
				Answers: evaluationTestAnswers(map[coredata.ThirdPartyEvaluationItem]*coredata.ThirdPartyEvaluationResult{
					coredata.ThirdPartyEvaluationItemContract: nil,
				}),
			},
			{
				ThirdParty:           payroll,
				ProcessingActivities: []string{"給与計算"},
				Assessment: &coredata.ThirdPartyRiskAssessment{
					CreatedAt: time.Date(2026, 5, 1, 1, 0, 0, 0, time.UTC),
					ExpiresAt: time.Date(2027, 5, 1, 1, 0, 0, 0, time.UTC),
				},
			},
			{
				ThirdParty:           shredder,
				ProcessingActivities: []string{"書類の廃棄"},
			},
		},
	}
}

func TestBuildThirdPartyEvaluationSheets(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 15, 9, 0, 0, 0, ledgerTimeZone)
	sheets := buildThirdPartyEvaluationSheets(evaluationTestData(now), now)

	require.Len(t, sheets, 2)

	list := sheets[0]
	assert.Equal(t, "委託先一覧", list.Name)
	assert.Equal(t, "委託先評価表", list.Title)
	assert.Equal(t, "サンプル株式会社　出力日：2026-10-15", list.Subtitle)
	require.Len(t, list.Rows, 4)
	for _, row := range list.Rows {
		assert.Len(t, row, len(list.Columns))
	}

	assert.Equal(
		t,
		[]string{
			"1",
			"クラウド会社",
			"",
			"顧客データの保管",
			"氏名、メールアドレス（約 1 万件）",
			"顧客管理、問い合わせ対応",
			"臨時（再委託先の変更）",
			"2026-04-02",
			"評価 一郎",
			"適 11・否 1・該当なし 1",
			"条件付きで委託可",
			"米国の制度を確認する",
			"管理 花子",
			"2026-04-03",
			"2027-04-01",
			"",
		},
		list.Rows[0],
	)

	sub := list.Rows[1]
	assert.Equal(t, "クラウド会社", sub[2], "the parent is named")
	assert.Equal(t, "2025-04-01", sub[7], "the assessment date stands in for the evaluation date")
	assert.Equal(t, "適 12・否 0・該当なし 0・未評価 1", sub[9])
	assert.Equal(t, "委託可", sub[10])
	assert.Equal(t, "次回評価予定を過ぎている、未評価の項目がある、承認者が未記入", sub[15])

	payroll := list.Rows[2]
	assert.Equal(t, "", payroll[9])
	assert.Equal(t, "（評価未記入）", payroll[10])
	assert.Equal(t, "チェックリストが未記入", payroll[15])

	shredder := list.Rows[3]
	assert.Equal(t, "書類の廃棄", shredder[5])
	assert.Equal(t, "（評価未登録）", shredder[10])
	assert.Equal(t, "評価がない", shredder[15])

	items := sheets[1]
	assert.Equal(t, "評価項目", items.Name)
	require.Len(t, items.Rows, 26, "13 items for each of the two evaluations")
	assert.Equal(
		t,
		[]string{"クラウド会社", "2026-04-02", "1", "認証の取得", thirdPartyEvaluationItemLabels[coredata.ThirdPartyEvaluationItemCertification].Criterion, "該当なし", ""},
		items.Rows[0],
	)
	assert.Equal(t, "外国でのデータの保管", items.Rows[11][3])
	assert.Equal(t, "否", items.Rows[11][5])
	assert.Equal(t, "保管する国：日本、米国", items.Rows[11][6])
	assert.Equal(t, "データセンター", items.Rows[25][0])
	assert.Equal(t, "契約", items.Rows[25][3])
	assert.Equal(t, "未評価", items.Rows[25][5])

	_, err := xlsxexport.BuildWorkbook(sheets...)
	require.NoError(t, err)
}

func TestBuildThirdPartyEvaluationSheetsEmpty(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 15, 9, 0, 0, 0, ledgerTimeZone)
	sheets := buildThirdPartyEvaluationSheets(thirdPartyEvaluationData{OrganizationName: "サンプル株式会社"}, now)

	require.Len(t, sheets, 2)
	require.Len(t, sheets[0].Rows, 1)
	assert.Equal(t, "（委託先なし）", sheets[0].Rows[0][1])
	assert.Len(t, sheets[0].Rows[0], len(sheets[0].Columns))
	require.Len(t, sheets[1].Rows, 1)
	assert.Equal(t, "（評価の記入なし）", sheets[1].Rows[0][0])
	assert.Len(t, sheets[1].Rows[0], len(sheets[1].Columns))

	_, err := xlsxexport.BuildWorkbook(sheets...)
	require.NoError(t, err)
}

func TestThirdPartyEvaluationItemLabels(t *testing.T) {
	t.Parallel()

	for _, item := range coredata.ThirdPartyEvaluationItems() {
		label, ok := thirdPartyEvaluationItemLabels[item]
		assert.True(t, ok, "item %s has a label", item)
		assert.NotEmpty(t, label.Label)
		assert.NotEmpty(t, label.Criterion)
	}
}

func evaluationTestRequest() *UpdateThirdPartyPrivacyEvaluationRequest {
	return &UpdateThirdPartyPrivacyEvaluationRequest{
		RiskAssessmentID: gid.New(gid.NewTenantID(), coredata.ThirdPartyRiskAssessmentEntityType),
	}
}

func TestUpdateThirdPartyPrivacyEvaluationRequestConsistency(t *testing.T) {
	t.Parallel()

	unsuitable := []ThirdPartyEvaluationAnswer{{
		Item:   coredata.ThirdPartyEvaluationItemContract,
		Result: evaluationTestPtr(coredata.ThirdPartyEvaluationResultUnsuitable),
	}}

	tests := []struct {
		name    string
		modify  func(r *UpdateThirdPartyPrivacyEvaluationRequest)
		wantErr bool
	}{
		{"empty is fine", func(r *UpdateThirdPartyPrivacyEvaluationRequest) {}, false},
		{
			"approved with an unsuitable item",
			func(r *UpdateThirdPartyPrivacyEvaluationRequest) {
				r.Judgement = evaluationTestPtr(coredata.ThirdPartyEvaluationJudgementApproved)
				r.Answers = unsuitable
			},
			true,
		},
		{
			"conditionally approved with an unsuitable item",
			func(r *UpdateThirdPartyPrivacyEvaluationRequest) {
				r.Judgement = evaluationTestPtr(coredata.ThirdPartyEvaluationJudgementConditionallyApproved)
				r.Conditions = "覚書を結ぶ"
				r.Answers = unsuitable
			},
			false,
		},
		{
			"conditionally approved without conditions",
			func(r *UpdateThirdPartyPrivacyEvaluationRequest) {
				r.Judgement = evaluationTestPtr(coredata.ThirdPartyEvaluationJudgementConditionallyApproved)
				r.Conditions = "   "
			},
			true,
		},
		{
			"ad hoc without a reason",
			func(r *UpdateThirdPartyPrivacyEvaluationRequest) {
				r.Kind = evaluationTestPtr(coredata.ThirdPartyEvaluationKindAdHoc)
			},
			true,
		},
		{
			"approved before evaluated",
			func(r *UpdateThirdPartyPrivacyEvaluationRequest) {
				r.EvaluatedOn = evaluationTestPtr(time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC))
				r.ApprovedOn = evaluationTestPtr(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
			},
			true,
		},
		{
			"approved on the evaluation day",
			func(r *UpdateThirdPartyPrivacyEvaluationRequest) {
				r.EvaluatedOn = evaluationTestPtr(time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC))
				r.ApprovedOn = evaluationTestPtr(time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC))
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Parallel()

				req := evaluationTestRequest()
				tt.modify(req)

				normalized := req.normalize()
				require.NoError(t, normalized.Validate())

				err := normalized.checkConsistency()
				if tt.wantErr {
					var inconsistent *ErrThirdPartyEvaluationInconsistent
					assert.ErrorAs(t, err, &inconsistent)
				} else {
					assert.NoError(t, err)
				}
			},
		)
	}
}

func TestUpdateThirdPartyPrivacyEvaluationRequestNormalize(t *testing.T) {
	t.Parallel()

	req := evaluationTestRequest()
	req.Kind = evaluationTestPtr(coredata.ThirdPartyEvaluationKindPeriodic)
	req.AdHocReason = "使われない理由"
	req.EntrustedWork = "  発送代行  "
	req.Answers = []ThirdPartyEvaluationAnswer{
		{Item: coredata.ThirdPartyEvaluationItemRules, Note: "  "},
		{Item: coredata.ThirdPartyEvaluationItemContract, Note: " 覚書あり "},
	}

	normalized := req.normalize()

	assert.Equal(t, "", normalized.AdHocReason, "a reason is kept only for an ad hoc evaluation")
	assert.Equal(t, "発送代行", normalized.EntrustedWork)
	require.Len(t, normalized.Answers, 1, "an answer with neither result nor note is dropped")
	assert.Equal(t, coredata.ThirdPartyEvaluationItemContract, normalized.Answers[0].Item)
	assert.Equal(t, "覚書あり", normalized.Answers[0].Note)
	assert.Equal(t, "使われない理由", req.AdHocReason, "the request is not changed")
}

func TestUpdateThirdPartyPrivacyEvaluationRequestValidate(t *testing.T) {
	t.Parallel()

	t.Run(
		"repeated item",
		func(t *testing.T) {
			t.Parallel()

			req := evaluationTestRequest()
			req.Answers = []ThirdPartyEvaluationAnswer{
				{Item: coredata.ThirdPartyEvaluationItemRules, Note: "a"},
				{Item: coredata.ThirdPartyEvaluationItemRules, Note: "b"},
			}

			_, ok := req.Validate().(validator.ValidationErrors)
			assert.True(t, ok)
		},
	)

	t.Run(
		"unknown item",
		func(t *testing.T) {
			t.Parallel()

			req := evaluationTestRequest()
			req.Answers = []ThirdPartyEvaluationAnswer{{Item: "UNKNOWN", Note: "a"}}

			assert.Error(t, req.Validate())
		},
	)

	t.Run(
		"evaluator of another entity type",
		func(t *testing.T) {
			t.Parallel()

			req := evaluationTestRequest()
			req.EvaluatorID = evaluationTestPtr(gid.New(gid.NewTenantID(), coredata.ThirdPartyEntityType))

			assert.Error(t, req.Validate())
		},
	)
}
