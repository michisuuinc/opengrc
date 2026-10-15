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
)

func TestBuildTrainingSheet(t *testing.T) {
	t.Parallel()

	score := 85
	passScore := 80
	passed := true
	department := "営業部"
	signedOn := trainingDate(t, "2026-10-04")
	source := coredata.TrainingAttendanceSourceSignature

	plan := &coredata.TrainingPlan{
		FiscalYear:   2026,
		Name:         "個人情報保護 定期教育",
		Kind:         coredata.TrainingPlanKindPeriodic,
		Method:       coredata.TrainingPlanMethodELearning,
		Instructor:   "個人情報保護管理者",
		StartsOn:     trainingDate(t, "2026-10-01"),
		EndsOn:       trainingDate(t, "2026-10-31"),
		TimeZone:     "Asia/Tokyo",
		TestRequired: true,
		PassScore:    &passScore,
	}
	emptyPlan := &coredata.TrainingPlan{
		FiscalYear: 2026,
		Name:       "新規入社者教育",
		Kind:       coredata.TrainingPlanKindOnboarding,
		Method:     coredata.TrainingPlanMethodClassroom,
		StartsOn:   trainingDate(t, "2026-04-01"),
		EndsOn:     trainingDate(t, "2027-03-31"),
		TimeZone:   "Asia/Tokyo",
	}

	data := trainingSheetData{
		OrganizationName: "テスト株式会社",
		Plans: []trainingSheetPlan{
			{
				Plan: plan,
				Participants: []TrainingParticipantProgress{
					{
						Participant: &coredata.TrainingParticipantDetail{
							TrainingParticipant: coredata.TrainingParticipant{TestScore: &score, Note: "=1+1"},
							FullName:            "山田 太郎",
							Department:          &department,
						},
						Progress: TrainingProgress{
							AttendedOn: &signedOn,
							Source:     &source,
							TestPassed: &passed,
							Status:     coredata.TrainingParticipantStatusCompleted,
						},
					},
					{
						Participant: &coredata.TrainingParticipantDetail{FullName: "佐藤 花子"},
						Progress:    TrainingProgress{Status: coredata.TrainingParticipantStatusNotAttended},
					},
				},
			},
			{Plan: emptyPlan},
		},
	}

	now := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	sheet := buildTrainingSheet(data, now)

	assert.Equal(t, "教育実施記録", sheet.Name)
	assert.Equal(t, "テスト株式会社　出力日：2026-11-02", sheet.Subtitle)
	require.Len(t, sheet.Rows, 3)

	header := make([]string, len(sheet.Columns))
	for i, c := range sheet.Columns {
		header[i] = c.Header
	}
	assert.Equal(t, []string{
		"No.", "年度", "教育の名前", "種類", "実施期間", "方法", "講師",
		"氏名", "部門", "受講日", "受講の確認", "点数", "合否", "状況", "備考",
	}, header)

	assert.Equal(t, []string{
		"1", "2026", "個人情報保護 定期教育", "定期教育", "2026-10-01〜2026-10-31", "e ラーニング", "個人情報保護管理者",
		"山田 太郎", "営業部", "2026-10-04", "署名", "85", "合格", "完了", "=1+1",
	}, sheet.Rows[0])
	assert.Equal(t, []string{
		"2", "2026", "個人情報保護 定期教育", "定期教育", "2026-10-01〜2026-10-31", "e ラーニング", "個人情報保護管理者",
		"佐藤 花子", "", "", "", "", "", "未受講", "",
	}, sheet.Rows[1])
	assert.Equal(t, "（受講者未登録）", sheet.Rows[2][7])
	assert.Equal(t, "新規入社者教育", sheet.Rows[2][2])
}
