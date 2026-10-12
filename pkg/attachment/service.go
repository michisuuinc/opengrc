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
	"encoding/json"
	"errors"
	"fmt"

	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
)

// Service reads attachment rows.
type Service struct {
	pg                *pg.Client
	parentReadActions ParentReadActions
}

func NewService(pgClient *pg.Client, actions ParentReadActions) *Service {
	return &Service{pg: pgClient, parentReadActions: actions}
}

// ParentReadAction returns the get action for parentID.
func (s *Service) ParentReadAction(parentID gid.GID) (string, bool) {
	return parentReadAction(parentID, s.parentReadActions)
}

// NewFileIDs returns file ids in content that have no attachment on parentID.
func (s *Service) NewFileIDs(
	ctx context.Context,
	scope coredata.Scoper,
	organizationID gid.GID,
	parentID gid.GID,
	content string,
) ([]gid.GID, error) {
	ids, err := FileIDs(content)
	if err != nil {
		if contentIsNotJSON(err) {
			return nil, nil
		}

		if invalid, ok := errors.AsType[*ErrInvalidFileID](err); ok {
			return nil, invalid
		}

		return nil, fmt.Errorf("cannot read attachment file ids: %w", err)
	}

	if len(ids) == 0 || parentID == gid.Nil {
		return ids, nil
	}

	var rows coredata.Attachments

	err = s.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			if err := rows.LoadByParentFileIDs(ctx, conn, scope, organizationID, parentID, ids); err != nil {
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("cannot load attachments: %w", err)
	}

	return withoutAttached(ids, rows), nil
}

// UnattachedFileIDs returns file ids in content that have no attachment row
// for parentID. The read uses conn, so it sees the caller's transaction.
func UnattachedFileIDs(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	organizationID gid.GID,
	parentID gid.GID,
	content string,
) ([]gid.GID, error) {
	ids, err := FileIDs(content)
	if err != nil {
		if contentIsNotJSON(err) {
			return nil, nil
		}

		if invalid, ok := errors.AsType[*ErrInvalidFileID](err); ok {
			return nil, invalid
		}

		return nil, fmt.Errorf("cannot read attachment file ids: %w", err)
	}

	if len(ids) == 0 || parentID == gid.Nil {
		return ids, nil
	}

	var rows coredata.Attachments
	if err := rows.LoadByParentFileIDs(ctx, conn, scope, organizationID, parentID, ids); err != nil {
		return nil, fmt.Errorf("cannot load attachments: %w", err)
	}

	return withoutAttached(ids, rows), nil
}

func withoutAttached(ids []gid.GID, rows coredata.Attachments) []gid.GID {
	attached := make(map[gid.GID]struct{}, len(rows))
	for _, row := range rows {
		attached[row.FileID] = struct{}{}
	}

	fresh := make([]gid.GID, 0, len(ids))
	for _, id := range ids {
		if _, ok := attached[id]; ok {
			continue
		}

		fresh = append(fresh, id)
	}

	return fresh
}

func contentIsNotJSON(err error) bool {
	if _, ok := errors.AsType[*json.SyntaxError](err); ok {
		return true
	}

	_, ok := errors.AsType[*json.UnmarshalTypeError](err)

	return ok
}
