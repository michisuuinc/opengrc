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
	// FindingPrivacyDetails holds the Japanese PrivacyMark internal audit
	// items that Finding does not carry: the category of the internal audit
	// procedure and the requirement the finding is based on. A finding has
	// at most one row. A nil Category means none was chosen.
	FindingPrivacyDetails struct {
		FindingID      gid.GID                 `db:"finding_id"`
		OrganizationID gid.GID                 `db:"organization_id"`
		Category       *FindingPrivacyCategory `db:"category"`
		Basis          string                  `db:"basis"`
		CreatedAt      time.Time               `db:"created_at"`
		UpdatedAt      time.Time               `db:"updated_at"`
	}

	FindingPrivacyDetailsList []*FindingPrivacyDetails
)

const findingPrivacyDetailsColumns = `
	finding_id,
	organization_id,
	category,
	basis,
	created_at,
	updated_at
`

func (d *FindingPrivacyDetails) LoadByFindingID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	findingID gid.GID,
) error {
	q := `
SELECT` + findingPrivacyDetailsColumns + `
FROM
	finding_privacy_details
WHERE
	%s
	AND finding_id = @finding_id
LIMIT 1;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"finding_id": findingID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query finding privacy details: %w", err)
	}

	details, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[FindingPrivacyDetails])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect finding privacy details: %w", err)
	}

	*d = details

	return nil
}

// LoadByAuditID loads the rows of every finding linked to the audit.
func (l *FindingPrivacyDetailsList) LoadByAuditID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	auditID gid.GID,
) error {
	q := `
SELECT
	d.finding_id,
	d.organization_id,
	d.category,
	d.basis,
	d.created_at,
	d.updated_at
FROM
	finding_privacy_details d
INNER JOIN findings_audits fa
	ON fa.finding_id = d.finding_id
WHERE
	d.%s
	AND fa.audit_id = @audit_id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"audit_id": auditID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query finding privacy details: %w", err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[FindingPrivacyDetails])
	if err != nil {
		return fmt.Errorf("cannot collect finding privacy details: %w", err)
	}

	*l = list

	return nil
}

// LockFinding locks the parent finding row for the rest of the transaction
// and returns its organization.
func (d *FindingPrivacyDetails) LockFinding(
	ctx context.Context,
	tx pg.Tx,
	scope Scoper,
	findingID gid.GID,
) (gid.GID, error) {
	q := `
SELECT
	organization_id
FROM
	findings
WHERE
	%s
	AND id = @finding_id
FOR UPDATE;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"finding_id": findingID}
	maps.Copy(args, scope.SQLArguments())

	var organizationID gid.GID
	if err := tx.QueryRow(ctx, q, args).Scan(&organizationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gid.GID{}, ErrResourceNotFound
		}

		return gid.GID{}, fmt.Errorf("cannot lock finding: %w", err)
	}

	return organizationID, nil
}

// Upsert inserts the row or replaces its items. The receiver is synced with
// the stored row, so CreatedAt keeps its first value.
func (d *FindingPrivacyDetails) Upsert(
	ctx context.Context,
	conn pg.Tx,
	scope Scoper,
) error {
	q := `
INSERT INTO finding_privacy_details (
	finding_id,
	tenant_id,
	organization_id,
	category,
	basis,
	created_at,
	updated_at
) VALUES (
	@finding_id,
	@tenant_id,
	@organization_id,
	@category,
	@basis,
	@created_at,
	@updated_at
)
ON CONFLICT (finding_id) DO UPDATE
SET
	category = EXCLUDED.category,
	basis = EXCLUDED.basis,
	updated_at = EXCLUDED.updated_at
WHERE
	finding_privacy_details.tenant_id = EXCLUDED.tenant_id
RETURNING` + findingPrivacyDetailsColumns

	args := pgx.StrictNamedArgs{
		"finding_id":      d.FindingID,
		"tenant_id":       scope.GetTenantID(),
		"organization_id": d.OrganizationID,
		"category":        d.Category,
		"basis":           d.Basis,
		"created_at":      d.CreatedAt,
		"updated_at":      d.UpdatedAt,
	}

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot upsert finding privacy details: %w", err)
	}

	details, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[FindingPrivacyDetails])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect upserted finding privacy details: %w", err)
	}

	*d = details

	return nil
}
