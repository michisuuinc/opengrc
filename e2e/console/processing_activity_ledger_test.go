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
	ledgerDetailsQuery = `
		query($id: ID!) {
			node(id: $id) {
				... on ProcessingActivity {
					id
					ledgerDetails {
						department
						approximateRecordCount
						collectionMethod
						storageLocation
						accessScope
						disposalMethod
						updatedAt
					}
				}
			}
		}
	`

	updateLedgerDetailsMutation = `
		mutation($input: UpdateProcessingActivityLedgerDetailsInput!) {
			updateProcessingActivityLedgerDetails(input: $input) {
				processingActivity {
					id
					ledgerDetails {
						department
						approximateRecordCount
						collectionMethod
						storageLocation
						accessScope
						disposalMethod
						updatedAt
					}
				}
			}
		}
	`

	exportLedgerMutation = `
		mutation($input: ExportProcessingActivityLedgerInput!) {
			exportProcessingActivityLedger(input: $input) {
				data
			}
		}
	`

	xlsxDataURLPrefix = "data:application/vnd.openxmlformats-officedocument.spreadsheetml.sheet;base64,"
)

type ledgerDetails struct {
	Department             *string `json:"department"`
	ApproximateRecordCount *string `json:"approximateRecordCount"`
	CollectionMethod       *string `json:"collectionMethod"`
	StorageLocation        *string `json:"storageLocation"`
	AccessScope            *string `json:"accessScope"`
	DisposalMethod         *string `json:"disposalMethod"`
	UpdatedAt              *string `json:"updatedAt"`
}

func updateLedgerDetails(t *testing.T, c *testutil.Client, input map[string]any) ledgerDetails {
	t.Helper()

	var result struct {
		UpdateProcessingActivityLedgerDetails struct {
			ProcessingActivity struct {
				LedgerDetails ledgerDetails `json:"ledgerDetails"`
			} `json:"processingActivity"`
		} `json:"updateProcessingActivityLedgerDetails"`
	}

	c.MustExecute(updateLedgerDetailsMutation, map[string]any{"input": input}, &result)

	return result.UpdateProcessingActivityLedgerDetails.ProcessingActivity.LedgerDetails
}

func exportLedgerRows(t *testing.T, c *testutil.Client) [][]string {
	t.Helper()

	var result struct {
		ExportProcessingActivityLedger struct {
			Data string `json:"data"`
		} `json:"exportProcessingActivityLedger"`
	}

	c.MustExecute(
		exportLedgerMutation,
		map[string]any{"input": map[string]any{"organizationId": c.GetOrganizationID().String()}},
		&result,
	)

	data := result.ExportProcessingActivityLedger.Data
	require.True(t, strings.HasPrefix(data, xlsxDataURLPrefix), "export should be an xlsx data URL")

	workbook, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, xlsxDataURLPrefix))
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(workbook))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	rows, err := f.GetRows("個人情報管理台帳")
	require.NoError(t, err)

	return rows
}

func TestProcessingActivityLedger_Details(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)

	t.Run(
		"is empty before any update",
		func(t *testing.T) {
			t.Parallel()

			paID := factory.NewProcessingActivity(owner).WithName(factory.SafeName("Ledger empty")).Create()

			var result struct {
				Node struct {
					LedgerDetails ledgerDetails `json:"ledgerDetails"`
				} `json:"node"`
			}

			owner.MustExecute(ledgerDetailsQuery, map[string]any{"id": paID}, &result)

			assert.Equal(t, ledgerDetails{}, result.Node.LedgerDetails)
		},
	)

	t.Run(
		"keeps omitted items and clears null items",
		func(t *testing.T) {
			t.Parallel()

			paID := factory.NewProcessingActivity(owner).WithName(factory.SafeName("Ledger update")).Create()

			first := updateLedgerDetails(t, owner, map[string]any{
				"processingActivityId":   paID,
				"department":             "人事部",
				"approximateRecordCount": "約300件",
				"collectionMethod":       "応募フォーム",
				"storageLocation":        "採用管理システム",
				"accessScope":            "人事部の採用担当者",
				"disposalMethod":         "システムから削除",
			})

			testutil.AssertOptionalStringEqual(t, new("人事部"), first.Department, "department")
			testutil.AssertOptionalStringEqual(t, new("システムから削除"), first.DisposalMethod, "disposalMethod")
			require.NotNil(t, first.UpdatedAt)

			second := updateLedgerDetails(t, owner, map[string]any{
				"processingActivityId": paID,
				"department":           "総務部",
				"disposalMethod":       nil,
			})

			testutil.AssertOptionalStringEqual(t, new("総務部"), second.Department, "department")
			testutil.AssertOptionalStringEqual(t, new("約300件"), second.ApproximateRecordCount, "approximateRecordCount")
			testutil.AssertOptionalStringEqual(t, new("採用管理システム"), second.StorageLocation, "storageLocation")
			assert.Nil(t, second.DisposalMethod)
		},
	)

	t.Run(
		"rejects too long values",
		func(t *testing.T) {
			t.Parallel()

			paID := factory.NewProcessingActivity(owner).WithName(factory.SafeName("Ledger invalid")).Create()

			_, err := owner.Do(updateLedgerDetailsMutation, map[string]any{
				"input": map[string]any{
					"processingActivityId": paID,
					"department":           strings.Repeat("あ", 1001),
				},
			})
			require.Error(t, err)
		},
	)
}

func TestProcessingActivityLedger_Export(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)

	paID := factory.NewProcessingActivity(owner).
		WithName("採用選考").
		WithPurpose("応募者の選考と連絡").
		Create()
	factory.NewProcessingActivity(owner).WithName("問い合わせ対応").Create()

	updateLedgerDetails(t, owner, map[string]any{
		"processingActivityId": paID,
		"department":           "人事部",
		"storageLocation":      "=HYPERLINK(\"http://example.com\")",
	})

	rows := exportLedgerRows(t, owner)
	require.Len(t, rows, 6, "title, subtitle, blank, header and one row per activity")

	header := rows[3]
	assert.Equal(t, "No.", header[0])
	assert.Equal(t, "業務の名称", header[1])
	assert.Equal(t, "担当部門", header[2])

	column := func(name string) int {
		for i, h := range header {
			if h == name {
				return i
			}
		}
		t.Fatalf("missing column %q", name)
		return -1
	}

	// Rows are ordered by name: 問い合わせ対応 sorts before 採用選考.
	recruiting := rows[5]
	assert.Equal(t, "採用選考", recruiting[column("業務の名称")])
	assert.Equal(t, "人事部", recruiting[column("担当部門")])
	assert.Equal(t, "応募者の選考と連絡", recruiting[column("利用目的")])
	assert.Equal(t, "=HYPERLINK(\"http://example.com\")", recruiting[column("保管の場所と媒体")])
}

func TestProcessingActivityLedger_RBAC(t *testing.T) {
	t.Parallel()

	org := testutil.NewOrganizationRoles(t)
	paID := factory.NewProcessingActivity(org.Client(t, testutil.RoleOwner)).
		WithName(factory.SafeName("Ledger RBAC")).
		Create()

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

				_, err := client.Do(updateLedgerDetailsMutation, map[string]any{
					"input": map[string]any{
						"processingActivityId": paID,
						"department":           "RBAC " + string(tt.role),
					},
				})
				if tt.updateForbidden {
					testutil.RequireForbiddenError(t, err, "%s should not update ledger details", tt.role)
				} else {
					require.NoError(t, err, "%s should update ledger details", tt.role)
				}

				_, err = client.Do(ledgerDetailsQuery, map[string]any{"id": paID})
				require.NoError(t, err, "%s should read ledger details", tt.role)

				_, err = client.Do(exportLedgerMutation, map[string]any{
					"input": map[string]any{"organizationId": client.GetOrganizationID().String()},
				})
				require.NoError(t, err, "%s should export the ledger", tt.role)
			},
		)
	}
}

func TestProcessingActivityLedger_TenantIsolation(t *testing.T) {
	t.Parallel()

	org1Owner := testutil.NewClient(t, testutil.RoleOwner)
	org2Owner := testutil.NewClient(t, testutil.RoleOwner)

	paID := factory.NewProcessingActivity(org1Owner).WithName("Org1 ledger").Create()

	t.Run(
		"cannot update ledger details of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(updateLedgerDetailsMutation, map[string]any{
				"input": map[string]any{
					"processingActivityId": paID,
					"department":           "Hijacked",
				},
			})
			require.Error(t, err)
		},
	)

	t.Run(
		"cannot export the ledger of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(exportLedgerMutation, map[string]any{
				"input": map[string]any{"organizationId": org1Owner.GetOrganizationID().String()},
			})
			require.Error(t, err)
		},
	)
}
