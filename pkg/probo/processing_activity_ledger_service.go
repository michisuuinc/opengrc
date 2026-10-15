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
	"go.probo.inc/probo/pkg/page"
	"go.probo.inc/probo/pkg/validator"
	"go.probo.inc/probo/pkg/xlsxexport"
)

// ProcessingActivityLedgerService manages the Japanese PrivacyMark register
// (個人情報管理台帳): the items the processing activity lacks, and the
// register's Excel export.
type ProcessingActivityLedgerService struct {
	svc *Service
}

// ledgerTimeZone dates the register in Japan Standard Time, the time zone of
// the PrivacyMark audit, whatever the server's zone is.
var ledgerTimeZone = time.FixedZone("JST", 9*60*60)

type UpdateProcessingActivityLedgerDetailsRequest struct {
	ProcessingActivityID   gid.GID
	Department             **string
	ApproximateRecordCount **string
	CollectionMethod       **string
	StorageLocation        **string
	AccessScope            **string
	DisposalMethod         **string
}

func (r *UpdateProcessingActivityLedgerDetailsRequest) Validate() error {
	v := validator.New()

	v.Check(r.ProcessingActivityID, "processing_activity_id", validator.Required(), validator.GID(coredata.ProcessingActivityEntityType))
	v.Check(r.Department, "department", validator.SafeText(TitleMaxLength))
	v.Check(r.ApproximateRecordCount, "approximate_record_count", validator.SafeText(TitleMaxLength))
	v.Check(r.CollectionMethod, "collection_method", validator.SafeText(TitleMaxLength))
	v.Check(r.StorageLocation, "storage_location", validator.SafeText(TitleMaxLength))
	v.Check(r.AccessScope, "access_scope", validator.SafeText(TitleMaxLength))
	v.Check(r.DisposalMethod, "disposal_method", validator.SafeText(TitleMaxLength))

	return v.Error()
}

// GetDetails returns the register items of a processing activity. An
// activity with no items yet gets an empty value, not an error.
func (s ProcessingActivityLedgerService) GetDetails(
	ctx context.Context,
	scope coredata.Scoper,
	processingActivityID gid.GID,
) (*coredata.ProcessingActivityLedgerDetails, error) {
	details := &coredata.ProcessingActivityLedgerDetails{}

	err := s.svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			return details.LoadByProcessingActivityID(ctx, conn, scope, processingActivityID)
		},
	)
	if errors.Is(err, coredata.ErrResourceNotFound) {
		return &coredata.ProcessingActivityLedgerDetails{ProcessingActivityID: processingActivityID}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("cannot load processing activity ledger details: %w", err)
	}

	return details, nil
}

// UpdateDetails changes the items set in the request and keeps the others.
func (s ProcessingActivityLedgerService) UpdateDetails(
	ctx context.Context,
	scope coredata.Scoper,
	req *UpdateProcessingActivityLedgerDetailsRequest,
) (*coredata.ProcessingActivityLedgerDetails, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	details := &coredata.ProcessingActivityLedgerDetails{}

	err := s.svc.pg.WithTx(
		ctx,
		func(ctx context.Context, tx pg.Tx) error {
			organizationID, err := details.LockProcessingActivity(ctx, tx, scope, req.ProcessingActivityID)
			if err != nil {
				return fmt.Errorf("cannot lock processing activity: %w", err)
			}

			now := time.Now()
			err = details.LoadByProcessingActivityID(ctx, tx, scope, req.ProcessingActivityID)
			switch {
			case errors.Is(err, coredata.ErrResourceNotFound):
				details = &coredata.ProcessingActivityLedgerDetails{
					ProcessingActivityID: req.ProcessingActivityID,
					OrganizationID:       organizationID,
					CreatedAt:            now,
				}
			case err != nil:
				return fmt.Errorf("cannot load processing activity ledger details: %w", err)
			}

			details = applyLedgerDetailsUpdate(*details, req)
			details.UpdatedAt = now

			if err := details.Upsert(ctx, tx, scope); err != nil {
				return fmt.Errorf("cannot upsert processing activity ledger details: %w", err)
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return details, nil
}

func applyLedgerDetailsUpdate(
	details coredata.ProcessingActivityLedgerDetails,
	req *UpdateProcessingActivityLedgerDetailsRequest,
) *coredata.ProcessingActivityLedgerDetails {
	fields := []struct {
		dst **string
		src **string
	}{
		{&details.Department, req.Department},
		{&details.ApproximateRecordCount, req.ApproximateRecordCount},
		{&details.CollectionMethod, req.CollectionMethod},
		{&details.StorageLocation, req.StorageLocation},
		{&details.AccessScope, req.AccessScope},
		{&details.DisposalMethod, req.DisposalMethod},
	}

	for _, f := range fields {
		if f.src != nil {
			*f.dst = *f.src
		}
	}

	return &details
}

// Export renders the organization's register as an .xlsx workbook.
func (s ProcessingActivityLedgerService) Export(
	ctx context.Context,
	scope coredata.Scoper,
	organizationID gid.GID,
) ([]byte, error) {
	var data ledgerData

	err := s.svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			var err error
			data, err = loadLedgerData(ctx, conn, scope, organizationID)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	workbook, err := xlsxexport.Build(buildLedgerSheet(data, time.Now().In(ledgerTimeZone)))
	if err != nil {
		return nil, fmt.Errorf("cannot build processing activity ledger workbook: %w", err)
	}

	return workbook, nil
}

func loadLedgerData(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	organizationID gid.GID,
) (ledgerData, error) {
	organization := &coredata.Organization{}
	if err := organization.LoadByID(ctx, conn, scope, organizationID); err != nil {
		return ledgerData{}, fmt.Errorf("cannot load organization: %w", err)
	}

	activities, err := page.LoadAll(
		ctx,
		page.OrderBy[coredata.ProcessingActivityOrderField]{
			Field:     coredata.ProcessingActivityOrderFieldName,
			Direction: page.OrderDirectionAsc,
		},
		func(ctx context.Context, cursor *page.Cursor[coredata.ProcessingActivityOrderField]) ([]*coredata.ProcessingActivity, error) {
			var batch coredata.ProcessingActivities
			if err := batch.LoadByOrganizationID(ctx, conn, scope, organizationID, cursor); err != nil {
				return nil, fmt.Errorf("cannot load processing activities: %w", err)
			}

			return batch, nil
		},
	)
	if err != nil {
		return ledgerData{}, err
	}

	ids := make([]gid.GID, len(activities))
	for i, a := range activities {
		ids[i] = a.ID
	}

	var detailsList coredata.ProcessingActivityLedgerDetailsList
	if err := detailsList.LoadByProcessingActivityIDs(ctx, conn, scope, ids); err != nil {
		return ledgerData{}, fmt.Errorf("cannot load processing activity ledger details: %w", err)
	}

	details := make(map[gid.GID]*coredata.ProcessingActivityLedgerDetails, len(detailsList))
	for _, d := range detailsList {
		details[d.ProcessingActivityID] = d
	}

	var thirdParties coredata.ThirdParties
	thirdPartyNames, err := thirdParties.LoadAllByProcessingActivities(ctx, conn, scope, organizationID)
	if err != nil {
		return ledgerData{}, fmt.Errorf("cannot load third parties: %w", err)
	}

	return ledgerData{
		OrganizationName: organization.Name,
		Activities:       activities,
		Details:          details,
		ThirdPartyNames:  thirdPartyNames,
	}, nil
}
