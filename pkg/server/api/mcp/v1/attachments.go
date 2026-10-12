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

package mcp_v1

import (
	"context"
	"errors"
	"fmt"

	"go.gearno.de/kit/log"
	"go.probo.inc/probo/pkg/attachment"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
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
			return fmt.Errorf("resource not found")
		}

		if _, ok := errors.AsType[*attachment.ErrInvalidFileID](err); ok {
			return fmt.Errorf("invalid attachment id")
		}

		r.logger.ErrorCtx(ctx, "cannot list new attachments", log.Error(err))

		return fmt.Errorf("internal server error")
	}

	if len(ids) == 0 {
		return nil
	}

	// The batch scope is this organization's tenant, the same tenant as scope.
	// A new attachment is allowed only for a file in this organization, and the
	// write that follows stays on the parent, so the parent scope is the one
	// passed down.
	if _, err := r.AuthorizeBatch(ctx, ids, attachment.ActionCreate); err != nil {
		return err
	}

	return nil
}
