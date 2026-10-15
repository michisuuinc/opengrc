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
	"go.probo.inc/probo/e2e/internal/testutil"
)

const (
	complaintRecordFields = `
		id
		privacyDetails {
			effectiveCategory
			firstRespondedOn
			finding { id referenceId }
			updatedAt
		}
	`

	updateComplaintRecordMutation = `
		mutation($input: UpdateRightsRequestPrivacyDetailsInput!) {
			updateRightsRequestPrivacyDetails(input: $input) {
				rightsRequest {` + complaintRecordFields + `}
			}
		}
	`

	complaintRecordQuery = `
		query($id: ID!) {
			node(id: $id) {
				... on RightsRequest {` + complaintRecordFields + `}
			}
		}
	`

	exportComplaintRecordsMutation = `
		mutation($input: ExportRightsRequestRecordsInput!) {
			exportComplaintRecords(input: $input) {
				data
			}
		}
	`
)

type complaintRecord struct {
	ID             string `json:"id"`
	PrivacyDetails struct {
		EffectiveCategory string  `json:"effectiveCategory"`
		FirstRespondedOn  *string `json:"firstRespondedOn"`
		Finding           *struct {
			ID          string `json:"id"`
			ReferenceID string `json:"referenceId"`
		} `json:"finding"`
		UpdatedAt *string `json:"updatedAt"`
	} `json:"privacyDetails"`
}

func updateComplaintRecord(t *testing.T, c *testutil.Client, input map[string]any) complaintRecord {
	t.Helper()

	var result struct {
		UpdateRightsRequestPrivacyDetails struct {
			RightsRequest complaintRecord `json:"rightsRequest"`
		} `json:"updateRightsRequestPrivacyDetails"`
	}

	c.MustExecute(updateComplaintRecordMutation, map[string]any{"input": input}, &result)

	return result.UpdateRightsRequestPrivacyDetails.RightsRequest
}

func loadComplaintRecord(t *testing.T, c *testutil.Client, rightsRequestID string) complaintRecord {
	t.Helper()

	var result struct {
		Node complaintRecord `json:"node"`
	}

	c.MustExecute(complaintRecordQuery, map[string]any{"id": rightsRequestID}, &result)

	return result.Node
}

func exportComplaintRecords(t *testing.T, c *testutil.Client, from any, to any) *excelize.File {
	t.Helper()

	var result struct {
		ExportComplaintRecords struct {
			Data string `json:"data"`
		} `json:"exportComplaintRecords"`
	}

	c.MustExecute(exportComplaintRecordsMutation, map[string]any{
		"input": map[string]any{
			"organizationId": c.GetOrganizationID().String(),
			"receivedFrom":   from,
			"receivedTo":     to,
		},
	}, &result)

	data := result.ExportComplaintRecords.Data
	require.True(t, strings.HasPrefix(data, xlsxDataURLPrefix), "export should be an xlsx data URL")

	workbook, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, xlsxDataURLPrefix))
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(workbook))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	return f
}

func TestComplaintRecord_Details(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)

	t.Run(
		"the first response and the finding are saved and cleared",
		func(t *testing.T) {
			t.Parallel()

			requestID := createRecordRightsRequest(t, owner, "COMPLAINT", "苦情 太郎")
			findingID := createAuditFinding(t, owner, "", "MINOR_NONCONFORMITY", "メールの誤送信")

			saved := updateComplaintRecord(t, owner, rightsRequestRecordInput(requestID, map[string]any{
				"receivedOn":       "2026-04-01",
				"firstRespondedOn": "2026-04-03",
				"findingId":        findingID,
			})).PrivacyDetails

			assert.Equal(t, "COMPLAINT", saved.EffectiveCategory)
			require.NotNil(t, saved.FirstRespondedOn)
			assert.Equal(t, "2026-04-03", *saved.FirstRespondedOn)
			require.NotNil(t, saved.Finding)
			assert.Equal(t, findingID, saved.Finding.ID)

			cleared := updateComplaintRecord(t, owner, rightsRequestRecordInput(requestID, nil)).PrivacyDetails
			assert.Nil(t, cleared.FirstRespondedOn)
			assert.Nil(t, cleared.Finding)
		},
	)

	t.Run(
		"deleting the finding unlinks it",
		func(t *testing.T) {
			t.Parallel()

			requestID := createRecordRightsRequest(t, owner, "COMPLAINT", "苦情 次郎")
			findingID := createAuditFinding(t, owner, "", "OBSERVATION", "削除される指摘事項")
			updateComplaintRecord(t, owner, rightsRequestRecordInput(requestID, map[string]any{"findingId": findingID}))

			owner.MustExecute(`
				mutation($input: DeleteFindingInput!) {
					deleteFinding(input: $input) { deletedFindingId }
				}
			`, map[string]any{"input": map[string]any{"findingId": findingID}}, nil)

			record := loadComplaintRecord(t, owner, requestID)
			assert.Nil(t, record.PrivacyDetails.Finding)
			assert.NotNil(t, record.PrivacyDetails.UpdatedAt, "the record stays")
		},
	)

	t.Run(
		"invalid input is rejected",
		func(t *testing.T) {
			t.Parallel()

			requestID := createRecordRightsRequest(t, owner, "COMPLAINT", "苦情 三郎")

			for name, overrides := range map[string]map[string]any{
				"first response before receipt": {"receivedOn": "2026-04-08", "firstRespondedOn": "2026-04-07"},
				"finding is not a finding":      {"findingId": requestID},
			} {
				_, err := owner.Do(updateComplaintRecordMutation, map[string]any{
					"input": rightsRequestRecordInput(requestID, overrides),
				})
				require.Error(t, err, name)
			}

			assert.Nil(t, loadComplaintRecord(t, owner, requestID).PrivacyDetails.UpdatedAt)
		},
	)
}

func TestComplaintRecord_Export(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)
	findingID := createAuditFinding(t, owner, "", "MINOR_NONCONFORMITY", "メールの誤送信")

	complaint := createRecordRightsRequest(t, owner, "COMPLAINT", "苦情 太郎")
	updateComplaintRecord(t, owner, rightsRequestRecordInput(complaint, map[string]any{
		"receivedOn":       "2026-04-01",
		"targetDepartment": "営業部",
		"firstRespondedOn": "2026-04-03",
		"findingId":        findingID,
	}))

	consultation := createRecordRightsRequest(t, owner, "ACCESS", "相談 花子")
	updateComplaintRecord(t, owner, rightsRequestRecordInput(consultation, map[string]any{
		"category":         "CONSULTATION",
		"receivedOn":       "2026-05-01",
		"targetDepartment": "営業部",
	}))

	disclosure := createRecordRightsRequest(t, owner, "ACCESS", "開示 次郎")
	updateComplaintRecord(t, owner, rightsRequestRecordInput(disclosure, map[string]any{"receivedOn": "2026-04-15"}))

	f := exportComplaintRecords(t, owner, "2026-04-01", "2026-09-30")

	rows := sheetDataRows(t, f, "苦情・相談の記録", "No.")
	require.Len(t, rows, 2, "only complaints and consultations")

	assert.Equal(t, "苦情 太郎", rows[0][4])
	assert.Equal(t, "苦情", rows[0][3])
	assert.Equal(t, "2026-04-03", rows[0][8])
	assert.Equal(t, "2 日", rows[0][9])
	linked := loadComplaintRecord(t, owner, complaint).PrivacyDetails.Finding
	require.NotNil(t, linked)
	assert.Equal(t, linked.ReferenceID, rows[0][14], "the finding is named by its reference ID")

	assert.Equal(t, "相談 花子", rows[1][4])
	assert.Equal(t, "相談", rows[1][3])

	departments := sheetDataRows(t, f, "部門別の件数", "対象の部門")
	assert.Equal(t, [][]string{{"営業部", "1", "1", "2", "2", "1"}, {"合計", "1", "1", "2", "2", "1"}}, departments)
}

func TestComplaintRecord_RBAC(t *testing.T) {
	t.Parallel()

	org := testutil.NewOrganizationRoles(t)

	for _, role := range []testutil.TestRole{testutil.RoleOwner, testutil.RoleAdmin, testutil.RoleViewer} {
		t.Run(
			string(role),
			func(t *testing.T) {
				t.Parallel()

				client := org.Client(t, role)

				_, err := client.Do(exportComplaintRecordsMutation, map[string]any{
					"input": map[string]any{"organizationId": client.GetOrganizationID().String()},
				})
				require.NoError(t, err, "%s should export the complaint records", role)
			},
		)
	}
}

func TestComplaintRecord_TenantIsolation(t *testing.T) {
	t.Parallel()

	org1Owner := testutil.NewClient(t, testutil.RoleOwner)
	org2Owner := testutil.NewClient(t, testutil.RoleOwner)

	org1FindingID := createAuditFinding(t, org1Owner, "", "OBSERVATION", "Org1 finding")
	org2RequestID := createRecordRightsRequest(t, org2Owner, "COMPLAINT", "Org2 太郎")

	t.Run(
		"cannot link a finding of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(updateComplaintRecordMutation, map[string]any{
				"input": rightsRequestRecordInput(org2RequestID, map[string]any{"findingId": org1FindingID}),
			})
			require.Error(t, err)

			assert.Nil(t, loadComplaintRecord(t, org2Owner, org2RequestID).PrivacyDetails.UpdatedAt)
		},
	)

	t.Run(
		"cannot export the complaint records of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(exportComplaintRecordsMutation, map[string]any{
				"input": map[string]any{"organizationId": org1Owner.GetOrganizationID().String()},
			})
			require.Error(t, err)
		},
	)
}
