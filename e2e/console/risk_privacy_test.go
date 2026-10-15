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
	riskPrivacyQuery = `
		query($id: ID!) {
			node(id: $id) {
				... on Risk {
					id
					privacyDetails {
						lifecycleStages
						updatedAt
					}
					privacyProcessingActivities {
						id
						name
					}
				}
			}
		}
	`

	updateRiskPrivacyMutation = `
		mutation($input: UpdateRiskPrivacyDetailsInput!) {
			updateRiskPrivacyDetails(input: $input) {
				risk {
					id
					privacyDetails {
						lifecycleStages
						updatedAt
					}
					privacyProcessingActivities {
						id
						name
					}
				}
			}
		}
	`

	exportRiskPrivacyMutation = `
		mutation($input: ExportRiskPrivacySheetInput!) {
			exportRiskPrivacySheet(input: $input) {
				data
			}
		}
	`
)

type riskPrivacy struct {
	PrivacyDetails struct {
		LifecycleStages []string `json:"lifecycleStages"`
		UpdatedAt       *string  `json:"updatedAt"`
	} `json:"privacyDetails"`
	PrivacyProcessingActivities []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"privacyProcessingActivities"`
}

func (r riskPrivacy) activityNames() []string {
	names := make([]string, len(r.PrivacyProcessingActivities))
	for i, a := range r.PrivacyProcessingActivities {
		names[i] = a.Name
	}
	return names
}

func updateRiskPrivacy(t *testing.T, c *testutil.Client, input map[string]any) riskPrivacy {
	t.Helper()

	var result struct {
		UpdateRiskPrivacyDetails struct {
			Risk riskPrivacy `json:"risk"`
		} `json:"updateRiskPrivacyDetails"`
	}

	c.MustExecute(updateRiskPrivacyMutation, map[string]any{"input": input}, &result)

	return result.UpdateRiskPrivacyDetails.Risk
}

func exportRiskPrivacyRows(t *testing.T, c *testutil.Client) [][]string {
	t.Helper()

	var result struct {
		ExportRiskPrivacySheet struct {
			Data string `json:"data"`
		} `json:"exportRiskPrivacySheet"`
	}

	c.MustExecute(
		exportRiskPrivacyMutation,
		map[string]any{"input": map[string]any{"organizationId": c.GetOrganizationID().String()}},
		&result,
	)

	data := result.ExportRiskPrivacySheet.Data
	require.True(t, strings.HasPrefix(data, xlsxDataURLPrefix), "export should be an xlsx data URL")

	workbook, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, xlsxDataURLPrefix))
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(workbook))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	rows, err := f.GetRows("リスク分析表")
	require.NoError(t, err)

	return rows
}

func TestRiskPrivacy_Details(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)

	t.Run(
		"is empty before any update",
		func(t *testing.T) {
			t.Parallel()

			riskID := factory.NewRisk(owner).WithName(factory.SafeName("Privacy empty")).Create()

			var result struct {
				Node riskPrivacy `json:"node"`
			}

			owner.MustExecute(riskPrivacyQuery, map[string]any{"id": riskID}, &result)

			assert.Empty(t, result.Node.PrivacyDetails.LifecycleStages)
			assert.NotNil(t, result.Node.PrivacyDetails.LifecycleStages, "stages should be an empty list, not null")
			assert.Nil(t, result.Node.PrivacyDetails.UpdatedAt)
			assert.Empty(t, result.Node.PrivacyProcessingActivities)
		},
	)

	t.Run(
		"sets stages and links, keeps omitted items, clears null links",
		func(t *testing.T) {
			t.Parallel()

			riskID := factory.NewRisk(owner).WithName(factory.SafeName("Privacy update")).Create()
			recruiting := factory.NewProcessingActivity(owner).WithName(factory.SafeName("採用選考")).Create()
			inquiries := factory.NewProcessingActivity(owner).WithName(factory.SafeName("問い合わせ対応")).Create()

			first := updateRiskPrivacy(t, owner, map[string]any{
				"riskId":                riskID,
				"lifecycleStages":       []string{"TRANSFER", "USE"},
				"processingActivityIds": []string{recruiting, inquiries},
			})

			assert.Equal(t, []string{"TRANSFER", "USE"}, first.PrivacyDetails.LifecycleStages)
			require.NotNil(t, first.PrivacyDetails.UpdatedAt)
			assert.ElementsMatch(t, []string{recruiting, inquiries}, []string{
				first.PrivacyProcessingActivities[0].ID,
				first.PrivacyProcessingActivities[1].ID,
			})

			second := updateRiskPrivacy(t, owner, map[string]any{
				"riskId":                riskID,
				"processingActivityIds": []string{inquiries},
			})

			assert.Equal(t, []string{"TRANSFER", "USE"}, second.PrivacyDetails.LifecycleStages, "omitted stages are kept")
			require.Len(t, second.PrivacyProcessingActivities, 1)
			assert.Equal(t, inquiries, second.PrivacyProcessingActivities[0].ID)

			third := updateRiskPrivacy(t, owner, map[string]any{
				"riskId":                riskID,
				"lifecycleStages":       []string{},
				"processingActivityIds": []string{},
			})

			assert.Empty(t, third.PrivacyDetails.LifecycleStages)
			assert.Empty(t, third.PrivacyProcessingActivities, "an empty list clears the links")

			updateRiskPrivacy(t, owner, map[string]any{
				"riskId":                riskID,
				"processingActivityIds": []string{recruiting},
			})
			fourth := updateRiskPrivacy(t, owner, map[string]any{
				"riskId":                riskID,
				"processingActivityIds": nil,
			})

			assert.Empty(t, fourth.PrivacyProcessingActivities, "null clears the links")
		},
	)

	t.Run(
		"rejects duplicate stages",
		func(t *testing.T) {
			t.Parallel()

			riskID := factory.NewRisk(owner).WithName(factory.SafeName("Privacy duplicate")).Create()

			_, err := owner.Do(updateRiskPrivacyMutation, map[string]any{
				"input": map[string]any{
					"riskId":          riskID,
					"lifecycleStages": []string{"USE", "USE"},
				},
			})
			require.Error(t, err)
		},
	)

	t.Run(
		"drops the link when the processing activity is deleted",
		func(t *testing.T) {
			t.Parallel()

			riskID := factory.NewRisk(owner).WithName(factory.SafeName("Privacy cascade")).Create()
			paID := factory.NewProcessingActivity(owner).WithName(factory.SafeName("Deleted activity")).Create()

			updateRiskPrivacy(t, owner, map[string]any{
				"riskId":                riskID,
				"processingActivityIds": []string{paID},
			})

			owner.MustExecute(`
				mutation($input: DeleteProcessingActivityInput!) {
					deleteProcessingActivity(input: $input) { deletedProcessingActivityId }
				}
			`, map[string]any{"input": map[string]any{"processingActivityId": paID}}, nil)

			var result struct {
				Node riskPrivacy `json:"node"`
			}

			owner.MustExecute(riskPrivacyQuery, map[string]any{"id": riskID}, &result)

			assert.Empty(t, result.Node.PrivacyProcessingActivities)
		},
	)
}

func TestRiskPrivacy_Export(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)

	recruiting := factory.NewProcessingActivity(owner).WithName("採用選考").Create()
	factory.NewProcessingActivity(owner).WithName("問い合わせ対応").Create()

	riskID := factory.NewRisk(owner).
		WithName("応募書類の誤送信").
		WithDescription("=HYPERLINK(\"http://example.com\")").
		WithTreatment("MITIGATED").
		WithLikelihood(3).
		WithImpact(4).
		Create()
	// A risk with no register entry is not in the sheet.
	factory.NewRisk(owner).WithName("サーバーの停止").Create()

	updateRiskPrivacy(t, owner, map[string]any{
		"riskId":                riskID,
		"lifecycleStages":       []string{"USE", "TRANSFER"},
		"processingActivityIds": []string{recruiting},
	})

	rows := exportRiskPrivacyRows(t, owner)
	require.Len(t, rows, 6, "title, subtitle, blank, header and one row per register entry")

	header := rows[3]
	assert.Equal(t, "No.", header[0])
	assert.Equal(t, "業務の名称", header[1])

	column := func(name string) int {
		for i, h := range header {
			if h == name {
				return i
			}
		}
		t.Fatalf("missing column %q", name)
		return -1
	}

	// Rows are ordered by register entry name: 問い合わせ対応 sorts before 採用選考.
	inquiries := rows[4]
	assert.Equal(t, "問い合わせ対応", inquiries[column("業務の名称")])
	assert.Equal(t, "（リスク未登録）", inquiries[column("想定されるリスク")])

	recruitingRow := rows[5]
	assert.Equal(t, "採用選考", recruitingRow[column("業務の名称")])
	assert.Equal(t, "移送・送信\n利用・加工", recruitingRow[column("段階")])
	assert.Equal(t, "応募書類の誤送信", recruitingRow[column("想定されるリスク")])
	assert.Equal(t, "=HYPERLINK(\"http://example.com\")", recruitingRow[column("リスクの内容")])
	assert.Equal(t, "3", recruitingRow[column("発生可能性")])
	assert.Equal(t, "4", recruitingRow[column("影響度")])
	assert.Equal(t, "低減", recruitingRow[column("対応方針")])
}

func TestRiskPrivacy_RBAC(t *testing.T) {
	t.Parallel()

	org := testutil.NewOrganizationRoles(t)
	ownerClient := org.Client(t, testutil.RoleOwner)
	riskID := factory.NewRisk(ownerClient).WithName(factory.SafeName("Privacy RBAC")).Create()

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

				_, err := client.Do(updateRiskPrivacyMutation, map[string]any{
					"input": map[string]any{
						"riskId":          riskID,
						"lifecycleStages": []string{"STORAGE"},
					},
				})
				if tt.updateForbidden {
					testutil.RequireForbiddenError(t, err, "%s should not update risk privacy details", tt.role)
				} else {
					require.NoError(t, err, "%s should update risk privacy details", tt.role)
				}

				_, err = client.Do(riskPrivacyQuery, map[string]any{"id": riskID})
				require.NoError(t, err, "%s should read risk privacy details", tt.role)

				_, err = client.Do(exportRiskPrivacyMutation, map[string]any{
					"input": map[string]any{"organizationId": client.GetOrganizationID().String()},
				})
				require.NoError(t, err, "%s should export the risk analysis", tt.role)
			},
		)
	}
}

func TestRiskPrivacy_TenantIsolation(t *testing.T) {
	t.Parallel()

	org1Owner := testutil.NewClient(t, testutil.RoleOwner)
	org2Owner := testutil.NewClient(t, testutil.RoleOwner)

	org1RiskID := factory.NewRisk(org1Owner).WithName("Org1 risk").Create()
	org1ActivityID := factory.NewProcessingActivity(org1Owner).WithName("Org1 activity").Create()
	org2RiskID := factory.NewRisk(org2Owner).WithName("Org2 risk").Create()

	t.Run(
		"cannot update risk privacy details of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(updateRiskPrivacyMutation, map[string]any{
				"input": map[string]any{
					"riskId":          org1RiskID,
					"lifecycleStages": []string{"USE"},
				},
			})
			require.Error(t, err)
		},
	)

	t.Run(
		"cannot link a processing activity of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(updateRiskPrivacyMutation, map[string]any{
				"input": map[string]any{
					"riskId":                org2RiskID,
					"processingActivityIds": []string{org1ActivityID},
				},
			})
			require.Error(t, err)

			var result struct {
				Node riskPrivacy `json:"node"`
			}

			org2Owner.MustExecute(riskPrivacyQuery, map[string]any{"id": org2RiskID}, &result)
			assert.Empty(t, result.Node.PrivacyProcessingActivities)
		},
	)

	t.Run(
		"cannot export the risk analysis of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(exportRiskPrivacyMutation, map[string]any{
				"input": map[string]any{"organizationId": org1Owner.GetOrganizationID().String()},
			})
			require.Error(t, err)
		},
	)
}
