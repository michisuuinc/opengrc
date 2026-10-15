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
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"go.probo.inc/probo/e2e/internal/factory"
	"go.probo.inc/probo/e2e/internal/testutil"
)

const (
	auditPrivacyFields = `
		id
		privacyDetails {
			scope
			criteria
			auditeeDepartments
			externalAuditors
			overallOpinion
			updatedAt
		}
		privacyAuditors {
			id
			fullName
		}
		privacyIndependenceConflicts {
			auditor { id }
			department
		}
	`

	auditPrivacyQuery = `
		query($id: ID!) {
			node(id: $id) {
				... on Audit {` + auditPrivacyFields + `}
			}
		}
	`

	updateAuditPrivacyMutation = `
		mutation($input: UpdateAuditPrivacyDetailsInput!) {
			updateAuditPrivacyDetails(input: $input) {
				audit {` + auditPrivacyFields + `}
			}
		}
	`

	findingPrivacyQuery = `
		query($id: ID!) {
			node(id: $id) {
				... on Finding {
					id
					privacyDetails {
						category
						effectiveCategory
						basis
						updatedAt
					}
				}
			}
		}
	`

	updateFindingPrivacyMutation = `
		mutation($input: UpdateFindingPrivacyDetailsInput!) {
			updateFindingPrivacyDetails(input: $input) {
				finding {
					id
					privacyDetails {
						category
						effectiveCategory
						basis
						updatedAt
					}
				}
			}
		}
	`

	exportAuditReportMutation = `
		mutation($input: ExportAuditReportInput!) {
			exportAuditReport(input: $input) {
				data
			}
		}
	`
)

type (
	auditPrivacy struct {
		ID             string `json:"id"`
		PrivacyDetails struct {
			Scope              string   `json:"scope"`
			Criteria           string   `json:"criteria"`
			AuditeeDepartments []string `json:"auditeeDepartments"`
			ExternalAuditors   []string `json:"externalAuditors"`
			OverallOpinion     string   `json:"overallOpinion"`
			UpdatedAt          *string  `json:"updatedAt"`
		} `json:"privacyDetails"`
		PrivacyAuditors []struct {
			ID       string `json:"id"`
			FullName string `json:"fullName"`
		} `json:"privacyAuditors"`
		PrivacyIndependenceConflicts []struct {
			Auditor struct {
				ID string `json:"id"`
			} `json:"auditor"`
			Department string `json:"department"`
		} `json:"privacyIndependenceConflicts"`
	}

	findingPrivacy struct {
		ID             string `json:"id"`
		PrivacyDetails struct {
			Category          *string `json:"category"`
			EffectiveCategory *string `json:"effectiveCategory"`
			Basis             string  `json:"basis"`
			UpdatedAt         *string `json:"updatedAt"`
		} `json:"privacyDetails"`
	}
)

func (a auditPrivacy) auditorIDs() []string {
	ids := make([]string, len(a.PrivacyAuditors))
	for i, p := range a.PrivacyAuditors {
		ids[i] = p.ID
	}
	return ids
}

func auditPrivacyInput(auditID string, overrides map[string]any) map[string]any {
	input := map[string]any{
		"auditId":            auditID,
		"scope":              "",
		"criteria":           "",
		"auditeeDepartments": []string{},
		"externalAuditors":   []string{},
		"overallOpinion":     "",
		"auditorIds":         []string{},
	}
	for k, v := range overrides {
		input[k] = v
	}
	return input
}

func createPrivacyAudit(t *testing.T, c *testutil.Client, name string) string {
	t.Helper()

	frameworkID := factory.NewFramework(c).Create()

	return factory.NewAudit(c, frameworkID).WithName(name).Create()
}

func updateAuditPrivacy(t *testing.T, c *testutil.Client, input map[string]any) auditPrivacy {
	t.Helper()

	var result struct {
		UpdateAuditPrivacyDetails struct {
			Audit auditPrivacy `json:"audit"`
		} `json:"updateAuditPrivacyDetails"`
	}

	c.MustExecute(updateAuditPrivacyMutation, map[string]any{"input": input}, &result)

	return result.UpdateAuditPrivacyDetails.Audit
}

func loadAuditPrivacy(t *testing.T, c *testutil.Client, auditID string) auditPrivacy {
	t.Helper()

	var result struct {
		Node auditPrivacy `json:"node"`
	}

	c.MustExecute(auditPrivacyQuery, map[string]any{"id": auditID}, &result)

	return result.Node
}

func createAuditFinding(t *testing.T, c *testutil.Client, auditID string, kind string, description string) string {
	t.Helper()

	var created struct {
		CreateFinding struct {
			FindingEdge struct {
				Node struct {
					ID string `json:"id"`
				} `json:"node"`
			} `json:"findingEdge"`
		} `json:"createFinding"`
	}

	c.MustExecute(`
		mutation($input: CreateFindingInput!) {
			createFinding(input: $input) {
				findingEdge { node { id } }
			}
		}
	`, map[string]any{
		"input": map[string]any{
			"organizationId":   c.GetOrganizationID().String(),
			"kind":             kind,
			"description":      description,
			"correctiveAction": "手順を見直す",
			"status":           "OPEN",
			"priority":         "MEDIUM",
		},
	}, &created)

	findingID := created.CreateFinding.FindingEdge.Node.ID

	if auditID != "" {
		c.MustExecute(`
			mutation($input: CreateFindingAuditMappingInput!) {
				createFindingAuditMapping(input: $input) {
					findingEdge { node { id } }
				}
			}
		`, map[string]any{
			"input": map[string]any{
				"findingId":   findingID,
				"auditId":     auditID,
				"referenceId": factory.SafeName("REF"),
			},
		}, nil)
	}

	return findingID
}

func updateFindingPrivacy(t *testing.T, c *testutil.Client, input map[string]any) findingPrivacy {
	t.Helper()

	var result struct {
		UpdateFindingPrivacyDetails struct {
			Finding findingPrivacy `json:"finding"`
		} `json:"updateFindingPrivacyDetails"`
	}

	c.MustExecute(updateFindingPrivacyMutation, map[string]any{"input": input}, &result)

	return result.UpdateFindingPrivacyDetails.Finding
}

// createProfileInDepartment adds a member through SCIM, the only way the
// department of a profile is set.
func createProfileInDepartment(t *testing.T, sc *scimClient, fullName string, department string) string {
	t.Helper()

	email := factory.SafeEmail()
	payload := buildUserPayload(email, fullName, factory.SafeName("ext"), true)
	payload["schemas"] = []string{
		"urn:ietf:params:scim:schemas:core:2.0:User",
		"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User",
	}
	payload["urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"] = map[string]any{
		"department": department,
	}

	body, status := sc.doRequest("POST", "/Users", payload)
	require.Equal(t, http.StatusCreated, status, body)

	var resource map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &resource))

	id, ok := resource["id"].(string)
	require.True(t, ok, "SCIM user should have an id")

	return id
}

func exportAuditReport(t *testing.T, c *testutil.Client, auditID string) *excelize.File {
	t.Helper()

	var result struct {
		ExportAuditReport struct {
			Data string `json:"data"`
		} `json:"exportAuditReport"`
	}

	c.MustExecute(exportAuditReportMutation, map[string]any{"input": map[string]any{"auditId": auditID}}, &result)

	data := result.ExportAuditReport.Data
	require.True(t, strings.HasPrefix(data, xlsxDataURLPrefix), "export should be an xlsx data URL")

	workbook, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, xlsxDataURLPrefix))
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(workbook))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	return f
}

func summaryValues(t *testing.T, f *excelize.File) map[string]string {
	t.Helper()

	rows, err := f.GetRows("監査概要")
	require.NoError(t, err)

	values := make(map[string]string)
	for _, row := range rows[4:] {
		if len(row) == 2 {
			values[row[0]] = row[1]
		} else if len(row) == 1 {
			values[row[0]] = ""
		}
	}

	return values
}

func TestAuditPrivacy_Details(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)

	t.Run(
		"an audit without details reads empty",
		func(t *testing.T) {
			t.Parallel()

			auditID := createPrivacyAudit(t, owner, factory.SafeName("Empty audit"))
			audit := loadAuditPrivacy(t, owner, auditID)

			assert.Equal(t, "", audit.PrivacyDetails.Scope)
			assert.Empty(t, audit.PrivacyDetails.AuditeeDepartments)
			assert.Empty(t, audit.PrivacyDetails.ExternalAuditors)
			assert.Nil(t, audit.PrivacyDetails.UpdatedAt)
			assert.Empty(t, audit.PrivacyAuditors)
			assert.Empty(t, audit.PrivacyIndependenceConflicts)
		},
	)

	t.Run(
		"details and auditors are saved, replaced and cleared",
		func(t *testing.T) {
			t.Parallel()

			auditID := createPrivacyAudit(t, owner, factory.SafeName("Saved audit"))
			first := factory.CreateUser(owner)
			second := factory.CreateUser(owner)

			audit := updateAuditPrivacy(t, owner, auditPrivacyInput(auditID, map[string]any{
				"scope":              "J.1〜J.11",
				"criteria":           "JIS Q 15001:2023、当社の規程",
				"auditeeDepartments": []string{" 営業部 ", "管理部"},
				"externalAuditors":   []string{"外部 太郎"},
				"overallOpinion":     "概ね適合している。",
				"auditorIds":         []string{second, first},
			}))

			assert.Equal(t, "J.1〜J.11", audit.PrivacyDetails.Scope)
			assert.Equal(t, []string{"営業部", "管理部"}, audit.PrivacyDetails.AuditeeDepartments)
			assert.Equal(t, []string{"外部 太郎"}, audit.PrivacyDetails.ExternalAuditors)
			assert.Equal(t, "概ね適合している。", audit.PrivacyDetails.OverallOpinion)
			assert.NotNil(t, audit.PrivacyDetails.UpdatedAt)
			assert.Equal(t, []string{second, first}, audit.auditorIDs())

			audit = updateAuditPrivacy(t, owner, auditPrivacyInput(auditID, map[string]any{
				"scope":      " J.1 ",
				"auditorIds": []string{first, second},
			}))
			assert.Equal(t, []string{first, second}, audit.auditorIDs(), "auditors keep the new order")
			assert.Equal(t, "J.1", audit.PrivacyDetails.Scope)

			audit = updateAuditPrivacy(t, owner, auditPrivacyInput(auditID, map[string]any{
				"scope":      "J.7",
				"auditorIds": []string{first},
			}))

			assert.Equal(t, "J.7", audit.PrivacyDetails.Scope)
			assert.Empty(t, audit.PrivacyDetails.AuditeeDepartments)
			assert.Equal(t, []string{first}, audit.auditorIDs())

			audit = updateAuditPrivacy(t, owner, auditPrivacyInput(auditID, nil))
			assert.Empty(t, audit.PrivacyAuditors)
			assert.Equal(t, "", audit.PrivacyDetails.Scope)
		},
	)

	t.Run(
		"invalid input is rejected and nothing changes",
		func(t *testing.T) {
			t.Parallel()

			auditID := createPrivacyAudit(t, owner, factory.SafeName("Invalid audit"))
			auditor := factory.CreateUser(owner)

			tooMany := make([]string, 21)
			for i := range tooMany {
				tooMany[i] = factory.SafeName("外部")
			}

			for name, overrides := range map[string]map[string]any{
				"duplicate department": {"auditeeDepartments": []string{"営業部", "営業部"}},
				"blank department":     {"auditeeDepartments": []string{"  "}},
				"too many external":    {"externalAuditors": tooMany},
				"duplicate auditor":    {"auditorIds": []string{auditor, auditor}},
				"unknown auditor":      {"auditorIds": []string{auditID}},
			} {
				_, err := owner.Do(updateAuditPrivacyMutation, map[string]any{
					"input": auditPrivacyInput(auditID, overrides),
				})
				require.Error(t, err, name)
			}

			audit := loadAuditPrivacy(t, owner, auditID)
			assert.Nil(t, audit.PrivacyDetails.UpdatedAt)
			assert.Empty(t, audit.PrivacyAuditors)
		},
	)

	t.Run(
		"deleting the audit deletes its details",
		func(t *testing.T) {
			t.Parallel()

			auditID := createPrivacyAudit(t, owner, factory.SafeName("Deleted audit"))
			updateAuditPrivacy(t, owner, auditPrivacyInput(auditID, map[string]any{
				"scope":      "J.1",
				"auditorIds": []string{factory.CreateUser(owner)},
			}))

			owner.MustExecute(`
				mutation($input: DeleteAuditInput!) {
					deleteAudit(input: $input) { deletedAuditId }
				}
			`, map[string]any{"input": map[string]any{"auditId": auditID}}, nil)

			_, err := owner.Do(auditPrivacyQuery, map[string]any{"id": auditID})
			require.Error(t, err)
		},
	)
}

func TestAuditPrivacy_IndependenceConflicts(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)
	sc := newSCIMClient(t, owner)

	sales := createProfileInDepartment(t, sc, "営業 一郎", "営業部")
	admin := createProfileInDepartment(t, sc, "管理 花子", "管理部")
	auditID := createPrivacyAudit(t, owner, factory.SafeName("Independence"))

	audit := updateAuditPrivacy(t, owner, auditPrivacyInput(auditID, map[string]any{
		"auditeeDepartments": []string{"営業部", "開発部"},
		"auditorIds":         []string{sales, admin},
	}))

	require.Len(t, audit.PrivacyIndependenceConflicts, 1)
	assert.Equal(t, sales, audit.PrivacyIndependenceConflicts[0].Auditor.ID)
	assert.Equal(t, "営業部", audit.PrivacyIndependenceConflicts[0].Department)
}

func TestAuditPrivacy_FindingDetails(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)

	t.Run(
		"the category falls back to the finding kind",
		func(t *testing.T) {
			t.Parallel()

			for kind, want := range map[string]*string{
				"MAJOR_NONCONFORMITY": new("NONCONFORMITY"),
				"MINOR_NONCONFORMITY": new("NONCONFORMITY"),
				"OBSERVATION":         new("OBSERVATION"),
				"EXCEPTION":           nil,
			} {
				findingID := createAuditFinding(t, owner, "", kind, "kind "+kind)

				var result struct {
					Node findingPrivacy `json:"node"`
				}
				owner.MustExecute(findingPrivacyQuery, map[string]any{"id": findingID}, &result)

				assert.Nil(t, result.Node.PrivacyDetails.Category, kind)
				assert.Equal(t, want, result.Node.PrivacyDetails.EffectiveCategory, kind)
				assert.Nil(t, result.Node.PrivacyDetails.UpdatedAt, kind)
			}
		},
	)

	t.Run(
		"the category and basis are saved and cleared",
		func(t *testing.T) {
			t.Parallel()

			findingID := createAuditFinding(t, owner, "", "OBSERVATION", "Saved finding")

			finding := updateFindingPrivacy(t, owner, map[string]any{
				"findingId": findingID,
				"category":  "IMPROVEMENT_OPPORTUNITY",
				"basis":     "J.7 安全管理措置",
			})
			assert.Equal(t, new("IMPROVEMENT_OPPORTUNITY"), finding.PrivacyDetails.Category)
			assert.Equal(t, new("IMPROVEMENT_OPPORTUNITY"), finding.PrivacyDetails.EffectiveCategory)
			assert.Equal(t, "J.7 安全管理措置", finding.PrivacyDetails.Basis)
			assert.NotNil(t, finding.PrivacyDetails.UpdatedAt)

			finding = updateFindingPrivacy(t, owner, map[string]any{
				"findingId": findingID,
				"category":  nil,
				"basis":     "",
			})
			assert.Nil(t, finding.PrivacyDetails.Category)
			assert.Equal(t, new("OBSERVATION"), finding.PrivacyDetails.EffectiveCategory)
			assert.Equal(t, "", finding.PrivacyDetails.Basis)
		},
	)

	t.Run(
		"an invalid category is rejected",
		func(t *testing.T) {
			t.Parallel()

			findingID := createAuditFinding(t, owner, "", "OBSERVATION", "Invalid finding")

			_, err := owner.Do(updateFindingPrivacyMutation, map[string]any{
				"input": map[string]any{"findingId": findingID, "category": "MAJOR", "basis": ""},
			})
			require.Error(t, err)
		},
	)
}

func TestAuditPrivacy_Export(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)
	auditor := factory.CreateUser(owner, factory.Attrs{"fullName": "監査 花子"})
	auditID := createPrivacyAudit(t, owner, "2026 年度 内部監査")

	updateAuditPrivacy(t, owner, auditPrivacyInput(auditID, map[string]any{
		"scope":              "J.1〜J.11",
		"criteria":           "JIS Q 15001:2023",
		"auditeeDepartments": []string{"営業部"},
		"externalAuditors":   []string{"外部 太郎"},
		"overallOpinion":     "概ね適合している。",
		"auditorIds":         []string{auditor},
	}))

	major := createAuditFinding(t, owner, auditID, "MAJOR_NONCONFORMITY", "台帳が更新されていない")
	improvement := createAuditFinding(t, owner, auditID, "OBSERVATION", "教育資料を改善できる")
	createAuditFinding(t, owner, "", "OBSERVATION", "別の監査の指摘")

	updateFindingPrivacy(t, owner, map[string]any{"findingId": major, "basis": "J.3"})
	updateFindingPrivacy(t, owner, map[string]any{
		"findingId": improvement,
		"category":  "IMPROVEMENT_OPPORTUNITY",
		"basis":     "J.11",
	})

	f := exportAuditReport(t, owner, auditID)
	assert.Equal(t, []string{"監査概要", "指摘事項一覧"}, f.GetSheetList())

	values := summaryValues(t, f)
	assert.Equal(t, "2026 年度 内部監査", values["監査の名前"])
	assert.Equal(t, "J.1〜J.11", values["監査の範囲"])
	assert.Equal(t, "監査 花子、外部 太郎（外部）", values["監査員"])
	assert.Equal(t, "営業部", values["被監査部門"])
	assert.Equal(t, "計 2 件（不適合（重大） 1 件、不適合 0 件、観察事項 0 件、改善の機会 1 件）", values["指摘事項の件数"])
	assert.Equal(t, "概ね適合している。", values["総合的な所見"])

	rows, err := f.GetRows("指摘事項一覧")
	require.NoError(t, err)
	require.Len(t, rows, 6)

	byDescription := make(map[string][]string)
	for _, row := range rows[4:] {
		byDescription[row[3]] = row
	}
	assert.Equal(t, "不適合（重大）", byDescription["台帳が更新されていない"][2])
	assert.Equal(t, "J.3", byDescription["台帳が更新されていない"][4])
	assert.Equal(t, "改善の機会", byDescription["教育資料を改善できる"][2])
	assert.Equal(t, "未完了", byDescription["教育資料を改善できる"][9])

	t.Run(
		"an audit without findings says so",
		func(t *testing.T) {
			t.Parallel()

			emptyID := createPrivacyAudit(t, owner, factory.SafeName("No findings"))
			rows, err := exportAuditReport(t, owner, emptyID).GetRows("指摘事項一覧")
			require.NoError(t, err)
			require.Len(t, rows, 5)
			assert.Contains(t, rows[4], "（指摘事項なし）")
		},
	)
}

func TestAuditPrivacy_RBAC(t *testing.T) {
	t.Parallel()

	org := testutil.NewOrganizationRoles(t)
	ownerClient := org.Client(t, testutil.RoleOwner)
	auditID := createPrivacyAudit(t, ownerClient, factory.SafeName("Privacy RBAC"))
	findingID := createAuditFinding(t, ownerClient, auditID, "OBSERVATION", "RBAC finding")

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

				_, err := client.Do(updateAuditPrivacyMutation, map[string]any{
					"input": auditPrivacyInput(auditID, map[string]any{"scope": "J.1"}),
				})
				_, findingErr := client.Do(updateFindingPrivacyMutation, map[string]any{
					"input": map[string]any{"findingId": findingID, "basis": "J.1"},
				})
				if tt.updateForbidden {
					testutil.RequireForbiddenError(t, err, "%s should not update audit privacy details", tt.role)
					testutil.RequireForbiddenError(t, findingErr, "%s should not update finding privacy details", tt.role)
				} else {
					require.NoError(t, err, "%s should update audit privacy details", tt.role)
					require.NoError(t, findingErr, "%s should update finding privacy details", tt.role)
				}

				_, err = client.Do(auditPrivacyQuery, map[string]any{"id": auditID})
				require.NoError(t, err, "%s should read audit privacy details", tt.role)

				_, err = client.Do(findingPrivacyQuery, map[string]any{"id": findingID})
				require.NoError(t, err, "%s should read finding privacy details", tt.role)

				_, err = client.Do(exportAuditReportMutation, map[string]any{
					"input": map[string]any{"auditId": auditID},
				})
				require.NoError(t, err, "%s should export the audit report", tt.role)
			},
		)
	}
}

func TestAuditPrivacy_TenantIsolation(t *testing.T) {
	t.Parallel()

	org1Owner := testutil.NewClient(t, testutil.RoleOwner)
	org2Owner := testutil.NewClient(t, testutil.RoleOwner)

	org1AuditID := createPrivacyAudit(t, org1Owner, "Org1 audit")
	org1FindingID := createAuditFinding(t, org1Owner, org1AuditID, "OBSERVATION", "Org1 finding")
	org1ProfileID := factory.CreateUser(org1Owner)
	org2AuditID := createPrivacyAudit(t, org2Owner, "Org2 audit")

	t.Run(
		"cannot update the audit details of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(updateAuditPrivacyMutation, map[string]any{
				"input": auditPrivacyInput(org1AuditID, map[string]any{"scope": "J.1"}),
			})
			require.Error(t, err)
		},
	)

	t.Run(
		"cannot choose an auditor of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(updateAuditPrivacyMutation, map[string]any{
				"input": auditPrivacyInput(org2AuditID, map[string]any{"auditorIds": []string{org1ProfileID}}),
			})
			require.Error(t, err)

			assert.Empty(t, loadAuditPrivacy(t, org2Owner, org2AuditID).PrivacyAuditors)
		},
	)

	t.Run(
		"cannot update the finding details of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(updateFindingPrivacyMutation, map[string]any{
				"input": map[string]any{"findingId": org1FindingID, "basis": "J.1"},
			})
			require.Error(t, err)
		},
	)

	t.Run(
		"cannot export the audit report of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(exportAuditReportMutation, map[string]any{
				"input": map[string]any{"auditId": org1AuditID},
			})
			require.Error(t, err)
		},
	)
}
