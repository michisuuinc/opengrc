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

// OrganizationLanguageSettings holds the organization's default language
// for emails whose recipient has no known language. An organization has at
// most one row; without one, such emails are in English.
type OrganizationLanguageSettings struct {
	OrganizationID  gid.GID   `db:"organization_id"`
	DefaultLanguage string    `db:"default_language"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// OrganizationLanguages are the languages an organization can choose.
func OrganizationLanguages() []string {
	return []string{"en", "ja"}
}

func (s *OrganizationLanguageSettings) LoadByOrganizationID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	organizationID gid.GID,
) error {
	q := `
SELECT
	organization_id,
	default_language,
	created_at,
	updated_at
FROM
	organization_language_settings
WHERE
	%s
	AND organization_id = @organization_id
LIMIT 1;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"organization_id": organizationID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query organization language settings: %w", err)
	}

	settings, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[OrganizationLanguageSettings])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect organization language settings: %w", err)
	}

	*s = settings

	return nil
}

// Upsert inserts the row or replaces the default language. The receiver is
// synced with the stored row.
func (s *OrganizationLanguageSettings) Upsert(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
) error {
	q := `
INSERT INTO organization_language_settings (
	organization_id,
	tenant_id,
	default_language,
	created_at,
	updated_at
) VALUES (
	@organization_id,
	@tenant_id,
	@default_language,
	@created_at,
	@updated_at
)
ON CONFLICT (organization_id) DO UPDATE
SET
	default_language = EXCLUDED.default_language,
	updated_at = EXCLUDED.updated_at
WHERE
	organization_language_settings.tenant_id = EXCLUDED.tenant_id
RETURNING
	organization_id,
	default_language,
	created_at,
	updated_at
`

	args := pgx.StrictNamedArgs{
		"organization_id":  s.OrganizationID,
		"tenant_id":        scope.GetTenantID(),
		"default_language": s.DefaultLanguage,
		"created_at":       s.CreatedAt,
		"updated_at":       s.UpdatedAt,
	}

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot upsert organization language settings: %w", err)
	}

	settings, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[OrganizationLanguageSettings])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect upserted organization language settings: %w", err)
	}

	*s = settings

	return nil
}

// LoadOrganizationDefaultLanguage returns the organization's default email
// language, or nil when it has none. It reads by organization ID alone, for
// the services that send emails outside a tenant scope; the value is not
// sensitive.
func LoadOrganizationDefaultLanguage(
	ctx context.Context,
	conn pg.Querier,
	organizationID gid.GID,
) (*string, error) {
	q := `
SELECT
	default_language
FROM
	organization_language_settings
WHERE
	organization_id = @organization_id
`

	var language string
	err := conn.QueryRow(ctx, q, pgx.StrictNamedArgs{"organization_id": organizationID}).Scan(&language)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("cannot load organization default language: %w", err)
	}

	return &language, nil
}
