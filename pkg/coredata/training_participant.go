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

package coredata

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/jackc/pgx/v5"
	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/iam/policy"
)

type (
	// TrainingParticipant is one person who must take a training plan, with
	// what was entered by hand: the attendance date for training without
	// signatures, the test score, and a pass or fail result that overrides
	// the one derived from the score.
	TrainingParticipant struct {
		ID             gid.GID    `db:"id"`
		OrganizationID gid.GID    `db:"organization_id"`
		TrainingPlanID gid.GID    `db:"training_plan_id"`
		ProfileID      gid.GID    `db:"profile_id"`
		AttendedOn     *time.Time `db:"attended_on"`
		TestScore      *int       `db:"test_score"`
		TestPassed     *bool      `db:"test_passed"`
		Note           string     `db:"note"`
		CreatedAt      time.Time  `db:"created_at"`
		UpdatedAt      time.Time  `db:"updated_at"`
	}

	// TrainingParticipantDetail adds the profile fields shown in lists and
	// in the training record export.
	TrainingParticipantDetail struct {
		TrainingParticipant
		FullName     string  `db:"full_name"`
		EmailAddress string  `db:"email_address"`
		Department   *string `db:"department"`
	}

	TrainingParticipantDetails []*TrainingParticipantDetail

	// TrainingMaterialSignature is a signature of a participant on any
	// version of a material document.
	TrainingMaterialSignature struct {
		ProfileID  gid.GID   `db:"profile_id"`
		DocumentID gid.GID   `db:"document_id"`
		SignedAt   time.Time `db:"signed_at"`
	}

	TrainingMaterialSignatures []*TrainingMaterialSignature
)

const trainingParticipantColumns = `
	id,
	organization_id,
	training_plan_id,
	profile_id,
	attended_on,
	test_score,
	test_passed,
	note,
	created_at,
	updated_at`

// AuthorizationAttributes returns the authorization attributes for policy evaluation.
func (p *TrainingParticipant) AuthorizationAttributes(
	ctx context.Context,
	conn pg.Querier,
	resourceIDs []gid.GID,
) (policy.AttributesByID, error) {
	return loadOrganizationAttributes(ctx, conn, "training_participants", resourceIDs)
}

func (p *TrainingParticipant) LoadByID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	trainingParticipantID gid.GID,
) error {
	q := `SELECT` + trainingParticipantColumns + `
FROM
	training_participants
WHERE
	%s
	AND id = @training_participant_id
LIMIT 1;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"training_participant_id": trainingParticipantID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query training participants: %w", err)
	}

	participant, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[TrainingParticipant])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect training participant: %w", err)
	}

	*p = participant

	return nil
}

// LoadByTrainingPlanIDs loads the participants of the given plans with
// their profile, ordered by plan and name.
func (p *TrainingParticipantDetails) LoadByTrainingPlanIDs(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	trainingPlanIDs []gid.GID,
) error {
	q := `
WITH participants AS (
	SELECT` + trainingParticipantColumns + `
	FROM
		training_participants
	WHERE
		%s
		AND training_plan_id = ANY(@training_plan_ids::TEXT[])
)
SELECT
	tp.*,
	mp.full_name,
	i.email_address,
	mp.department
FROM
	participants tp
	INNER JOIN iam_membership_profiles mp ON mp.id = tp.profile_id
	INNER JOIN identities i ON i.id = mp.identity_id
ORDER BY
	tp.training_plan_id, mp.full_name, tp.id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"training_plan_ids": gidStrings(trainingPlanIDs)}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query training participants: %w", err)
	}

	participants, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[TrainingParticipantDetail])
	if err != nil {
		return fmt.Errorf("cannot collect training participants: %w", err)
	}

	*p = participants

	return nil
}

// InsertTrainingParticipants adds profiles to a plan. Profiles that are
// already participants are left as they are.
func InsertTrainingParticipants(
	ctx context.Context,
	conn pg.Tx,
	scope Scoper,
	organizationID gid.GID,
	trainingPlanID gid.GID,
	participantIDs []gid.GID,
	profileIDs []gid.GID,
) (int, error) {
	if len(profileIDs) == 0 {
		return 0, nil
	}
	if len(participantIDs) != len(profileIDs) {
		return 0, fmt.Errorf("cannot insert training participants: %d ids for %d profiles", len(participantIDs), len(profileIDs))
	}

	q := `
INSERT INTO training_participants (
	id,
	tenant_id,
	organization_id,
	training_plan_id,
	profile_id,
	created_at,
	updated_at
)
SELECT
	input.id,
	@tenant_id,
	@organization_id,
	@training_plan_id,
	input.profile_id,
	@now,
	@now
FROM
	UNNEST(@ids::TEXT[], @profile_ids::TEXT[]) AS input(id, profile_id)
ON CONFLICT (training_plan_id, profile_id) DO NOTHING
`

	now := time.Now()
	args := pgx.StrictNamedArgs{
		"ids":              gidStrings(participantIDs),
		"profile_ids":      gidStrings(profileIDs),
		"tenant_id":        scope.GetTenantID(),
		"organization_id":  organizationID,
		"training_plan_id": trainingPlanID,
		"now":              now,
	}

	result, err := conn.Exec(ctx, q, args)
	if err != nil {
		return 0, fmt.Errorf("cannot insert training participants: %w", err)
	}

	return int(result.RowsAffected()), nil
}

func (p *TrainingParticipant) Update(
	ctx context.Context,
	conn pg.Tx,
	scope Scoper,
) error {
	q := `
UPDATE training_participants
SET
	attended_on = @attended_on,
	test_score = @test_score,
	test_passed = @test_passed,
	note = @note,
	updated_at = @updated_at
WHERE
	%s
	AND id = @id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{
		"id":          p.ID,
		"attended_on": p.AttendedOn,
		"test_score":  p.TestScore,
		"test_passed": p.TestPassed,
		"note":        p.Note,
		"updated_at":  p.UpdatedAt,
	}
	maps.Copy(args, scope.SQLArguments())

	result, err := conn.Exec(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot update training participant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrResourceNotFound
	}

	return nil
}

func (p *TrainingParticipant) Delete(
	ctx context.Context,
	conn pg.Tx,
	scope Scoper,
) error {
	q := `
DELETE FROM training_participants
WHERE
	%s
	AND id = @id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"id": p.ID}
	maps.Copy(args, scope.SQLArguments())

	result, err := conn.Exec(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot delete training participant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrResourceNotFound
	}

	return nil
}

// LoadByTrainingPlanID loads every signature of the plan's participants on
// the plan's materials, on any version. Whether a signature falls in the
// plan period is decided by the caller, so that signatures made before the
// period can be reported too.
func (s *TrainingMaterialSignatures) LoadByTrainingPlanID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	trainingPlanID gid.GID,
) error {
	q := `
WITH participants AS (
	SELECT profile_id
	FROM training_participants
	WHERE %[1]s AND training_plan_id = @training_plan_id
),
materials AS (
	SELECT document_id
	FROM training_plan_materials
	WHERE %[1]s
		AND training_plan_id = @training_plan_id
		AND document_id IN (SELECT id FROM documents WHERE deleted_at IS NULL)
),
signatures AS (
	SELECT document_version_id, signed_by_profile_id, signed_at
	FROM document_version_signatures
	WHERE %[1]s
		AND state = 'SIGNED'
		AND signed_at IS NOT NULL
		AND signed_by_profile_id IN (SELECT profile_id FROM participants)
)
SELECT
	s.signed_by_profile_id AS profile_id,
	dv.document_id,
	s.signed_at
FROM
	signatures s
	INNER JOIN document_versions dv ON dv.id = s.document_version_id
WHERE
	dv.document_id IN (SELECT document_id FROM materials)
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"training_plan_id": trainingPlanID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query training material signatures: %w", err)
	}

	signatures, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[TrainingMaterialSignature])
	if err != nil {
		return fmt.Errorf("cannot collect training material signatures: %w", err)
	}

	*s = signatures

	return nil
}

// CountTrainingParticipants returns how many participants a plan has.
func CountTrainingParticipants(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	trainingPlanID gid.GID,
) (int, error) {
	q := fmt.Sprintf(`
SELECT COUNT(id)
FROM training_participants
WHERE %s AND training_plan_id = @training_plan_id
`, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"training_plan_id": trainingPlanID}
	maps.Copy(args, scope.SQLArguments())

	var count int
	if err := conn.QueryRow(ctx, q, args).Scan(&count); err != nil {
		return 0, fmt.Errorf("cannot count training participants: %w", err)
	}

	return count, nil
}

func gidStrings(ids []gid.GID) []string {
	values := make([]string, len(ids))
	for i, id := range ids {
		values[i] = id.String()
	}

	return values
}
