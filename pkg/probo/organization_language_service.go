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

package probo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/validator"
)

// OrganizationLanguageService manages the organization's default email
// language, used for recipients with no known language.
type OrganizationLanguageService struct {
	svc *Service
}

type UpdateOrganizationEmailLanguageRequest struct {
	OrganizationID gid.GID
	Language       string
}

func (r *UpdateOrganizationEmailLanguageRequest) Validate() error {
	v := validator.New()

	v.Check(r.OrganizationID, "organization_id", validator.Required(), validator.GID(coredata.OrganizationEntityType))
	v.Check(r.Language, "language", validator.Required(), validator.OneOfSlice(coredata.OrganizationLanguages()))

	return v.Error()
}

// GetEmailLanguage returns the organization's default email language, or
// nil when none was chosen (emails are then in English).
func (s OrganizationLanguageService) GetEmailLanguage(
	ctx context.Context,
	scope coredata.Scoper,
	organizationID gid.GID,
) (*string, error) {
	settings := &coredata.OrganizationLanguageSettings{}

	err := s.svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			return settings.LoadByOrganizationID(ctx, conn, scope, organizationID)
		},
	)
	if errors.Is(err, coredata.ErrResourceNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("cannot load organization language settings: %w", err)
	}

	return &settings.DefaultLanguage, nil
}

// UpdateEmailLanguage sets the organization's default email language.
func (s OrganizationLanguageService) UpdateEmailLanguage(
	ctx context.Context,
	scope coredata.Scoper,
	req *UpdateOrganizationEmailLanguageRequest,
) error {
	if err := req.Validate(); err != nil {
		return err
	}

	return s.svc.pg.WithTx(
		ctx,
		func(ctx context.Context, tx pg.Tx) error {
			organization := &coredata.Organization{}
			if err := organization.LoadByID(ctx, tx, scope, req.OrganizationID); err != nil {
				return fmt.Errorf("cannot load organization: %w", err)
			}

			now := time.Now()
			settings := &coredata.OrganizationLanguageSettings{
				OrganizationID:  organization.ID,
				DefaultLanguage: req.Language,
				CreatedAt:       now,
				UpdatedAt:       now,
			}

			if err := settings.Upsert(ctx, tx, scope); err != nil {
				return fmt.Errorf("cannot upsert organization language settings: %w", err)
			}

			return nil
		},
	)
}
