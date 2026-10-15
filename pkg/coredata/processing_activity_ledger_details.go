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
	// ProcessingActivityLedgerDetails holds the Japanese PrivacyMark register
	// (個人情報管理台帳) items that ProcessingActivity does not carry. A
	// processing activity has at most one row.
	ProcessingActivityLedgerDetails struct {
		ProcessingActivityID   gid.GID   `db:"processing_activity_id"`
		OrganizationID         gid.GID   `db:"organization_id"`
		Department             *string   `db:"department"`
		ApproximateRecordCount *string   `db:"approximate_record_count"`
		CollectionMethod       *string   `db:"collection_method"`
		StorageLocation        *string   `db:"storage_location"`
		AccessScope            *string   `db:"access_scope"`
		DisposalMethod         *string   `db:"disposal_method"`
		CreatedAt              time.Time `db:"created_at"`
		UpdatedAt              time.Time `db:"updated_at"`
	}

	ProcessingActivityLedgerDetailsList []*ProcessingActivityLedgerDetails
)

func (d *ProcessingActivityLedgerDetails) LoadByProcessingActivityID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	processingActivityID gid.GID,
) error {
	q := `
SELECT
	processing_activity_id,
	organization_id,
	department,
	approximate_record_count,
	collection_method,
	storage_location,
	access_scope,
	disposal_method,
	created_at,
	updated_at
FROM
	processing_activity_ledger_details
WHERE
	%s
	AND processing_activity_id = @processing_activity_id
LIMIT 1;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"processing_activity_id": processingActivityID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query processing activity ledger details: %w", err)
	}

	details, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[ProcessingActivityLedgerDetails])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect processing activity ledger details: %w", err)
	}

	*d = details

	return nil
}

// LockProcessingActivity locks the parent processing activity row for the
// rest of the transaction and returns its organization. Concurrent updates of
// the same register items then run one after the other, including the first
// insert.
func (d *ProcessingActivityLedgerDetails) LockProcessingActivity(
	ctx context.Context,
	tx pg.Tx,
	scope Scoper,
	processingActivityID gid.GID,
) (gid.GID, error) {
	q := `
SELECT
	organization_id
FROM
	processing_activities
WHERE
	%s
	AND id = @processing_activity_id
FOR UPDATE;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"processing_activity_id": processingActivityID}
	maps.Copy(args, scope.SQLArguments())

	var organizationID gid.GID
	if err := tx.QueryRow(ctx, q, args).Scan(&organizationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gid.GID{}, ErrResourceNotFound
		}

		return gid.GID{}, fmt.Errorf("cannot lock processing activity: %w", err)
	}

	return organizationID, nil
}

// LoadByProcessingActivityIDs loads the rows of the given processing
// activities. The result is bounded by the caller's ID list.
func (l *ProcessingActivityLedgerDetailsList) LoadByProcessingActivityIDs(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	processingActivityIDs []gid.GID,
) error {
	if len(processingActivityIDs) == 0 {
		*l = ProcessingActivityLedgerDetailsList{}
		return nil
	}

	q := `
SELECT
	processing_activity_id,
	organization_id,
	department,
	approximate_record_count,
	collection_method,
	storage_location,
	access_scope,
	disposal_method,
	created_at,
	updated_at
FROM
	processing_activity_ledger_details
WHERE
	%s
	AND processing_activity_id = ANY(@processing_activity_ids)
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"processing_activity_ids": processingActivityIDs}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query processing activity ledger details: %w", err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[ProcessingActivityLedgerDetails])
	if err != nil {
		return fmt.Errorf("cannot collect processing activity ledger details: %w", err)
	}

	*l = list

	return nil
}

// Upsert inserts the row or replaces its items. The receiver is synced with
// the stored row, so CreatedAt keeps its first value.
func (d *ProcessingActivityLedgerDetails) Upsert(
	ctx context.Context,
	conn pg.Tx,
	scope Scoper,
) error {
	q := `
INSERT INTO processing_activity_ledger_details (
	processing_activity_id,
	tenant_id,
	organization_id,
	department,
	approximate_record_count,
	collection_method,
	storage_location,
	access_scope,
	disposal_method,
	created_at,
	updated_at
) VALUES (
	@processing_activity_id,
	@tenant_id,
	@organization_id,
	@department,
	@approximate_record_count,
	@collection_method,
	@storage_location,
	@access_scope,
	@disposal_method,
	@created_at,
	@updated_at
)
ON CONFLICT (processing_activity_id) DO UPDATE
SET
	department = EXCLUDED.department,
	approximate_record_count = EXCLUDED.approximate_record_count,
	collection_method = EXCLUDED.collection_method,
	storage_location = EXCLUDED.storage_location,
	access_scope = EXCLUDED.access_scope,
	disposal_method = EXCLUDED.disposal_method,
	updated_at = EXCLUDED.updated_at
WHERE
	processing_activity_ledger_details.tenant_id = EXCLUDED.tenant_id
RETURNING
	processing_activity_id,
	organization_id,
	department,
	approximate_record_count,
	collection_method,
	storage_location,
	access_scope,
	disposal_method,
	created_at,
	updated_at
`

	args := pgx.StrictNamedArgs{
		"processing_activity_id":   d.ProcessingActivityID,
		"tenant_id":                scope.GetTenantID(),
		"organization_id":          d.OrganizationID,
		"department":               d.Department,
		"approximate_record_count": d.ApproximateRecordCount,
		"collection_method":        d.CollectionMethod,
		"storage_location":         d.StorageLocation,
		"access_scope":             d.AccessScope,
		"disposal_method":          d.DisposalMethod,
		"created_at":               d.CreatedAt,
		"updated_at":               d.UpdatedAt,
	}

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot upsert processing activity ledger details: %w", err)
	}

	details, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[ProcessingActivityLedgerDetails])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect upserted processing activity ledger details: %w", err)
	}

	*d = details

	return nil
}
