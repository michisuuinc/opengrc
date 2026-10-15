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
	// TrainingPlanMaterial links a training plan to a document used as its
	// material. Participants complete the training by signing every material.
	TrainingPlanMaterial struct {
		DocumentID gid.GID `db:"document_id"`
		Title      string  `db:"title"`
	}

	TrainingPlanMaterials []*TrainingPlanMaterial
)

// LoadByTrainingPlanID loads the materials of a plan with the title of the
// document's latest version. Deleted documents are left out.
func (m *TrainingPlanMaterials) LoadByTrainingPlanID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	trainingPlanID gid.GID,
) error {
	q := `
WITH materials AS (
	SELECT
		document_id
	FROM
		training_plan_materials
	WHERE
		%s
		AND training_plan_id = @training_plan_id
)
SELECT
	m.document_id,
	COALESCE(lv.title, '') AS title
FROM
	materials m
	INNER JOIN documents d ON d.id = m.document_id AND d.deleted_at IS NULL
	LEFT JOIN LATERAL (
		SELECT title
		FROM document_versions
		WHERE document_id = m.document_id
		ORDER BY major DESC, minor DESC
		LIMIT 1
	) lv ON TRUE
ORDER BY
	title, m.document_id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"training_plan_id": trainingPlanID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query training plan materials: %w", err)
	}

	materials, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[TrainingPlanMaterial])
	if err != nil {
		return fmt.Errorf("cannot collect training plan materials: %w", err)
	}

	*m = materials

	return nil
}

func (m TrainingPlanMaterials) DocumentIDs() []gid.GID {
	ids := make([]gid.GID, len(m))
	for i, material := range m {
		ids[i] = material.DocumentID
	}

	return ids
}

// ReplaceTrainingPlanMaterials replaces the whole set of materials of a plan.
func ReplaceTrainingPlanMaterials(
	ctx context.Context,
	conn pg.Tx,
	scope Scoper,
	organizationID gid.GID,
	trainingPlanID gid.GID,
	documentIDs []gid.GID,
) error {
	deleteQuery := fmt.Sprintf(`
DELETE FROM training_plan_materials
WHERE
	%s
	AND training_plan_id = @training_plan_id
`, scope.SQLFragment())

	deleteArgs := pgx.StrictNamedArgs{"training_plan_id": trainingPlanID}
	maps.Copy(deleteArgs, scope.SQLArguments())

	if _, err := conn.Exec(ctx, deleteQuery, deleteArgs); err != nil {
		return fmt.Errorf("cannot delete training plan materials: %w", err)
	}

	if len(documentIDs) == 0 {
		return nil
	}

	insertQuery := `
INSERT INTO training_plan_materials (
	training_plan_id,
	document_id,
	tenant_id,
	organization_id,
	created_at
)
SELECT
	@training_plan_id,
	document_id,
	@tenant_id,
	@organization_id,
	@created_at
FROM
	UNNEST(@document_ids::TEXT[]) AS document_id
ON CONFLICT DO NOTHING
`

	insertArgs := pgx.StrictNamedArgs{
		"training_plan_id": trainingPlanID,
		"document_ids":     gidStrings(documentIDs),
		"tenant_id":        scope.GetTenantID(),
		"organization_id":  organizationID,
		"created_at":       time.Now(),
	}

	if _, err := conn.Exec(ctx, insertQuery, insertArgs); err != nil {
		return fmt.Errorf("cannot insert training plan materials: %w", err)
	}

	return nil
}
