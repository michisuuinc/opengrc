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
package console_test

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"go.probo.inc/probo/e2e/internal/factory"
	"go.probo.inc/probo/e2e/internal/testutil"
)

const (
	rightsRequestRecordFields = `
		id
		privacyDetails {
			category
			effectiveCategory
			receivedOn
			effectiveReceivedOn
			receiptChannel
			effectiveReceiptChannel
			byAgent
			targetDepartment
			handler { id fullName }
			verificationMethod
			verifiedOn
			verificationNote
			supplementRequestedOn
			supplementReceivedOn
			decision
			decisionReason
			answerMethod
			answeredOn
			approver { id fullName }
			handlingDays
			updatedAt
		}
	`

	rightsRequestRecordQuery = `
		query($id: ID!) {
			node(id: $id) {
				... on RightsRequest {` + rightsRequestRecordFields + `}
			}
		}
	`

	updateRightsRequestRecordMutation = `
		mutation($input: UpdateRightsRequestPrivacyDetailsInput!) {
			updateRightsRequestPrivacyDetails(input: $input) {
				rightsRequest {` + rightsRequestRecordFields + `}
			}
		}
	`

	exportRightsRequestRecordsMutation = `
		mutation($input: ExportRightsRequestRecordsInput!) {
			exportRightsRequestRecords(input: $input) {
				data
			}
		}
	`
)

type (
	rightsRequestRecordProfile struct {
		ID       string `json:"id"`
		FullName string `json:"fullName"`
	}

	rightsRequestRecord struct {
		ID             string `json:"id"`
		PrivacyDetails struct {
			Category                *string                     `json:"category"`
			EffectiveCategory       string                      `json:"effectiveCategory"`
			ReceivedOn              *string                     `json:"receivedOn"`
			EffectiveReceivedOn     string                      `json:"effectiveReceivedOn"`
			ReceiptChannel          *string                     `json:"receiptChannel"`
			EffectiveReceiptChannel *string                     `json:"effectiveReceiptChannel"`
			ByAgent                 bool                        `json:"byAgent"`
			TargetDepartment        string                      `json:"targetDepartment"`
			Handler                 *rightsRequestRecordProfile `json:"handler"`
			VerificationMethod      *string                     `json:"verificationMethod"`
			VerifiedOn              *string                     `json:"verifiedOn"`
			VerificationNote        string                      `json:"verificationNote"`
			SupplementRequestedOn   *string                     `json:"supplementRequestedOn"`
			SupplementReceivedOn    *string                     `json:"supplementReceivedOn"`
			Decision                *string                     `json:"decision"`
			DecisionReason          string                      `json:"decisionReason"`
			AnswerMethod            *string                     `json:"answerMethod"`
			AnsweredOn              *string                     `json:"answeredOn"`
			Approver                *rightsRequestRecordProfile `json:"approver"`
			HandlingDays            *int                        `json:"handlingDays"`
			UpdatedAt               *string                     `json:"updatedAt"`
		} `json:"privacyDetails"`
	}
)

func rightsRequestRecordInput(rightsRequestID string, overrides map[string]any) map[string]any {
	input := map[string]any{
		"rightsRequestId":  rightsRequestID,
		"byAgent":          false,
		"targetDepartment": "",
		"verificationNote": "",
		"decisionReason":   "",
	}
	for k, v := range overrides {
		input[k] = v
	}
	return input
}

func createRecordRightsRequest(t *testing.T, c *testutil.Client, requestType string, dataSubject string) string {
	t.Helper()

	var result struct {
		CreateRightsRequest struct {
			RightsRequestEdge struct {
				Node struct {
					ID string `json:"id"`
				} `json:"node"`
			} `json:"rightsRequestEdge"`
		} `json:"createRightsRequest"`
	}

	c.MustExecute(`
		mutation($input: CreateRightsRequestInput!) {
			createRightsRequest(input: $input) {
				rightsRequestEdge { node { id } }
			}
		}
	`, map[string]any{
		"input": map[string]any{
			"organizationId": c.GetOrganizationID().String(),
			"requestType":    requestType,
			"requestState":   "TODO",
			"dataSubject":    dataSubject,
		},
	}, &result)

	return result.CreateRightsRequest.RightsRequestEdge.Node.ID
}

func updateRightsRequestRecord(t *testing.T, c *testutil.Client, input map[string]any) rightsRequestRecord {
	t.Helper()

	var result struct {
		UpdateRightsRequestPrivacyDetails struct {
			RightsRequest rightsRequestRecord `json:"rightsRequest"`
		} `json:"updateRightsRequestPrivacyDetails"`
	}

	c.MustExecute(updateRightsRequestRecordMutation, map[string]any{"input": input}, &result)

	return result.UpdateRightsRequestPrivacyDetails.RightsRequest
}

func loadRightsRequestRecord(t *testing.T, c *testutil.Client, rightsRequestID string) rightsRequestRecord {
	t.Helper()

	var result struct {
		Node rightsRequestRecord `json:"node"`
	}

	c.MustExecute(rightsRequestRecordQuery, map[string]any{"id": rightsRequestID}, &result)

	return result.Node
}

func exportRightsRequestRecords(t *testing.T, c *testutil.Client, from any, to any) *excelize.File {
	t.Helper()

	var result struct {
		ExportRightsRequestRecords struct {
			Data string `json:"data"`
		} `json:"exportRightsRequestRecords"`
	}

	c.MustExecute(exportRightsRequestRecordsMutation, map[string]any{
		"input": map[string]any{
			"organizationId": c.GetOrganizationID().String(),
			"receivedFrom":   from,
			"receivedTo":     to,
		},
	}, &result)

	data := result.ExportRightsRequestRecords.Data
	require.True(t, strings.HasPrefix(data, xlsxDataURLPrefix), "export should be an xlsx data URL")

	workbook, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, xlsxDataURLPrefix))
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(workbook))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	return f
}

func TestRightsRequestRecord_Details(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)

	t.Run(
		"a request without a record follows its type and registration date",
		func(t *testing.T) {
			t.Parallel()

			requestID := createRecordRightsRequest(t, owner, "OBJECTION", "山田 太郎")

			d := loadRightsRequestRecord(t, owner, requestID).PrivacyDetails

			assert.Nil(t, d.UpdatedAt)
			assert.Nil(t, d.Category)
			assert.Equal(t, "SUSPENSION", d.EffectiveCategory)
			assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, d.EffectiveReceivedOn)
			assert.Nil(t, d.EffectiveReceiptChannel)
			assert.Nil(t, d.HandlingDays)
		},
	)

	t.Run(
		"the record is saved and replaced",
		func(t *testing.T) {
			t.Parallel()

			requestID := createRecordRightsRequest(t, owner, "ACCESS", "山田 太郎")
			handler := factory.CreateUser(owner, factory.Attrs{"fullName": "窓口 一郎"})
			approver := factory.CreateUser(owner, factory.Attrs{"fullName": "管理 花子"})

			saved := updateRightsRequestRecord(t, owner, rightsRequestRecordInput(requestID, map[string]any{
				"category":              "THIRD_PARTY_RECORD_DISCLOSURE",
				"receivedOn":            "2026-04-01",
				"receiptChannel":        "POSTAL",
				"byAgent":               true,
				"targetDepartment":      " 営業部 ",
				"handlerId":             handler,
				"verificationMethod":    "ID_DOCUMENT",
				"verifiedOn":            "2026-04-02",
				"verificationNote":      "運転免許証",
				"supplementRequestedOn": "2026-04-03",
				"supplementReceivedOn":  "2026-04-08",
				"decision":              "PARTIALLY_FULFILLED",
				"decisionReason":        "第三者の情報を含むため一部不開示",
				"answerMethod":          "POSTAL",
				"answeredOn":            "2026-04-15",
				"approverId":            approver,
			}))

			d := saved.PrivacyDetails
			require.NotNil(t, d.UpdatedAt)
			assert.Equal(t, "THIRD_PARTY_RECORD_DISCLOSURE", *d.Category)
			assert.Equal(t, "THIRD_PARTY_RECORD_DISCLOSURE", d.EffectiveCategory)
			assert.Equal(t, "2026-04-01", d.EffectiveReceivedOn)
			assert.True(t, d.ByAgent)
			assert.Equal(t, "営業部", d.TargetDepartment)
			require.NotNil(t, d.Handler)
			assert.Equal(t, "窓口 一郎", d.Handler.FullName)
			require.NotNil(t, d.Approver)
			assert.Equal(t, approver, d.Approver.ID)
			assert.Equal(t, "2026-04-15", *d.AnsweredOn)
			require.NotNil(t, d.HandlingDays)
			assert.Equal(t, 9, *d.HandlingDays, "14 days less the 5-day wait for documents")

			assert.Equal(t, saved, loadRightsRequestRecord(t, owner, requestID))

			replaced := updateRightsRequestRecord(t, owner, rightsRequestRecordInput(requestID, map[string]any{
				"decision":       "FULFILLED",
				"decisionReason": "使われない理由",
			})).PrivacyDetails

			assert.Equal(t, "FULFILLED", *replaced.Decision)
			assert.Empty(t, replaced.DecisionReason, "a request answered as requested keeps no reason")
			assert.Nil(t, replaced.Category)
			assert.Equal(t, "DISCLOSURE", replaced.EffectiveCategory)
			assert.Nil(t, replaced.Handler)
			assert.Nil(t, replaced.AnsweredOn)
			assert.Nil(t, replaced.HandlingDays)
		},
	)

	t.Run(
		"inconsistent or invalid input is rejected and nothing changes",
		func(t *testing.T) {
			t.Parallel()

			requestID := createRecordRightsRequest(t, owner, "ACCESS", "山田 太郎")

			for name, overrides := range map[string]map[string]any{
				"refused without a reason":        {"decision": "NOT_FULFILLED"},
				"documents received before asked": {"supplementRequestedOn": "2026-04-08", "supplementReceivedOn": "2026-04-03"},
				"documents never asked for":       {"supplementReceivedOn": "2026-04-03"},
				"answered before received":        {"receivedOn": "2026-04-08", "answeredOn": "2026-04-03"},
				"invalid date":                    {"receivedOn": "2026/04/08"},
				"handler is not a profile":        {"handlerId": requestID},
			} {
				_, err := owner.Do(updateRightsRequestRecordMutation, map[string]any{
					"input": rightsRequestRecordInput(requestID, overrides),
				})
				require.Error(t, err, name)
			}

			assert.Nil(t, loadRightsRequestRecord(t, owner, requestID).PrivacyDetails.UpdatedAt)
		},
	)

	t.Run(
		"deleting the request deletes its record",
		func(t *testing.T) {
			t.Parallel()

			requestID := createRecordRightsRequest(t, owner, "ACCESS", "山田 太郎")
			updateRightsRequestRecord(t, owner, rightsRequestRecordInput(requestID, map[string]any{"receivedOn": "2026-04-01"}))

			owner.MustExecute(`
				mutation($input: DeleteRightsRequestInput!) {
					deleteRightsRequest(input: $input) { deletedRightsRequestId }
				}
			`, map[string]any{"input": map[string]any{"rightsRequestId": requestID}}, nil)

			_, err := owner.Do(rightsRequestRecordQuery, map[string]any{"id": requestID})
			require.Error(t, err)
		},
	)
}

func TestRightsRequestRecord_Export(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)
	handler := factory.CreateUser(owner, factory.Attrs{"fullName": "窓口 一郎"})

	april := createRecordRightsRequest(t, owner, "ACCESS", "四月 太郎")
	updateRightsRequestRecord(t, owner, rightsRequestRecordInput(april, map[string]any{
		"receivedOn":         "2026-04-01",
		"receiptChannel":     "EMAIL",
		"handlerId":          handler,
		"verificationMethod": "ACCOUNT",
		"answeredOn":         "2026-04-11",
		"decision":           "FULFILLED",
	}))

	may := createRecordRightsRequest(t, owner, "COMPLAINT", "五月 花子")
	updateRightsRequestRecord(t, owner, rightsRequestRecordInput(may, map[string]any{
		"category":   "CONSULTATION",
		"receivedOn": "2026-05-01",
	}))

	march := createRecordRightsRequest(t, owner, "DELETION", "三月 次郎")
	updateRightsRequestRecord(t, owner, rightsRequestRecordInput(march, map[string]any{"receivedOn": "2026-03-31"}))

	t.Run(
		"the period bounds the receipt dates",
		func(t *testing.T) {
			t.Parallel()

			f := exportRightsRequestRecords(t, owner, "2026-04-01", "2026-09-30")

			rows := sheetDataRows(t, f, "請求の記録", "No.")
			require.Len(t, rows, 2, "the March request is outside the period")

			assert.Equal(t, "2026-04-01", rows[0][1])
			assert.Equal(t, "メール", rows[0][2])
			assert.Equal(t, "開示", rows[0][3])
			assert.Equal(t, "四月 太郎", rows[0][4])
			assert.Equal(t, "窓口 一郎", rows[0][8])
			assert.Equal(t, "2026-04-11", rows[0][19])
			assert.Equal(t, "10 日", rows[0][21])

			assert.Equal(t, "2026-05-01", rows[1][1])
			assert.Equal(t, "相談", rows[1][3])

			summary := sheetDataRows(t, f, "集計", "請求の種類")
			require.Len(t, summary, 9)
			assert.Equal(t, []string{"開示", "1", "1", "0", "10.0 日", "10 日", "0", "0"}, summary[1])
			assert.Equal(t, []string{"合計", "2", "1", "1", "10.0 日", "10 日", "0", "0"}, summary[8])
		},
	)

	t.Run(
		"without a period every request is exported",
		func(t *testing.T) {
			t.Parallel()

			rows := sheetDataRows(t, exportRightsRequestRecords(t, owner, nil, nil), "請求の記録", "No.")
			require.Len(t, rows, 3)
			assert.Equal(t, "三月 次郎", rows[0][4], "sorted by receipt date")
		},
	)

	t.Run(
		"a period that ends before it starts is rejected",
		func(t *testing.T) {
			t.Parallel()

			_, err := owner.Do(exportRightsRequestRecordsMutation, map[string]any{
				"input": map[string]any{
					"organizationId": owner.GetOrganizationID().String(),
					"receivedFrom":   "2026-09-30",
					"receivedTo":     "2026-04-01",
				},
			})
			require.Error(t, err)
		},
	)
}

func TestRightsRequestRecord_RBAC(t *testing.T) {
	t.Parallel()

	org := testutil.NewOrganizationRoles(t)
	ownerClient := org.Client(t, testutil.RoleOwner)
	requestID := createRecordRightsRequest(t, ownerClient, "ACCESS", "RBAC 太郎")

	tests := []struct {
		name            string
		role            testutil.TestRole
		updateForbidden bool
	}{
		{name: "owner", role: testutil.RoleOwner},
		{name: "admin", role: testutil.RoleAdmin},
		{name: "viewer", role: testutil.RoleViewer, updateForbidden: true},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				t.Parallel()

				client := org.Client(t, tt.role)

				_, err := client.Do(updateRightsRequestRecordMutation, map[string]any{
					"input": rightsRequestRecordInput(requestID, map[string]any{"targetDepartment": "営業部"}),
				})
				if tt.updateForbidden {
					testutil.RequireForbiddenError(t, err, "%s should not update the record", tt.role)
				} else {
					require.NoError(t, err, "%s should update the record", tt.role)
				}

				_, err = client.Do(rightsRequestRecordQuery, map[string]any{"id": requestID})
				require.NoError(t, err, "%s should read the record", tt.role)

				_, err = client.Do(exportRightsRequestRecordsMutation, map[string]any{
					"input": map[string]any{"organizationId": client.GetOrganizationID().String()},
				})
				require.NoError(t, err, "%s should export the records", tt.role)
			},
		)
	}
}

func TestRightsRequestRecord_TenantIsolation(t *testing.T) {
	t.Parallel()

	org1Owner := testutil.NewClient(t, testutil.RoleOwner)
	org2Owner := testutil.NewClient(t, testutil.RoleOwner)

	org1RequestID := createRecordRightsRequest(t, org1Owner, "ACCESS", "Org1 太郎")
	org1ProfileID := factory.CreateUser(org1Owner)
	org2RequestID := createRecordRightsRequest(t, org2Owner, "ACCESS", "Org2 太郎")

	t.Run(
		"cannot read or update the record of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(rightsRequestRecordQuery, map[string]any{"id": org1RequestID})
			require.Error(t, err)

			_, err = org2Owner.Do(updateRightsRequestRecordMutation, map[string]any{
				"input": rightsRequestRecordInput(org1RequestID, map[string]any{"targetDepartment": "営業部"}),
			})
			require.Error(t, err)
		},
	)

	t.Run(
		"cannot choose a handler or approver of another organization",
		func(t *testing.T) {
			t.Parallel()

			for _, field := range []string{"handlerId", "approverId"} {
				_, err := org2Owner.Do(updateRightsRequestRecordMutation, map[string]any{
					"input": rightsRequestRecordInput(org2RequestID, map[string]any{field: org1ProfileID}),
				})
				require.Error(t, err, field)
			}

			assert.Nil(t, loadRightsRequestRecord(t, org2Owner, org2RequestID).PrivacyDetails.UpdatedAt)
		},
	)

	t.Run(
		"cannot export the records of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(exportRightsRequestRecordsMutation, map[string]any{
				"input": map[string]any{"organizationId": org1Owner.GetOrganizationID().String()},
			})
			require.Error(t, err)
		},
	)
}
