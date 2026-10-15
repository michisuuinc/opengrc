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
	"go.probo.inc/probo/pkg/page"
)

type (
	// TrainingPlan is one training (教育) of the Japanese PrivacyMark yearly
	// training plan. StartsOn and EndsOn are calendar dates in TimeZone.
	TrainingPlan struct {
		ID             gid.GID            `db:"id"`
		OrganizationID gid.GID            `db:"organization_id"`
		FiscalYear     int                `db:"fiscal_year"`
		Name           string             `db:"name"`
		Kind           TrainingPlanKind   `db:"kind"`
		Description    string             `db:"description"`
		Method         TrainingPlanMethod `db:"method"`
		Instructor     string             `db:"instructor"`
		StartsOn       time.Time          `db:"starts_on"`
		EndsOn         time.Time          `db:"ends_on"`
		TimeZone       string             `db:"time_zone"`
		TestRequired   bool               `db:"test_required"`
		PassScore      *int               `db:"pass_score"`
		CreatedAt      time.Time          `db:"created_at"`
		UpdatedAt      time.Time          `db:"updated_at"`
	}

	TrainingPlans []*TrainingPlan
)

const trainingPlanColumns = `
	id,
	organization_id,
	fiscal_year,
	name,
	kind,
	description,
	method,
	instructor,
	starts_on,
	ends_on,
	time_zone,
	test_required,
	pass_score,
	created_at,
	updated_at`

func (p *TrainingPlan) CursorKey(field TrainingPlanOrderField) page.CursorKey {
	switch field {
	case TrainingPlanOrderFieldCreatedAt:
		return page.NewCursorKey(p.ID, p.CreatedAt)
	case TrainingPlanOrderFieldStartsOn:
		return page.NewCursorKey(p.ID, p.StartsOn)
	case TrainingPlanOrderFieldName:
		return page.NewCursorKey(p.ID, p.Name)
	}

	panic(fmt.Sprintf("unsupported order by: %s", field))
}

// AuthorizationAttributes returns the authorization attributes for policy evaluation.
func (p *TrainingPlan) AuthorizationAttributes(
	ctx context.Context,
	conn pg.Querier,
	resourceIDs []gid.GID,
) (policy.AttributesByID, error) {
	return loadOrganizationAttributes(ctx, conn, "training_plans", resourceIDs)
}

func (p *TrainingPlan) LoadByID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	trainingPlanID gid.GID,
) error {
	q := `SELECT` + trainingPlanColumns + `
FROM
	training_plans
WHERE
	%s
	AND id = @training_plan_id
LIMIT 1;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"training_plan_id": trainingPlanID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query training plans: %w", err)
	}

	plan, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[TrainingPlan])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect training plan: %w", err)
	}

	*p = plan

	return nil
}

// trainingPlanYearFragment narrows a query to one fiscal year when given.
func trainingPlanYearFragment(fiscalYear *int) (string, pgx.StrictNamedArgs) {
	if fiscalYear == nil {
		return "TRUE", pgx.StrictNamedArgs{}
	}

	return "fiscal_year = @fiscal_year", pgx.StrictNamedArgs{"fiscal_year": *fiscalYear}
}

func (p *TrainingPlans) CountByOrganizationID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	organizationID gid.GID,
	fiscalYear *int,
) (int, error) {
	q := `
SELECT
	COUNT(id)
FROM
	training_plans
WHERE
	%s
	AND organization_id = @organization_id
	AND %s
`

	yearFragment, yearArgs := trainingPlanYearFragment(fiscalYear)
	q = fmt.Sprintf(q, scope.SQLFragment(), yearFragment)

	args := pgx.StrictNamedArgs{"organization_id": organizationID}
	maps.Copy(args, scope.SQLArguments())
	maps.Copy(args, yearArgs)

	var count int
	if err := conn.QueryRow(ctx, q, args).Scan(&count); err != nil {
		return 0, fmt.Errorf("cannot count training plans: %w", err)
	}

	return count, nil
}

func (p *TrainingPlans) LoadByOrganizationID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	organizationID gid.GID,
	fiscalYear *int,
	cursor *page.Cursor[TrainingPlanOrderField],
) error {
	q := `SELECT` + trainingPlanColumns + `
FROM
	training_plans
WHERE
	%s
	AND organization_id = @organization_id
	AND %s
	AND %s
`

	yearFragment, yearArgs := trainingPlanYearFragment(fiscalYear)
	q = fmt.Sprintf(q, scope.SQLFragment(), yearFragment, cursor.SQLFragment())

	args := pgx.StrictNamedArgs{"organization_id": organizationID}
	maps.Copy(args, scope.SQLArguments())
	maps.Copy(args, yearArgs)
	maps.Copy(args, cursor.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query training plans: %w", err)
	}

	plans, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[TrainingPlan])
	if err != nil {
		return fmt.Errorf("cannot collect training plans: %w", err)
	}

	*p = plans

	return nil
}

func (p *TrainingPlan) Insert(
	ctx context.Context,
	conn pg.Tx,
	scope Scoper,
) error {
	q := `
INSERT INTO training_plans (
	id,
	tenant_id,
	organization_id,
	fiscal_year,
	name,
	kind,
	description,
	method,
	instructor,
	starts_on,
	ends_on,
	time_zone,
	test_required,
	pass_score,
	created_at,
	updated_at
) VALUES (
	@id,
	@tenant_id,
	@organization_id,
	@fiscal_year,
	@name,
	@kind,
	@description,
	@method,
	@instructor,
	@starts_on,
	@ends_on,
	@time_zone,
	@test_required,
	@pass_score,
	@created_at,
	@updated_at
)
`

	args := pgx.StrictNamedArgs{
		"id":              p.ID,
		"tenant_id":       scope.GetTenantID(),
		"organization_id": p.OrganizationID,
		"fiscal_year":     p.FiscalYear,
		"name":            p.Name,
		"kind":            p.Kind,
		"description":     p.Description,
		"method":          p.Method,
		"instructor":      p.Instructor,
		"starts_on":       p.StartsOn,
		"ends_on":         p.EndsOn,
		"time_zone":       p.TimeZone,
		"test_required":   p.TestRequired,
		"pass_score":      p.PassScore,
		"created_at":      p.CreatedAt,
		"updated_at":      p.UpdatedAt,
	}

	if _, err := conn.Exec(ctx, q, args); err != nil {
		return fmt.Errorf("cannot insert training plan: %w", err)
	}

	return nil
}

func (p *TrainingPlan) Update(
	ctx context.Context,
	conn pg.Tx,
	scope Scoper,
) error {
	q := `
UPDATE training_plans
SET
	fiscal_year = @fiscal_year,
	name = @name,
	kind = @kind,
	description = @description,
	method = @method,
	instructor = @instructor,
	starts_on = @starts_on,
	ends_on = @ends_on,
	time_zone = @time_zone,
	test_required = @test_required,
	pass_score = @pass_score,
	updated_at = @updated_at
WHERE
	%s
	AND id = @id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{
		"id":            p.ID,
		"fiscal_year":   p.FiscalYear,
		"name":          p.Name,
		"kind":          p.Kind,
		"description":   p.Description,
		"method":        p.Method,
		"instructor":    p.Instructor,
		"starts_on":     p.StartsOn,
		"ends_on":       p.EndsOn,
		"time_zone":     p.TimeZone,
		"test_required": p.TestRequired,
		"pass_score":    p.PassScore,
		"updated_at":    p.UpdatedAt,
	}
	maps.Copy(args, scope.SQLArguments())

	result, err := conn.Exec(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot update training plan: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrResourceNotFound
	}

	return nil
}

func (p *TrainingPlan) Delete(
	ctx context.Context,
	conn pg.Tx,
	scope Scoper,
) error {
	q := `
DELETE FROM training_plans
WHERE
	%s
	AND id = @id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"id": p.ID}
	maps.Copy(args, scope.SQLArguments())

	result, err := conn.Exec(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot delete training plan: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrResourceNotFound
	}

	return nil
}

// LoadAllByOrganizationID loads every plan of the organization, optionally
// of one fiscal year, by fiscal year, start date and name.
func (p *TrainingPlans) LoadAllByOrganizationID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	organizationID gid.GID,
	fiscalYear *int,
) error {
	q := `SELECT` + trainingPlanColumns + `
FROM
	training_plans
WHERE
	%s
	AND organization_id = @organization_id
	AND %s
ORDER BY
	fiscal_year, starts_on, name, id
`

	yearFragment, yearArgs := trainingPlanYearFragment(fiscalYear)
	q = fmt.Sprintf(q, scope.SQLFragment(), yearFragment)

	args := pgx.StrictNamedArgs{"organization_id": organizationID}
	maps.Copy(args, scope.SQLArguments())
	maps.Copy(args, yearArgs)

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query training plans: %w", err)
	}

	plans, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[TrainingPlan])
	if err != nil {
		return fmt.Errorf("cannot collect training plans: %w", err)
	}

	*p = plans

	return nil
}
