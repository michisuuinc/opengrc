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

	"github.com/jackc/pgx/v5"
	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/gid"
)

// LoadNamesByRisk returns, for each risk of the organization that has
// internal controls, the names of those controls sorted by name. It feeds the
// Japanese PrivacyMark risk analysis (リスク分析表) export.
func (m *InternalControls) LoadNamesByRisk(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	organizationID gid.GID,
) (map[gid.GID][]string, error) {
	q := `
SELECT
	ric.risk_id,
	ic.name
FROM
	risks_internal_controls ric
INNER JOIN
	internal_controls ic ON ic.id = ric.internal_control_id
WHERE
	ric.%s
	AND ic.%s
	AND ric.organization_id = @organization_id
ORDER BY
	ric.risk_id, ic.name
`

	q = fmt.Sprintf(q, scope.SQLFragment(), scope.SQLFragment())

	args := pgx.StrictNamedArgs{"organization_id": organizationID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return nil, fmt.Errorf("cannot query internal control names by risk: %w", err)
	}
	defer rows.Close()

	names := make(map[gid.GID][]string)
	for rows.Next() {
		var (
			riskID gid.GID
			name   string
		)
		if err := rows.Scan(&riskID, &name); err != nil {
			return nil, fmt.Errorf("cannot scan internal control name: %w", err)
		}
		names[riskID] = append(names[riskID], name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cannot iterate internal control names: %w", err)
	}

	return names, nil
}
