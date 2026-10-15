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

// LoadNamesByThirdParty returns, for each third party of the organization
// that register entries are outsourced to, the names of those entries sorted
// by name. It feeds the Japanese PrivacyMark outsourcing evaluation
// (委託先評価表) export.
func (m *ProcessingActivities) LoadNamesByThirdParty(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	organizationID gid.GID,
) (map[gid.GID][]string, error) {
	q := `
SELECT
	patp.third_party_id,
	pa.name
FROM
	processing_activity_third_parties patp
INNER JOIN
	processing_activities pa ON pa.id = patp.processing_activity_id
WHERE
	patp.%s
	AND pa.%s
	AND pa.organization_id = @organization_id
ORDER BY
	patp.third_party_id, pa.name
`

	q = fmt.Sprintf(q, scope.SQLFragment(), scope.SQLFragment())

	args := pgx.StrictNamedArgs{"organization_id": organizationID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return nil, fmt.Errorf("cannot query processing activity names by third party: %w", err)
	}
	defer rows.Close()

	names := make(map[gid.GID][]string)
	for rows.Next() {
		var (
			thirdPartyID gid.GID
			name         string
		)
		if err := rows.Scan(&thirdPartyID, &name); err != nil {
			return nil, fmt.Errorf("cannot scan processing activity name: %w", err)
		}
		names[thirdPartyID] = append(names[thirdPartyID], name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cannot iterate processing activity names: %w", err)
	}

	return names, nil
}
