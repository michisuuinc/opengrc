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

func recordTestDate(month time.Month, day int) *time.Time {
	date := time.Date(2026, month, day, 0, 0, 0, 0, time.UTC)
	return &date
}

func recordTestRequest(requestType coredata.RightsRequestType, state coredata.RightsRequestState) *coredata.RightsRequest {
	subject := "山田 太郎"

	return &coredata.RightsRequest{
		ID:           gid.New(gid.NewTenantID(), coredata.RightsRequestEntityType),
		RequestType:  requestType,
		RequestState: state,
		DataSubject:  &subject,
		CreatedAt:    time.Date(2026, 5, 31, 16, 0, 0, 0, time.UTC),
	}
}

func TestRightsRequestHandlingDays(t *testing.T) {
	t.Parallel()

	received := *recordTestDate(4, 1)

	tests := []struct {
		name    string
		details *coredata.RightsRequestPrivacyDetails
		want    *int
	}{
		{"no details", nil, nil},
		{"not answered", &coredata.RightsRequestPrivacyDetails{}, nil},
		{"answered", &coredata.RightsRequestPrivacyDetails{AnsweredOn: recordTestDate(4, 15)}, evaluationTestPtr(14)},
		{
			"wait for documents excluded",
			&coredata.RightsRequestPrivacyDetails{
				SupplementRequestedOn: recordTestDate(4, 3),
				SupplementReceivedOn:  recordTestDate(4, 8),
				AnsweredOn:            recordTestDate(4, 15),
			},
			evaluationTestPtr(9),
		},
		{
			"documents never received",
			&coredata.RightsRequestPrivacyDetails{
				SupplementRequestedOn: recordTestDate(4, 3),
				AnsweredOn:            recordTestDate(4, 15),
			},
			evaluationTestPtr(2),
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Parallel()

				assert.Equal(t, tt.want, RightsRequestHandlingDays(received, tt.details))
			},
		)
	}
}

func TestRightsRequestEffectiveValues(t *testing.T) {
	t.Parallel()

	request := recordTestRequest(coredata.RightsRequestTypeObjection, coredata.RightsRequestStateTodo)

	assert.Equal(t, coredata.RightsRequestPrivacyCategorySuspension, RightsRequestEffectiveCategory(request, nil))
	assert.Equal(
		t,
		coredata.RightsRequestPrivacyCategoryConsultation,
		RightsRequestEffectiveCategory(request, &coredata.RightsRequestPrivacyDetails{
			Category: evaluationTestPtr(coredata.RightsRequestPrivacyCategoryConsultation),
		}),
	)

	assert.Equal(t, *recordTestDate(6, 1), RightsRequestEffectiveReceivedOn(request, nil), "registered on June 1 in Japan")
	assert.Equal(
		t,
		*recordTestDate(5, 20),
		RightsRequestEffectiveReceivedOn(request, &coredata.RightsRequestPrivacyDetails{ReceivedOn: recordTestDate(5, 20)}),
	)

	assert.Nil(t, RightsRequestEffectiveReceiptChannel(request, nil))
	portal := gid.New(gid.NewTenantID(), coredata.CompliancePortalEntityType)
	request.CompliancePortalID = &portal
	assert.Equal(t, coredata.RightsRequestReceiptChannelWebForm, *RightsRequestEffectiveReceiptChannel(request, nil))
}

func TestBuildRightsRequestRecordSheets(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 15, 9, 0, 0, 0, ledgerTimeZone)

	disclosure := recordTestRequest(coredata.RightsRequestTypeAccess, coredata.RightsRequestStateDone)
	action := "保有個人データを開示した"
	disclosure.ActionTaken = &action
	deadline := time.Date(2026, 4, 30, 15, 0, 0, 0, time.UTC)
	disclosure.Deadline = &deadline

	complaint := recordTestRequest(coredata.RightsRequestTypeComplaint, coredata.RightsRequestStateInProgress)
	overdue := now.Add(-48 * time.Hour)
	complaint.Deadline = &overdue

	undocumented := recordTestRequest(coredata.RightsRequestTypeDeletion, coredata.RightsRequestStateDone)

	data := rightsRequestRecordData{
		OrganizationName: "サンプル株式会社",
		From:             recordTestDate(4, 1),
		To:               recordTestDate(9, 30),
		Records: []rightsRequestRecord{
			{
				Request: disclosure,
				Details: &coredata.RightsRequestPrivacyDetails{
					ReceivedOn:            recordTestDate(4, 1),
					ReceiptChannel:        evaluationTestPtr(coredata.RightsRequestReceiptChannelPostal),
					ByAgent:               true,
					TargetDepartment:      "営業部",
					VerificationMethod:    evaluationTestPtr(coredata.RightsRequestVerificationMethodIDDocument),
					VerificationNote:      "運転免許証",
					VerifiedOn:            recordTestDate(4, 2),
					SupplementRequestedOn: recordTestDate(4, 3),
					SupplementReceivedOn:  recordTestDate(4, 8),
					Decision:              evaluationTestPtr(coredata.RightsRequestDecisionPartiallyFulfilled),
					DecisionReason:        "第三者の情報を含むため一部不開示",
					AnswerMethod:          evaluationTestPtr(coredata.RightsRequestAnswerMethodPostal),
					AnsweredOn:            recordTestDate(4, 15),
					ApproverProfileID:     evaluationTestPtr(gid.New(gid.NewTenantID(), coredata.MembershipProfileEntityType)),
				},
				ReceivedOn:   *recordTestDate(4, 1),
				HandlerName:  "窓口 一郎",
				ApproverName: "管理 花子",
			},
			{
				Request: complaint,
				Details: &coredata.RightsRequestPrivacyDetails{
					Category: evaluationTestPtr(coredata.RightsRequestPrivacyCategoryConsultation),
				},
				ReceivedOn: *recordTestDate(5, 1),
			},
			{
				Request:    undocumented,
				ReceivedOn: *recordTestDate(6, 1),
			},
		},
	}

	sheets := buildRightsRequestRecordSheets(data, now)
	require.Len(t, sheets, 2)

	records := sheets[0]
	assert.Equal(t, "請求の記録", records.Name)
	assert.Equal(t, "サンプル株式会社　受付日：2026-04-01〜2026-09-30　出力日：2026-10-15", records.Subtitle)
	require.Len(t, records.Rows, 3)
	for _, row := range records.Rows {
		assert.Len(t, row, len(records.Columns))
	}

	assert.Equal(
		t,
		[]string{
			"1", "2026-04-01", "郵送", "開示", "山田 太郎", "代理人", "", "営業部", "窓口 一郎",
			"本人確認書類（運転免許証）", "2026-04-02", "2026-04-03", "2026-04-08", "2026-05-01", "完了",
			"保有個人データを開示した", "一部対応（一部不開示等）", "第三者の情報を含むため一部不開示",
			"書面（郵送）", "2026-04-15", "管理 花子", "9 日", "",
		},
		records.Rows[0],
	)

	assert.Equal(t, "相談", records.Rows[1][3])
	assert.Equal(t, "期限を過ぎている", records.Rows[1][22], "a consultation needs no verification")

	assert.Equal(t, "利用停止・消去", records.Rows[2][3])
	assert.Equal(t, "記録の項目が未記入", records.Rows[2][22])

	summary := sheets[1]
	assert.Equal(t, "集計", summary.Name)
	require.Len(t, summary.Rows, 9, "8 kinds and the total")
	assert.Equal(t, []string{"開示", "1", "1", "0", "9.0 日", "9 日", "1", "0"}, summary.Rows[1])
	assert.Equal(t, []string{"相談", "1", "0", "1", "", "", "0", "0"}, summary.Rows[7])
	assert.Equal(t, []string{"合計", "3", "1", "2", "9.0 日", "9 日", "1", "0"}, summary.Rows[8])

	_, err := xlsxexport.BuildWorkbook(sheets...)
	require.NoError(t, err)
}

func TestBuildRightsRequestRecordSheetsEmpty(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 15, 9, 0, 0, 0, ledgerTimeZone)
	sheets := buildRightsRequestRecordSheets(rightsRequestRecordData{OrganizationName: "サンプル株式会社"}, now)

	assert.Equal(t, "サンプル株式会社　受付日：すべて　出力日：2026-10-15", sheets[0].Subtitle)
	require.Len(t, sheets[0].Rows, 1)
	assert.Equal(t, "（請求なし）", sheets[0].Rows[0][3])
	assert.Len(t, sheets[0].Rows[0], len(sheets[0].Columns))
	assert.Equal(t, []string{"合計", "0", "0", "0", "", "", "0", "0"}, sheets[1].Rows[8])

	_, err := xlsxexport.BuildWorkbook(sheets...)
	require.NoError(t, err)
}

func TestUpdateRightsRequestPrivacyDetailsRequestRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(r *UpdateRightsRequestPrivacyDetailsRequest)
		wantErr bool
	}{
		{"empty is fine", func(r *UpdateRightsRequestPrivacyDetailsRequest) {}, false},
		{
			"refused without a reason",
			func(r *UpdateRightsRequestPrivacyDetailsRequest) {
				r.Decision = evaluationTestPtr(coredata.RightsRequestDecisionNotFulfilled)
				r.DecisionReason = "  "
			},
			true,
		},
		{
			"partly answered with a reason",
			func(r *UpdateRightsRequestPrivacyDetailsRequest) {
				r.Decision = evaluationTestPtr(coredata.RightsRequestDecisionPartiallyFulfilled)
				r.DecisionReason = "第三者の情報を含む"
			},
			false,
		},
		{
			"documents received but never asked for",
			func(r *UpdateRightsRequestPrivacyDetailsRequest) {
				r.SupplementReceivedOn = recordTestDate(4, 8)
			},
			true,
		},
		{
			"documents received before asked for",
			func(r *UpdateRightsRequestPrivacyDetailsRequest) {
				r.SupplementRequestedOn = recordTestDate(4, 8)
				r.SupplementReceivedOn = recordTestDate(4, 3)
			},
			true,
		},
		{
			"answered before received",
			func(r *UpdateRightsRequestPrivacyDetailsRequest) {
				r.ReceivedOn = recordTestDate(4, 8)
				r.AnsweredOn = recordTestDate(4, 3)
			},
			true,
		},
		{
			"answered on the day received",
			func(r *UpdateRightsRequestPrivacyDetailsRequest) {
				r.ReceivedOn = recordTestDate(4, 8)
				r.AnsweredOn = evaluationTestPtr(time.Date(2026, 4, 8, 23, 0, 0, 0, time.UTC))
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Parallel()

				req := &UpdateRightsRequestPrivacyDetailsRequest{
					RightsRequestID: gid.New(gid.NewTenantID(), coredata.RightsRequestEntityType),
				}
				tt.modify(req)

				normalized := req.normalize()
				require.NoError(t, normalized.Validate())

				err := normalized.checkConsistency()
				if tt.wantErr {
					var inconsistent *ErrRightsRequestRecordInconsistent
					assert.ErrorAs(t, err, &inconsistent)
				} else {
					assert.NoError(t, err)
				}
			},
		)
	}

	req := &UpdateRightsRequestPrivacyDetailsRequest{
		Decision:       evaluationTestPtr(coredata.RightsRequestDecisionFulfilled),
		DecisionReason: "使われない理由",
	}
	assert.Empty(t, req.normalize().DecisionReason, "a request answered as requested keeps no reason")
}
