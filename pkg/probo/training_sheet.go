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
	"time"

	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/xlsxexport"
)

const (
	trainingSheetName      = "教育実施記録"
	trainingNoParticipants = "（受講者未登録）"
)

type (
	trainingSheetPlan struct {
		Plan         *coredata.TrainingPlan
		Participants []TrainingParticipantProgress
	}

	trainingSheetData struct {
		OrganizationName string
		Plans            []trainingSheetPlan
	}
)

var (
	trainingKindLabels = map[coredata.TrainingPlanKind]string{
		coredata.TrainingPlanKindPeriodic:   "定期教育",
		coredata.TrainingPlanKindOnboarding: "新規入社者教育",
		coredata.TrainingPlanKindRoleBased:  "役割別教育",
		coredata.TrainingPlanKindAdHoc:      "臨時教育",
	}
	trainingMethodLabels = map[coredata.TrainingPlanMethod]string{
		coredata.TrainingPlanMethodClassroom: "集合研修",
		coredata.TrainingPlanMethodOnline:    "オンライン研修",
		coredata.TrainingPlanMethodELearning: "e ラーニング",
		coredata.TrainingPlanMethodMaterials: "資料の配布と説明",
	}
	trainingStatusLabels = map[coredata.TrainingParticipantStatus]string{
		coredata.TrainingParticipantStatusCompleted:   "完了",
		coredata.TrainingParticipantStatusTestPending: "テスト未実施",
		coredata.TrainingParticipantStatusFailed:      "不合格",
		coredata.TrainingParticipantStatusNotAttended: "未受講",
	}
	trainingSourceLabels = map[coredata.TrainingAttendanceSource]string{
		coredata.TrainingAttendanceSourceSignature: "署名",
		coredata.TrainingAttendanceSourceManual:    "手入力",
	}
	trainingColumns = []xlsxexport.Column{
		{Header: "No.", Width: 5},
		{Header: "年度", Width: 7},
		{Header: "教育の名前", Width: 24},
		{Header: "種類", Width: 12},
		{Header: "実施期間", Width: 22},
		{Header: "方法", Width: 12},
		{Header: "講師", Width: 14},
		{Header: "氏名", Width: 14},
		{Header: "部門", Width: 12},
		{Header: "受講日", Width: 11},
		{Header: "受講の確認", Width: 9},
		{Header: "点数", Width: 6},
		{Header: "合否", Width: 7},
		{Header: "状況", Width: 10},
		{Header: "備考", Width: 20},
	}
)

// buildTrainingSheet lists one row per plan and participant. A plan with no
// participant gets one row so that missing records show up.
func buildTrainingSheet(data trainingSheetData, now time.Time) xlsxexport.Sheet {
	rows := make([][]string, 0)

	for _, p := range data.Plans {
		planCells := trainingPlanCells(p.Plan)

		if len(p.Participants) == 0 {
			row := append([]string{strconv.Itoa(len(rows) + 1)}, planCells...)
			row = append(row, trainingNoParticipants)
			rows = append(rows, padRow(row, len(trainingColumns)))
			continue
		}

		for _, participant := range p.Participants {
			row := append([]string{strconv.Itoa(len(rows) + 1)}, planCells...)
			row = append(row, trainingParticipantCells(participant)...)
			rows = append(rows, row)
		}
	}

	return xlsxexport.Sheet{
		Name:     trainingSheetName,
		Title:    trainingSheetName,
		Subtitle: data.OrganizationName + "　出力日：" + now.Format(ledgerDateLayout),
		Columns:  trainingColumns,
		Rows:     rows,
	}
}

func trainingPlanCells(plan *coredata.TrainingPlan) []string {
	return []string{
		strconv.Itoa(plan.FiscalYear),
		plan.Name,
		trainingKindLabels[plan.Kind],
		plan.StartsOn.Format(ledgerDateLayout) + "〜" + plan.EndsOn.Format(ledgerDateLayout),
		trainingMethodLabels[plan.Method],
		plan.Instructor,
	}
}

func trainingParticipantCells(p TrainingParticipantProgress) []string {
	participant := p.Participant

	cells := []string{participant.FullName, ledgerBlank, ledgerBlank, ledgerBlank, ledgerBlank, ledgerBlank}
	if participant.Department != nil {
		cells[1] = *participant.Department
	}
	if p.Progress.AttendedOn != nil {
		cells[2] = p.Progress.AttendedOn.Format(ledgerDateLayout)
	}
	if p.Progress.Source != nil {
		cells[3] = trainingSourceLabels[*p.Progress.Source]
	}
	if participant.TestScore != nil {
		cells[4] = strconv.Itoa(*participant.TestScore)
	}
	if p.Progress.TestPassed != nil {
		cells[5] = "不合格"
		if *p.Progress.TestPassed {
			cells[5] = "合格"
		}
	}

	return append(cells, trainingStatusLabels[p.Progress.Status], participant.Note)
}
