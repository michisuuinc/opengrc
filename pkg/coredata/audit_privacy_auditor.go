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
	// AuditPrivacyAuditor links an audit to a member who audited it, for the
	// Japanese PrivacyMark internal audit report (内部監査報告書).
	AuditPrivacyAuditor struct {
		AuditID        gid.GID   `db:"audit_id"`
		ProfileID      gid.GID   `db:"profile_id"`
		OrganizationID gid.GID   `db:"organization_id"`
		CreatedAt      time.Time `db:"created_at"`
	}

	AuditPrivacyAuditors []*AuditPrivacyAuditor
)

// LoadByAuditID loads the auditors of the audit in the order they were
// added.
func (l *AuditPrivacyAuditors) LoadByAuditID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	auditID gid.GID,
) error {
	q := `
SELECT
	audit_id,
	profile_id,
	organization_id,
	created_at
FROM
	audit_privacy_auditors
WHERE
	%s
	AND audit_id = @audit_id
ORDER BY
	created_at,
	profile_id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"audit_id": auditID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query audit privacy auditors: %w", err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[AuditPrivacyAuditor])
	if err != nil {
		return fmt.Errorf("cannot collect audit privacy auditors: %w", err)
	}

	*l = list

	return nil
}

// Replace makes the given profiles the only auditors of the audit, in the
// given order. Every profile must belong to the organization, or
// ErrResourceNotFound is returned and nothing is changed. The IDs must not
// repeat.
func (l *AuditPrivacyAuditors) Replace(
	ctx context.Context,
	tx pg.Tx,
	scope Scoper,
	auditID gid.GID,
	organizationID gid.GID,
	profileIDs []gid.GID,
	createdAt time.Time,
) error {
	ids := gidStrings(profileIDs)

	if err := checkProfilesInOrganization(ctx, tx, scope, organizationID, ids); err != nil {
		return err
	}

	// Every row is rewritten so that the creation times follow the new
	// order, which is the order the auditors are listed in.
	deleteQuery := `
DELETE FROM
	audit_privacy_auditors
WHERE
	%s
	AND audit_id = @audit_id
`

	deleteQuery = fmt.Sprintf(deleteQuery, scope.SQLFragment())

	deleteArgs := pgx.StrictNamedArgs{"audit_id": auditID}
	maps.Copy(deleteArgs, scope.SQLArguments())

	if _, err := tx.Exec(ctx, deleteQuery, deleteArgs); err != nil {
		return fmt.Errorf("cannot delete audit privacy auditors: %w", err)
	}

	if len(ids) == 0 {
		return nil
	}

	// Later IDs get a later creation time.
	insertQuery := `
INSERT INTO audit_privacy_auditors (
	audit_id,
	profile_id,
	tenant_id,
	organization_id,
	created_at
)
SELECT
	@audit_id,
	input.id,
	@tenant_id,
	@organization_id,
	@created_at::TIMESTAMPTZ + (input.ord * INTERVAL '1 microsecond')
FROM
	UNNEST(@profile_ids::TEXT[]) WITH ORDINALITY AS input(id, ord)
`

	insertArgs := pgx.StrictNamedArgs{
		"audit_id":        auditID,
		"tenant_id":       scope.GetTenantID(),
		"organization_id": organizationID,
		"profile_ids":     ids,
		"created_at":      createdAt,
	}

	if _, err := tx.Exec(ctx, insertQuery, insertArgs); err != nil {
		return fmt.Errorf("cannot insert audit privacy auditors: %w", err)
	}

	return nil
}

func checkProfilesInOrganization(
	ctx context.Context,
	tx pg.Tx,
	scope Scoper,
	organizationID gid.GID,
	profileIDs []string,
) error {
	if len(profileIDs) == 0 {
		return nil
	}

	q := `
SELECT
	COUNT(*)
FROM
	iam_membership_profiles
WHERE
	%s
	AND organization_id = @organization_id
	AND id = ANY(@profile_ids::TEXT[])
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{
		"organization_id": organizationID,
		"profile_ids":     profileIDs,
	}
	maps.Copy(args, scope.SQLArguments())

	var count int
	if err := tx.QueryRow(ctx, q, args).Scan(&count); err != nil {
		return fmt.Errorf("cannot count membership profiles: %w", err)
	}

	if count != len(profileIDs) {
		return ErrResourceNotFound
	}

	return nil
}
