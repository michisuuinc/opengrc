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

package files_v1

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.gearno.de/kit/log"
	"go.probo.inc/probo/pkg/attachment"
	"go.probo.inc/probo/pkg/baseurl"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/iam"
	"go.probo.inc/probo/pkg/securecookie"
	"go.probo.inc/probo/pkg/server/api/authn"
)

func TestRenderAttachmentAuthError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		status  int
		wantLog bool
	}{
		{
			name:   "insufficient permissions",
			err:    iam.NewInsufficientPermissionsError(gid.Nil, gid.Nil, "core:document-version:get"),
			status: http.StatusNotFound,
		},
		{
			name:   "insufficient scope",
			err:    iam.NewInsufficientOAuth2ScopeError(gid.Nil, coredata.OAuth2Scope("documents:read")),
			status: http.StatusNotFound,
		},
		{
			name:   "missing resource",
			err:    coredata.ErrResourceNotFound,
			status: http.StatusNotFound,
		},
		{
			name:    "database failure",
			err:     fmt.Errorf("cannot load memberships for principal: %w", errors.New("connection refused")),
			status:  http.StatusInternalServerError,
			wantLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logged bytes.Buffer

			h := &Handler{logger: log.NewLogger(log.WithOutput(&logged))}
			rec := httptest.NewRecorder()

			h.renderAttachmentAuthError(context.Background(), rec, "file-id", tt.err)

			assert.Equal(t, tt.status, rec.Code)
			assert.Equal(t, tt.wantLog, bytes.Contains(logged.Bytes(), []byte("connection refused")))
		})
	}
}

func TestRenderAuthError(t *testing.T) {
	t.Parallel()

	baseURL, err := baseurl.Parse("https://example.com")
	require.NoError(t, err)

	tests := []struct {
		name    string
		err     error
		status  int
		wantLog bool
	}{
		{
			name:   "insufficient scope",
			err:    iam.NewInsufficientOAuth2ScopeError(gid.Nil, coredata.OAuth2Scope("documents:read")),
			status: http.StatusForbidden,
		},
		{
			name:   "insufficient permissions",
			err:    iam.NewInsufficientPermissionsError(gid.Nil, gid.Nil, "core:file:get"),
			status: http.StatusForbidden,
		},
		{
			name:   "missing resource",
			err:    coredata.ErrResourceNotFound,
			status: http.StatusNotFound,
		},
		{
			name:   "wrapped missing resource",
			err:    fmt.Errorf("cannot load resource attributes: %w", coredata.ErrResourceNotFound),
			status: http.StatusNotFound,
		},
		{
			name:   "assumption required",
			err:    iam.NewAssumptionRequiredError(gid.Nil, gid.Nil),
			status: http.StatusNotFound,
		},
		{
			name:    "database failure",
			err:     fmt.Errorf("cannot load memberships for principal: %w", errors.New("connection refused")),
			status:  http.StatusInternalServerError,
			wantLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logged bytes.Buffer

			h := &Handler{
				logger:  log.NewLogger(log.WithOutput(&logged)),
				baseURL: baseURL,
			}
			rec := httptest.NewRecorder()

			h.renderAuthError(context.Background(), rec, "file-id", tt.err)

			assert.Equal(t, tt.status, rec.Code)
			assert.Equal(t, tt.wantLog, bytes.Contains(logged.Bytes(), []byte("connection refused")))
		})
	}
}

func TestKeepAttachmentAuthError(t *testing.T) {
	t.Parallel()

	denial := iam.NewInsufficientPermissionsError(gid.Nil, gid.Nil, "core:task:get")
	outage := fmt.Errorf("cannot load memberships for principal: %w", errors.New("connection refused"))

	assert.Equal(t, denial, keepAttachmentAuthError(nil, denial))
	assert.Equal(t, outage, keepAttachmentAuthError(denial, outage))
	assert.Equal(t, outage, keepAttachmentAuthError(outage, denial))
	assert.Equal(t, denial, keepAttachmentAuthError(denial, iam.NewInsufficientOAuth2ScopeError(gid.Nil, coredata.OAuth2Scope("documents:read"))))
}

func TestAttachmentParents(t *testing.T) {
	t.Parallel()

	tenant := gid.NewTenantID()
	taskID := gid.New(tenant, coredata.TaskEntityType)
	unknownID := gid.New(tenant, coredata.OrganizationEntityType)
	svc := attachment.NewService(nil, attachment.ParentReadActions{Task: "core:task:get"})

	t.Run("known parent keeps its read action", func(t *testing.T) {
		t.Parallel()

		parents := attachmentParents(svc, coredata.Attachments{{ParentID: taskID}})
		require.Len(t, parents, 1)
		assert.Equal(t, taskID, parents[0].id)
		assert.Equal(t, "core:task:get", parents[0].action)
	})

	t.Run("unmapped parent has no read action", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, attachmentParents(svc, coredata.Attachments{{ParentID: unknownID}}))
	})

	t.Run("empty action has no read action", func(t *testing.T) {
		t.Parallel()

		empty := attachment.NewService(nil, attachment.ParentReadActions{})
		assert.Empty(t, attachmentParents(empty, coredata.Attachments{{ParentID: taskID}}))
	})
}

func TestHandleGetPublicFile_InvalidGID(t *testing.T) {
	t.Parallel()

	h := &Handler{logger: log.NewLogger(log.WithOutput(io.Discard))}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/public/not-a-valid-gid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("fileID", "not-a-valid-gid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.handleGetPublicFile(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleGetFile_InvalidGID(t *testing.T) {
	t.Parallel()

	h := &Handler{logger: log.NewLogger(log.WithOutput(io.Discard))}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/not-a-valid-gid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("fileID", "not-a-valid-gid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.handleGetFile(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleGetStaticFile(t *testing.T) {
	t.Parallel()

	mux := NewMux(
		log.NewLogger(log.WithOutput(io.Discard)),
		nil,
		nil,
		nil,
		nil,
		securecookie.Config{},
		"test-secret",
		baseurl.MustParse("https://example.com"),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/static/probo.png", nil)
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Cache-Control"), "max-age=3600")

	etag := rec.Header().Get("ETag")
	require.NotEmpty(t, etag)
	require.True(t, strings.HasPrefix(etag, `"`) && strings.HasSuffix(etag, `"`))

	recNotModified := httptest.NewRecorder()
	reqNotModified := httptest.NewRequest(http.MethodGet, "/static/probo.png", nil)
	reqNotModified.Header.Set("If-None-Match", etag)
	mux.ServeHTTP(recNotModified, reqNotModified)

	require.Equal(t, http.StatusNotModified, recNotModified.Code)

	recMissing := httptest.NewRecorder()
	reqMissing := httptest.NewRequest(http.MethodGet, "/static/does-not-exist.png", nil)
	mux.ServeHTTP(recMissing, reqMissing)

	require.Equal(t, http.StatusNotFound, recMissing.Code)
}

func TestHandleGetFile_UnauthenticatedReturns401(t *testing.T) {
	t.Parallel()

	mux := NewMux(
		log.NewLogger(log.WithOutput(io.Discard)),
		nil,
		nil,
		nil,
		nil,
		securecookie.Config{},
		"test-secret",
		baseurl.MustParse("https://example.com"),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/some-valid-looking-id", nil)
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAttachmentRoute_UnmappedParentReturns404(t *testing.T) {
	t.Parallel()

	tenant := gid.NewTenantID()
	fileID := gid.New(tenant, coredata.FileEntityType)
	parentID := gid.New(tenant, coredata.OrganizationEntityType)
	mux := attachmentMux(t, fileID, coredata.Attachments{{
		FileID:         fileID,
		OrganizationID: gid.New(tenant, coredata.OrganizationEntityType),
		ParentID:       parentID,
	}}, "")

	rec := httptest.NewRecorder()
	req := attachmentRequest(tenant, fileID)
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, rec.Header().Get("Location"))
}

func TestAttachmentRoute_AuthorizedParentRedirects(t *testing.T) {
	t.Parallel()

	tenant := gid.NewTenantID()
	fileID := gid.New(tenant, coredata.FileEntityType)
	parentID := gid.New(tenant, coredata.TaskEntityType)

	const downloadURL = "https://files.example/report.pdf"

	mux := attachmentMux(t, fileID, coredata.Attachments{{
		FileID:         fileID,
		OrganizationID: gid.New(tenant, coredata.OrganizationEntityType),
		ParentID:       parentID,
	}}, downloadURL)

	rec := httptest.NewRecorder()
	req := attachmentRequest(tenant, fileID)
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	assert.Equal(t, downloadURL, rec.Header().Get("Location"))
}

func attachmentMux(t *testing.T, fileID gid.GID, rows coredata.Attachments, downloadURL string) *chi.Mux {
	t.Helper()

	h := &Handler{
		logger: log.NewLogger(log.WithOutput(io.Discard)),
		files:  stubDelivery{url: downloadURL},
		stored: stubFiles{
			file: &coredata.File{
				ID:             fileID,
				OrganizationID: rows[0].OrganizationID,
				Visibility:     coredata.FileVisibilityAttachment,
			},
			rows: rows,
		},
		attachments: attachment.NewService(nil, attachment.ParentReadActions{Task: "core:task:get"}),
		authorizer:  stubAuthorizer{},
	}

	mux := chi.NewRouter()
	mux.Get("/attachments/{fileID}", h.handleGetAttachmentFile)

	return mux
}

func attachmentRequest(tenant gid.TenantID, fileID gid.GID) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/attachments/"+fileID.String(), nil)
	identity := &coredata.Identity{ID: gid.New(tenant, coredata.IdentityEntityType)}

	return req.WithContext(authn.ContextWithIdentity(req.Context(), identity))
}

type stubFiles struct {
	file *coredata.File
	rows coredata.Attachments
}

func (s stubFiles) Get(context.Context, coredata.Scoper, gid.GID) (*coredata.File, error) {
	return s.file, nil
}

func (s stubFiles) ListAttachments(context.Context, coredata.Scoper, gid.GID, gid.GID) (coredata.Attachments, error) {
	return s.rows, nil
}

func (s stubFiles) UploadedBy(context.Context, coredata.Scoper, *coredata.File, gid.GID) (bool, error) {
	return false, nil
}

type stubDelivery struct {
	url string
}

func (s stubDelivery) ServePublicFile(context.Context, http.ResponseWriter, *http.Request, gid.GID) error {
	return nil
}

func (s stubDelivery) GeneratePresignedURL(context.Context, *coredata.File, time.Duration) (string, error) {
	return s.url, nil
}

type stubAuthorizer struct{}

func (stubAuthorizer) Authorize(context.Context, iam.AuthorizeParams) (*coredata.Scope, error) {
	return nil, nil
}
