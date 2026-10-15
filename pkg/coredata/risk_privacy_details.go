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
)

type (
	// RiskPrivacyDetails holds the Japanese PrivacyMark risk analysis
	// (リスク分析表) items that Risk does not carry. A risk has at most one
	// row. The register entries the risk applies to are stored separately in
	// RiskProcessingActivities.
	RiskPrivacyDetails struct {
		RiskID          gid.GID                      `db:"risk_id"`
		OrganizationID  gid.GID                      `db:"organization_id"`
		LifecycleStages []PersonalDataLifecycleStage `db:"lifecycle_stages"`
		CreatedAt       time.Time                    `db:"created_at"`
		UpdatedAt       time.Time                    `db:"updated_at"`
	}

	RiskPrivacyDetailsList []*RiskPrivacyDetails
)

func (d *RiskPrivacyDetails) LoadByRiskID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	riskID gid.GID,
) error {
	q := `
SELECT
	risk_id,
	organization_id,
	lifecycle_stages,
	created_at,
	updated_at
FROM
	risk_privacy_details
WHERE
	%s
	AND risk_id = @risk_id
LIMIT 1;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"risk_id": riskID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query risk privacy details: %w", err)
	}

	details, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[RiskPrivacyDetails])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect risk privacy details: %w", err)
	}

	*d = details

	return nil
}

// LockRisk locks the parent risk row for the rest of the transaction and
// returns its organization. Concurrent updates of the same risk's items then
// run one after the other, including the first insert.
func (d *RiskPrivacyDetails) LockRisk(
	ctx context.Context,
	tx pg.Tx,
	scope Scoper,
	riskID gid.GID,
) (gid.GID, error) {
	q := `
SELECT
	organization_id
FROM
	risks
WHERE
	%s
	AND id = @risk_id
FOR UPDATE;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"risk_id": riskID}
	maps.Copy(args, scope.SQLArguments())

	var organizationID gid.GID
	if err := tx.QueryRow(ctx, q, args).Scan(&organizationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gid.GID{}, ErrResourceNotFound
		}

		return gid.GID{}, fmt.Errorf("cannot lock risk: %w", err)
	}

	return organizationID, nil
}

// LoadByOrganizationID loads the rows of every risk of the organization.
func (l *RiskPrivacyDetailsList) LoadByOrganizationID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	organizationID gid.GID,
) error {
	q := `
SELECT
	risk_id,
	organization_id,
	lifecycle_stages,
	created_at,
	updated_at
FROM
	risk_privacy_details
WHERE
	%s
	AND organization_id = @organization_id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"organization_id": organizationID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query risk privacy details: %w", err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[RiskPrivacyDetails])
	if err != nil {
		return fmt.Errorf("cannot collect risk privacy details: %w", err)
	}

	*l = list

	return nil
}

// Upsert inserts the row or replaces its items. The receiver is synced with
// the stored row, so CreatedAt keeps its first value.
func (d *RiskPrivacyDetails) Upsert(
	ctx context.Context,
	conn pg.Tx,
	scope Scoper,
) error {
	q := `
INSERT INTO risk_privacy_details (
	risk_id,
	tenant_id,
	organization_id,
	lifecycle_stages,
	created_at,
	updated_at
) VALUES (
	@risk_id,
	@tenant_id,
	@organization_id,
	@lifecycle_stages,
	@created_at,
	@updated_at
)
ON CONFLICT (risk_id) DO UPDATE
SET
	lifecycle_stages = EXCLUDED.lifecycle_stages,
	updated_at = EXCLUDED.updated_at
WHERE
	risk_privacy_details.tenant_id = EXCLUDED.tenant_id
RETURNING
	risk_id,
	organization_id,
	lifecycle_stages,
	created_at,
	updated_at
`

	stages := d.LifecycleStages
	if stages == nil {
		stages = []PersonalDataLifecycleStage{}
	}

	args := pgx.StrictNamedArgs{
		"risk_id":          d.RiskID,
		"tenant_id":        scope.GetTenantID(),
		"organization_id":  d.OrganizationID,
		"lifecycle_stages": stages,
		"created_at":       d.CreatedAt,
		"updated_at":       d.UpdatedAt,
	}

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot upsert risk privacy details: %w", err)
	}

	details, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[RiskPrivacyDetails])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect upserted risk privacy details: %w", err)
	}

	*d = details

	return nil
}
