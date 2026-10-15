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
	"fmt"
	"time"
	// Plan periods are calendar dates in an IANA time zone; embed the zone
	// database so this works in images without /usr/share/zoneinfo.
	_ "time/tzdata"

	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
)

// TrainingProgress is how far one participant is in a training plan. It is
// derived from the participant's signatures and manual entries, never stored.
type TrainingProgress struct {
	AttendedOn      *time.Time
	Source          *coredata.TrainingAttendanceSource
	SignedMaterials int
	// SignedOutsidePeriod counts the materials the participant signed only
	// outside the plan period. A signature is kept for a document version, so
	// they cannot sign those materials again until a new version is published.
	SignedOutsidePeriod int
	// TestPassed is the result that counts: the one entered by hand, or the
	// one derived from the score and the plan's pass score.
	TestPassed *bool
	Status     coredata.TrainingParticipantStatus
}

// TrainingPeriod returns the plan period as [from, to): from the start of
// StartsOn to the end of EndsOn in the plan's time zone.
func TrainingPeriod(plan *coredata.TrainingPlan) (time.Time, time.Time, error) {
	location, err := time.LoadLocation(plan.TimeZone)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("cannot load time zone %q: %w", plan.TimeZone, err)
	}

	from := time.Date(plan.StartsOn.Year(), plan.StartsOn.Month(), plan.StartsOn.Day(), 0, 0, 0, 0, location)
	to := time.Date(plan.EndsOn.Year(), plan.EndsOn.Month(), plan.EndsOn.Day(), 0, 0, 0, 0, location).AddDate(0, 0, 1)

	return from, to, nil
}

// ResolveTrainingTestPassed returns the test result that counts: an explicit
// result wins, otherwise the score is compared with the plan's pass score.
func ResolveTrainingTestPassed(plan *coredata.TrainingPlan, score *int, passed *bool) *bool {
	if passed != nil {
		return passed
	}
	if score == nil || plan.PassScore == nil {
		return nil
	}

	result := *score >= *plan.PassScore

	return &result
}

// EvaluateTrainingProgress decides whether a participant attended the plan
// and completed it. A participant attended when a date was entered by hand,
// or when they signed every material within the plan period; the attendance
// date is then the day of the last signature in the plan's time zone.
func EvaluateTrainingProgress(
	plan *coredata.TrainingPlan,
	participant *coredata.TrainingParticipant,
	materialIDs []gid.GID,
	signatures []*coredata.TrainingMaterialSignature,
) TrainingProgress {
	progress := TrainingProgress{
		Status:     coredata.TrainingParticipantStatusNotAttended,
		TestPassed: ResolveTrainingTestPassed(plan, participant.TestScore, participant.TestPassed),
	}

	from, to, err := TrainingPeriod(plan)
	if err != nil {
		return progress
	}

	inPeriod, outsideOnly := signedMaterials(participant.ProfileID, materialIDs, signatures, from, to)
	progress.SignedMaterials = len(inPeriod)
	progress.SignedOutsidePeriod = outsideOnly

	switch {
	case participant.AttendedOn != nil:
		source := coredata.TrainingAttendanceSourceManual
		progress.AttendedOn = participant.AttendedOn
		progress.Source = &source
	case len(materialIDs) > 0 && len(inPeriod) == len(materialIDs):
		var last time.Time
		for _, signedAt := range inPeriod {
			if signedAt.After(last) {
				last = signedAt
			}
		}

		local := last.In(from.Location())
		attendedOn := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
		source := coredata.TrainingAttendanceSourceSignature
		progress.AttendedOn = &attendedOn
		progress.Source = &source
	default:
		return progress
	}

	switch {
	case !plan.TestRequired:
		progress.Status = coredata.TrainingParticipantStatusCompleted
	case progress.TestPassed == nil:
		progress.Status = coredata.TrainingParticipantStatusTestPending
	case *progress.TestPassed:
		progress.Status = coredata.TrainingParticipantStatusCompleted
	default:
		progress.Status = coredata.TrainingParticipantStatusFailed
	}

	return progress
}

// signedMaterials returns, for each material the profile signed within
// [from, to), the time of the latest such signature, and how many materials
// the profile signed only outside that period.
func signedMaterials(
	profileID gid.GID,
	materialIDs []gid.GID,
	signatures []*coredata.TrainingMaterialSignature,
	from time.Time,
	to time.Time,
) (map[gid.GID]time.Time, int) {
	inPeriod := make(map[gid.GID]time.Time)
	outside := make(map[gid.GID]bool)

	isMaterial := make(map[gid.GID]bool, len(materialIDs))
	for _, id := range materialIDs {
		isMaterial[id] = true
	}

	for _, signature := range signatures {
		if signature.ProfileID != profileID || !isMaterial[signature.DocumentID] {
			continue
		}
		if signature.SignedAt.Before(from) || !signature.SignedAt.Before(to) {
			outside[signature.DocumentID] = true
			continue
		}
		if signature.SignedAt.After(inPeriod[signature.DocumentID]) {
			inPeriod[signature.DocumentID] = signature.SignedAt
		}
	}

	outsideOnly := 0
	for documentID := range outside {
		if _, ok := inPeriod[documentID]; !ok {
			outsideOnly++
		}
	}

	return inPeriod, outsideOnly
}
