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
)

func trainingDate(t *testing.T, value string) time.Time {
	t.Helper()

	date, err := time.Parse(time.DateOnly, value)
	require.NoError(t, err)

	return date
}

func TestTrainingPeriod(t *testing.T) {
	t.Parallel()

	plan := &coredata.TrainingPlan{
		StartsOn: trainingDate(t, "2026-10-01"),
		EndsOn:   trainingDate(t, "2026-10-31"),
		TimeZone: "Asia/Tokyo",
	}

	from, to, err := TrainingPeriod(plan)
	require.NoError(t, err)
	assert.Equal(t, "2026-09-30T15:00:00Z", from.UTC().Format(time.RFC3339))
	assert.Equal(t, "2026-10-31T15:00:00Z", to.UTC().Format(time.RFC3339))

	_, _, err = TrainingPeriod(&coredata.TrainingPlan{TimeZone: "Not/AZone"})
	assert.Error(t, err)
}

func TestResolveTrainingTestPassed(t *testing.T) {
	t.Parallel()

	score := func(v int) *int { return &v }
	passed := func(v bool) *bool { return &v }
	withPassScore := &coredata.TrainingPlan{TestRequired: true, PassScore: score(80)}
	withoutPassScore := &coredata.TrainingPlan{TestRequired: true}

	tests := []struct {
		name   string
		plan   *coredata.TrainingPlan
		score  *int
		passed *bool
		want   *bool
	}{
		{name: "nothing entered", plan: withPassScore, want: nil},
		{name: "score at pass score passes", plan: withPassScore, score: score(80), want: passed(true)},
		{name: "score below pass score fails", plan: withPassScore, score: score(79), want: passed(false)},
		{name: "explicit result wins", plan: withPassScore, score: score(50), passed: passed(true), want: passed(true)},
		{name: "score without pass score is undecided", plan: withoutPassScore, score: score(90), want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ResolveTrainingTestPassed(tt.plan, tt.score, tt.passed))
		})
	}
}

func TestEvaluateTrainingProgress(t *testing.T) {
	t.Parallel()

	tenantID := gid.NewTenantID()
	profileID := gid.New(tenantID, coredata.MembershipProfileEntityType)
	otherProfileID := gid.New(tenantID, coredata.MembershipProfileEntityType)
	policyID := gid.New(tenantID, coredata.DocumentEntityType)
	rulesID := gid.New(tenantID, coredata.DocumentEntityType)
	materials := []gid.GID{policyID, rulesID}

	plan := func(testRequired bool) *coredata.TrainingPlan {
		return &coredata.TrainingPlan{
			StartsOn:     trainingDate(t, "2026-10-01"),
			EndsOn:       trainingDate(t, "2026-10-31"),
			TimeZone:     "Asia/Tokyo",
			TestRequired: testRequired,
		}
	}
	at := func(value string) time.Time {
		v, err := time.Parse(time.RFC3339, value)
		require.NoError(t, err)
		return v
	}
	signature := func(profile, document gid.GID, signedAt string) *coredata.TrainingMaterialSignature {
		return &coredata.TrainingMaterialSignature{ProfileID: profile, DocumentID: document, SignedAt: at(signedAt)}
	}
	boolPtr := func(v bool) *bool { return &v }
	datePtr := func(v string) *time.Time { d := trainingDate(t, v); return &d }

	t.Run("all materials signed in period completes on the last signature", func(t *testing.T) {
		t.Parallel()

		progress := EvaluateTrainingProgress(
			plan(false),
			&coredata.TrainingParticipant{ProfileID: profileID},
			materials,
			[]*coredata.TrainingMaterialSignature{
				signature(profileID, policyID, "2026-10-02T01:00:00Z"),
				// 2026-10-04 08:30 in Tokyo.
				signature(profileID, rulesID, "2026-10-03T23:30:00Z"),
				signature(otherProfileID, rulesID, "2026-10-05T00:00:00Z"),
			},
		)

		assert.Equal(t, coredata.TrainingParticipantStatusCompleted, progress.Status)
		require.NotNil(t, progress.AttendedOn)
		assert.Equal(t, "2026-10-04", progress.AttendedOn.Format(time.DateOnly))
		require.NotNil(t, progress.Source)
		assert.Equal(t, coredata.TrainingAttendanceSourceSignature, *progress.Source)
		assert.Equal(t, 2, progress.SignedMaterials)
	})

	t.Run("a missing material is not attended", func(t *testing.T) {
		t.Parallel()

		progress := EvaluateTrainingProgress(
			plan(false),
			&coredata.TrainingParticipant{ProfileID: profileID},
			materials,
			[]*coredata.TrainingMaterialSignature{signature(profileID, policyID, "2026-10-02T01:00:00Z")},
		)

		assert.Equal(t, coredata.TrainingParticipantStatusNotAttended, progress.Status)
		assert.Nil(t, progress.AttendedOn)
		assert.Equal(t, 1, progress.SignedMaterials)
	})

	t.Run("signatures outside the period do not count", func(t *testing.T) {
		t.Parallel()

		progress := EvaluateTrainingProgress(
			plan(false),
			&coredata.TrainingParticipant{ProfileID: profileID},
			materials,
			[]*coredata.TrainingMaterialSignature{
				// 2026-09-30 23:59 in Tokyo, one minute before the period.
				signature(profileID, policyID, "2026-09-30T14:59:00Z"),
				// 2026-11-01 00:00 in Tokyo, just after the period.
				signature(profileID, rulesID, "2026-10-31T15:00:00Z"),
			},
		)

		assert.Equal(t, coredata.TrainingParticipantStatusNotAttended, progress.Status)
		assert.Equal(t, 0, progress.SignedMaterials)
		assert.Equal(t, 2, progress.SignedOutsidePeriod)
	})

	t.Run("a material signed in and outside the period counts as signed", func(t *testing.T) {
		t.Parallel()

		progress := EvaluateTrainingProgress(
			plan(false),
			&coredata.TrainingParticipant{ProfileID: profileID},
			materials,
			[]*coredata.TrainingMaterialSignature{
				signature(profileID, policyID, "2025-10-02T01:00:00Z"),
				signature(profileID, policyID, "2026-10-02T01:00:00Z"),
			},
		)

		assert.Equal(t, 1, progress.SignedMaterials)
		assert.Equal(t, 0, progress.SignedOutsidePeriod)
	})

	t.Run("a plan without materials needs a manual date", func(t *testing.T) {
		t.Parallel()

		progress := EvaluateTrainingProgress(plan(false), &coredata.TrainingParticipant{ProfileID: profileID}, nil, nil)
		assert.Equal(t, coredata.TrainingParticipantStatusNotAttended, progress.Status)
	})

	t.Run("a manual date wins over signatures", func(t *testing.T) {
		t.Parallel()

		progress := EvaluateTrainingProgress(
			plan(false),
			&coredata.TrainingParticipant{ProfileID: profileID, AttendedOn: datePtr("2026-10-10")},
			materials,
			[]*coredata.TrainingMaterialSignature{
				signature(profileID, policyID, "2026-10-02T01:00:00Z"),
				signature(profileID, rulesID, "2026-10-03T01:00:00Z"),
			},
		)

		assert.Equal(t, coredata.TrainingParticipantStatusCompleted, progress.Status)
		assert.Equal(t, "2026-10-10", progress.AttendedOn.Format(time.DateOnly))
		assert.Equal(t, coredata.TrainingAttendanceSourceManual, *progress.Source)
	})

	t.Run("test results decide the status once attended", func(t *testing.T) {
		t.Parallel()

		attended := datePtr("2026-10-10")
		cases := []struct {
			name   string
			passed *bool
			want   coredata.TrainingParticipantStatus
		}{
			{name: "no result", passed: nil, want: coredata.TrainingParticipantStatusTestPending},
			{name: "failed", passed: boolPtr(false), want: coredata.TrainingParticipantStatusFailed},
			{name: "passed", passed: boolPtr(true), want: coredata.TrainingParticipantStatusCompleted},
		}
		for _, c := range cases {
			progress := EvaluateTrainingProgress(
				plan(true),
				&coredata.TrainingParticipant{ProfileID: profileID, AttendedOn: attended, TestPassed: c.passed},
				materials,
				nil,
			)
			assert.Equal(t, c.want, progress.Status, c.name)
		}
	})

	t.Run("the result follows the score and the current pass score", func(t *testing.T) {
		t.Parallel()

		passScore := 80
		score := 75
		testPlan := plan(true)
		testPlan.PassScore = &passScore
		participant := &coredata.TrainingParticipant{ProfileID: profileID, AttendedOn: datePtr("2026-10-10"), TestScore: &score}

		progress := EvaluateTrainingProgress(testPlan, participant, nil, nil)
		assert.Equal(t, coredata.TrainingParticipantStatusFailed, progress.Status)

		lowered := 70
		testPlan.PassScore = &lowered
		progress = EvaluateTrainingProgress(testPlan, participant, nil, nil)
		assert.Equal(t, coredata.TrainingParticipantStatusCompleted, progress.Status)
		require.NotNil(t, progress.TestPassed)
		assert.True(t, *progress.TestPassed)
	})

	t.Run("a passed test without attendance is not attended", func(t *testing.T) {
		t.Parallel()

		progress := EvaluateTrainingProgress(
			plan(true),
			&coredata.TrainingParticipant{ProfileID: profileID, TestPassed: boolPtr(true)},
			materials,
			nil,
		)
		assert.Equal(t, coredata.TrainingParticipantStatusNotAttended, progress.Status)
	})
}
