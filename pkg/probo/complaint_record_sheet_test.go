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
	"go.probo.inc/probo/pkg/xlsxexport"
)

func TestBuildComplaintRecordSheets(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 15, 9, 0, 0, 0, ledgerTimeZone)

	complaint := recordTestRequest(coredata.RightsRequestTypeComplaint, coredata.RightsRequestStateDone)
	content := "メールの誤送信について"
	complaint.Details = &content
	action := "経緯を説明し、再発防止策を伝えた"
	complaint.ActionTaken = &action

	consultation := recordTestRequest(coredata.RightsRequestTypeComplaint, coredata.RightsRequestStateInProgress)
	unlabelled := recordTestRequest(coredata.RightsRequestTypeComplaint, coredata.RightsRequestStateTodo)

	data := rightsRequestRecordData{
		OrganizationName: "サンプル株式会社",
		Records: []rightsRequestRecord{
			{
				Request: complaint,
				Details: &coredata.RightsRequestPrivacyDetails{
					ReceivedOn:        recordTestDate(4, 1),
					ReceiptChannel:    evaluationTestPtr(coredata.RightsRequestReceiptChannelEmail),
					TargetDepartment:  "営業部",
					FirstRespondedOn:  recordTestDate(4, 3),
					AnsweredOn:        recordTestDate(4, 20),
					ApproverProfileID: evaluationTestPtr(gid.New(gid.NewTenantID(), coredata.MembershipProfileEntityType)),
					FindingID:         evaluationTestPtr(gid.New(gid.NewTenantID(), coredata.FindingEntityType)),
				},
				ReceivedOn:       *recordTestDate(4, 1),
				HandlerName:      "窓口 一郎",
				ApproverName:     "管理 花子",
				FindingReference: "FND-3",
			},
			{
				Request: consultation,
				Details: &coredata.RightsRequestPrivacyDetails{
					Category:         evaluationTestPtr(coredata.RightsRequestPrivacyCategoryConsultation),
					TargetDepartment: "営業部",
				},
				ReceivedOn: *recordTestDate(5, 1),
			},
			{Request: unlabelled, ReceivedOn: *recordTestDate(6, 1)},
		},
	}

	sheets := buildComplaintRecordSheets(data, now)
	require.Len(t, sheets, 2)

	records := sheets[0]
	assert.Equal(t, "苦情・相談の記録", records.Name)
	require.Len(t, records.Rows, 3)
	for _, row := range records.Rows {
		assert.Len(t, row, len(records.Columns))
	}

	assert.Equal(
		t,
		[]string{
			"1", "2026-04-01", "メール", "苦情", "山田 太郎", "メールの誤送信について", "営業部", "窓口 一郎",
			"2026-04-03", "2 日", "完了", "経緯を説明し、再発防止策を伝えた", "2026-04-20", "管理 花子", "FND-3", "",
		},
		records.Rows[0],
	)
	assert.Equal(t, "相談", records.Rows[1][3])
	assert.Equal(t, "一次回答が未記入", records.Rows[1][15], "no identity verification is asked for")
	assert.Equal(t, "記録の項目が未記入", records.Rows[2][15])

	departments := sheets[1]
	assert.Equal(t, "部門別の件数", departments.Name)
	assert.Equal(
		t,
		[][]string{
			{"営業部", "1", "1", "2", "1", "1"},
			{"（部門未記入）", "1", "0", "1", "1", "0"},
			{"合計", "2", "1", "3", "2", "1"},
		},
		departments.Rows,
	)

	_, err := xlsxexport.BuildWorkbook(sheets...)
	require.NoError(t, err)
}

func TestBuildComplaintRecordSheetsEmpty(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 15, 9, 0, 0, 0, ledgerTimeZone)
	sheets := buildComplaintRecordSheets(rightsRequestRecordData{OrganizationName: "サンプル株式会社"}, now)

	require.Len(t, sheets[0].Rows, 1)
	assert.Equal(t, "（苦情・相談なし）", sheets[0].Rows[0][5])
	assert.Len(t, sheets[0].Rows[0], len(sheets[0].Columns))
	assert.Equal(t, [][]string{{"合計", "0", "0", "0", "0", "0"}}, sheets[1].Rows)

	_, err := xlsxexport.BuildWorkbook(sheets...)
	require.NoError(t, err)
}

func TestUpdateRightsRequestPrivacyDetailsRequestFirstResponse(t *testing.T) {
	t.Parallel()

	req := &UpdateRightsRequestPrivacyDetailsRequest{
		RightsRequestID:  gid.New(gid.NewTenantID(), coredata.RightsRequestEntityType),
		ReceivedOn:       recordTestDate(4, 8),
		FirstRespondedOn: recordTestDate(4, 7),
		FindingID:        evaluationTestPtr(gid.New(gid.NewTenantID(), coredata.FindingEntityType)),
	}

	normalized := req.normalize()
	require.NoError(t, normalized.Validate())

	var inconsistent *ErrRightsRequestRecordInconsistent
	assert.ErrorAs(t, normalized.checkConsistency(), &inconsistent, "a first response cannot come before receipt")

	req.FindingID = evaluationTestPtr(gid.New(gid.NewTenantID(), coredata.RiskEntityType))
	assert.Error(t, req.normalize().Validate(), "the finding must be a finding")
}
