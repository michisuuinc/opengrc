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
	"reflect"
	"slices"
	"strings"
	"time"

	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/page"
	"go.probo.inc/probo/pkg/validator"
	"go.probo.inc/probo/pkg/xlsxexport"
)

// RiskPrivacyService manages the Japanese PrivacyMark risk analysis
// (リスク分析表): the lifecycle stages a risk applies to, the register
// entries (processing activities) it was identified for, and the Excel
// export.
type RiskPrivacyService struct {
	svc *Service
}

// riskPrivacyMaxProcessingActivities bounds the processing activities one
// risk can be linked to in a single update.
const riskPrivacyMaxProcessingActivities = 1000

type UpdateRiskPrivacyDetailsRequest struct {
	RiskID                gid.GID
	LifecycleStages       *[]coredata.PersonalDataLifecycleStage
	ProcessingActivityIDs *[]gid.GID
}

func (r *UpdateRiskPrivacyDetailsRequest) Validate() error {
	v := validator.New()

	v.Check(r.RiskID, "risk_id", validator.Required(), validator.GID(coredata.RiskEntityType))
	v.Check(r.LifecycleStages, "lifecycle_stages", validator.NoDuplicates())
	v.CheckEach(r.LifecycleStages, "lifecycle_stages", func(index int, item any) {
		v.Check(item, fmt.Sprintf("lifecycle_stages[%d]", index), validator.Required(), validator.OneOfSlice(coredata.PersonalDataLifecycleStages()))
	})
	v.Check(r.ProcessingActivityIDs, "processing_activity_ids", validator.NoDuplicates(), maxItems(riskPrivacyMaxProcessingActivities))
	v.CheckEach(r.ProcessingActivityIDs, "processing_activity_ids", func(index int, item any) {
		v.Check(item, fmt.Sprintf("processing_activity_ids[%d]", index), validator.Required(), validator.GID(coredata.ProcessingActivityEntityType))
	})

	return v.Error()
}

// maxItems rejects a list with more than n items. Validator.Check passes
// the list already dereferenced, so the value is a slice, not a pointer.
func maxItems(n int) validator.ValidatorFunc {
	return func(value any) *validator.ValidationError {
		if value == nil {
			return nil
		}

		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Slice || rv.Len() <= n {
			return nil
		}

		return &validator.ValidationError{
			Code:    validator.ErrorCodeTooLong,
			Message: fmt.Sprintf("must not contain more than %d items", n),
		}
	}
}

// GetDetails returns the risk analysis items of a risk. A risk with no items
// yet gets an empty value, not an error.
func (s RiskPrivacyService) GetDetails(
	ctx context.Context,
	scope coredata.Scoper,
	riskID gid.GID,
) (*coredata.RiskPrivacyDetails, error) {
	details := &coredata.RiskPrivacyDetails{}

	err := s.svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			return details.LoadByRiskID(ctx, conn, scope, riskID)
		},
	)
	if errors.Is(err, coredata.ErrResourceNotFound) {
		return &coredata.RiskPrivacyDetails{
			RiskID:          riskID,
			LifecycleStages: []coredata.PersonalDataLifecycleStage{},
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("cannot load risk privacy details: %w", err)
	}

	return details, nil
}

// ListProcessingActivities returns the processing activities linked to the
// risk, by name.
func (s RiskPrivacyService) ListProcessingActivities(
	ctx context.Context,
	scope coredata.Scoper,
	riskID gid.GID,
) (coredata.ProcessingActivities, error) {
	var activities coredata.ProcessingActivities

	err := s.svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			var links coredata.RiskProcessingActivities
			if err := links.LoadByRiskID(ctx, conn, scope, riskID); err != nil {
				return fmt.Errorf("cannot load risk processing activities: %w", err)
			}

			ids := make([]gid.GID, len(links))
			for i, l := range links {
				ids[i] = l.ProcessingActivityID
			}

			if err := activities.LoadByIDs(ctx, conn, scope, ids); err != nil {
				return fmt.Errorf("cannot load processing activities: %w", err)
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(activities, func(a, b *coredata.ProcessingActivity) int {
		return strings.Compare(a.Name, b.Name)
	})

	return activities, nil
}

// UpdateDetails changes the items set in the request and keeps the others.
// The processing activities, when set, replace the linked ones.
func (s RiskPrivacyService) UpdateDetails(
	ctx context.Context,
	scope coredata.Scoper,
	req *UpdateRiskPrivacyDetailsRequest,
) error {
	if err := req.Validate(); err != nil {
		return err
	}

	return s.svc.pg.WithTx(
		ctx,
		func(ctx context.Context, tx pg.Tx) error {
			details := &coredata.RiskPrivacyDetails{}

			organizationID, err := details.LockRisk(ctx, tx, scope, req.RiskID)
			if err != nil {
				return fmt.Errorf("cannot lock risk: %w", err)
			}

			now := time.Now()

			if req.ProcessingActivityIDs != nil {
				var links coredata.RiskProcessingActivities
				if err := links.Replace(ctx, tx, scope, req.RiskID, organizationID, *req.ProcessingActivityIDs, now); err != nil {
					return fmt.Errorf("cannot replace risk processing activities: %w", err)
				}
			}

			if req.LifecycleStages == nil {
				return nil
			}

			err = details.LoadByRiskID(ctx, tx, scope, req.RiskID)
			switch {
			case errors.Is(err, coredata.ErrResourceNotFound):
				details = &coredata.RiskPrivacyDetails{
					RiskID:         req.RiskID,
					OrganizationID: organizationID,
					CreatedAt:      now,
				}
			case err != nil:
				return fmt.Errorf("cannot load risk privacy details: %w", err)
			}

			updated := *details
			updated.LifecycleStages = slices.Clone(*req.LifecycleStages)
			updated.UpdatedAt = now

			if err := updated.Upsert(ctx, tx, scope); err != nil {
				return fmt.Errorf("cannot upsert risk privacy details: %w", err)
			}

			return nil
		},
	)
}

// Export renders the organization's risk analysis as an .xlsx workbook.
func (s RiskPrivacyService) Export(
	ctx context.Context,
	scope coredata.Scoper,
	organizationID gid.GID,
) ([]byte, error) {
	var data riskPrivacyData

	err := s.svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			var err error
			data, err = loadRiskPrivacyData(ctx, conn, scope, organizationID)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	workbook, err := xlsxexport.Build(buildRiskPrivacySheet(data, time.Now().In(ledgerTimeZone)))
	if err != nil {
		return nil, fmt.Errorf("cannot build risk analysis workbook: %w", err)
	}

	return workbook, nil
}

func loadRiskPrivacyData(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	organizationID gid.GID,
) (riskPrivacyData, error) {
	organization := &coredata.Organization{}
	if err := organization.LoadByID(ctx, conn, scope, organizationID); err != nil {
		return riskPrivacyData{}, fmt.Errorf("cannot load organization: %w", err)
	}

	activities, err := loadAllProcessingActivities(ctx, conn, scope, organizationID)
	if err != nil {
		return riskPrivacyData{}, err
	}

	risks, err := loadAllRisks(ctx, conn, scope, organizationID)
	if err != nil {
		return riskPrivacyData{}, err
	}

	var links coredata.RiskProcessingActivities
	if err := links.LoadByOrganizationID(ctx, conn, scope, organizationID); err != nil {
		return riskPrivacyData{}, fmt.Errorf("cannot load risk processing activities: %w", err)
	}

	riskIDs := make(map[gid.GID][]gid.GID)
	for _, l := range links {
		riskIDs[l.ProcessingActivityID] = append(riskIDs[l.ProcessingActivityID], l.RiskID)
	}

	var detailsList coredata.RiskPrivacyDetailsList
	if err := detailsList.LoadByOrganizationID(ctx, conn, scope, organizationID); err != nil {
		return riskPrivacyData{}, fmt.Errorf("cannot load risk privacy details: %w", err)
	}

	stages := make(map[gid.GID][]coredata.PersonalDataLifecycleStage, len(detailsList))
	for _, d := range detailsList {
		stages[d.RiskID] = d.LifecycleStages
	}

	var internalControls coredata.InternalControls
	controlNames, err := internalControls.LoadNamesByRisk(ctx, conn, scope, organizationID)
	if err != nil {
		return riskPrivacyData{}, fmt.Errorf("cannot load internal control names: %w", err)
	}

	ownerNames, err := loadRiskOwnerNames(ctx, conn, scope, risks)
	if err != nil {
		return riskPrivacyData{}, err
	}

	riskMap := make(map[gid.GID]*coredata.Risk, len(risks))
	for _, r := range risks {
		riskMap[r.ID] = r
	}

	return riskPrivacyData{
		OrganizationName:     organization.Name,
		Activities:           activities,
		Risks:                riskMap,
		RiskIDs:              riskIDs,
		Stages:               stages,
		InternalControlNames: controlNames,
		OwnerNames:           ownerNames,
	}, nil
}

func loadAllProcessingActivities(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	organizationID gid.GID,
) ([]*coredata.ProcessingActivity, error) {
	return page.LoadAll(
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
}

func loadAllRisks(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	organizationID gid.GID,
) ([]*coredata.Risk, error) {
	return page.LoadAll(
		ctx,
		page.OrderBy[coredata.RiskOrderField]{
			Field:     coredata.RiskOrderFieldReferenceID,
			Direction: page.OrderDirectionAsc,
		},
		func(ctx context.Context, cursor *page.Cursor[coredata.RiskOrderField]) ([]*coredata.Risk, error) {
			var batch coredata.Risks
			if err := batch.LoadByOrganizationID(ctx, conn, scope, organizationID, cursor, coredata.NewRiskFilter(nil)); err != nil {
				return nil, fmt.Errorf("cannot load risks: %w", err)
			}

			return batch, nil
		},
	)
}

func loadRiskOwnerNames(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	risks []*coredata.Risk,
) (map[gid.GID]string, error) {
	ownerIDs := make([]gid.GID, 0, len(risks))
	seen := make(map[gid.GID]struct{}, len(risks))
	for _, r := range risks {
		if r.OwnerID == nil {
			continue
		}
		if _, ok := seen[*r.OwnerID]; !ok {
			seen[*r.OwnerID] = struct{}{}
			ownerIDs = append(ownerIDs, *r.OwnerID)
		}
	}

	names := make(map[gid.GID]string, len(ownerIDs))
	if len(ownerIDs) == 0 {
		return names, nil
	}

	var profiles coredata.MembershipProfiles
	if err := profiles.LoadByIDs(ctx, conn, scope, ownerIDs); err != nil && !errors.Is(err, coredata.ErrResourceNotFound) {
		return nil, fmt.Errorf("cannot load risk owners: %w", err)
	}

	for _, p := range profiles {
		names[p.ID] = p.FullName
	}

	return names, nil
}
