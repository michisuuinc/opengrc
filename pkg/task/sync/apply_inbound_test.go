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

package tasksync

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/internal/test"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/prosemirror"
	"go.probo.inc/probo/pkg/task/sync/linear"
)

func TestWebhookIssueIdentity(t *testing.T) {
	t.Parallel()

	t.Run("linked issue candidate", func(t *testing.T) {
		t.Parallel()

		issueID, organizationID, keep, err := webhookIssueIdentity([]byte(`{
			"type":"Issue",
			"organizationId":"org-1",
			"data":{"id":"issue-1","title":"Hello"}
		}`))
		require.NoError(t, err)
		assert.True(t, keep)
		assert.Equal(t, "issue-1", issueID)
		assert.Equal(t, "org-1", organizationID)
	})

	t.Run("comment on a linked issue is stored", func(t *testing.T) {
		t.Parallel()

		issueID, organizationID, keep, err := webhookIssueIdentity([]byte(`{
			"type":"Comment",
			"organizationId":"org-1",
			"data":{"id":"comment-1","issueId":"issue-1","body":"Hi"}
		}`))
		require.NoError(t, err)
		assert.True(t, keep)
		assert.Equal(t, "issue-1", issueID)
		assert.Equal(t, "org-1", organizationID)
	})

	t.Run("comment without an issue is not stored", func(t *testing.T) {
		t.Parallel()

		_, _, keep, err := webhookIssueIdentity([]byte(`{
			"type":"Comment",
			"organizationId":"org-1",
			"data":{"id":"comment-1","body":"Hi"}
		}`))
		require.NoError(t, err)
		assert.False(t, keep)
	})

	t.Run("issue without id is not stored", func(t *testing.T) {
		t.Parallel()

		_, _, keep, err := webhookIssueIdentity([]byte(`{
			"type":"Issue",
			"organizationId":"org-1",
			"data":{"title":"Hello"}
		}`))
		require.NoError(t, err)
		assert.False(t, keep)
	})

	t.Run("issue without organization is not stored", func(t *testing.T) {
		t.Parallel()

		_, _, keep, err := webhookIssueIdentity([]byte(`{
			"type":"Issue",
			"data":{"id":"issue-1"}
		}`))
		require.NoError(t, err)
		assert.False(t, keep)
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		_, _, _, err := webhookIssueIdentity([]byte(`{`))
		require.Error(t, err)
	})
}

func TestIsAppActor(t *testing.T) {
	t.Parallel()

	metadata, err := json.Marshal(map[string]string{"app_actor_id": "app-1"})
	require.NoError(t, err)

	assert.True(t, isAppActor(metadata, "app-1"))
	assert.False(t, isAppActor(metadata, "user-1"))
	assert.False(t, isAppActor(nil, "app-1"))
	assert.False(t, isAppActor(metadata, ""))
}

func TestInboundEventIsStale(t *testing.T) {
	t.Parallel()

	older := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	newer := older.Add(time.Minute)

	assert.True(t, inboundEventIsStale(nil, &newer))
	assert.False(t, inboundEventIsStale(&newer, nil))
	assert.False(t, inboundEventIsStale(nil, nil))
	assert.False(t, inboundEventIsStale(&newer, &older))
	assert.True(t, inboundEventIsStale(&older, &newer))
	assert.True(t, inboundEventIsStale(&newer, &newer))
}

func TestInboundActivityActorID_EmptyOrInvalidEmail(t *testing.T) {
	t.Parallel()

	id, err := inboundActivityActorID(
		t.Context(),
		nil,
		nil,
		gid.GID{},
		linear.WebhookActor{},
	)
	require.NoError(t, err)
	assert.Nil(t, id)

	id, err = inboundActivityActorID(
		t.Context(),
		nil,
		nil,
		gid.GID{},
		linear.WebhookActor{Email: "   "},
	)
	require.NoError(t, err)
	assert.Nil(t, id)

	id, err = inboundActivityActorID(
		t.Context(),
		nil,
		nil,
		gid.GID{},
		linear.WebhookActor{Email: "not-an-email"},
	)
	require.NoError(t, err)
	assert.Nil(t, id)
}

func TestRecordInboundActivities_SkippedWhenUnwired(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	err := svc.recordInboundActivities(
		t.Context(),
		nil,
		nil,
		gid.GID{},
		&coredata.Task{},
		&coredata.Task{},
		linear.WebhookActor{},
		time.Now(),
	)
	require.NoError(t, err)
}

func TestLinearDateTime(t *testing.T) {
	t.Parallel()

	got := linearDateTime("2026-09-14T12:00:00.000Z")
	require.NotNil(t, got)
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, time.September, got.Month())
	assert.Equal(t, 14, got.Day())

	assert.Nil(t, linearDateTime(""))
	assert.Nil(t, linearDateTime("not-a-date"))
}

func TestPickLinkForLinearOrganization(t *testing.T) {
	t.Parallel()

	linkA := &coredata.TaskExternalLink{
		Destination: json.RawMessage(`{"team_id":"team-1","linear_organization_id":"org-a"}`),
	}
	linkB := &coredata.TaskExternalLink{
		Destination: json.RawMessage(`{"team_id":"team-1","linear_organization_id":"org-b"}`),
	}

	got, ok := PickLinkForLinearOrganization(coredata.TaskExternalLinks{}, "org-a")
	assert.False(t, ok)
	assert.Nil(t, got)

	got, ok = PickLinkForLinearOrganization(coredata.TaskExternalLinks{linkA}, "org-a")
	require.True(t, ok)
	assert.Equal(t, linkA, got)

	got, ok = PickLinkForLinearOrganization(coredata.TaskExternalLinks{linkA}, "org-other")
	assert.False(t, ok)
	assert.Nil(t, got)

	got, ok = PickLinkForLinearOrganization(coredata.TaskExternalLinks{linkA, linkB}, "org-b")
	require.True(t, ok)
	assert.Equal(t, linkB, got)

	got, ok = PickLinkForLinearOrganization(coredata.TaskExternalLinks{linkA, linkB}, "org-missing")
	assert.False(t, ok)
	assert.Nil(t, got)

	got, ok = PickLinkForLinearOrganization(coredata.TaskExternalLinks{linkA, linkB}, "")
	assert.False(t, ok)
	assert.Nil(t, got)

	got, ok = PickLinkForLinearOrganization(coredata.TaskExternalLinks{linkA}, "")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestMapInboundIssue_PartialPayloadKeepsOmittedFields(t *testing.T) {
	t.Parallel()

	deadline := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	content, err := MarkdownToContent("Keep this body")
	require.NoError(t, err)

	task := &coredata.Task{
		Name:     "Original",
		Content:  content,
		State:    coredata.TaskStateTodo,
		Priority: coredata.TaskPriorityHigh,
		Deadline: &deadline,
	}

	envelope, err := linear.ParseEnvelope([]byte(`{
		"action":"update",
		"type":"Issue",
		"data":{"title":"Renamed","dueDate":null}
	}`))
	require.NoError(t, err)

	data, err := envelope.IssueData()
	require.NoError(t, err)

	mapped, err := mapInboundIssue(task, data, nil)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", mapped.Name)
	assert.Equal(t, content, mapped.Content)
	assert.Equal(t, coredata.TaskStateTodo, mapped.State)
	assert.Equal(t, coredata.TaskPriorityHigh, mapped.Priority)
	assert.Nil(t, mapped.Deadline)
}

func TestMapInboundIssue_UnsetLinearPriorityKeepsExisting(t *testing.T) {
	t.Parallel()

	task := &coredata.Task{
		Name:     "Original",
		State:    coredata.TaskStateTodo,
		Priority: coredata.TaskPriorityHigh,
	}

	envelope, err := linear.ParseEnvelope([]byte(`{
		"action":"update",
		"type":"Issue",
		"data":{"title":"Renamed","priority":0}
	}`))
	require.NoError(t, err)

	data, err := envelope.IssueData()
	require.NoError(t, err)
	require.True(t, data.Has("priority"))
	assert.Equal(t, 0, data.Priority)

	mapped, err := mapInboundIssue(task, data, nil)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", mapped.Name)
	assert.Equal(t, coredata.TaskPriorityHigh, mapped.Priority)

	hash := ContentHash(mapped.Name, mapped.Markdown, mapped.State, mapped.Priority, mapped.Deadline, nil)
	assert.Equal(
		t,
		ContentHash("Renamed", mapped.Markdown, coredata.TaskStateTodo, coredata.TaskPriorityHigh, nil, nil),
		hash,
	)
	assert.NotEqual(
		t,
		ContentHash("Renamed", mapped.Markdown, coredata.TaskStateTodo, coredata.TaskPriorityMedium, nil, nil),
		hash,
	)
}

func TestMapInboundIssue_DescriptionHashMatchesOutboundRender(t *testing.T) {
	t.Parallel()

	task := &coredata.Task{
		Name:     "Title",
		State:    coredata.TaskStateTodo,
		Priority: coredata.TaskPriorityMedium,
	}

	envelope, err := linear.ParseEnvelope([]byte(`{
		"action":"update",
		"type":"Issue",
		"data":{"description":"Hello **world**"}
	}`))
	require.NoError(t, err)

	data, err := envelope.IssueData()
	require.NoError(t, err)

	mapped, err := mapInboundIssue(task, data, nil)
	require.NoError(t, err)

	inboundHash := ContentHash(
		mapped.Name,
		mapped.Markdown,
		mapped.State,
		mapped.Priority,
		mapped.Deadline,
		task.AssignedToID,
	)

	task.Content = mapped.Content
	outboundHash, err := taskContentHash(task)
	require.NoError(t, err)
	assert.Equal(t, inboundHash, outboundHash)

	rawHash := ContentHash(
		mapped.Name,
		data.Description,
		mapped.State,
		mapped.Priority,
		mapped.Deadline,
		task.AssignedToID,
	)
	assert.NotEqual(t, rawHash, outboundHash)
}

func TestApplyInboundIssue_TypedAttachment(t *testing.T) {
	t.Parallel()

	t.Run("links an attachment file in the organization", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		fx := seedInboundIssueWithFile(t, coredata.FileVisibilityAttachment, true)

		require.NoError(t, applyInboundAttachmentDescription(ctx, fx))

		var count int

		require.NoError(t, fx.client.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
			return conn.QueryRow(
				ctx,
				`SELECT count(*) FROM attachments WHERE task_id = $1 AND file_id = $2`,
				fx.taskID,
				fx.fileID,
			).Scan(&count)
		}))
		assert.Equal(t, 1, count)

		content := taskContent(t, fx)
		assert.Contains(t, content, fx.fileID.String())
	})

	t.Run("rejects a file from another organization", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		fx := seedInboundIssueWithFile(t, coredata.FileVisibilityAttachment, false)

		err := applyInboundAttachmentDescription(ctx, fx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in this organization")
		assert.NotContains(t, taskContent(t, fx), fx.fileID.String())
	})

	t.Run("rejects a file that is not an attachment", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		fx := seedInboundIssueWithFile(t, coredata.FileVisibilityPrivate, true)

		err := applyInboundAttachmentDescription(ctx, fx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not an attachment file")
		assert.NotContains(t, taskContent(t, fx), fx.fileID.String())
	})
}

type inboundIssueFixture struct {
	client         *pg.Client
	scope          coredata.Scoper
	organizationID gid.GID
	taskID         gid.GID
	connectorID    gid.GID
	fileID         gid.GID
	externalID     string
	linearOrgID    string
}

func seedInboundIssueWithFile(t *testing.T, visibility coredata.FileVisibility, fileInTaskOrg bool) inboundIssueFixture {
	t.Helper()

	ctx := t.Context()
	client := test.PGClient(t)
	tenantID := gid.NewTenantID()
	organizationID := gid.New(tenantID, coredata.OrganizationEntityType)

	fileOrganizationID := organizationID
	if !fileInTaskOrg {
		fileOrganizationID = gid.New(tenantID, coredata.OrganizationEntityType)
	}

	taskID := gid.New(tenantID, coredata.TaskEntityType)
	connectorID := gid.New(tenantID, coredata.ConnectorEntityType)
	fileID := gid.New(tenantID, coredata.FileEntityType)
	now := time.Now().UTC().Truncate(time.Microsecond)
	fx := inboundIssueFixture{
		client:         client,
		scope:          coredata.NewScope(tenantID),
		organizationID: organizationID,
		taskID:         taskID,
		connectorID:    connectorID,
		fileID:         fileID,
		externalID:     "issue-" + taskID.String(),
		linearOrgID:    "lin-" + organizationID.String(),
	}

	require.NoError(t, client.WithTx(ctx, func(ctx context.Context, tx pg.Tx) error {
		if _, err := tx.Exec(
			ctx,
			`INSERT INTO organizations (id, tenant_id, name, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $4)`,
			organizationID,
			tenantID,
			"inbound-issue-"+organizationID.String(),
			now,
		); err != nil {
			return err
		}

		if fileOrganizationID != organizationID {
			if _, err := tx.Exec(
				ctx,
				`INSERT INTO organizations (id, tenant_id, name, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $4)`,
				fileOrganizationID,
				tenantID,
				"inbound-issue-file-"+fileOrganizationID.String(),
				now,
			); err != nil {
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
			     $1, $2, $3, 'links', '{"type":"doc","content":[]}'::jsonb, $4,
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

		if _, err := tx.Exec(
			ctx,
			`INSERT INTO files (
			     id, tenant_id, organization_id, bucket_name, mime_type, file_name,
			     file_key, file_size, visibility, created_at, updated_at
			 )
			 VALUES ($1, $2, $3, 'test-bucket', 'application/pdf', 'report.pdf', gen_random_uuid(), 12, $4, $5, $5)`,
			fileID,
			tenantID,
			fileOrganizationID,
			visibility,
			now,
		); err != nil {
			return err
		}

		if _, err := tx.Exec(
			ctx,
			`INSERT INTO connectors (
			     id, tenant_id, organization_id, provider, protocol, name,
			     settings, encrypted_connection, created_at, updated_at
			 )
			 VALUES (
			     $1, $2, $3, 'LINEAR_SYNC', 'OAUTH2', 'Linear sync',
			     '{}', '\x', $4, $4
			 )`,
			connectorID,
			tenantID,
			organizationID,
			now,
		); err != nil {
			return err
		}

		_, err := tx.Exec(
			ctx,
			`INSERT INTO task_external_links (
			     task_id, tenant_id, organization_id, connector_id, provider,
			     external_id, external_identifier, external_url, destination, origin,
			     metadata, created_at, updated_at
			 )
			 VALUES (
			     $1, $2, $3, $4, 'LINEAR_SYNC',
			     $5, 'ENG-3', 'https://linear.app/issue/ENG-3', $6, 'PROBO',
			     '{}', $7, $7
			 )`,
			taskID,
			tenantID,
			organizationID,
			connectorID,
			fx.externalID,
			`{"linear_organization_id":"`+fx.linearOrgID+`"}`,
			now,
		)

		return err
	}))

	t.Cleanup(func() {
		assert.NoError(t, client.WithTx(context.Background(), func(ctx context.Context, tx pg.Tx) error {
			task := taskID.String()

			if _, err := tx.Exec(
				ctx,
				`DELETE FROM webhook_events
				 WHERE webhook_data_id IN (
				     SELECT id FROM webhook_data
				     WHERE data->>'id' = $1 OR updated_from->>'id' = $1
				 )`,
				task,
			); err != nil {
				return err
			}

			if _, err := tx.Exec(
				ctx,
				`DELETE FROM webhook_data
				 WHERE data->>'id' = $1 OR updated_from->>'id' = $1`,
				task,
			); err != nil {
				return err
			}

			if _, err := tx.Exec(
				ctx,
				`DELETE FROM task_sync_jobs WHERE payload->>'task_id' = $1`,
				task,
			); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM attachments WHERE task_id = $1`, taskID); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM task_external_links WHERE task_id = $1`, taskID); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, taskID); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM files WHERE id = $1`, fileID); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM connectors WHERE id = $1`, connectorID); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, organizationID); err != nil {
				return err
			}

			if fileOrganizationID == organizationID {
				return nil
			}

			_, err := tx.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, fileOrganizationID)

			return err
		}))
	})

	return fx
}

func applyInboundAttachmentDescription(ctx context.Context, fx inboundIssueFixture) error {
	body, err := json.Marshal(map[string]any{
		"action":         "update",
		"type":           "Issue",
		"organizationId": fx.linearOrgID,
		"actor":          map[string]string{"id": "user-1"},
		"data": map[string]any{
			"id":          fx.externalID,
			"description": "[report.pdf](" + prosemirror.AttachmentPath(fx.fileID.String()) + ")",
			"updatedAt":   "2026-10-12T12:00:00Z",
		},
	})
	if err != nil {
		return err
	}

	envelope, err := linear.ParseEnvelope(body)
	if err != nil {
		return err
	}

	return (&Service{pg: fx.client}).ApplyInboundIssue(ctx, envelope)
}

func taskContent(t *testing.T, fx inboundIssueFixture) string {
	t.Helper()

	var content string

	require.NoError(t, fx.client.WithConn(t.Context(), func(ctx context.Context, conn pg.Querier) error {
		return conn.QueryRow(ctx, `SELECT content::text FROM tasks WHERE id = $1`, fx.taskID).Scan(&content)
	}))

	return content
}

func TestFinishLinearLink_TypedAttachment(t *testing.T) {
	t.Parallel()

	t.Run("links an attachment file from the Linear description", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		fx := seedInboundIssueWithFile(t, coredata.FileVisibilityAttachment, true)

		require.NoError(t, finishLinearLinkWithAttachment(ctx, fx))

		var count int

		require.NoError(t, fx.client.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
			return conn.QueryRow(
				ctx,
				`SELECT count(*) FROM attachments WHERE task_id = $1 AND file_id = $2`,
				fx.taskID,
				fx.fileID,
			).Scan(&count)
		}))
		assert.Equal(t, 1, count)
		assert.Contains(t, taskContent(t, fx), fx.fileID.String())
	})

	t.Run("rejects a file from another organization", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		fx := seedInboundIssueWithFile(t, coredata.FileVisibilityAttachment, false)

		err := finishLinearLinkWithAttachment(ctx, fx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in this organization")
		assert.NotContains(t, taskContent(t, fx), fx.fileID.String())
	})
}

func finishLinearLinkWithAttachment(ctx context.Context, fx inboundIssueFixture) error {
	description := "[report.pdf](" + prosemirror.AttachmentPath(fx.fileID.String()) + ")"
	issue := &linear.Issue{
		ID:          fx.externalID,
		Identifier:  "ENG-3",
		URL:         "https://linear.app/issue/ENG-3",
		Title:       "From Linear",
		Description: description,
		UpdatedAt:   time.Now(),
	}

	content, markdown, err := linearIssueContent(issue)
	if err != nil {
		return err
	}

	destination, err := json.Marshal(coredata.TaskExternalLinkDestination{
		TeamID:               "team-1",
		LinearOrganizationID: fx.linearOrgID,
	})
	if err != nil {
		return err
	}

	_, _, _, err = (&Service{pg: fx.client}).finishLinearLink(
		ctx,
		fx.scope,
		&coredata.Task{ID: fx.taskID, OrganizationID: fx.organizationID},
		&coredata.Connector{ID: fx.connectorID},
		destination,
		"",
		issue,
		content,
		markdown,
		"linear-attachment",
		nil,
	)

	return err
}

func TestCommentContentHashStable(t *testing.T) {
	t.Parallel()

	assert.Equal(t, CommentContentHash("hello"), CommentContentHash("hello"))
	assert.NotEqual(t, CommentContentHash("hello"), CommentContentHash("Hello"))
}
