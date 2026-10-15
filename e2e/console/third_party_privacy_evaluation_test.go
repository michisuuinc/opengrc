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
	thirdPartyEvaluationFields = `
		id
		privacyEvaluation {
			kind
			adHocReason
			entrustedWork
			personalData
			storageCountries
			evaluator { id fullName }
			evaluatedOn
			answers { item result note }
			judgement
			conditions
			approver { id fullName }
			approvedOn
			updatedAt
		}
	`

	thirdPartyEvaluationQuery = `
		query($id: ID!) {
			node(id: $id) {
				... on ThirdPartyRiskAssessment {` + thirdPartyEvaluationFields + `}
			}
		}
	`

	updateThirdPartyEvaluationMutation = `
		mutation($input: UpdateThirdPartyPrivacyEvaluationInput!) {
			updateThirdPartyPrivacyEvaluation(input: $input) {
				thirdPartyRiskAssessment {` + thirdPartyEvaluationFields + `}
			}
		}
	`

	exportThirdPartyEvaluationMutation = `
		mutation($input: ExportThirdPartyEvaluationSheetInput!) {
			exportThirdPartyEvaluationSheet(input: $input) {
				data
			}
		}
	`

	thirdPartyEvaluationExpiresAt = "2030-01-01T00:00:00Z"
)

var thirdPartyEvaluationItems = []string{
	"CERTIFICATION",
	"MANAGEMENT_STRUCTURE",
	"RULES",
	"EMPLOYEE_TRAINING",
	"PHYSICAL_SECURITY",
	"ACCESS_CONTROL",
	"TECHNICAL_SECURITY",
	"SUBCONTRACTING",
	"INCIDENT_RESPONSE",
	"PAST_INCIDENTS",
	"RETURN_AND_DELETION",
	"FOREIGN_STORAGE",
	"CONTRACT",
}

type (
	thirdPartyEvaluationProfile struct {
		ID       string `json:"id"`
		FullName string `json:"fullName"`
	}

	thirdPartyEvaluationAnswer struct {
		Item   string  `json:"item"`
		Result *string `json:"result"`
		Note   string  `json:"note"`
	}

	thirdPartyEvaluation struct {
		ID                string `json:"id"`
		PrivacyEvaluation struct {
			Kind             *string                      `json:"kind"`
			AdHocReason      string                       `json:"adHocReason"`
			EntrustedWork    string                       `json:"entrustedWork"`
			PersonalData     string                       `json:"personalData"`
			StorageCountries string                       `json:"storageCountries"`
			Evaluator        *thirdPartyEvaluationProfile `json:"evaluator"`
			EvaluatedOn      *string                      `json:"evaluatedOn"`
			Answers          []thirdPartyEvaluationAnswer `json:"answers"`
			Judgement        *string                      `json:"judgement"`
			Conditions       string                       `json:"conditions"`
			Approver         *thirdPartyEvaluationProfile `json:"approver"`
			ApprovedOn       *string                      `json:"approvedOn"`
			UpdatedAt        *string                      `json:"updatedAt"`
		} `json:"privacyEvaluation"`
	}
)

func (e thirdPartyEvaluation) answer(item string) thirdPartyEvaluationAnswer {
	for _, a := range e.PrivacyEvaluation.Answers {
		if a.Item == item {
			return a
		}
	}

	return thirdPartyEvaluationAnswer{}
}

func thirdPartyEvaluationInput(riskAssessmentID string, overrides map[string]any) map[string]any {
	input := map[string]any{
		"riskAssessmentId": riskAssessmentID,
		"adHocReason":      "",
		"entrustedWork":    "",
		"personalData":     "",
		"storageCountries": "",
		"answers":          []map[string]any{},
		"conditions":       "",
	}
	for k, v := range overrides {
		input[k] = v
	}
	return input
}

// suitableAnswers answers every item as suitable, except the given ones.
func suitableAnswers(overrides map[string]string) []map[string]any {
	answers := make([]map[string]any, 0, len(thirdPartyEvaluationItems))
	for _, item := range thirdPartyEvaluationItems {
		result := "SUITABLE"
		if r, ok := overrides[item]; ok {
			result = r
		}
		answers = append(answers, map[string]any{"item": item, "result": result, "note": ""})
	}
	return answers
}

func createEvaluatedThirdParty(t *testing.T, c *testutil.Client, name string) (string, string) {
	t.Helper()

	thirdPartyID := factory.NewThirdParty(c).WithName(name).Create()

	return thirdPartyID, createEvaluationRiskAssessment(t, c, thirdPartyID)
}

func createEvaluationRiskAssessment(t *testing.T, c *testutil.Client, thirdPartyID string) string {
	t.Helper()

	var result struct {
		CreateThirdPartyRiskAssessment struct {
			ThirdPartyRiskAssessmentEdge struct {
				Node struct {
					ID string `json:"id"`
				} `json:"node"`
			} `json:"thirdPartyRiskAssessmentEdge"`
		} `json:"createThirdPartyRiskAssessment"`
	}

	c.MustExecute(`
		mutation($input: CreateThirdPartyRiskAssessmentInput!) {
			createThirdPartyRiskAssessment(input: $input) {
				thirdPartyRiskAssessmentEdge { node { id } }
			}
		}
	`, map[string]any{
		"input": map[string]any{
			"thirdPartyId":    thirdPartyID,
			"expiresAt":       thirdPartyEvaluationExpiresAt,
			"dataSensitivity": "HIGH",
			"businessImpact":  "MEDIUM",
		},
	}, &result)

	return result.CreateThirdPartyRiskAssessment.ThirdPartyRiskAssessmentEdge.Node.ID
}

// createOutsourcedActivity creates a register entry outsourced to the third
// parties.
func createOutsourcedActivity(t *testing.T, c *testutil.Client, name string, thirdPartyIDs ...string) {
	t.Helper()

	c.MustExecute(`
		mutation($input: CreateProcessingActivityInput!) {
			createProcessingActivity(input: $input) {
				processingActivityEdge { node { id } }
			}
		}
	`, map[string]any{
		"input": map[string]any{
			"organizationId":                       c.GetOrganizationID().String(),
			"name":                                 name,
			"specialOrCriminalData":                "NO",
			"lawfulBasis":                          "CONSENT",
			"internationalTransfers":               false,
			"dataProtectionImpactAssessmentNeeded": "NOT_NEEDED",
			"transferImpactAssessmentNeeded":       "NOT_NEEDED",
			"role":                                 "CONTROLLER",
			"thirdPartyIds":                        thirdPartyIDs,
		},
	}, nil)
}

func updateThirdPartyEvaluation(t *testing.T, c *testutil.Client, input map[string]any) thirdPartyEvaluation {
	t.Helper()

	var result struct {
		UpdateThirdPartyPrivacyEvaluation struct {
			ThirdPartyRiskAssessment thirdPartyEvaluation `json:"thirdPartyRiskAssessment"`
		} `json:"updateThirdPartyPrivacyEvaluation"`
	}

	c.MustExecute(updateThirdPartyEvaluationMutation, map[string]any{"input": input}, &result)

	return result.UpdateThirdPartyPrivacyEvaluation.ThirdPartyRiskAssessment
}

func loadThirdPartyEvaluation(t *testing.T, c *testutil.Client, riskAssessmentID string) thirdPartyEvaluation {
	t.Helper()

	var result struct {
		Node thirdPartyEvaluation `json:"node"`
	}

	c.MustExecute(thirdPartyEvaluationQuery, map[string]any{"id": riskAssessmentID}, &result)

	return result.Node
}

func exportThirdPartyEvaluation(t *testing.T, c *testutil.Client) *excelize.File {
	t.Helper()

	var result struct {
		ExportThirdPartyEvaluationSheet struct {
			Data string `json:"data"`
		} `json:"exportThirdPartyEvaluationSheet"`
	}

	c.MustExecute(exportThirdPartyEvaluationMutation, map[string]any{
		"input": map[string]any{"organizationId": c.GetOrganizationID().String()},
	}, &result)

	data := result.ExportThirdPartyEvaluationSheet.Data
	require.True(t, strings.HasPrefix(data, xlsxDataURLPrefix), "export should be an xlsx data URL")

	workbook, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, xlsxDataURLPrefix))
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(workbook))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	return f
}

// sheetDataRows returns the rows below the header row, whose first cell is
// header.
func sheetDataRows(t *testing.T, f *excelize.File, sheet string, header string) [][]string {
	t.Helper()

	rows, err := f.GetRows(sheet)
	require.NoError(t, err)

	for i, row := range rows {
		if len(row) > 0 && row[0] == header {
			return rows[i+1:]
		}
	}

	require.Failf(t, "header row not found", "sheet %s", sheet)

	return nil
}

func TestThirdPartyEvaluation_Details(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)

	t.Run(
		"an assessment without an evaluation lists every item unanswered",
		func(t *testing.T) {
			t.Parallel()

			_, assessmentID := createEvaluatedThirdParty(t, owner, factory.SafeName("Empty"))

			evaluation := loadThirdPartyEvaluation(t, owner, assessmentID)

			assert.Equal(t, assessmentID, evaluation.ID)
			assert.Nil(t, evaluation.PrivacyEvaluation.UpdatedAt)
			assert.Nil(t, evaluation.PrivacyEvaluation.Kind)
			assert.Nil(t, evaluation.PrivacyEvaluation.Evaluator)
			require.Len(t, evaluation.PrivacyEvaluation.Answers, len(thirdPartyEvaluationItems))
			for i, a := range evaluation.PrivacyEvaluation.Answers {
				assert.Equal(t, thirdPartyEvaluationItems[i], a.Item)
				assert.Nil(t, a.Result)
				assert.Empty(t, a.Note)
			}
		},
	)

	t.Run(
		"the evaluation is saved and replaced",
		func(t *testing.T) {
			t.Parallel()

			_, assessmentID := createEvaluatedThirdParty(t, owner, factory.SafeName("Saved"))
			evaluator := factory.CreateUser(owner, factory.Attrs{"fullName": "評価 一郎"})
			approver := factory.CreateUser(owner, factory.Attrs{"fullName": "管理 花子"})

			answers := suitableAnswers(map[string]string{"FOREIGN_STORAGE": "UNSUITABLE"})
			answers[0]["note"] = "  認証書の写し  "

			saved := updateThirdPartyEvaluation(t, owner, thirdPartyEvaluationInput(assessmentID, map[string]any{
				"kind":             "AD_HOC",
				"adHocReason":      " 再委託先の変更 ",
				"entrustedWork":    "顧客データの保管",
				"personalData":     "氏名、メールアドレス",
				"storageCountries": "日本、米国",
				"evaluatorId":      evaluator,
				"evaluatedOn":      "2026-04-02",
				"answers":          answers,
				"judgement":        "CONDITIONALLY_APPROVED",
				"conditions":       "米国の制度を確認する",
				"approverId":       approver,
				"approvedOn":       "2026-04-03",
			}))

			e := saved.PrivacyEvaluation
			require.NotNil(t, e.UpdatedAt)
			assert.Equal(t, "AD_HOC", *e.Kind)
			assert.Equal(t, "再委託先の変更", e.AdHocReason)
			assert.Equal(t, "日本、米国", e.StorageCountries)
			require.NotNil(t, e.Evaluator)
			assert.Equal(t, "評価 一郎", e.Evaluator.FullName)
			assert.Equal(t, "2026-04-02", *e.EvaluatedOn)
			assert.Equal(t, "CONDITIONALLY_APPROVED", *e.Judgement)
			require.NotNil(t, e.Approver)
			assert.Equal(t, approver, e.Approver.ID)
			assert.Equal(t, "2026-04-03", *e.ApprovedOn)
			assert.Equal(t, "認証書の写し", saved.answer("CERTIFICATION").Note)
			assert.Equal(t, "UNSUITABLE", *saved.answer("FOREIGN_STORAGE").Result)

			assert.Equal(t, saved, loadThirdPartyEvaluation(t, owner, assessmentID))

			replaced := updateThirdPartyEvaluation(t, owner, thirdPartyEvaluationInput(assessmentID, map[string]any{
				"kind":        "PERIODIC",
				"adHocReason": "使われない理由",
				"answers": []map[string]any{
					{"item": "CONTRACT", "result": "NOT_APPLICABLE", "note": ""},
				},
			}))

			r := replaced.PrivacyEvaluation
			assert.Equal(t, "PERIODIC", *r.Kind)
			assert.Empty(t, r.AdHocReason, "a reason is kept only for an ad hoc evaluation")
			assert.Nil(t, r.Evaluator)
			assert.Nil(t, r.Judgement)
			assert.Nil(t, r.ApprovedOn)
			assert.Nil(t, replaced.answer("CERTIFICATION").Result, "answers left out are cleared")
			assert.Equal(t, "NOT_APPLICABLE", *replaced.answer("CONTRACT").Result)
		},
	)

	t.Run(
		"inconsistent or invalid input is rejected and nothing changes",
		func(t *testing.T) {
			t.Parallel()

			_, assessmentID := createEvaluatedThirdParty(t, owner, factory.SafeName("Invalid"))
			_, otherAssessmentID := createEvaluatedThirdParty(t, owner, factory.SafeName("Other"))
			evaluator := factory.CreateUser(owner)

			for name, overrides := range map[string]map[string]any{
				"approved with an unsuitable item": {
					"judgement": "APPROVED",
					"answers":   suitableAnswers(map[string]string{"CONTRACT": "UNSUITABLE"}),
				},
				"conditional approval without conditions": {"judgement": "CONDITIONALLY_APPROVED", "conditions": "  "},
				"ad hoc without a reason":                 {"kind": "AD_HOC"},
				"approved before evaluated":               {"evaluatedOn": "2026-04-02", "approvedOn": "2026-04-01"},
				"invalid date":                            {"evaluatedOn": "2026/04/02"},
				"repeated item": {"answers": []map[string]any{
					{"item": "RULES", "result": "SUITABLE", "note": ""},
					{"item": "RULES", "result": "UNSUITABLE", "note": ""},
				}},
				"evaluator is not a member": {"evaluatorId": otherAssessmentID},
				"approver is not a profile": {"approverId": assessmentID},
			} {
				_, err := owner.Do(updateThirdPartyEvaluationMutation, map[string]any{
					"input": thirdPartyEvaluationInput(assessmentID, overrides),
				})
				require.Error(t, err, name)
			}

			// A valid evaluator still saves.
			updateThirdPartyEvaluation(t, owner, thirdPartyEvaluationInput(otherAssessmentID, map[string]any{
				"evaluatorId": evaluator,
			}))

			assert.Nil(t, loadThirdPartyEvaluation(t, owner, assessmentID).PrivacyEvaluation.UpdatedAt)
		},
	)

	t.Run(
		"deleting the third party deletes the evaluation",
		func(t *testing.T) {
			t.Parallel()

			thirdPartyID, assessmentID := createEvaluatedThirdParty(t, owner, factory.SafeName("Deleted"))
			updateThirdPartyEvaluation(t, owner, thirdPartyEvaluationInput(assessmentID, map[string]any{
				"entrustedWork": "発送代行",
				"answers":       suitableAnswers(nil),
			}))

			owner.MustExecute(`
				mutation($input: DeleteThirdPartyInput!) {
					deleteThirdParty(input: $input) { deletedThirdPartyId }
				}
			`, map[string]any{"input": map[string]any{"thirdPartyId": thirdPartyID}}, nil)

			_, err := owner.Do(thirdPartyEvaluationQuery, map[string]any{"id": assessmentID})
			require.Error(t, err)
		},
	)
}

func TestThirdPartyEvaluation_Export(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)
	evaluator := factory.CreateUser(owner, factory.Attrs{"fullName": "評価 一郎"})
	approver := factory.CreateUser(owner, factory.Attrs{"fullName": "管理 花子"})

	cloudID, cloudAssessment := createEvaluatedThirdParty(t, owner, "クラウド会社")
	payrollID, _ := createEvaluatedThirdParty(t, owner, "給与計算会社")
	shredderID := factory.NewThirdParty(owner).WithName("廃棄業者").Create()
	factory.NewThirdParty(owner).WithName("無関係の取引先").Create()

	createOutsourcedActivity(t, owner, "顧客管理", cloudID, payrollID)
	createOutsourcedActivity(t, owner, "書類の廃棄", shredderID)

	updateThirdPartyEvaluation(t, owner, thirdPartyEvaluationInput(cloudAssessment, map[string]any{
		"kind":             "NEW",
		"entrustedWork":    "顧客データの保管",
		"storageCountries": "日本",
		"evaluatorId":      evaluator,
		"evaluatedOn":      "2026-04-02",
		"answers":          suitableAnswers(map[string]string{"CERTIFICATION": "NOT_APPLICABLE"}),
		"judgement":        "APPROVED",
		"approverId":       approver,
		"approvedOn":       "2026-04-03",
	}))

	f := exportThirdPartyEvaluation(t, owner)

	list := sheetDataRows(t, f, "委託先一覧", "No.")
	require.Len(t, list, 3, "only third parties outsourced to or evaluated are listed")

	byName := make(map[string][]string, len(list))
	for _, row := range list {
		require.GreaterOrEqual(t, len(row), 2)
		byName[row[1]] = row
	}

	cloud := byName["クラウド会社"]
	require.NotNil(t, cloud)
	assert.Equal(t, "顧客データの保管", cloud[3])
	assert.Equal(t, "顧客管理", cloud[5])
	assert.Equal(t, "新規", cloud[6])
	assert.Equal(t, "2026-04-02", cloud[7])
	assert.Equal(t, "評価 一郎", cloud[8])
	assert.Equal(t, "適 12・否 0・該当なし 1", cloud[9])
	assert.Equal(t, "委託可", cloud[10])
	assert.Equal(t, "管理 花子", cloud[12])
	assert.Equal(t, "2026-04-03", cloud[13])
	assert.Equal(t, "2030-01-01", cloud[14])

	payroll := byName["給与計算会社"]
	require.NotNil(t, payroll)
	assert.Equal(t, "（評価未記入）", payroll[10])

	shredder := byName["廃棄業者"]
	require.NotNil(t, shredder)
	assert.Equal(t, "（評価未登録）", shredder[10])
	assert.Equal(t, "評価がない", shredder[15])

	items := sheetDataRows(t, f, "評価項目", "委託先")
	require.Len(t, items, 13, "the checklist of the one evaluation")
	assert.Equal(t, "クラウド会社", items[0][0])
	assert.Equal(t, "認証の取得", items[0][3])
	assert.Equal(t, "該当なし", items[0][5])
	assert.Equal(t, "外国でのデータの保管", items[11][3])
	assert.Equal(t, "保管する国：日本", items[11][6])
}

func TestThirdPartyEvaluation_RBAC(t *testing.T) {
	t.Parallel()

	org := testutil.NewOrganizationRoles(t)
	ownerClient := org.Client(t, testutil.RoleOwner)
	_, assessmentID := createEvaluatedThirdParty(t, ownerClient, factory.SafeName("Evaluation RBAC"))

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

				_, err := client.Do(updateThirdPartyEvaluationMutation, map[string]any{
					"input": thirdPartyEvaluationInput(assessmentID, map[string]any{"entrustedWork": "発送代行"}),
				})
				if tt.updateForbidden {
					testutil.RequireForbiddenError(t, err, "%s should not update the evaluation", tt.role)
				} else {
					require.NoError(t, err, "%s should update the evaluation", tt.role)
				}

				_, err = client.Do(thirdPartyEvaluationQuery, map[string]any{"id": assessmentID})
				require.NoError(t, err, "%s should read the evaluation", tt.role)

				_, err = client.Do(exportThirdPartyEvaluationMutation, map[string]any{
					"input": map[string]any{"organizationId": client.GetOrganizationID().String()},
				})
				require.NoError(t, err, "%s should export the outsourcing evaluation", tt.role)
			},
		)
	}
}

func TestThirdPartyEvaluation_TenantIsolation(t *testing.T) {
	t.Parallel()

	org1Owner := testutil.NewClient(t, testutil.RoleOwner)
	org2Owner := testutil.NewClient(t, testutil.RoleOwner)

	_, org1AssessmentID := createEvaluatedThirdParty(t, org1Owner, "Org1 third party")
	org1ProfileID := factory.CreateUser(org1Owner)
	_, org2AssessmentID := createEvaluatedThirdParty(t, org2Owner, "Org2 third party")

	t.Run(
		"cannot read or update the evaluation of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(thirdPartyEvaluationQuery, map[string]any{"id": org1AssessmentID})
			require.Error(t, err)

			_, err = org2Owner.Do(updateThirdPartyEvaluationMutation, map[string]any{
				"input": thirdPartyEvaluationInput(org1AssessmentID, map[string]any{"entrustedWork": "発送代行"}),
			})
			require.Error(t, err)
		},
	)

	t.Run(
		"cannot choose an evaluator or approver of another organization",
		func(t *testing.T) {
			t.Parallel()

			for _, field := range []string{"evaluatorId", "approverId"} {
				_, err := org2Owner.Do(updateThirdPartyEvaluationMutation, map[string]any{
					"input": thirdPartyEvaluationInput(org2AssessmentID, map[string]any{field: org1ProfileID}),
				})
				require.Error(t, err, field)
			}

			assert.Nil(t, loadThirdPartyEvaluation(t, org2Owner, org2AssessmentID).PrivacyEvaluation.UpdatedAt)
		},
	)

	t.Run(
		"cannot export the outsourcing evaluation of another organization",
		func(t *testing.T) {
			t.Parallel()

			_, err := org2Owner.Do(exportThirdPartyEvaluationMutation, map[string]any{
				"input": map[string]any{"organizationId": org1Owner.GetOrganizationID().String()},
			})
			require.Error(t, err)
		},
	)
}
