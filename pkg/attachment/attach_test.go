// Copyright (c) 2026 Probo Inc <hello@probo.com>.
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

package attachment

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/internal/test"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/prosemirror"
)

func TestNewFileIDs(t *testing.T) {
	t.Parallel()

	tenant := gid.NewTenantID()
	attachedID := gid.New(tenant, coredata.FileEntityType)
	freshID := gid.New(tenant, coredata.FileEntityType)

	content := fmt.Sprintf(
		`{"type":"doc","content":[{"type":"file","attrs":{"fileId":"%s","fileName":"a.pdf","mimeType":"application/pdf","size":1}},{"type":"file","attrs":{"fileId":"%s","fileName":"b.pdf","mimeType":"application/pdf","size":1}}]}`,
		attachedID,
		freshID,
	)

	t.Run("every file is new when the parent is nil", func(t *testing.T) {
		t.Parallel()

		ids, err := NewService(nil, ParentReadActions{}).NewFileIDs(t.Context(), nil, gid.Nil, gid.Nil, content)
		require.NoError(t, err)
		assert.Equal(t, []gid.GID{attachedID, freshID}, ids)
	})

	t.Run("empty content", func(t *testing.T) {
		t.Parallel()

		ids, err := NewService(nil, ParentReadActions{}).NewFileIDs(t.Context(), nil, gid.Nil, gid.Nil, "")
		require.NoError(t, err)
		assert.Nil(t, ids)
	})

	t.Run("malformed file id is rejected before a query", func(t *testing.T) {
		t.Parallel()

		content := `{"type":"doc","content":[{"type":"file","attrs":{"fileId":"not-a-gid","fileName":"a.pdf","mimeType":"application/pdf","size":1}}]}`
		ids, err := UnattachedFileIDs(t.Context(), nil, nil, gid.Nil, gid.Nil, content)
		require.Error(t, err)
		assert.Nil(t, ids)

		var invalid *ErrInvalidFileID
		require.ErrorAs(t, err, &invalid)
	})

	t.Run("malformed file id", func(t *testing.T) {
		t.Parallel()

		content := `{"type":"doc","content":[{"type":"file","attrs":{"fileId":"not-a-gid","fileName":"a.pdf","mimeType":"application/pdf","size":1}}]}`
		ids, err := NewService(nil, ParentReadActions{}).NewFileIDs(t.Context(), nil, gid.Nil, gid.Nil, content)
		require.Error(t, err)
		assert.Nil(t, ids)

		var invalid *ErrInvalidFileID
		require.ErrorAs(t, err, &invalid)
	})

	t.Run("content that is not json", func(t *testing.T) {
		t.Parallel()

		for _, content := range []string{"<script>xss</script>", "Test\x00"} {
			ids, err := NewService(nil, ParentReadActions{}).NewFileIDs(t.Context(), nil, gid.Nil, gid.Nil, content)
			require.NoError(t, err)
			assert.Nil(t, ids)
		}
	})
}

func TestNewFileIDs_Parent(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	client := test.PGClient(t)
	tenantID := gid.NewTenantID()
	scope := coredata.NewScope(tenantID)
	organizationID := gid.New(tenantID, coredata.OrganizationEntityType)
	attachedParentID := gid.New(tenantID, coredata.TaskEntityType)
	emptyParentID := gid.New(tenantID, coredata.TaskEntityType)
	attachedID := gid.New(tenantID, coredata.FileEntityType)
	freshID := gid.New(tenantID, coredata.FileEntityType)
	now := time.Now().UTC().Truncate(time.Microsecond)

	require.NoError(t, client.WithTx(ctx, func(ctx context.Context, tx pg.Tx) error {
		_, err := tx.Exec(
			ctx,
			`INSERT INTO organizations (id, tenant_id, name, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $4)`,
			organizationID,
			tenantID,
			"attachments-"+organizationID.String(),
			now,
		)
		if err != nil {
			return err
		}

		_, err = tx.Exec(
			ctx,
			`INSERT INTO files (
			     id, tenant_id, organization_id, bucket_name, mime_type, file_name,
			     file_key, file_size, visibility, created_at, updated_at
			 )
			 VALUES ($1, $2, $3, 'test-bucket', 'application/pdf', 'a.pdf', gen_random_uuid(), 1, $4, $5, $5)`,
			attachedID,
			tenantID,
			organizationID,
			coredata.FileVisibilityAttachment,
			now,
		)
		if err != nil {
			return err
		}

		for rank, parentID := range []gid.GID{attachedParentID, emptyParentID} {
			_, err = tx.Exec(
				ctx,
				`INSERT INTO tasks (
				     id, tenant_id, organization_id, name, content, reference_id,
				     state, priority, rank, created_at, updated_at
				 )
				 VALUES (
				     $1, $2, $3, 'task', '{"type":"doc","content":[]}'::jsonb, $4,
				     'TODO', 'MEDIUM', $5, $6, $6
				 )`,
				parentID,
				tenantID,
				organizationID,
				parentID.String(),
				rank+1,
				now,
			)
			if err != nil {
				return err
			}
		}

		return coredata.Attachment{
			FileID:         attachedID,
			OrganizationID: organizationID,
			ParentID:       attachedParentID,
			CreatedAt:      now,
		}.Insert(ctx, tx, scope)
	}))

	t.Cleanup(func() {
		_ = client.WithTx(context.Background(), func(ctx context.Context, tx pg.Tx) error {
			if _, err := tx.Exec(ctx, `DELETE FROM tasks WHERE organization_id = $1`, organizationID); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM files WHERE organization_id = $1`, organizationID); err != nil {
				return err
			}

			_, err := tx.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, organizationID)

			return err
		})
	})

	content := fmt.Sprintf(
		`{"type":"doc","content":[{"type":"file","attrs":{"fileId":"%s","fileName":"a.pdf","mimeType":"application/pdf","size":1}},{"type":"file","attrs":{"fileId":"%s","fileName":"b.pdf","mimeType":"application/pdf","size":1}}]}`,
		attachedID,
		freshID,
	)
	svc := NewService(client, ParentReadActions{})

	t.Run("existing attachment is omitted", func(t *testing.T) {
		t.Parallel()

		ids, err := svc.NewFileIDs(t.Context(), scope, organizationID, attachedParentID, content)
		require.NoError(t, err)
		assert.Equal(t, []gid.GID{freshID}, ids)
	})

	t.Run("every file is new when this parent has no rows", func(t *testing.T) {
		t.Parallel()

		ids, err := svc.NewFileIDs(t.Context(), scope, organizationID, emptyParentID, content)
		require.NoError(t, err)
		assert.Equal(t, []gid.GID{attachedID, freshID}, ids)
	})
}

func TestAttach_RejectsDeletedFile(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	client := test.PGClient(t)
	tenantID := gid.NewTenantID()
	scope := coredata.NewScope(tenantID)
	organizationID := gid.New(tenantID, coredata.OrganizationEntityType)
	taskID := gid.New(tenantID, coredata.TaskEntityType)
	fileID := gid.New(tenantID, coredata.FileEntityType)
	now := time.Now().UTC().Truncate(time.Microsecond)

	require.NoError(t, client.WithTx(ctx, func(ctx context.Context, tx pg.Tx) error {
		if _, err := tx.Exec(
			ctx,
			`INSERT INTO organizations (id, tenant_id, name, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $4)`,
			organizationID,
			tenantID,
			"attachment-deleted-"+organizationID.String(),
			now,
		); err != nil {
			return err
		}

		if _, err := tx.Exec(
			ctx,
			`INSERT INTO tasks (
			     id, tenant_id, organization_id, name, content, reference_id,
			     state, priority, rank, created_at, updated_at
			 )
			 VALUES (
			     $1, $2, $3, 'task', '{"type":"doc","content":[]}'::jsonb, $4,
			     'TODO', 'MEDIUM', 1, $5, $5
			 )`,
			taskID,
			tenantID,
			organizationID,
			taskID.String(),
			now,
		); err != nil {
			return err
		}

		_, err := tx.Exec(
			ctx,
			`INSERT INTO files (
			     id, tenant_id, organization_id, bucket_name, mime_type, file_name,
			     file_key, file_size, visibility, created_at, updated_at, deleted_at
			 )
			 VALUES (
			     $1, $2, $3, 'test-bucket', 'application/pdf', 'a.pdf',
			     gen_random_uuid(), 1, $4, $5, $5, $5
			 )`,
			fileID,
			tenantID,
			organizationID,
			coredata.FileVisibilityAttachment,
			now,
		)

		return err
	}))

	t.Cleanup(func() {
		_ = client.WithTx(context.Background(), func(ctx context.Context, tx pg.Tx) error {
			if _, err := tx.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, taskID); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM files WHERE id = $1`, fileID); err != nil {
				return err
			}

			_, err := tx.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, organizationID)

			return err
		})
	})

	content := fmt.Sprintf(
		`{"type":"doc","content":[{"type":"file","attrs":{"fileId":"%s","fileName":"a.pdf","mimeType":"application/pdf","size":1}}]}`,
		fileID,
	)

	err := client.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
		_, err := Attach(ctx, conn, scope, organizationID, taskID, content)

		return err
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "is deleted")

	var count int

	require.NoError(t, client.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
		return conn.QueryRow(
			ctx,
			`SELECT count(*) FROM attachments WHERE task_id = $1 AND file_id = $2`,
			taskID,
			fileID,
		).Scan(&count)
	}))
	assert.Zero(t, count)
}

func TestAttach_BindsContentAndPrunesRemovedFile(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	client := test.PGClient(t)
	tenantID := gid.NewTenantID()
	scope := coredata.NewScope(tenantID)
	organizationID := gid.New(tenantID, coredata.OrganizationEntityType)
	taskID := gid.New(tenantID, coredata.TaskEntityType)
	keptID := gid.New(tenantID, coredata.FileEntityType)
	removedID := gid.New(tenantID, coredata.FileEntityType)
	now := time.Now().UTC().Truncate(time.Microsecond)

	require.NoError(t, client.WithTx(ctx, func(ctx context.Context, tx pg.Tx) error {
		if _, err := tx.Exec(
			ctx,
			`INSERT INTO organizations (id, tenant_id, name, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $4)`,
			organizationID,
			tenantID,
			"attachment-bind-"+organizationID.String(),
			now,
		); err != nil {
			return err
		}

		if _, err := tx.Exec(
			ctx,
			`INSERT INTO tasks (
			     id, tenant_id, organization_id, name, content, reference_id,
			     state, priority, rank, created_at, updated_at
			 )
			 VALUES (
			     $1, $2, $3, 'task', '{"type":"doc","content":[]}'::jsonb, $4,
			     'TODO', 'MEDIUM', 1, $5, $5
			 )`,
			taskID,
			tenantID,
			organizationID,
			taskID.String(),
			now,
		); err != nil {
			return err
		}

		for _, file := range []struct {
			id   gid.GID
			name string
			size int64
		}{
			{id: keptID, name: "report.pdf", size: 42},
			{id: removedID, name: "old.pdf", size: 7},
		} {
			if _, err := tx.Exec(
				ctx,
				`INSERT INTO files (
				     id, tenant_id, organization_id, bucket_name, mime_type, file_name,
				     file_key, file_size, visibility, created_at, updated_at
				 )
				 VALUES (
				     $1, $2, $3, 'test-bucket', 'application/pdf', $4,
				     gen_random_uuid(), $5, $6, $7, $7
				 )`,
				file.id,
				tenantID,
				organizationID,
				file.name,
				file.size,
				coredata.FileVisibilityAttachment,
				now,
			); err != nil {
				return err
			}
		}

		return coredata.Attachment{
			FileID:         removedID,
			OrganizationID: organizationID,
			ParentID:       taskID,
			CreatedAt:      now,
		}.Insert(ctx, tx, scope)
	}))

	t.Cleanup(func() {
		_ = client.WithTx(context.Background(), func(ctx context.Context, tx pg.Tx) error {
			if _, err := tx.Exec(ctx, `DELETE FROM attachments WHERE task_id = $1`, taskID); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, taskID); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM files WHERE organization_id = $1`, organizationID); err != nil {
				return err
			}

			_, err := tx.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, organizationID)

			return err
		})
	})

	content := fmt.Sprintf(
		`{"type":"doc","content":[{"type":"file","attrs":{"fileId":"%s","fileName":"draft.pdf","mimeType":"text/plain","size":1}}]}`,
		keptID,
	)

	var bound string

	require.NoError(t, client.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
		var err error

		bound, err = Attach(ctx, conn, scope, organizationID, taskID, content)

		return err
	}))

	node, err := prosemirror.Parse(bound)
	require.NoError(t, err)
	require.Len(t, node.Content, 1)

	attrs, err := node.Content[0].FileAttrs()
	require.NoError(t, err)
	assert.Equal(t, keptID.String(), attrs.FileID)
	assert.Equal(t, "report.pdf", attrs.FileName)
	assert.Equal(t, "application/pdf", attrs.MimeType)
	assert.Equal(t, int64(42), attrs.Size)

	var attached []string

	require.NoError(t, client.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
		rows, err := conn.Query(
			ctx,
			`SELECT file_id FROM attachments WHERE task_id = $1`,
			taskID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var fileID string
			if err := rows.Scan(&fileID); err != nil {
				return err
			}

			attached = append(attached, fileID)
		}

		return rows.Err()
	}))
	assert.Equal(t, []string{keptID.String()}, attached)
}

func TestDeleteByDocumentIDs_RemovesEveryVersionAttachment(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	client := test.PGClient(t)
	tenantID := gid.NewTenantID()
	scope := coredata.NewScope(tenantID)
	organizationID := gid.New(tenantID, coredata.OrganizationEntityType)
	deletedDocumentID := gid.New(tenantID, coredata.DocumentEntityType)
	keptDocumentID := gid.New(tenantID, coredata.DocumentEntityType)
	publishedVersionID := gid.New(tenantID, coredata.DocumentVersionEntityType)
	draftVersionID := gid.New(tenantID, coredata.DocumentVersionEntityType)
	keptVersionID := gid.New(tenantID, coredata.DocumentVersionEntityType)
	taskID := gid.New(tenantID, coredata.TaskEntityType)
	publishedFileID := gid.New(tenantID, coredata.FileEntityType)
	draftFileID := gid.New(tenantID, coredata.FileEntityType)
	keptFileID := gid.New(tenantID, coredata.FileEntityType)
	now := time.Now().UTC().Truncate(time.Microsecond)
	publishedMajor := 1
	publishedMinor := 0

	require.NoError(t, client.WithTx(ctx, func(ctx context.Context, tx pg.Tx) error {
		if _, err := tx.Exec(
			ctx,
			`INSERT INTO organizations (id, tenant_id, name, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $4)`,
			organizationID,
			tenantID,
			"attachment-document-delete-"+organizationID.String(),
			now,
		); err != nil {
			return err
		}

		for _, documentID := range []gid.GID{deletedDocumentID, keptDocumentID} {
			if err := (coredata.Document{
				ID:                    documentID,
				OrganizationID:        organizationID,
				CurrentPublishedMajor: &publishedMajor,
				CurrentPublishedMinor: &publishedMinor,
				WriteMode:             coredata.DocumentWriteModeAuthored,
				Status:                coredata.DocumentStatusActive,
				CreatedAt:             now,
				UpdatedAt:             now,
			}).Insert(ctx, tx, scope); err != nil {
				return err
			}
		}

		publishedAt := now
		versions := []coredata.DocumentVersion{
			{
				ID:             publishedVersionID,
				OrganizationID: organizationID,
				DocumentID:     deletedDocumentID,
				Title:          "Published",
				Major:          1,
				Minor:          0,
				Classification: coredata.DocumentClassificationInternal,
				DocumentType:   coredata.DocumentTypePolicy,
				Content:        "published",
				Status:         coredata.DocumentVersionStatusPublished,
				Orientation:    coredata.DocumentVersionOrientationPortrait,
				PublishedAt:    &publishedAt,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			{
				ID:             draftVersionID,
				OrganizationID: organizationID,
				DocumentID:     deletedDocumentID,
				Title:          "Draft",
				Major:          1,
				Minor:          1,
				Classification: coredata.DocumentClassificationInternal,
				DocumentType:   coredata.DocumentTypePolicy,
				Content:        "draft",
				Status:         coredata.DocumentVersionStatusDraft,
				Orientation:    coredata.DocumentVersionOrientationPortrait,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			{
				ID:             keptVersionID,
				OrganizationID: organizationID,
				DocumentID:     keptDocumentID,
				Title:          "Kept",
				Major:          1,
				Minor:          0,
				Classification: coredata.DocumentClassificationInternal,
				DocumentType:   coredata.DocumentTypePolicy,
				Content:        "kept",
				Status:         coredata.DocumentVersionStatusPublished,
				Orientation:    coredata.DocumentVersionOrientationPortrait,
				PublishedAt:    &publishedAt,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
		}

		for _, version := range versions {
			if err := version.Insert(ctx, tx, scope); err != nil {
				return err
			}
		}

		if _, err := tx.Exec(
			ctx,
			`INSERT INTO tasks (
			     id, tenant_id, organization_id, name, content, reference_id,
			     state, priority, rank, created_at, updated_at
			 )
			 VALUES (
			     $1, $2, $3, 'task', '{"type":"doc","content":[]}'::jsonb, $4,
			     'TODO', 'MEDIUM', 1, $5, $5
			 )`,
			taskID,
			tenantID,
			organizationID,
			taskID.String(),
			now,
		); err != nil {
			return err
		}

		files := []struct {
			id   gid.GID
			name string
		}{
			{id: publishedFileID, name: "published.pdf"},
			{id: draftFileID, name: "draft.pdf"},
			{id: keptFileID, name: "kept.pdf"},
		}

		for _, file := range files {
			if _, err := tx.Exec(
				ctx,
				`INSERT INTO files (
				     id, tenant_id, organization_id, bucket_name, mime_type, file_name,
				     file_key, file_size, visibility, created_at, updated_at
				 )
				 VALUES (
				     $1, $2, $3, 'test-bucket', 'application/pdf', $4,
				     gen_random_uuid(), 1, $5, $6, $6
				 )`,
				file.id,
				tenantID,
				organizationID,
				file.name,
				coredata.FileVisibilityAttachment,
				now,
			); err != nil {
				return err
			}
		}

		links := []coredata.Attachment{
			{FileID: publishedFileID, OrganizationID: organizationID, ParentID: publishedVersionID, CreatedAt: now},
			{FileID: draftFileID, OrganizationID: organizationID, ParentID: draftVersionID, CreatedAt: now},
			{FileID: publishedFileID, OrganizationID: organizationID, ParentID: taskID, CreatedAt: now},
			{FileID: keptFileID, OrganizationID: organizationID, ParentID: keptVersionID, CreatedAt: now},
		}

		for _, link := range links {
			if err := link.Insert(ctx, tx, scope); err != nil {
				return err
			}
		}

		return nil
	}))

	t.Cleanup(func() {
		_ = client.WithTx(context.Background(), func(ctx context.Context, tx pg.Tx) error {
			if _, err := tx.Exec(ctx, `DELETE FROM attachments WHERE organization_id = $1`, organizationID); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM document_versions WHERE organization_id = $1`, organizationID); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM documents WHERE organization_id = $1`, organizationID); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, taskID); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM files WHERE organization_id = $1`, organizationID); err != nil {
				return err
			}

			_, err := tx.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, organizationID)

			return err
		})
	})

	require.NoError(t, client.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
		return (coredata.Attachments{}).DeleteByDocumentIDs(ctx, conn, scope, []gid.GID{deletedDocumentID})
	}))

	var (
		deletedCount int
		keptCount    int
		taskCount    int
		fileCount    int
	)

	require.NoError(t, client.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
		if err := conn.QueryRow(
			ctx,
			`SELECT count(*) FROM attachments WHERE document_version_id = ANY($1::text[])`,
			[]string{publishedVersionID.String(), draftVersionID.String()},
		).Scan(&deletedCount); err != nil {
			return err
		}

		if err := conn.QueryRow(
			ctx,
			`SELECT count(*) FROM attachments WHERE document_version_id = $1`,
			keptVersionID,
		).Scan(&keptCount); err != nil {
			return err
		}

		if err := conn.QueryRow(
			ctx,
			`SELECT count(*) FROM attachments WHERE task_id = $1 AND file_id = $2`,
			taskID,
			publishedFileID,
		).Scan(&taskCount); err != nil {
			return err
		}

		return conn.QueryRow(
			ctx,
			`SELECT count(*) FROM files WHERE id = ANY($1::text[])`,
			[]string{publishedFileID.String(), draftFileID.String(), keptFileID.String()},
		).Scan(&fileCount)
	}))

	assert.Zero(t, deletedCount)
	assert.Equal(t, 1, keptCount)
	assert.Equal(t, 1, taskCount)
	assert.Equal(t, 3, fileCount)
}
