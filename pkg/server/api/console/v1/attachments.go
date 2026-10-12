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

package console_v1

import (
	"context"
	"errors"

	"go.gearno.de/kit/log"
	"go.probo.inc/probo/pkg/attachment"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/server/gqlutils"
)

func (r *Resolver) authorizeAttachments(
	ctx context.Context,
	scope coredata.Scoper,
	organizationID gid.GID,
	content string,
	parentID gid.GID,
) error {
	ids, err := r.attachment.NewFileIDs(ctx, scope, organizationID, parentID, content)
	if err != nil {
		if errors.Is(err, coredata.ErrResourceNotFound) {
			return gqlutils.NotFound(ctx, err)
		}

		if _, ok := errors.AsType[*attachment.ErrInvalidFileID](err); ok {
			return gqlutils.Invalid(ctx, err)
		}

		r.logger.ErrorCtx(ctx, "cannot list new attachments", log.Error(err))

		return gqlutils.Internal(ctx)
	}

	if len(ids) == 0 {
		return nil
	}

	if _, err := r.batchAuthorize(ctx, attachment.ActionCreate, ids); err != nil {
		return err
	}

	return nil
}
