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
	// ThirdPartyPrivacyEvaluationItem is the answer to one item of the
	// outsourcing evaluation checklist (委託先評価チェックリスト): the result
	// and the documents checked. Items with no answer have no row.
	ThirdPartyPrivacyEvaluationItem struct {
		RiskAssessmentID gid.GID                     `db:"risk_assessment_id"`
		Item             ThirdPartyEvaluationItem    `db:"item"`
		OrganizationID   gid.GID                     `db:"organization_id"`
		Result           *ThirdPartyEvaluationResult `db:"result"`
		Note             string                      `db:"note"`
		CreatedAt        time.Time                   `db:"created_at"`
	}

	ThirdPartyPrivacyEvaluationItems []*ThirdPartyPrivacyEvaluationItem
)

const thirdPartyPrivacyEvaluationItemColumns = `
	risk_assessment_id,
	item,
	organization_id,
	result,
	note,
	created_at
`

func (l *ThirdPartyPrivacyEvaluationItems) LoadByRiskAssessmentID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	riskAssessmentID gid.GID,
) error {
	q := `
SELECT
` + thirdPartyPrivacyEvaluationItemColumns + `
FROM
	third_party_privacy_evaluation_items
WHERE
	%s
	AND risk_assessment_id = @risk_assessment_id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"risk_assessment_id": riskAssessmentID}
	maps.Copy(args, scope.SQLArguments())

	return l.query(ctx, conn, q, args)
}

// LoadByOrganizationID loads the answers of every evaluation of the
// organization.
func (l *ThirdPartyPrivacyEvaluationItems) LoadByOrganizationID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	organizationID gid.GID,
) error {
	q := `
SELECT
` + thirdPartyPrivacyEvaluationItemColumns + `
FROM
	third_party_privacy_evaluation_items
WHERE
	%s
	AND organization_id = @organization_id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"organization_id": organizationID}
	maps.Copy(args, scope.SQLArguments())

	return l.query(ctx, conn, q, args)
}

func (l *ThirdPartyPrivacyEvaluationItems) query(
	ctx context.Context,
	conn pg.Querier,
	q string,
	args pgx.StrictNamedArgs,
) error {
	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query third party privacy evaluation items: %w", err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[ThirdPartyPrivacyEvaluationItem])
	if err != nil {
		return fmt.Errorf("cannot collect third party privacy evaluation items: %w", err)
	}

	*l = list

	return nil
}

// Replace makes the given answers the only answers of the evaluation. The
// evaluation row must exist. The items must not repeat.
func (l ThirdPartyPrivacyEvaluationItems) Replace(
	ctx context.Context,
	tx pg.Tx,
	scope Scoper,
	riskAssessmentID gid.GID,
	organizationID gid.GID,
	createdAt time.Time,
) error {
	deleteQuery := `
DELETE FROM
	third_party_privacy_evaluation_items
WHERE
	%s
	AND risk_assessment_id = @risk_assessment_id
`

	deleteQuery = fmt.Sprintf(deleteQuery, scope.SQLFragment())

	deleteArgs := pgx.StrictNamedArgs{"risk_assessment_id": riskAssessmentID}
	maps.Copy(deleteArgs, scope.SQLArguments())

	if _, err := tx.Exec(ctx, deleteQuery, deleteArgs); err != nil {
		return fmt.Errorf("cannot delete third party privacy evaluation items: %w", err)
	}

	if len(l) == 0 {
		return nil
	}

	items := make([]string, len(l))
	results := make([]*string, len(l))
	notes := make([]string, len(l))
	for i, answer := range l {
		items[i] = answer.Item.String()
		if answer.Result != nil {
			result := answer.Result.String()
			results[i] = &result
		}
		notes[i] = answer.Note
	}

	insertQuery := `
INSERT INTO third_party_privacy_evaluation_items (
	risk_assessment_id,
	item,
	tenant_id,
	organization_id,
	result,
	note,
	created_at
)
SELECT
	@risk_assessment_id,
	input.item,
	@tenant_id,
	@organization_id,
	input.result,
	input.note,
	@created_at
FROM
	UNNEST(@items::TEXT[], @results::TEXT[], @notes::TEXT[]) AS input(item, result, note)
`

	insertArgs := pgx.StrictNamedArgs{
		"risk_assessment_id": riskAssessmentID,
		"tenant_id":          scope.GetTenantID(),
		"organization_id":    organizationID,
		"items":              items,
		"results":            results,
		"notes":              notes,
		"created_at":         createdAt,
	}

	if _, err := tx.Exec(ctx, insertQuery, insertArgs); err != nil {
		return fmt.Errorf("cannot insert third party privacy evaluation items: %w", err)
	}

	return nil
}
