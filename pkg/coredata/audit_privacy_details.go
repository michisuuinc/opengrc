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
	// AuditPrivacyDetails holds the Japanese PrivacyMark internal audit
	// report (内部監査報告書) items that Audit does not carry. An audit has at
	// most one row. The internal auditors are stored separately in
	// AuditPrivacyAuditors.
	AuditPrivacyDetails struct {
		AuditID            gid.GID   `db:"audit_id"`
		OrganizationID     gid.GID   `db:"organization_id"`
		Scope              string    `db:"scope"`
		Criteria           string    `db:"criteria"`
		AuditeeDepartments []string  `db:"auditee_departments"`
		ExternalAuditors   []string  `db:"external_auditors"`
		OverallOpinion     string    `db:"overall_opinion"`
		CreatedAt          time.Time `db:"created_at"`
		UpdatedAt          time.Time `db:"updated_at"`
	}
)

func (d *AuditPrivacyDetails) LoadByAuditID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	auditID gid.GID,
) error {
	q := `
SELECT
	audit_id,
	organization_id,
	scope,
	criteria,
	auditee_departments,
	external_auditors,
	overall_opinion,
	created_at,
	updated_at
FROM
	audit_privacy_details
WHERE
	%s
	AND audit_id = @audit_id
LIMIT 1;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"audit_id": auditID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query audit privacy details: %w", err)
	}

	details, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[AuditPrivacyDetails])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect audit privacy details: %w", err)
	}

	*d = details

	return nil
}

// LockAudit locks the parent audit row for the rest of the transaction and
// returns its organization. Concurrent updates of the same audit's items then
// run one after the other, including the first insert.
func (d *AuditPrivacyDetails) LockAudit(
	ctx context.Context,
	tx pg.Tx,
	scope Scoper,
	auditID gid.GID,
) (gid.GID, error) {
	q := `
SELECT
	organization_id
FROM
	audits
WHERE
	%s
	AND id = @audit_id
FOR UPDATE;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"audit_id": auditID}
	maps.Copy(args, scope.SQLArguments())

	var organizationID gid.GID
	if err := tx.QueryRow(ctx, q, args).Scan(&organizationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gid.GID{}, ErrResourceNotFound
		}

		return gid.GID{}, fmt.Errorf("cannot lock audit: %w", err)
	}

	return organizationID, nil
}

// Upsert inserts the row or replaces its items. The receiver is synced with
// the stored row, so CreatedAt keeps its first value.
func (d *AuditPrivacyDetails) Upsert(
	ctx context.Context,
	conn pg.Tx,
	scope Scoper,
) error {
	q := `
INSERT INTO audit_privacy_details (
	audit_id,
	tenant_id,
	organization_id,
	scope,
	criteria,
	auditee_departments,
	external_auditors,
	overall_opinion,
	created_at,
	updated_at
) VALUES (
	@audit_id,
	@tenant_id,
	@organization_id,
	@scope,
	@criteria,
	@auditee_departments,
	@external_auditors,
	@overall_opinion,
	@created_at,
	@updated_at
)
ON CONFLICT (audit_id) DO UPDATE
SET
	scope = EXCLUDED.scope,
	criteria = EXCLUDED.criteria,
	auditee_departments = EXCLUDED.auditee_departments,
	external_auditors = EXCLUDED.external_auditors,
	overall_opinion = EXCLUDED.overall_opinion,
	updated_at = EXCLUDED.updated_at
WHERE
	audit_privacy_details.tenant_id = EXCLUDED.tenant_id
RETURNING
	audit_id,
	organization_id,
	scope,
	criteria,
	auditee_departments,
	external_auditors,
	overall_opinion,
	created_at,
	updated_at
`

	args := pgx.StrictNamedArgs{
		"audit_id":            d.AuditID,
		"tenant_id":           scope.GetTenantID(),
		"organization_id":     d.OrganizationID,
		"scope":               d.Scope,
		"criteria":            d.Criteria,
		"auditee_departments": nonNilStrings(d.AuditeeDepartments),
		"external_auditors":   nonNilStrings(d.ExternalAuditors),
		"overall_opinion":     d.OverallOpinion,
		"created_at":          d.CreatedAt,
		"updated_at":          d.UpdatedAt,
	}

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot upsert audit privacy details: %w", err)
	}

	details, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[AuditPrivacyDetails])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect upserted audit privacy details: %w", err)
	}

	*d = details

	return nil
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}
