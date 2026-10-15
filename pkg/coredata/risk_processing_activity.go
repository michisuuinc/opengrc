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
	"fmt"
	"maps"
	"time"

	"github.com/jackc/pgx/v5"
	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/gid"
)

type (
	// RiskProcessingActivity links a risk to a register entry (processing
	// activity) it was identified for, for the Japanese PrivacyMark risk
	// analysis (リスク分析表).
	RiskProcessingActivity struct {
		RiskID               gid.GID   `db:"risk_id"`
		ProcessingActivityID gid.GID   `db:"processing_activity_id"`
		OrganizationID       gid.GID   `db:"organization_id"`
		CreatedAt            time.Time `db:"created_at"`
	}

	RiskProcessingActivities []*RiskProcessingActivity
)

func (l *RiskProcessingActivities) LoadByRiskID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	riskID gid.GID,
) error {
	q := `
SELECT
	risk_id,
	processing_activity_id,
	organization_id,
	created_at
FROM
	risk_processing_activities
WHERE
	%s
	AND risk_id = @risk_id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"risk_id": riskID}
	maps.Copy(args, scope.SQLArguments())

	return l.query(ctx, conn, q, args)
}

func (l *RiskProcessingActivities) LoadByOrganizationID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	organizationID gid.GID,
) error {
	q := `
SELECT
	risk_id,
	processing_activity_id,
	organization_id,
	created_at
FROM
	risk_processing_activities
WHERE
	%s
	AND organization_id = @organization_id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"organization_id": organizationID}
	maps.Copy(args, scope.SQLArguments())

	return l.query(ctx, conn, q, args)
}

func (l *RiskProcessingActivities) query(
	ctx context.Context,
	conn pg.Querier,
	q string,
	args pgx.StrictNamedArgs,
) error {
	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query risk processing activities: %w", err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[RiskProcessingActivity])
	if err != nil {
		return fmt.Errorf("cannot collect risk processing activities: %w", err)
	}

	*l = list

	return nil
}

// Replace makes the given processing activities the only ones linked to the
// risk. Links that stay keep their creation time. Every processing activity
// must belong to the organization, or ErrResourceNotFound is returned and
// nothing is changed. The IDs must not repeat.
func (l *RiskProcessingActivities) Replace(
	ctx context.Context,
	tx pg.Tx,
	scope Scoper,
	riskID gid.GID,
	organizationID gid.GID,
	processingActivityIDs []gid.GID,
	createdAt time.Time,
) error {
	ids := make([]string, len(processingActivityIDs))
	for i, id := range processingActivityIDs {
		ids[i] = id.String()
	}

	if err := checkProcessingActivitiesInOrganization(ctx, tx, scope, organizationID, ids); err != nil {
		return err
	}

	deleteQuery := `
DELETE FROM
	risk_processing_activities
WHERE
	%s
	AND risk_id = @risk_id
	AND processing_activity_id <> ALL(@processing_activity_ids::TEXT[])
`

	deleteQuery = fmt.Sprintf(deleteQuery, scope.SQLFragment())

	deleteArgs := pgx.StrictNamedArgs{
		"risk_id":                 riskID,
		"processing_activity_ids": ids,
	}
	maps.Copy(deleteArgs, scope.SQLArguments())

	if _, err := tx.Exec(ctx, deleteQuery, deleteArgs); err != nil {
		return fmt.Errorf("cannot delete risk processing activities: %w", err)
	}

	if len(ids) == 0 {
		return nil
	}

	insertQuery := `
INSERT INTO risk_processing_activities (
	risk_id,
	processing_activity_id,
	tenant_id,
	organization_id,
	created_at
)
SELECT
	@risk_id,
	id,
	@tenant_id,
	@organization_id,
	@created_at
FROM
	UNNEST(@processing_activity_ids::TEXT[]) AS id
ON CONFLICT (risk_id, processing_activity_id) DO NOTHING
`

	insertArgs := pgx.StrictNamedArgs{
		"risk_id":                 riskID,
		"tenant_id":               scope.GetTenantID(),
		"organization_id":         organizationID,
		"processing_activity_ids": ids,
		"created_at":              createdAt,
	}

	if _, err := tx.Exec(ctx, insertQuery, insertArgs); err != nil {
		return fmt.Errorf("cannot insert risk processing activities: %w", err)
	}

	return nil
}

func checkProcessingActivitiesInOrganization(
	ctx context.Context,
	tx pg.Tx,
	scope Scoper,
	organizationID gid.GID,
	processingActivityIDs []string,
) error {
	if len(processingActivityIDs) == 0 {
		return nil
	}

	q := `
SELECT
	COUNT(*)
FROM
	processing_activities
WHERE
	%s
	AND organization_id = @organization_id
	AND id = ANY(@processing_activity_ids::TEXT[])
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{
		"organization_id":         organizationID,
		"processing_activity_ids": processingActivityIDs,
	}
	maps.Copy(args, scope.SQLArguments())

	var count int
	if err := tx.QueryRow(ctx, q, args).Scan(&count); err != nil {
		return fmt.Errorf("cannot count processing activities: %w", err)
	}

	if count != len(processingActivityIDs) {
		return ErrResourceNotFound
	}

	return nil
}
