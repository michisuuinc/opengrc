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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"go.probo.inc/probo/e2e/internal/factory"
	"go.probo.inc/probo/e2e/internal/testutil"
)

const (
	trainingTimeZone = "Asia/Tokyo"

	trainingPlanFields = `
		id
		fiscalYear
		name
		kind
		method
		startsOn
		endsOn
		timeZone
		testRequired
		passScore
		materials { documentId title }
		summary { totalCount completedCount notAttendedCount recordedCount }
		participants {
			id
			profileId
			fullName
			status
			attendedOn
			attendanceSource
			signedMaterialCount
			signedOutsidePeriodCount
			manualAttendedOn
			testScore
			testPassed
			manualTestPassed
			hasRecord
			note
		}
	`

	createTrainingPlanMutation = `
		mutation($input: CreateTrainingPlanInput!) {
			createTrainingPlan(input: $input) {
				trainingPlanEdge { node { ` + trainingPlanFields + ` } }
			}
		}
	`

	updateTrainingPlanMutation = `
		mutation($input: UpdateTrainingPlanInput!) {
			updateTrainingPlan(input: $input) {
				trainingPlan { ` + trainingPlanFields + ` }
			}
		}
	`

	deleteTrainingPlanMutation = `
		mutation($input: DeleteTrainingPlanInput!) {
			deleteTrainingPlan(input: $input) { deletedTrainingPlanId }
		}
	`

	setTrainingPlanMaterialsMutation = `
		mutation($input: SetTrainingPlanMaterialsInput!) {
			setTrainingPlanMaterials(input: $input) {
				trainingPlan { ` + trainingPlanFields + ` }
			}
		}
	`

	addTrainingParticipantsMutation = `
		mutation($input: AddTrainingParticipantsInput!) {
			addTrainingParticipants(input: $input) {
				addedCount
				trainingPlan { ` + trainingPlanFields + ` }
			}
		}
	`

	removeTrainingParticipantMutation = `
		mutation($input: RemoveTrainingParticipantInput!) {
			removeTrainingParticipant(input: $input) {
				deletedTrainingParticipantId
				trainingPlan { ` + trainingPlanFields + ` }
			}
		}
	`

	updateTrainingParticipantMutation = `
		mutation($input: UpdateTrainingParticipantInput!) {
			updateTrainingParticipant(input: $input) {
				trainingPlan { ` + trainingPlanFields + ` }
			}
		}
	`

	requestTrainingSignaturesMutation = `
		mutation($input: RequestTrainingSignaturesInput!) {
			requestTrainingSignatures(input: $input) {
				requestedCount
				trainingPlan { ` + trainingPlanFields + ` }
			}
		}
	`

	exportTrainingRecordsMutation = `
		mutation($input: ExportTrainingRecordsInput!) {
			exportTrainingRecords(input: $input) { data }
		}
	`

	trainingPlanQuery = `
		query($id: ID!) {
			node(id: $id) {
				... on TrainingPlan { ` + trainingPlanFields + ` }
			}
		}
	`

	trainingPlansQuery = `
		query($id: ID!, $fiscalYear: Int) {
			node(id: $id) {
				... on Organization {
					trainingPlans(first: 50, filter: { fiscalYear: $fiscalYear }) {
						totalCount
						edges { node { id name fiscalYear } }
					}
				}
			}
		}
	`
)

type (
	trainingParticipant struct {
		ID                  string  `json:"id"`
		ProfileID           string  `json:"profileId"`
		FullName            string  `json:"fullName"`
		Status              string  `json:"status"`
		AttendedOn          *string `json:"attendedOn"`
		AttendanceSource    *string `json:"attendanceSource"`
		SignedMaterialCount int     `json:"signedMaterialCount"`
		SignedOutside       int     `json:"signedOutsidePeriodCount"`
		ManualAttendedOn    *string `json:"manualAttendedOn"`
		TestScore           *int    `json:"testScore"`
		TestPassed          *bool   `json:"testPassed"`
		ManualTestPassed    *bool   `json:"manualTestPassed"`
		HasRecord           bool    `json:"hasRecord"`
		Note                string  `json:"note"`
	}

	trainingPlan struct {
		ID           string `json:"id"`
		FiscalYear   int    `json:"fiscalYear"`
		Name         string `json:"name"`
		Kind         string `json:"kind"`
		Method       string `json:"method"`
		StartsOn     string `json:"startsOn"`
		EndsOn       string `json:"endsOn"`
		TimeZone     string `json:"timeZone"`
		TestRequired bool   `json:"testRequired"`
		PassScore    *int   `json:"passScore"`
		Materials    []struct {
			DocumentID string `json:"documentId"`
			Title      string `json:"title"`
		} `json:"materials"`
		Summary struct {
			TotalCount       int `json:"totalCount"`
			CompletedCount   int `json:"completedCount"`
			NotAttendedCount int `json:"notAttendedCount"`
			RecordedCount    int `json:"recordedCount"`
		} `json:"summary"`
		Participants []trainingParticipant `json:"participants"`
	}
)

func (p trainingPlan) participant(t *testing.T, profileID string) trainingParticipant {
	t.Helper()

	for _, participant := range p.Participants {
		if participant.ProfileID == profileID {
			return participant
		}
	}

	t.Fatalf("participant %s not found", profileID)

	return trainingParticipant{}
}

// trainingToday returns today in the plan time zone, so a plan from
// yesterday to tomorrow always contains the signatures made by the test.
func trainingToday() time.Time {
	location, _ := time.LoadLocation(trainingTimeZone)

	return time.Now().In(location)
}

func trainingPlanInput(c *testutil.Client, overrides map[string]any) map[string]any {
	today := trainingToday()
	input := map[string]any{
		"organizationId": c.GetOrganizationID().String(),
		"fiscalYear":     today.Year(),
		"name":           factory.SafeName("個人情報保護 定期教育"),
		"kind":           "PERIODIC",
		"method":         "E_LEARNING",
		"instructor":     "個人情報保護管理者",
		"startsOn":       today.AddDate(0, 0, -1).Format(time.DateOnly),
		"endsOn":         today.AddDate(0, 0, 1).Format(time.DateOnly),
		"timeZone":       trainingTimeZone,
		"testRequired":   false,
	}
	for key, value := range overrides {
		input[key] = value
	}

	return input
}

func createTrainingPlan(t *testing.T, c *testutil.Client, overrides map[string]any) trainingPlan {
	t.Helper()

	var result struct {
		CreateTrainingPlan struct {
			TrainingPlanEdge struct {
				Node trainingPlan `json:"node"`
			} `json:"trainingPlanEdge"`
		} `json:"createTrainingPlan"`
	}

	c.MustExecute(createTrainingPlanMutation, map[string]any{"input": trainingPlanInput(c, overrides)}, &result)

	return result.CreateTrainingPlan.TrainingPlanEdge.Node
}

func loadTrainingPlan(t *testing.T, c *testutil.Client, id string) trainingPlan {
	t.Helper()

	var result struct {
		Node trainingPlan `json:"node"`
	}

	c.MustExecute(trainingPlanQuery, map[string]any{"id": id}, &result)

	return result.Node
}

func setTrainingMaterials(t *testing.T, c *testutil.Client, planID string, documentIDs ...string) trainingPlan {
	t.Helper()

	var result struct {
		SetTrainingPlanMaterials struct {
			TrainingPlan trainingPlan `json:"trainingPlan"`
		} `json:"setTrainingPlanMaterials"`
	}

	c.MustExecute(setTrainingPlanMaterialsMutation, map[string]any{
		"input": map[string]any{"trainingPlanId": planID, "documentIds": documentIDs},
	}, &result)

	return result.SetTrainingPlanMaterials.TrainingPlan
}

func addTrainingParticipants(t *testing.T, c *testutil.Client, planID string, profileIDs ...string) (int, trainingPlan) {
	t.Helper()

	var result struct {
		AddTrainingParticipants struct {
			AddedCount   int          `json:"addedCount"`
			TrainingPlan trainingPlan `json:"trainingPlan"`
		} `json:"addTrainingParticipants"`
	}

	c.MustExecute(addTrainingParticipantsMutation, map[string]any{
		"input": map[string]any{"trainingPlanId": planID, "profileIds": profileIDs},
	}, &result)

	return result.AddTrainingParticipants.AddedCount, result.AddTrainingParticipants.TrainingPlan
}

func updateTrainingParticipant(t *testing.T, c *testutil.Client, input map[string]any) trainingPlan {
	t.Helper()

	var result struct {
		UpdateTrainingParticipant struct {
			TrainingPlan trainingPlan `json:"trainingPlan"`
		} `json:"updateTrainingParticipant"`
	}

	c.MustExecute(updateTrainingParticipantMutation, map[string]any{"input": input}, &result)

	return result.UpdateTrainingParticipant.TrainingPlan
}

func requestTrainingSignatures(t *testing.T, c *testutil.Client, planID string) (int, trainingPlan) {
	t.Helper()

	var result struct {
		RequestTrainingSignatures struct {
			RequestedCount int          `json:"requestedCount"`
			TrainingPlan   trainingPlan `json:"trainingPlan"`
		} `json:"requestTrainingSignatures"`
	}

	c.MustExecute(requestTrainingSignaturesMutation, map[string]any{
		"input": map[string]any{"trainingPlanId": planID},
	}, &result)

	return result.RequestTrainingSignatures.RequestedCount, result.RequestTrainingSignatures.TrainingPlan
}

// publishedTrainingMaterial creates and publishes a document and returns its
// ID and published version ID.
func publishedTrainingMaterial(t *testing.T, owner *testutil.Client) (string, string) {
	t.Helper()

	docID, _ := createTestDocument(t, owner)
	approveTestDocument(t, owner, docID)

	return docID, latestDocumentVersionID(t, owner, docID)
}

func exportTrainingRows(t *testing.T, c *testutil.Client, fiscalYear int) [][]string {
	t.Helper()

	var result struct {
		ExportTrainingRecords struct {
			Data string `json:"data"`
		} `json:"exportTrainingRecords"`
	}

	c.MustExecute(exportTrainingRecordsMutation, map[string]any{
		"input": map[string]any{"organizationId": c.GetOrganizationID().String(), "fiscalYear": fiscalYear},
	}, &result)

	data := result.ExportTrainingRecords.Data
	require.True(t, strings.HasPrefix(data, xlsxDataURLPrefix), "export should be an xlsx data URL")

	workbook, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, xlsxDataURLPrefix))
	require.NoError(t, err)

	f, err := excelize.OpenReader(bytes.NewReader(workbook))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	rows, err := f.GetRows("教育実施記録")
	require.NoError(t, err)

	return rows
}

func TestTraining_PlanLifecycle(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)

	plan := createTrainingPlan(t, owner, map[string]any{"fiscalYear": 2031, "testRequired": true, "passScore": 80})
	assert.Equal(t, 2031, plan.FiscalYear)
	assert.Equal(t, "PERIODIC", plan.Kind)
	assert.Equal(t, trainingTimeZone, plan.TimeZone)
	require.NotNil(t, plan.PassScore)
	assert.Equal(t, 80, *plan.PassScore)
	assert.Empty(t, plan.Participants)

	createTrainingPlan(t, owner, map[string]any{"fiscalYear": 2032})

	var list struct {
		Node struct {
			TrainingPlans struct {
				TotalCount int `json:"totalCount"`
				Edges      []struct {
					Node struct {
						ID string `json:"id"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"trainingPlans"`
		} `json:"node"`
	}

	owner.MustExecute(trainingPlansQuery, map[string]any{"id": owner.GetOrganizationID().String(), "fiscalYear": 2031}, &list)
	assert.Equal(t, 1, list.Node.TrainingPlans.TotalCount)
	require.Len(t, list.Node.TrainingPlans.Edges, 1)
	assert.Equal(t, plan.ID, list.Node.TrainingPlans.Edges[0].Node.ID)

	var updated struct {
		UpdateTrainingPlan struct {
			TrainingPlan trainingPlan `json:"trainingPlan"`
		} `json:"updateTrainingPlan"`
	}

	owner.MustExecute(updateTrainingPlanMutation, map[string]any{
		"input": map[string]any{"trainingPlanId": plan.ID, "name": "臨時教育（規程改訂）", "kind": "AD_HOC"},
	}, &updated)
	assert.Equal(t, "臨時教育（規程改訂）", updated.UpdateTrainingPlan.TrainingPlan.Name)
	assert.Equal(t, "AD_HOC", updated.UpdateTrainingPlan.TrainingPlan.Kind)
	require.NotNil(t, updated.UpdateTrainingPlan.TrainingPlan.PassScore, "an omitted pass score is kept")
	assert.Equal(t, 80, *updated.UpdateTrainingPlan.TrainingPlan.PassScore)

	owner.MustExecute(updateTrainingPlanMutation, map[string]any{
		"input": map[string]any{"trainingPlanId": plan.ID, "passScore": nil},
	}, &updated)
	assert.Nil(t, updated.UpdateTrainingPlan.TrainingPlan.PassScore, "a null pass score is cleared")

	owner.MustExecute(deleteTrainingPlanMutation, map[string]any{
		"input": map[string]any{"trainingPlanId": plan.ID},
	}, &struct{}{})

	_, err := owner.Do(trainingPlanQuery, map[string]any{"id": plan.ID})
	require.Error(t, err, "a deleted plan is not found")
}

func TestTraining_InvalidPlans(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)

	tests := []struct {
		name      string
		overrides map[string]any
	}{
		{name: "ends before it starts", overrides: map[string]any{"startsOn": "2026-10-31", "endsOn": "2026-10-01"}},
		{name: "date is not a date", overrides: map[string]any{"startsOn": "2026/10/01"}},
		{name: "unknown time zone", overrides: map[string]any{"timeZone": "Mars/Base"}},
		{name: "pass score above 100", overrides: map[string]any{"passScore": 101}},
		{name: "fiscal year out of range", overrides: map[string]any{"fiscalYear": 1999}},
		{name: "empty name", overrides: map[string]any{"name": ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := owner.Do(createTrainingPlanMutation, map[string]any{"input": trainingPlanInput(owner, tt.overrides)})
			require.Error(t, err)
		})
	}
}

func TestTraining_SignatureCompletesTraining(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)
	employee := testutil.NewClientInOrg(t, testutil.RoleEmployee, owner)
	employeeProfileID := employee.GetProfileID().String()

	policyID, policyVersionID := publishedTrainingMaterial(t, owner)
	rulesID, rulesVersionID := publishedTrainingMaterial(t, owner)

	plan := createTrainingPlan(t, owner, nil)
	plan = setTrainingMaterials(t, owner, plan.ID, policyID, rulesID)
	require.Len(t, plan.Materials, 2)

	added, plan := addTrainingParticipants(t, owner, plan.ID, employeeProfileID)
	assert.Equal(t, 1, added)
	assert.Equal(t, "NOT_ATTENDED", plan.participant(t, employeeProfileID).Status)

	added, _ = addTrainingParticipants(t, owner, plan.ID, employeeProfileID)
	assert.Equal(t, 0, added, "adding a participant twice is a no-op")

	requested, _ := requestTrainingSignatures(t, owner, plan.ID)
	assert.Equal(t, 1, requested)

	signDocumentVersion(t, employee, policyVersionID)

	plan = loadTrainingPlan(t, owner, plan.ID)
	participant := plan.participant(t, employeeProfileID)
	assert.Equal(t, "NOT_ATTENDED", participant.Status, "one material is still unsigned")
	assert.Equal(t, 1, participant.SignedMaterialCount)

	signDocumentVersion(t, employee, rulesVersionID)

	plan = loadTrainingPlan(t, owner, plan.ID)
	participant = plan.participant(t, employeeProfileID)
	assert.Equal(t, "COMPLETED", participant.Status)
	require.NotNil(t, participant.AttendedOn)
	assert.Equal(t, trainingToday().Format(time.DateOnly), *participant.AttendedOn)
	require.NotNil(t, participant.AttendanceSource)
	assert.Equal(t, "SIGNATURE", *participant.AttendanceSource)
	assert.Equal(t, 1, plan.Summary.CompletedCount)
	assert.Equal(t, 0, plan.Summary.NotAttendedCount)

	requested, _ = requestTrainingSignatures(t, owner, plan.ID)
	assert.Equal(t, 0, requested, "participants who attended are not asked again")
}

func TestTraining_SignatureOutsidePeriodDoesNotCount(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)
	employee := testutil.NewClientInOrg(t, testutil.RoleEmployee, owner)
	employeeProfileID := employee.GetProfileID().String()

	docID, versionID := publishedTrainingMaterial(t, owner)

	plan := createTrainingPlan(t, owner, map[string]any{
		"fiscalYear": 2020,
		"startsOn":   "2020-04-01",
		"endsOn":     "2021-03-31",
	})
	setTrainingMaterials(t, owner, plan.ID, docID)
	addTrainingParticipants(t, owner, plan.ID, employeeProfileID)
	requestTrainingSignatures(t, owner, plan.ID)
	signDocumentVersion(t, employee, versionID)

	participant := loadTrainingPlan(t, owner, plan.ID).participant(t, employeeProfileID)
	assert.Equal(t, "NOT_ATTENDED", participant.Status)
	assert.Equal(t, 0, participant.SignedMaterialCount)
	assert.Equal(t, 1, participant.SignedOutside, "the signature outside the period is reported")
}

func TestTraining_ManualAttendanceAndTest(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)
	employee := testutil.NewClientInOrg(t, testutil.RoleEmployee, owner)
	employeeProfileID := employee.GetProfileID().String()

	plan := createTrainingPlan(t, owner, map[string]any{"method": "CLASSROOM", "testRequired": true, "passScore": 80})
	_, plan = addTrainingParticipants(t, owner, plan.ID, employeeProfileID)
	participantID := plan.participant(t, employeeProfileID).ID

	tests := []struct {
		name       string
		input      map[string]any
		wantStatus string
		wantPassed *bool
	}{
		{
			name:       "attended without a test result",
			input:      map[string]any{"attendedOn": "2026-10-10"},
			wantStatus: "TEST_PENDING",
		},
		{
			name:       "score below the pass score fails",
			input:      map[string]any{"attendedOn": "2026-10-10", "testScore": 70},
			wantStatus: "FAILED",
			wantPassed: new(false),
		},
		{
			name:       "score at the pass score completes",
			input:      map[string]any{"attendedOn": "2026-10-10", "testScore": 80, "note": "再テストで合格"},
			wantStatus: "COMPLETED",
			wantPassed: new(true),
		},
		{
			name:       "a passed test without attendance is not attended",
			input:      map[string]any{"testScore": 90},
			wantStatus: "NOT_ATTENDED",
			wantPassed: new(true),
		},
	}

	// Sequential on purpose: each case updates the same participant.
	for _, tt := range tests {
		input := map[string]any{"trainingParticipantId": participantID}
		for key, value := range tt.input {
			input[key] = value
		}

		participant := updateTrainingParticipant(t, owner, input).participant(t, employeeProfileID)
		assert.Equal(t, tt.wantStatus, participant.Status, tt.name)
		assert.Equal(t, tt.wantPassed, participant.TestPassed, tt.name)
	}

	participant := loadTrainingPlan(t, owner, plan.ID).participant(t, employeeProfileID)
	assert.Nil(t, participant.ManualAttendedOn, "an omitted date is cleared")
	assert.Nil(t, participant.ManualTestPassed, "a derived result is not stored")

	// A corrected pass score re-evaluates the stored score.
	updateTrainingParticipant(t, owner, map[string]any{
		"trainingParticipantId": participantID,
		"attendedOn":            "2026-10-10",
		"testScore":             85,
	})
	owner.MustExecute(updateTrainingPlanMutation, map[string]any{
		"input": map[string]any{"trainingPlanId": plan.ID, "passScore": 90},
	}, &struct{}{})
	participant = loadTrainingPlan(t, owner, plan.ID).participant(t, employeeProfileID)
	assert.Equal(t, "FAILED", participant.Status)

	// A result entered by hand wins over the score.
	participant = updateTrainingParticipant(t, owner, map[string]any{
		"trainingParticipantId": participantID,
		"attendedOn":            "2026-10-10",
		"testScore":             85,
		"testPassed":            true,
	}).participant(t, employeeProfileID)
	assert.Equal(t, "COMPLETED", participant.Status)
	require.NotNil(t, participant.ManualTestPassed)
	assert.True(t, *participant.ManualTestPassed)

	_, err := owner.Do(updateTrainingParticipantMutation, map[string]any{
		"input": map[string]any{
			"trainingParticipantId": participantID,
			"attendedOn":            trainingToday().AddDate(0, 0, 2).Format(time.DateOnly),
		},
	})
	require.Error(t, err, "an attendance date in the future is rejected")

	_, err = owner.Do(updateTrainingParticipantMutation, map[string]any{
		"input": map[string]any{"trainingParticipantId": participantID, "testScore": 101},
	})
	require.Error(t, err)

	_, err = owner.Do(removeTrainingParticipantMutation, map[string]any{
		"input": map[string]any{"trainingParticipantId": participantID},
	})
	require.Error(t, err, "a participant with a record cannot be removed")
}

func TestTraining_RecordsAreKept(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)
	attended := testutil.NewClientInOrg(t, testutil.RoleEmployee, owner)
	addedByMistake := testutil.NewClientInOrg(t, testutil.RoleEmployee, owner)
	attendedProfileID := attended.GetProfileID().String()
	mistakeProfileID := addedByMistake.GetProfileID().String()

	plan := createTrainingPlan(t, owner, map[string]any{"method": "CLASSROOM"})
	_, plan = addTrainingParticipants(t, owner, plan.ID, attendedProfileID, mistakeProfileID)
	plan = updateTrainingParticipant(t, owner, map[string]any{
		"trainingParticipantId": plan.participant(t, attendedProfileID).ID,
		"attendedOn":            "2026-10-10",
	})
	assert.Equal(t, 1, plan.Summary.RecordedCount)
	assert.True(t, plan.participant(t, attendedProfileID).HasRecord)
	assert.False(t, plan.participant(t, mistakeProfileID).HasRecord)

	_, err := owner.Do(removeTrainingParticipantMutation, map[string]any{
		"input": map[string]any{"trainingParticipantId": plan.participant(t, attendedProfileID).ID},
	})
	require.Error(t, err, "a participant who attended cannot be removed")
	assert.Contains(t, err.Error(), "training records must be kept")

	_, err = owner.Do(deleteTrainingPlanMutation, map[string]any{
		"input": map[string]any{"trainingPlanId": plan.ID},
	})
	require.Error(t, err, "a plan with records cannot be deleted")

	var removed struct {
		RemoveTrainingParticipant struct {
			TrainingPlan trainingPlan `json:"trainingPlan"`
		} `json:"removeTrainingParticipant"`
	}

	owner.MustExecute(removeTrainingParticipantMutation, map[string]any{
		"input": map[string]any{"trainingParticipantId": plan.participant(t, mistakeProfileID).ID},
	}, &removed)
	require.Len(t, removed.RemoveTrainingParticipant.TrainingPlan.Participants, 1, "a participant without a record can be removed")

	// The plan with a record is still there.
	loadTrainingPlan(t, owner, plan.ID)

	unrecorded := createTrainingPlan(t, owner, nil)
	addTrainingParticipants(t, owner, unrecorded.ID, mistakeProfileID)
	owner.MustExecute(deleteTrainingPlanMutation, map[string]any{
		"input": map[string]any{"trainingPlanId": unrecorded.ID},
	}, &struct{}{})
}

func TestTraining_RequestSignaturesNeedsMaterials(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)
	plan := createTrainingPlan(t, owner, nil)

	_, err := owner.Do(requestTrainingSignaturesMutation, map[string]any{
		"input": map[string]any{"trainingPlanId": plan.ID},
	})
	require.Error(t, err)
}

func TestTraining_Export(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)
	employee := testutil.NewClientInOrg(t, testutil.RoleEmployee, owner)

	plan := createTrainingPlan(t, owner, map[string]any{"fiscalYear": 2033, "name": "2033年度 定期教育", "method": "CLASSROOM"})
	_, plan = addTrainingParticipants(t, owner, plan.ID, employee.GetProfileID().String())
	updateTrainingParticipant(t, owner, map[string]any{
		"trainingParticipantId": plan.Participants[0].ID,
		"attendedOn":            "2026-05-10",
	})
	createTrainingPlan(t, owner, map[string]any{"fiscalYear": 2033, "name": "2033年度 新規入社者教育", "kind": "ONBOARDING"})
	createTrainingPlan(t, owner, map[string]any{"fiscalYear": 2034, "name": "2034年度 定期教育"})

	rows := exportTrainingRows(t, owner, 2033)
	require.GreaterOrEqual(t, len(rows), 6)
	assert.Equal(t, "No.", rows[3][0])

	data := rows[4:]
	require.Len(t, data, 2, "one row per participant, and one row for the plan without participants")

	byName := map[string][]string{}
	for _, row := range data {
		byName[row[2]] = row
	}

	attended := byName["2033年度 定期教育"]
	require.NotNil(t, attended)
	assert.Equal(t, "集合研修", attended[5])
	assert.Equal(t, plan.Participants[0].FullName, attended[7])
	assert.Equal(t, "2026-05-10", attended[9])
	assert.Equal(t, "手入力", attended[10])
	assert.Equal(t, "完了", attended[13])

	empty := byName["2033年度 新規入社者教育"]
	require.NotNil(t, empty)
	assert.Equal(t, "（受講者未登録）", empty[7])
}

func TestTraining_RBAC(t *testing.T) {
	t.Parallel()

	org := testutil.NewOrganizationRoles(t)
	ownerClient := org.Client(t, testutil.RoleOwner)
	plan := createTrainingPlan(t, ownerClient, nil)

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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := org.Client(t, tt.role)

			_, err := client.Do(updateTrainingPlanMutation, map[string]any{
				"input": map[string]any{"trainingPlanId": plan.ID, "instructor": string(tt.role)},
			})
			if tt.updateForbidden {
				testutil.RequireForbiddenError(t, err, "%s should not update training plans", tt.role)
			} else {
				require.NoError(t, err, "%s should update training plans", tt.role)
			}

			_, err = client.Do(trainingPlanQuery, map[string]any{"id": plan.ID})
			require.NoError(t, err, "%s should read training plans", tt.role)

			_, err = client.Do(exportTrainingRecordsMutation, map[string]any{
				"input": map[string]any{"organizationId": client.GetOrganizationID().String()},
			})
			require.NoError(t, err, "%s should export training records", tt.role)
		})
	}

	t.Run("employee", func(t *testing.T) {
		t.Parallel()

		employee := testutil.NewClientInOrg(t, testutil.RoleEmployee, ownerClient)

		_, err := employee.Do(trainingPlanQuery, map[string]any{"id": plan.ID})
		require.Error(t, err, "employees cannot read training plans")

		_, err = employee.Do(createTrainingPlanMutation, map[string]any{"input": trainingPlanInput(employee, nil)})
		require.Error(t, err, "employees cannot create training plans")

		_, err = employee.Do(exportTrainingRecordsMutation, map[string]any{
			"input": map[string]any{"organizationId": employee.GetOrganizationID().String()},
		})
		require.Error(t, err, "employees cannot export training records")
	})
}

func TestTraining_TenantIsolation(t *testing.T) {
	t.Parallel()

	org1Owner := testutil.NewClient(t, testutil.RoleOwner)
	org2Owner := testutil.NewClient(t, testutil.RoleOwner)

	org1Plan := createTrainingPlan(t, org1Owner, nil)
	org1DocumentID, _ := publishedTrainingMaterial(t, org1Owner)
	org2Plan := createTrainingPlan(t, org2Owner, nil)

	t.Run("cannot read or change a plan of another organization", func(t *testing.T) {
		t.Parallel()

		_, err := org2Owner.Do(trainingPlanQuery, map[string]any{"id": org1Plan.ID})
		require.Error(t, err)

		_, err = org2Owner.Do(updateTrainingPlanMutation, map[string]any{
			"input": map[string]any{"trainingPlanId": org1Plan.ID, "name": "hijacked"},
		})
		require.Error(t, err)

		_, err = org2Owner.Do(deleteTrainingPlanMutation, map[string]any{
			"input": map[string]any{"trainingPlanId": org1Plan.ID},
		})
		require.Error(t, err)
	})

	t.Run("cannot use documents or people of another organization", func(t *testing.T) {
		t.Parallel()

		_, err := org2Owner.Do(setTrainingPlanMaterialsMutation, map[string]any{
			"input": map[string]any{"trainingPlanId": org2Plan.ID, "documentIds": []string{org1DocumentID}},
		})
		require.Error(t, err)

		_, err = org2Owner.Do(addTrainingParticipantsMutation, map[string]any{
			"input": map[string]any{"trainingPlanId": org2Plan.ID, "profileIds": []string{org1Owner.GetProfileID().String()}},
		})
		require.Error(t, err)

		plan := loadTrainingPlan(t, org2Owner, org2Plan.ID)
		assert.Empty(t, plan.Materials)
		assert.Empty(t, plan.Participants)
	})

	t.Run("cannot export another organization", func(t *testing.T) {
		t.Parallel()

		_, err := org2Owner.Do(exportTrainingRecordsMutation, map[string]any{
			"input": map[string]any{"organizationId": org1Owner.GetOrganizationID().String()},
		})
		require.Error(t, err)
	})
}

func TestTraining_RequestSignaturesNeedsPublishedMaterials(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)
	employee := testutil.NewClientInOrg(t, testutil.RoleEmployee, owner)

	draftID, _ := createTestDocument(t, owner)
	plan := createTrainingPlan(t, owner, nil)
	setTrainingMaterials(t, owner, plan.ID, draftID)
	addTrainingParticipants(t, owner, plan.ID, employee.GetProfileID().String())

	_, err := owner.Do(requestTrainingSignaturesMutation, map[string]any{
		"input": map[string]any{"trainingPlanId": plan.ID},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish it first")
}

func TestTraining_Pagination(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)
	for _, startsOn := range []string{"2035-04-01", "2035-06-01", "2035-09-01"} {
		createTrainingPlan(t, owner, map[string]any{"fiscalYear": 2035, "startsOn": startsOn, "endsOn": startsOn})
	}

	const query = `
		query($id: ID!, $after: CursorKey, $field: TrainingPlanOrderField!) {
			node(id: $id) {
				... on Organization {
					trainingPlans(first: 1, after: $after, orderBy: { field: $field, direction: ASC }) {
						edges { cursor node { startsOn name } }
						pageInfo { hasNextPage }
					}
				}
			}
		}
	`

	for _, field := range []string{"STARTS_ON", "NAME", "CREATED_AT"} {
		var seen []string
		var after any

		for range 4 {
			var result struct {
				Node struct {
					TrainingPlans struct {
						Edges []struct {
							Cursor string `json:"cursor"`
							Node   struct {
								StartsOn string `json:"startsOn"`
							} `json:"node"`
						} `json:"edges"`
						PageInfo struct {
							HasNextPage bool `json:"hasNextPage"`
						} `json:"pageInfo"`
					} `json:"trainingPlans"`
				} `json:"node"`
			}

			owner.MustExecute(query, map[string]any{"id": owner.GetOrganizationID().String(), "after": after, "field": field}, &result)
			edges := result.Node.TrainingPlans.Edges
			require.Len(t, edges, 1, field)
			seen = append(seen, edges[0].Node.StartsOn)
			if !result.Node.TrainingPlans.PageInfo.HasNextPage {
				break
			}
			after = edges[0].Cursor
		}

		assert.ElementsMatch(t, []string{"2035-04-01", "2035-06-01", "2035-09-01"}, seen, field)
		if field == "STARTS_ON" {
			assert.Equal(t, []string{"2035-04-01", "2035-06-01", "2035-09-01"}, seen)
		}
	}
}
