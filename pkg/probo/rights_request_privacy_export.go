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
	"slices"
	"time"

	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/page"
	"go.probo.inc/probo/pkg/xlsxexport"
)

type ExportRightsRequestRecordsRequest struct {
	OrganizationID gid.GID
	// ReceivedFrom and ReceivedTo bound the receipt dates, both included.
	// Either may be nil.
	ReceivedFrom *time.Time
	ReceivedTo   *time.Time
}

// ErrRightsRequestInvalidPeriod is a receipt period that ends before it
// starts.
type ErrRightsRequestInvalidPeriod struct{}

func (e *ErrRightsRequestInvalidPeriod) Error() string {
	return "the receipt period must not end before it starts"
}

// Export renders the record of requests from individuals (本人からの請求の
// 記録) received in the period as an .xlsx workbook, with a summary per
// kind.
func (s RightsRequestPrivacyService) Export(
	ctx context.Context,
	scope coredata.Scoper,
	req ExportRightsRequestRecordsRequest,
) ([]byte, error) {
	data, err := s.loadRecords(ctx, scope, req, func(coredata.RightsRequestPrivacyCategory) bool { return true })
	if err != nil {
		return nil, err
	}

	workbook, err := xlsxexport.BuildWorkbook(buildRightsRequestRecordSheets(data, time.Now().In(ledgerTimeZone))...)
	if err != nil {
		return nil, fmt.Errorf("cannot build rights request record workbook: %w", err)
	}

	return workbook, nil
}

// ExportComplaints renders the complaint and consultation record (苦情・相談の
// 記録) received in the period as an .xlsx workbook, with the counts per
// department.
func (s RightsRequestPrivacyService) ExportComplaints(
	ctx context.Context,
	scope coredata.Scoper,
	req ExportRightsRequestRecordsRequest,
) ([]byte, error) {
	data, err := s.loadRecords(ctx, scope, req, isComplaintOrConsultation)
	if err != nil {
		return nil, err
	}

	workbook, err := xlsxexport.BuildWorkbook(buildComplaintRecordSheets(data, time.Now().In(ledgerTimeZone))...)
	if err != nil {
		return nil, fmt.Errorf("cannot build complaint record workbook: %w", err)
	}

	return workbook, nil
}

// loadRecords checks the period and loads the requests received in it whose
// category passes include.
func (s RightsRequestPrivacyService) loadRecords(
	ctx context.Context,
	scope coredata.Scoper,
	req ExportRightsRequestRecordsRequest,
	include func(coredata.RightsRequestPrivacyCategory) bool,
) (rightsRequestRecordData, error) {
	if req.ReceivedFrom != nil {
		from := trainingDateOnly(*req.ReceivedFrom)
		req.ReceivedFrom = &from
	}

	if req.ReceivedTo != nil {
		to := trainingDateOnly(*req.ReceivedTo)
		req.ReceivedTo = &to
	}

	if req.ReceivedFrom != nil && req.ReceivedTo != nil && req.ReceivedTo.Before(*req.ReceivedFrom) {
		return rightsRequestRecordData{}, &ErrRightsRequestInvalidPeriod{}
	}

	var data rightsRequestRecordData

	err := s.svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			var err error
			data, err = loadRightsRequestRecordData(ctx, conn, scope, req, include)
			return err
		},
	)

	return data, err
}

func loadRightsRequestRecordData(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	req ExportRightsRequestRecordsRequest,
	include func(coredata.RightsRequestPrivacyCategory) bool,
) (rightsRequestRecordData, error) {
	organization := &coredata.Organization{}
	if err := organization.LoadByID(ctx, conn, scope, req.OrganizationID); err != nil {
		return rightsRequestRecordData{}, fmt.Errorf("cannot load organization: %w", err)
	}

	requests, err := page.LoadAll(
		ctx,
		page.OrderBy[coredata.RightsRequestOrderField]{
			Field:     coredata.RightsRequestOrderFieldCreatedAt,
			Direction: page.OrderDirectionAsc,
		},
		func(ctx context.Context, cursor *page.Cursor[coredata.RightsRequestOrderField]) ([]*coredata.RightsRequest, error) {
			var batch coredata.RightsRequests
			if err := batch.LoadByOrganizationID(ctx, conn, scope, req.OrganizationID, cursor); err != nil {
				return nil, fmt.Errorf("cannot load rights requests: %w", err)
			}

			return batch, nil
		},
	)
	if err != nil {
		return rightsRequestRecordData{}, err
	}

	var detailsList coredata.RightsRequestPrivacyDetailsList
	if err := detailsList.LoadByOrganizationID(ctx, conn, scope, req.OrganizationID); err != nil {
		return rightsRequestRecordData{}, fmt.Errorf("cannot load rights request privacy details: %w", err)
	}

	detailsByID := make(map[gid.GID]*coredata.RightsRequestPrivacyDetails, len(detailsList))
	for _, d := range detailsList {
		detailsByID[d.RightsRequestID] = d
	}

	records := make([]rightsRequestRecord, 0, len(requests))
	for _, r := range requests {
		details := detailsByID[r.ID]
		receivedOn := RightsRequestEffectiveReceivedOn(r, details)

		if (req.ReceivedFrom != nil && receivedOn.Before(*req.ReceivedFrom)) ||
			(req.ReceivedTo != nil && receivedOn.After(*req.ReceivedTo)) ||
			!include(RightsRequestEffectiveCategory(r, details)) {
			continue
		}

		records = append(records, rightsRequestRecord{Request: r, Details: details, ReceivedOn: receivedOn})
	}

	// By receipt date, then by registration for the same day.
	slices.SortStableFunc(records, func(a, b rightsRequestRecord) int {
		return a.ReceivedOn.Compare(b.ReceivedOn)
	})

	if err := fillRightsRequestProfileNames(ctx, conn, scope, records); err != nil {
		return rightsRequestRecordData{}, err
	}

	if err := fillRightsRequestFindingReferences(ctx, conn, scope, records); err != nil {
		return rightsRequestRecordData{}, err
	}

	return rightsRequestRecordData{
		OrganizationName: organization.Name,
		From:             req.ReceivedFrom,
		To:               req.ReceivedTo,
		Records:          records,
	}, nil
}

func fillRightsRequestProfileNames(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	records []rightsRequestRecord,
) error {
	ids := []gid.GID{}
	seen := map[gid.GID]bool{}
	for _, r := range records {
		if r.Details == nil {
			continue
		}

		for _, id := range []*gid.GID{r.Details.HandlerProfileID, r.Details.ApproverProfileID} {
			if id != nil && !seen[*id] {
				seen[*id] = true
				ids = append(ids, *id)
			}
		}
	}

	if len(ids) == 0 {
		return nil
	}

	var profiles coredata.MembershipProfiles
	if err := profiles.LoadByIDs(ctx, conn, scope, ids); err != nil && !errors.Is(err, coredata.ErrResourceNotFound) {
		return fmt.Errorf("cannot load handlers: %w", err)
	}

	names := make(map[gid.GID]string, len(profiles))
	for _, p := range profiles {
		names[p.ID] = p.FullName
	}

	for i := range records {
		d := records[i].Details
		if d == nil {
			continue
		}

		if d.HandlerProfileID != nil {
			records[i].HandlerName = names[*d.HandlerProfileID]
		}

		if d.ApproverProfileID != nil {
			records[i].ApproverName = names[*d.ApproverProfileID]
		}
	}

	return nil
}

// fillRightsRequestFindingReferences names the finding linked to each
// request by its reference ID. A finding deleted meanwhile is left blank.
func fillRightsRequestFindingReferences(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	records []rightsRequestRecord,
) error {
	references := map[gid.GID]string{}

	for i := range records {
		d := records[i].Details
		if d == nil || d.FindingID == nil {
			continue
		}

		reference, ok := references[*d.FindingID]
		if !ok {
			finding := &coredata.Finding{}
			err := finding.LoadByID(ctx, conn, scope, *d.FindingID)
			switch {
			case err == nil:
				reference = finding.ReferenceID
			case !errors.Is(err, coredata.ErrResourceNotFound):
				return fmt.Errorf("cannot load finding: %w", err)
			}
			references[*d.FindingID] = reference
		}

		records[i].FindingReference = reference
	}

	return nil
}
