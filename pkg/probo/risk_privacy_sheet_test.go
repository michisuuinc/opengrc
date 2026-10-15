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
)

func riskPrivacyTestRisk(referenceID, name string) *coredata.Risk {
	return &coredata.Risk{
		ID:          gid.New(gid.NewTenantID(), coredata.RiskEntityType),
		ReferenceID: referenceID,
		Name:        name,
	}
}

func riskPrivacyRows(t *testing.T, data riskPrivacyData) []map[string]string {
	t.Helper()

	sheet := buildRiskPrivacySheet(data, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))

	rows := make([]map[string]string, len(sheet.Rows))
	for i, cells := range sheet.Rows {
		require.Len(t, cells, len(sheet.Columns))
		row := make(map[string]string, len(sheet.Columns))
		for j, c := range sheet.Columns {
			row[c.Header] = cells[j]
		}
		rows[i] = row
	}

	return rows
}

func TestBuildRiskPrivacySheet_Heading(t *testing.T) {
	t.Parallel()

	sheet := buildRiskPrivacySheet(
		riskPrivacyData{OrganizationName: "サンプル株式会社"},
		time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC),
	)

	assert.Equal(t, "リスク分析表", sheet.Name)
	assert.Equal(t, "リスク分析表", sheet.Title)
	assert.Equal(t, "サンプル株式会社　出力日：2026-10-15", sheet.Subtitle)
	assert.Equal(t, "No.", sheet.Columns[0].Header)
	assert.Len(t, sheet.Columns, len(riskPrivacyColumns)+3)
	assert.Empty(t, sheet.Rows)
}

func TestBuildRiskPrivacySheet_Rows(t *testing.T) {
	t.Parallel()

	t.Run(
		"fills one row per register entry and risk",
		func(t *testing.T) {
			t.Parallel()

			a := ledgerTestActivity("採用選考")
			a.SpecialOrCriminalData = coredata.ProcessingActivitySpecialOrCriminalDatumYes

			leak := riskPrivacyTestRisk("R-002", "応募書類の誤送信")
			leak.Description = new("メールの宛先を誤る")
			leak.InherentLikelihood = new(3)
			leak.InherentImpact = new(4)
			leak.InherentRiskScore = new(12)
			leak.ResidualLikelihood = new(1)
			leak.ResidualImpact = new(4)
			leak.ResidualRiskScore = new(4)
			leak.Treatment = new(coredata.RiskTreatmentMitigated)
			leak.Note = "年1回見直し"
			ownerID := gid.New(gid.NewTenantID(), coredata.MembershipProfileEntityType)
			leak.OwnerID = &ownerID

			disposal := riskPrivacyTestRisk("R-001", "不採用者の書類の廃棄漏れ")
			disposal.Treatment = new(coredata.RiskTreatmentAccepted)

			rows := riskPrivacyRows(t, riskPrivacyData{
				Activities: []*coredata.ProcessingActivity{a},
				Risks:      map[gid.GID]*coredata.Risk{leak.ID: leak, disposal.ID: disposal},
				RiskIDs:    map[gid.GID][]gid.GID{a.ID: {disposal.ID, leak.ID}},
				Stages: map[gid.GID][]coredata.PersonalDataLifecycleStage{
					leak.ID: {
						coredata.PersonalDataLifecycleStageUse,
						coredata.PersonalDataLifecycleStageTransfer,
					},
					disposal.ID: {coredata.PersonalDataLifecycleStageDisposal},
				},
				InternalControlNames: map[gid.GID][]string{leak.ID: {"メール送信前の宛先確認", "添付ファイルの暗号化"}},
				OwnerNames:           map[gid.GID]string{ownerID: "山田 太郎"},
			})

			require.Len(t, rows, 2)

			first := rows[0]
			assert.Equal(t, "1", first["No."])
			assert.Equal(t, "採用選考", first["業務の名称"])
			assert.Equal(t, "あり", first["要配慮個人情報"])
			assert.Equal(t, "R-002", first["リスクID"], "the risk with the earliest stage comes first")
			assert.Equal(t, "移送・送信\n利用・加工", first["段階"])
			assert.Equal(t, "応募書類の誤送信", first["想定されるリスク"])
			assert.Equal(t, "メールの宛先を誤る", first["リスクの内容"])
			assert.Equal(t, "3", first["発生可能性"])
			assert.Equal(t, "4", first["影響度"])
			assert.Equal(t, "12", first["リスク値"])
			assert.Equal(t, "低減", first["対応方針"])
			assert.Equal(t, "メール送信前の宛先確認\n添付ファイルの暗号化", first["対策（内部統制）"])
			assert.Equal(t, "1", first["残留リスクの発生可能性"])
			assert.Equal(t, "4", first["残留リスクの影響度"])
			assert.Equal(t, "4", first["残留リスク値"])
			assert.Equal(t, "山田 太郎", first["責任者"])
			assert.Equal(t, "年1回見直し", first["備考"])

			second := rows[1]
			assert.Equal(t, "2", second["No."])
			assert.Equal(t, "採用選考", second["業務の名称"])
			assert.Equal(t, "R-001", second["リスクID"])
			assert.Equal(t, "消去・廃棄", second["段階"])
			assert.Equal(t, "受容", second["対応方針"])
			for _, header := range []string{"発生可能性", "影響度", "リスク値", "対策（内部統制）", "残留リスク値", "責任者", "リスクの内容"} {
				assert.Empty(t, second[header], header)
			}
		},
	)

	t.Run(
		"marks register entries with no risk",
		func(t *testing.T) {
			t.Parallel()

			a := ledgerTestActivity("問い合わせ対応")

			rows := riskPrivacyRows(t, riskPrivacyData{Activities: []*coredata.ProcessingActivity{a}})

			require.Len(t, rows, 1)
			assert.Equal(t, "問い合わせ対応", rows[0]["業務の名称"])
			assert.Equal(t, "（リスク未登録）", rows[0]["想定されるリスク"])
			assert.Empty(t, rows[0]["リスクID"])
			assert.Empty(t, rows[0]["段階"])
		},
	)

	t.Run(
		"puts risks with no stage last and numbers rows across entries",
		func(t *testing.T) {
			t.Parallel()

			a := ledgerTestActivity("顧客管理")
			b := ledgerTestActivity("従業員管理")
			noStage := riskPrivacyTestRisk("R-001", "目的外利用")
			collection := riskPrivacyTestRisk("R-009", "不正な取得")

			rows := riskPrivacyRows(t, riskPrivacyData{
				Activities: []*coredata.ProcessingActivity{a, b},
				Risks:      map[gid.GID]*coredata.Risk{noStage.ID: noStage, collection.ID: collection},
				RiskIDs: map[gid.GID][]gid.GID{
					a.ID: {noStage.ID, collection.ID},
					b.ID: {noStage.ID},
				},
				Stages: map[gid.GID][]coredata.PersonalDataLifecycleStage{
					collection.ID: {coredata.PersonalDataLifecycleStageCollection},
				},
			})

			require.Len(t, rows, 3)
			assert.Equal(t, []string{"R-009", "R-001", "R-001"}, []string{rows[0]["リスクID"], rows[1]["リスクID"], rows[2]["リスクID"]})
			assert.Equal(t, []string{"顧客管理", "顧客管理", "従業員管理"}, []string{rows[0]["業務の名称"], rows[1]["業務の名称"], rows[2]["業務の名称"]})
			assert.Equal(t, "3", rows[2]["No."])
		},
	)
}

func TestRiskPrivacyTreatmentLabel(t *testing.T) {
	t.Parallel()

	assert.Empty(t, riskPrivacyTreatmentLabel(nil))
	assert.Equal(t, "移転", riskPrivacyTreatmentLabel(new(coredata.RiskTreatmentTransferred)))
	assert.Equal(t, "回避", riskPrivacyTreatmentLabel(new(coredata.RiskTreatmentAvoided)))
}

func TestUpdateRiskPrivacyDetailsRequest_Validate(t *testing.T) {
	t.Parallel()

	riskID := gid.New(gid.NewTenantID(), coredata.RiskEntityType)
	activityID := gid.New(gid.NewTenantID(), coredata.ProcessingActivityEntityType)

	valid := &UpdateRiskPrivacyDetailsRequest{
		RiskID:                riskID,
		LifecycleStages:       &[]coredata.PersonalDataLifecycleStage{coredata.PersonalDataLifecycleStageUse},
		ProcessingActivityIDs: &[]gid.GID{activityID},
	}
	assert.NoError(t, valid.Validate())

	duplicated := &UpdateRiskPrivacyDetailsRequest{
		RiskID: riskID,
		LifecycleStages: &[]coredata.PersonalDataLifecycleStage{
			coredata.PersonalDataLifecycleStageUse,
			coredata.PersonalDataLifecycleStageUse,
		},
	}
	assert.Error(t, duplicated.Validate())

	unknownStage := &UpdateRiskPrivacyDetailsRequest{
		RiskID:          riskID,
		LifecycleStages: &[]coredata.PersonalDataLifecycleStage{"ARCHIVE"},
	}
	assert.Error(t, unknownStage.Validate())

	wrongType := &UpdateRiskPrivacyDetailsRequest{
		RiskID:                riskID,
		ProcessingActivityIDs: &[]gid.GID{riskID},
	}
	assert.Error(t, wrongType.Validate())
}
