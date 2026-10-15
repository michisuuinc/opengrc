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
	"strings"
	"time"

	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/page"
	"go.probo.inc/probo/pkg/validator"
	"go.probo.inc/probo/pkg/xlsxexport"
)

// AuditPrivacyService manages the Japanese PrivacyMark internal audit report
// (内部監査報告書): the scope, criteria, auditors, audited departments and
// overall opinion of an audit, the category and basis of its findings, and
// the Excel export.
type AuditPrivacyService struct {
	svc *Service
}

const (
	auditPrivacyMaxAuditors         = 20
	auditPrivacyMaxExternalAuditors = 20
	auditPrivacyMaxDepartments      = 50
)

type (
	UpdateAuditPrivacyDetailsRequest struct {
		AuditID            gid.GID
		Scope              string
		Criteria           string
		AuditeeDepartments []string
		ExternalAuditors   []string
		OverallOpinion     string
		AuditorIDs         []gid.GID
	}

	UpdateFindingPrivacyDetailsRequest struct {
		FindingID gid.GID
		Category  *coredata.FindingPrivacyCategory
		Basis     string
	}

	// AuditIndependenceConflict is an internal auditor who belongs to one of
	// the audited departments. The internal audit procedure forbids
	// auditing one's own department.
	AuditIndependenceConflict struct {
		Auditor    *coredata.MembershipProfile
		Department string
	}
)

func (r *UpdateAuditPrivacyDetailsRequest) Validate() error {
	v := validator.New()

	v.Check(r.AuditID, "audit_id", validator.Required(), validator.GID(coredata.AuditEntityType))
	checkOptionalTrainingText(v, r.Scope, "scope", validator.SafeText(ContentMaxLength))
	checkOptionalTrainingText(v, r.Criteria, "criteria", validator.SafeText(TitleMaxLength))
	checkOptionalTrainingText(v, r.OverallOpinion, "overall_opinion", validator.SafeText(ContentMaxLength))
	checkNameList(v, r.AuditeeDepartments, "auditee_departments", auditPrivacyMaxDepartments)
	checkNameList(v, r.ExternalAuditors, "external_auditors", auditPrivacyMaxExternalAuditors)
	v.Check(&r.AuditorIDs, "auditor_ids", validator.NoDuplicates(), maxItems(auditPrivacyMaxAuditors))
	v.CheckEach(r.AuditorIDs, "auditor_ids", func(index int, item any) {
		v.Check(item, fmt.Sprintf("auditor_ids[%d]", index), validator.Required(), validator.GID(coredata.MembershipProfileEntityType))
	})

	return v.Error()
}

func (r *UpdateFindingPrivacyDetailsRequest) Validate() error {
	v := validator.New()

	v.Check(r.FindingID, "finding_id", validator.Required(), validator.GID(coredata.FindingEntityType))
	v.Check(r.Category, "category", validator.OneOfSlice(coredata.FindingPrivacyCategories()))
	checkOptionalTrainingText(v, r.Basis, "basis", validator.SafeTextNoNewLine(TitleMaxLength))

	return v.Error()
}

// checkNameList validates a list of short names: none empty, none repeated
// and at most limit of them.
func checkNameList(v *validator.Validator, names []string, field string, limit int) {
	v.Check(&names, field, validator.NoDuplicates(), maxItems(limit))
	v.CheckEach(names, field, func(index int, item any) {
		v.Check(item, fmt.Sprintf("%s[%d]", field, index), validator.Required(), validator.SafeTextNoNewLine(NameMaxLength))
	})
}

// trimNames trims each name. Validation then rejects names left empty.
func trimNames(names []string) []string {
	trimmed := make([]string, len(names))
	for i, name := range names {
		trimmed[i] = strings.TrimSpace(name)
	}

	return trimmed
}

// GetDetails returns the report items of an audit. An audit with no items
// yet gets an empty value, not an error.
func (s AuditPrivacyService) GetDetails(
	ctx context.Context,
	scope coredata.Scoper,
	auditID gid.GID,
) (*coredata.AuditPrivacyDetails, error) {
	details := &coredata.AuditPrivacyDetails{}

	err := s.svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			return details.LoadByAuditID(ctx, conn, scope, auditID)
		},
	)
	if errors.Is(err, coredata.ErrResourceNotFound) {
		return &coredata.AuditPrivacyDetails{
			AuditID:            auditID,
			AuditeeDepartments: []string{},
			ExternalAuditors:   []string{},
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("cannot load audit privacy details: %w", err)
	}

	return details, nil
}

// ListAuditors returns the internal auditors of the audit in the order they
// were set.
func (s AuditPrivacyService) ListAuditors(
	ctx context.Context,
	scope coredata.Scoper,
	auditID gid.GID,
) (coredata.MembershipProfiles, error) {
	var profiles coredata.MembershipProfiles

	err := s.svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			var err error
			profiles, err = loadAuditors(ctx, conn, scope, auditID)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	return profiles, nil
}

func loadAuditors(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	auditID gid.GID,
) (coredata.MembershipProfiles, error) {
	var links coredata.AuditPrivacyAuditors
	if err := links.LoadByAuditID(ctx, conn, scope, auditID); err != nil {
		return nil, fmt.Errorf("cannot load audit privacy auditors: %w", err)
	}

	if len(links) == 0 {
		return coredata.MembershipProfiles{}, nil
	}

	ids := make([]gid.GID, len(links))
	for i, l := range links {
		ids[i] = l.ProfileID
	}

	var profiles coredata.MembershipProfiles
	if err := profiles.LoadByIDs(ctx, conn, scope, ids); err != nil && !errors.Is(err, coredata.ErrResourceNotFound) {
		return nil, fmt.Errorf("cannot load auditors: %w", err)
	}

	byID := make(map[gid.GID]*coredata.MembershipProfile, len(profiles))
	for _, p := range profiles {
		byID[p.ID] = p
	}

	ordered := make(coredata.MembershipProfiles, 0, len(ids))
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			ordered = append(ordered, p)
		}
	}

	return ordered, nil
}

// IndependenceConflicts returns the internal auditors whose department is
// one of the audited departments. Department names are compared without
// surrounding spaces and case.
func (s AuditPrivacyService) IndependenceConflicts(
	ctx context.Context,
	scope coredata.Scoper,
	auditID gid.GID,
) ([]AuditIndependenceConflict, error) {
	details, err := s.GetDetails(ctx, scope, auditID)
	if err != nil {
		return nil, err
	}

	if len(details.AuditeeDepartments) == 0 {
		return []AuditIndependenceConflict{}, nil
	}

	auditors, err := s.ListAuditors(ctx, scope, auditID)
	if err != nil {
		return nil, err
	}

	return findIndependenceConflicts(auditors, details.AuditeeDepartments), nil
}

func findIndependenceConflicts(
	auditors coredata.MembershipProfiles,
	departments []string,
) []AuditIndependenceConflict {
	normalized := make(map[string]string, len(departments))
	for _, d := range departments {
		normalized[normalizeDepartment(d)] = d
	}

	conflicts := []AuditIndependenceConflict{}
	for _, auditor := range auditors {
		if auditor.Department == nil {
			continue
		}

		if d, ok := normalized[normalizeDepartment(*auditor.Department)]; ok && d != "" {
			conflicts = append(conflicts, AuditIndependenceConflict{Auditor: auditor, Department: d})
		}
	}

	return conflicts
}

func normalizeDepartment(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// UpdateDetails replaces the report items and the internal auditors of the
// audit.
func (s AuditPrivacyService) UpdateDetails(
	ctx context.Context,
	scope coredata.Scoper,
	req *UpdateAuditPrivacyDetailsRequest,
) error {
	normalized := *req
	normalized.Scope = strings.TrimSpace(req.Scope)
	normalized.Criteria = strings.TrimSpace(req.Criteria)
	normalized.OverallOpinion = strings.TrimSpace(req.OverallOpinion)
	normalized.AuditeeDepartments = trimNames(req.AuditeeDepartments)
	normalized.ExternalAuditors = trimNames(req.ExternalAuditors)
	normalized.AuditorIDs = slices.Clone(req.AuditorIDs)

	if err := normalized.Validate(); err != nil {
		return err
	}

	return s.svc.pg.WithTx(
		ctx,
		func(ctx context.Context, tx pg.Tx) error {
			details := &coredata.AuditPrivacyDetails{}

			organizationID, err := details.LockAudit(ctx, tx, scope, normalized.AuditID)
			if err != nil {
				return fmt.Errorf("cannot lock audit: %w", err)
			}

			now := time.Now()

			var auditors coredata.AuditPrivacyAuditors
			if err := auditors.Replace(ctx, tx, scope, normalized.AuditID, organizationID, normalized.AuditorIDs, now); err != nil {
				return fmt.Errorf("cannot replace audit privacy auditors: %w", err)
			}

			createdAt := now
			err = details.LoadByAuditID(ctx, tx, scope, normalized.AuditID)
			switch {
			case err == nil:
				createdAt = details.CreatedAt
			case !errors.Is(err, coredata.ErrResourceNotFound):
				return fmt.Errorf("cannot load audit privacy details: %w", err)
			}

			updated := &coredata.AuditPrivacyDetails{
				AuditID:            normalized.AuditID,
				OrganizationID:     organizationID,
				Scope:              normalized.Scope,
				Criteria:           normalized.Criteria,
				AuditeeDepartments: normalized.AuditeeDepartments,
				ExternalAuditors:   normalized.ExternalAuditors,
				OverallOpinion:     normalized.OverallOpinion,
				CreatedAt:          createdAt,
				UpdatedAt:          now,
			}

			if err := updated.Upsert(ctx, tx, scope); err != nil {
				return fmt.Errorf("cannot upsert audit privacy details: %w", err)
			}

			return nil
		},
	)
}

// GetFindingDetails returns the internal audit items of a finding. A
// finding with no items yet gets an empty value, not an error.
func (s AuditPrivacyService) GetFindingDetails(
	ctx context.Context,
	scope coredata.Scoper,
	findingID gid.GID,
) (*coredata.FindingPrivacyDetails, error) {
	details := &coredata.FindingPrivacyDetails{}

	err := s.svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			return details.LoadByFindingID(ctx, conn, scope, findingID)
		},
	)
	if errors.Is(err, coredata.ErrResourceNotFound) {
		return &coredata.FindingPrivacyDetails{FindingID: findingID}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("cannot load finding privacy details: %w", err)
	}

	return details, nil
}

// UpdateFindingDetails replaces the internal audit items of the finding.
func (s AuditPrivacyService) UpdateFindingDetails(
	ctx context.Context,
	scope coredata.Scoper,
	request *UpdateFindingPrivacyDetailsRequest,
) error {
	normalized := *request
	normalized.Basis = strings.TrimSpace(request.Basis)
	req := &normalized

	if err := req.Validate(); err != nil {
		return err
	}

	return s.svc.pg.WithTx(
		ctx,
		func(ctx context.Context, tx pg.Tx) error {
			details := &coredata.FindingPrivacyDetails{}

			organizationID, err := details.LockFinding(ctx, tx, scope, req.FindingID)
			if err != nil {
				return fmt.Errorf("cannot lock finding: %w", err)
			}

			now := time.Now()

			createdAt := now
			err = details.LoadByFindingID(ctx, tx, scope, req.FindingID)
			switch {
			case err == nil:
				createdAt = details.CreatedAt
			case !errors.Is(err, coredata.ErrResourceNotFound):
				return fmt.Errorf("cannot load finding privacy details: %w", err)
			}

			updated := &coredata.FindingPrivacyDetails{
				FindingID:      req.FindingID,
				OrganizationID: organizationID,
				Category:       req.Category,
				Basis:          req.Basis,
				CreatedAt:      createdAt,
				UpdatedAt:      now,
			}

			if err := updated.Upsert(ctx, tx, scope); err != nil {
				return fmt.Errorf("cannot upsert finding privacy details: %w", err)
			}

			return nil
		},
	)
}

// Export renders the internal audit report of the audit as an .xlsx
// workbook.
func (s AuditPrivacyService) Export(
	ctx context.Context,
	scope coredata.Scoper,
	auditID gid.GID,
) ([]byte, error) {
	var data auditReportData

	err := s.svc.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			var err error
			data, err = loadAuditReportData(ctx, conn, scope, auditID)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	workbook, err := xlsxexport.BuildWorkbook(buildAuditReportSheets(data, time.Now().In(ledgerTimeZone))...)
	if err != nil {
		return nil, fmt.Errorf("cannot build audit report workbook: %w", err)
	}

	return workbook, nil
}

func loadAuditReportData(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	auditID gid.GID,
) (auditReportData, error) {
	audit := &coredata.Audit{}
	if err := audit.LoadByID(ctx, conn, scope, auditID); err != nil {
		return auditReportData{}, fmt.Errorf("cannot load audit: %w", err)
	}

	organization := &coredata.Organization{}
	if err := organization.LoadByID(ctx, conn, scope, audit.OrganizationID); err != nil {
		return auditReportData{}, fmt.Errorf("cannot load organization: %w", err)
	}

	framework := &coredata.Framework{}
	if err := framework.LoadByID(ctx, conn, scope, audit.FrameworkID); err != nil {
		return auditReportData{}, fmt.Errorf("cannot load framework: %w", err)
	}

	details := &coredata.AuditPrivacyDetails{}
	if err := details.LoadByAuditID(ctx, conn, scope, auditID); err != nil {
		if !errors.Is(err, coredata.ErrResourceNotFound) {
			return auditReportData{}, fmt.Errorf("cannot load audit privacy details: %w", err)
		}
		details = nil
	}

	auditors, err := loadAuditors(ctx, conn, scope, auditID)
	if err != nil {
		return auditReportData{}, err
	}

	auditorNames := make([]string, len(auditors))
	for i, a := range auditors {
		auditorNames[i] = a.FullName
	}

	findings, err := loadAuditReportFindings(ctx, conn, scope, auditID)
	if err != nil {
		return auditReportData{}, err
	}

	return auditReportData{
		OrganizationName: organization.Name,
		Audit:            audit,
		FrameworkName:    framework.Name,
		Details:          details,
		AuditorNames:     auditorNames,
		Findings:         findings,
	}, nil
}

func loadAuditReportFindings(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	auditID gid.GID,
) ([]auditReportFinding, error) {
	findings, err := page.LoadAll(
		ctx,
		page.OrderBy[coredata.FindingOrderField]{
			Field:     coredata.FindingOrderFieldReferenceId,
			Direction: page.OrderDirectionAsc,
		},
		func(ctx context.Context, cursor *page.Cursor[coredata.FindingOrderField]) ([]*coredata.Finding, error) {
			var batch coredata.Findings
			if err := batch.LoadByAuditID(ctx, conn, scope, auditID, cursor, coredata.NewFindingFilter(nil, nil, nil, nil, nil)); err != nil {
				return nil, fmt.Errorf("cannot load findings: %w", err)
			}

			return batch, nil
		},
	)
	if err != nil {
		return nil, err
	}

	var detailsList coredata.FindingPrivacyDetailsList
	if err := detailsList.LoadByAuditID(ctx, conn, scope, auditID); err != nil {
		return nil, fmt.Errorf("cannot load finding privacy details: %w", err)
	}

	detailsByID := make(map[gid.GID]*coredata.FindingPrivacyDetails, len(detailsList))
	for _, d := range detailsList {
		detailsByID[d.FindingID] = d
	}

	ownerNames, err := loadFindingOwnerNames(ctx, conn, scope, findings)
	if err != nil {
		return nil, err
	}

	result := make([]auditReportFinding, len(findings))
	for i, f := range findings {
		result[i] = auditReportFinding{Finding: f, Details: detailsByID[f.ID]}
		if f.OwnerID != nil {
			result[i].OwnerName = ownerNames[*f.OwnerID]
		}
	}

	return result, nil
}

func loadFindingOwnerNames(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	findings []*coredata.Finding,
) (map[gid.GID]string, error) {
	ownerIDs := make([]gid.GID, 0, len(findings))
	seen := make(map[gid.GID]struct{}, len(findings))
	for _, f := range findings {
		if f.OwnerID == nil {
			continue
		}
		if _, ok := seen[*f.OwnerID]; !ok {
			seen[*f.OwnerID] = struct{}{}
			ownerIDs = append(ownerIDs, *f.OwnerID)
		}
	}

	names := make(map[gid.GID]string, len(ownerIDs))
	if len(ownerIDs) == 0 {
		return names, nil
	}

	var profiles coredata.MembershipProfiles
	if err := profiles.LoadByIDs(ctx, conn, scope, ownerIDs); err != nil && !errors.Is(err, coredata.ErrResourceNotFound) {
		return nil, fmt.Errorf("cannot load finding owners: %w", err)
	}

	for _, p := range profiles {
		names[p.ID] = p.FullName
	}

	return names, nil
}
