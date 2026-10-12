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
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.gearno.de/kit/log"
	"go.probo.inc/probo/pkg/attachment"
	"go.probo.inc/probo/pkg/baseurl"
	"go.probo.inc/probo/pkg/bearertoken"
	"go.probo.inc/probo/pkg/brand"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/filemanager"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/iam"
	"go.probo.inc/probo/pkg/probo"
	"go.probo.inc/probo/pkg/securecookie"
	"go.probo.inc/probo/pkg/server/api/authn"
	"go.probo.inc/probo/pkg/server/jsonx"
)

const presignedURLExpiry = 1 * time.Hour

type storedFiles interface {
	Get(ctx context.Context, scope coredata.Scoper, fileID gid.GID) (*coredata.File, error)
	ListAttachments(ctx context.Context, scope coredata.Scoper, organizationID gid.GID, fileID gid.GID) (coredata.Attachments, error)
	UploadedBy(ctx context.Context, scope coredata.Scoper, file *coredata.File, identityID gid.GID) (bool, error)
}

type fileDelivery interface {
	ServePublicFile(ctx context.Context, w http.ResponseWriter, r *http.Request, fileID gid.GID) error
	GeneratePresignedURL(ctx context.Context, file *coredata.File, expiresIn time.Duration) (string, error)
}

type accessAuthorizer interface {
	Authorize(ctx context.Context, params iam.AuthorizeParams) (*coredata.Scope, error)
}

type Handler struct {
	logger      *log.Logger
	files       fileDelivery
	stored      storedFiles
	attachments *attachment.Service
	authorizer  accessAuthorizer
	assets      *brand.Assets
	baseURL     *baseurl.BaseURL
}

func NewMux(
	logger *log.Logger,
	fileSvc *filemanager.Service,
	proboSvc *probo.Service,
	attachmentSvc *attachment.Service,
	iamSvc *iam.Service,
	cookieConfig securecookie.Config,
	tokenSecret string,
	baseURL *baseurl.BaseURL,
) *chi.Mux {
	var delivery fileDelivery
	if fileSvc != nil {
		delivery = fileSvc
	}

	var stored storedFiles
	if proboSvc != nil {
		stored = proboSvc.Files
	}

	var authorizer accessAuthorizer
	if iamSvc != nil {
		authorizer = iamSvc.Authorizer
	}

	h := &Handler{
		logger:      logger,
		files:       delivery,
		stored:      stored,
		attachments: attachmentSvc,
		authorizer:  authorizer,
		assets:      brand.NewAssets(),
		baseURL:     baseURL,
	}

	r := chi.NewRouter()

	r.Get("/static/{file}", h.handleGetStaticFile)
	r.Get("/public/{fileID}", h.handleGetPublicFile)

	r.Group(func(r chi.Router) {
		r.Use(authn.NewSessionMiddleware(iamSvc, cookieConfig))
		r.Use(authn.NewAPIKeyMiddleware(iamSvc, tokenSecret))
		r.Use(authn.NewOAuth2AccessTokenMiddleware(iamSvc))
		r.Use(authn.NewIdentityPresenceMiddleware(baseURL))
		r.Get("/attachments/{fileID}", h.handleGetAttachmentFile)
		r.Get("/{fileID}", h.handleGetFile)
	})

	return r
}

func (h *Handler) handleGetStaticFile(w http.ResponseWriter, r *http.Request) {
	file := chi.URLParam(r, "file")

	if _, statErr := h.assets.Stat(file); statErr != nil {
		jsonx.RenderNotFound(w, fmt.Errorf("file not found"))
		return
	}

	// ServeAssets honors the ETag header we set above for If-None-Match (and
	// If-Range), so it emits 304 Not Modified and handles range requests without
	// any extra conditional logic here.
	h.assets.ServeAssets(w, r, file)
}

func (h *Handler) handleGetPublicFile(w http.ResponseWriter, r *http.Request) {
	fileIDStr := chi.URLParam(r, "fileID")

	fileID, err := gid.ParseGID(fileIDStr)
	if err != nil {
		jsonx.RenderNotFound(w, fmt.Errorf("file not found"))
		return
	}

	err = h.files.ServePublicFile(r.Context(), w, r, fileID)
	if errors.Is(err, filemanager.ErrPublicFileNotFound) {
		jsonx.RenderNotFound(w, fmt.Errorf("file not found"))
		return
	}

	if err != nil {
		h.logger.ErrorCtx(
			r.Context(),
			"cannot serve public file",
			log.Error(err),
			log.String("file_id", fileIDStr),
		)
		jsonx.RenderInternalServerError(w)
	}
}

func (h *Handler) handleGetFile(w http.ResponseWriter, r *http.Request) {
	fileIDStr := chi.URLParam(r, "fileID")

	fileID, err := gid.ParseGID(fileIDStr)
	if err != nil {
		jsonx.RenderNotFound(w, fmt.Errorf("file not found"))
		return
	}

	ctx := r.Context()
	identity := authn.IdentityFromContext(ctx)
	session := authn.SessionFromContext(ctx)

	params := iam.AuthorizeParams{
		Principal:          identity.ID,
		Resource:           fileID,
		Action:             probo.ActionFileGet,
		ResourceAttributes: make(map[string]string),
	}
	if session != nil {
		params.Session = &session.ID
	}

	scope, err := h.authorizer.Authorize(ctx, params)
	if err != nil {
		h.renderAuthError(ctx, w, fileIDStr, err)

		return
	}

	f, err := h.stored.Get(ctx, scope, fileID)
	if err != nil {
		if errors.Is(err, coredata.ErrResourceNotFound) {
			jsonx.RenderNotFound(w, fmt.Errorf("file not found"))
			return
		}

		h.logger.ErrorCtx(ctx, "cannot get file", log.Error(err), log.String("file_id", fileIDStr))
		jsonx.RenderInternalServerError(w)

		return
	}

	if f.Visibility == coredata.FileVisibilityAttachment {
		jsonx.RenderNotFound(w, fmt.Errorf("file not found"))
		return
	}

	presignedURL, err := h.files.GeneratePresignedURL(ctx, f, presignedURLExpiry)
	if err != nil {
		h.logger.ErrorCtx(ctx, "cannot generate file URL", log.Error(err), log.String("file_id", fileIDStr))
		jsonx.RenderInternalServerError(w)

		return
	}

	http.Redirect(w, r, presignedURL, http.StatusTemporaryRedirect)
}

func (h *Handler) handleGetAttachmentFile(w http.ResponseWriter, r *http.Request) {
	fileIDStr := chi.URLParam(r, "fileID")

	fileID, err := gid.ParseGID(fileIDStr)
	if err != nil {
		jsonx.RenderNotFound(w, fmt.Errorf("file not found"))
		return
	}

	ctx := r.Context()
	lookup := coredata.NewScopeFromObjectID(fileID)

	f, err := h.stored.Get(ctx, lookup, fileID)
	if err != nil {
		if errors.Is(err, coredata.ErrResourceNotFound) {
			jsonx.RenderNotFound(w, fmt.Errorf("file not found"))
			return
		}

		h.logger.ErrorCtx(ctx, "cannot get attachment file", log.Error(err), log.String("file_id", fileIDStr))
		jsonx.RenderInternalServerError(w)

		return
	}

	if f.Visibility != coredata.FileVisibilityAttachment {
		jsonx.RenderNotFound(w, fmt.Errorf("file not found"))
		return
	}

	attachments, err := h.stored.ListAttachments(ctx, lookup, f.OrganizationID, fileID)
	if err != nil {
		h.logger.ErrorCtx(ctx, "cannot list attachments", log.Error(err), log.String("file_id", fileIDStr))
		jsonx.RenderInternalServerError(w)

		return
	}

	identity := authn.IdentityFromContext(ctx)
	if identity == nil {
		jsonx.RenderNotFound(w, fmt.Errorf("file not found"))
		return
	}

	// A file with no attachment row is still in the editor, before the first
	// save. Its uploader may read it after the same scope, assumption, and
	// membership checks as a file read. A row whose parent has no read action
	// is a refusal, so that path is only for a file that has no parent.
	if len(attachments) == 0 {
		if err := h.authorize(ctx, identity.ID, fileID, probo.ActionFileGet); err != nil {
			h.renderAttachmentAuthError(ctx, w, fileIDStr, err)

			return
		}

		uploaded, err := h.stored.UploadedBy(ctx, lookup, f, identity.ID)
		if err != nil {
			h.logger.ErrorCtx(ctx, "cannot check attachment uploader", log.Error(err), log.String("file_id", fileIDStr))
			jsonx.RenderInternalServerError(w)

			return
		}

		if !uploaded {
			jsonx.RenderNotFound(w, fmt.Errorf("file not found"))

			return
		}
	} else {
		parents := attachmentParents(h.attachments, attachments)
		if len(parents) == 0 {
			h.logger.ErrorCtx(
				ctx,
				"cannot authorize attachment parent",
				log.String("file_id", fileIDStr),
			)
			jsonx.RenderNotFound(w, fmt.Errorf("file not found"))

			return
		}

		var authErr error

		for _, parent := range parents {
			err := h.authorize(ctx, identity.ID, parent.id, parent.action)
			if err == nil {
				authErr = nil

				break
			}

			// A later refusal must not hide an earlier infrastructure error.
			authErr = keepAttachmentAuthError(authErr, err)
		}

		if authErr != nil {
			h.renderAttachmentAuthError(ctx, w, fileIDStr, authErr)

			return
		}
	}

	presignedURL, err := h.files.GeneratePresignedURL(ctx, f, presignedURLExpiry)
	if err != nil {
		h.logger.ErrorCtx(ctx, "cannot generate attachment URL", log.Error(err), log.String("file_id", fileIDStr))
		jsonx.RenderInternalServerError(w)

		return
	}

	http.Redirect(w, r, presignedURL, http.StatusTemporaryRedirect)
}

func (h *Handler) authorize(ctx context.Context, principal gid.GID, resource gid.GID, action string) error {
	session := authn.SessionFromContext(ctx)

	params := iam.AuthorizeParams{
		Principal:          principal,
		Resource:           resource,
		Action:             action,
		ResourceAttributes: make(map[string]string),
	}
	if session != nil {
		params.Session = &session.ID
	}

	_, err := h.authorizer.Authorize(ctx, params)

	return err
}

// renderAttachmentAuthError hides an authorization refusal as a missing file.
func (h *Handler) renderAttachmentAuthError(ctx context.Context, w http.ResponseWriter, fileID string, err error) {
	if attachmentAuthorizationDenied(err) {
		jsonx.RenderNotFound(w, fmt.Errorf("file not found"))

		return
	}

	h.logger.ErrorCtx(
		ctx,
		"cannot authorize file access",
		log.Error(err),
		log.String("file_id", fileID),
	)
	jsonx.RenderInternalServerError(w)
}

// keepAttachmentAuthError remembers the first infrastructure failure. A
// refusal is kept only until a non-denial error arrives.
func keepAttachmentAuthError(current, next error) error {
	if current == nil || (attachmentAuthorizationDenied(current) && !attachmentAuthorizationDenied(next)) {
		return next
	}

	return current
}

func attachmentAuthorizationDenied(err error) bool {
	if fileAuthorizationDenied(err) {
		return true
	}

	if _, ok := errors.AsType[*iam.ErrInsufficientPermissions](err); ok {
		return true
	}

	if _, ok := errors.AsType[*iam.ErrInsufficientOAuth2Scope](err); ok {
		return true
	}

	return false
}

func (h *Handler) renderAuthError(ctx context.Context, w http.ResponseWriter, fileID string, err error) {
	if scopeErr, ok := errors.AsType[*iam.ErrInsufficientOAuth2Scope](err); ok {
		bearertoken.SetBearerInsufficientScope(w, h.baseURL, scopeErr.Scopes...)
		jsonx.RenderForbidden(w)

		return
	}

	if _, ok := errors.AsType[*iam.ErrInsufficientPermissions](err); ok {
		jsonx.RenderForbidden(w)

		return
	}

	// Authorize reports a missing parent and other refusals through the same
	// error as a database failure. Only the refusals may look like a missing
	// file; anything else is an outage and must not be hidden as a 404.
	if fileAuthorizationDenied(err) {
		jsonx.RenderNotFound(w, fmt.Errorf("file not found"))

		return
	}

	h.logger.ErrorCtx(
		ctx,
		"cannot authorize file access",
		log.Error(err),
		log.String("file_id", fileID),
	)
	jsonx.RenderInternalServerError(w)
}

func fileAuthorizationDenied(err error) bool {
	if errors.Is(err, coredata.ErrResourceNotFound) {
		return true
	}

	if _, ok := errors.AsType[*iam.ErrAssumptionRequired](err); ok {
		return true
	}

	if _, ok := errors.AsType[*iam.ErrOrganizationNotFound](err); ok {
		return true
	}

	if _, ok := errors.AsType[*iam.ErrSessionNotFound](err); ok {
		return true
	}

	if _, ok := errors.AsType[*iam.ErrSessionExpired](err); ok {
		return true
	}

	return false
}

type attachmentParent struct {
	id     gid.GID
	action string
}

func attachmentParents(svc *attachment.Service, rows coredata.Attachments) []attachmentParent {
	parents := make([]attachmentParent, 0, len(rows))
	seen := make(map[gid.GID]struct{}, len(rows))

	for _, row := range rows {
		action, ok := svc.ParentReadAction(row.ParentID)
		if !ok {
			continue
		}

		if _, ok := seen[row.ParentID]; ok {
			continue
		}

		seen[row.ParentID] = struct{}{}
		parents = append(parents, attachmentParent{id: row.ParentID, action: action})
	}

	return parents
}
