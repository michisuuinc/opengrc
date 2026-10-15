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
	"fmt"
	"time"

	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/page"
	"go.probo.inc/probo/pkg/validator"
	"go.probo.inc/probo/pkg/xlsxexport"
)

// TrainingService manages training (教育) for the Japanese PrivacyMark:
// yearly training plans, their materials (documents) and participants, and
// whether each participant attended and passed.
type TrainingService struct {
	svc *Service
}

const (
	// trainingMaxIDs bounds the profiles sent in one request.
	trainingMaxIDs = 1000
	// trainingMaxMaterials and trainingMaxParticipants bound one plan, and so
	// the signature requests one plan can create.
	trainingMaxMaterials    = 20
	trainingMaxParticipants = 2000
	// trainingSignatureBatch is how many participants are asked to sign in
	// one transaction.
	trainingSignatureBatch = 100

	trainingMinFiscalYear = 2000
	trainingMaxFiscalYear = 2100
)

type (
	ErrTrainingPlanHasNoMaterials struct{}

	ErrTrainingInvalidPeriod struct{}

	ErrTrainingInvalidTimeZone struct {
		TimeZone string
	}

	ErrTrainingTooManyParticipants struct{}

	ErrTrainingAttendedInFuture struct{}

	// ErrTrainingHasRecords keeps the training records required by the
	// PrivacyMark: a participant with a record cannot be removed, and a plan
	// with such participants cannot be deleted.
	ErrTrainingHasRecords struct{}

	ErrTrainingMaterialNotPublished struct {
		Title string
	}

	// TrainingParticipantProgress is a participant with their profile and
	// their progress in the plan.
	TrainingParticipantProgress struct {
		Participant *coredata.TrainingParticipantDetail
		Progress    TrainingProgress
	}

	// TrainingPlanSummary counts the participants of a plan by status.
	TrainingPlanSummary struct {
		Total       int
		Completed   int
		NotAttended int
		Recorded    int
	}
)

func (e *ErrTrainingPlanHasNoMaterials) Error() string {
	return "the training plan has no materials to sign"
}

func (e *ErrTrainingInvalidPeriod) Error() string {
	return "the training period must not end before it starts"
}

func (e *ErrTrainingInvalidTimeZone) Error() string {
	return fmt.Sprintf("unknown time zone %q", e.TimeZone)
}

func (e *ErrTrainingTooManyParticipants) Error() string {
	return fmt.Sprintf("a training plan can have at most %d participants", trainingMaxParticipants)
}

func (e *ErrTrainingHasRecords) Error() string {
	return "training records must be kept: a participant who attended or has a test result cannot be removed, nor their training plan deleted"
}

func (e *ErrTrainingAttendedInFuture) Error() string {
	return "the attendance date must not be in the future"
}

func (e *ErrTrainingMaterialNotPublished) Error() string {
	return fmt.Sprintf("the material %q has no published version to sign; publish it first", e.Title)
}

type CreateTrainingPlanRequest struct {
	OrganizationID gid.GID
	FiscalYear     int
	Name           string
	Kind           coredata.TrainingPlanKind
	Description    string
	Method         coredata.TrainingPlanMethod
	Instructor     string
	StartsOn       time.Time
	EndsOn         time.Time
	TimeZone       string
	TestRequired   bool
	PassScore      *int
}

type UpdateTrainingPlanRequest struct {
	ID           gid.GID
	FiscalYear   *int
	Name         *string
	Kind         *coredata.TrainingPlanKind
	Description  *string
	Method       *coredata.TrainingPlanMethod
	Instructor   *string
	StartsOn     *time.Time
	EndsOn       *time.Time
	TimeZone     *string
	TestRequired *bool
	// PassScore is changed only when PassScoreSet; nil then clears it.
	PassScoreSet bool
	PassScore    *int
}

// UpdateTrainingParticipantRequest replaces what is entered by hand. A nil
// TestPassed means the result is derived from the score.
type UpdateTrainingParticipantRequest struct {
	ID         gid.GID
	AttendedOn *time.Time
	TestScore  *int
	TestPassed *bool
	Note       string
}

func (r *CreateTrainingPlanRequest) Validate() error {
	v := validator.New()

	v.Check(r.OrganizationID, "organization_id", validator.Required(), validator.GID(coredata.OrganizationEntityType))
	v.Check(r.FiscalYear, "fiscal_year", validator.Min(trainingMinFiscalYear), validator.Max(trainingMaxFiscalYear))
	v.Check(r.Name, "name", validator.Required(), validator.SafeTextNoNewLine(TitleMaxLength))
	v.Check(r.Kind, "kind", validator.Required(), validator.OneOfSlice(coredata.TrainingPlanKinds()))
	v.Check(r.Method, "method", validator.Required(), validator.OneOfSlice(coredata.TrainingPlanMethods()))
	checkOptionalTrainingText(v, r.Description, "description", validator.SafeText(ContentMaxLength))
	checkOptionalTrainingText(v, r.Instructor, "instructor", validator.SafeTextNoNewLine(TitleMaxLength))
	v.Check(r.TimeZone, "time_zone", validator.Required(), validator.SafeTextNoNewLine(NameMaxLength))
	v.Check(r.PassScore, "pass_score", validator.Min(0), validator.Max(100))

	return v.Error()
}

func (r *UpdateTrainingPlanRequest) Validate() error {
	v := validator.New()

	v.Check(r.ID, "id", validator.Required(), validator.GID(coredata.TrainingPlanEntityType))
	v.Check(r.FiscalYear, "fiscal_year", validator.Min(trainingMinFiscalYear), validator.Max(trainingMaxFiscalYear))
	v.Check(r.Name, "name", validator.NotEmpty(), validator.SafeTextNoNewLine(TitleMaxLength))
	v.Check(r.Kind, "kind", validator.OneOfSlice(coredata.TrainingPlanKinds()))
	v.Check(r.Method, "method", validator.OneOfSlice(coredata.TrainingPlanMethods()))
	if r.Description != nil {
		checkOptionalTrainingText(v, *r.Description, "description", validator.SafeText(ContentMaxLength))
	}
	if r.Instructor != nil {
		checkOptionalTrainingText(v, *r.Instructor, "instructor", validator.SafeTextNoNewLine(TitleMaxLength))
	}
	v.Check(r.TimeZone, "time_zone", validator.NotEmpty(), validator.SafeTextNoNewLine(NameMaxLength))
	v.Check(r.PassScore, "pass_score", validator.Min(0), validator.Max(100))

	return v.Error()
}

func (r *UpdateTrainingParticipantRequest) Validate() error {
	v := validator.New()

	v.Check(r.ID, "id", validator.Required(), validator.GID(coredata.TrainingParticipantEntityType))
	v.Check(r.TestScore, "test_score", validator.Min(0), validator.Max(100))
	checkOptionalTrainingText(v, r.Note, "note", validator.SafeText(ContentMaxLength))

	return v.Error()
}

// checkOptionalTrainingText checks a text field that may be left empty.
func checkOptionalTrainingText(v *validator.Validator, value string, field string, check validator.ValidatorFunc) {
	if value != "" {
		v.Check(value, field, check)
	}
}

func validateTrainingIDs(ids []gid.GID, field string, entityType uint16, limit int) error {
	v := validator.New()

	v.Check(&ids, field, validator.NoDuplicates(), maxItems(limit))
	v.CheckEach(ids, field, func(index int, item any) {
		v.Check(item, fmt.Sprintf("%s[%d]", field, index), validator.Required(), validator.GID(entityType))
	})

	return v.Error()
}

// checkTrainingPlan validates what the request validators cannot: the
// period order and the time zone.
func checkTrainingPlan(plan *coredata.TrainingPlan) error {
	if plan.EndsOn.Before(plan.StartsOn) {
		return &ErrTrainingInvalidPeriod{}
	}
	if _, err := time.LoadLocation(plan.TimeZone); err != nil || plan.TimeZone == "Local" {
		return &ErrTrainingInvalidTimeZone{TimeZone: plan.TimeZone}
	}

	return nil
}

// trainingLatestAttendanceDate is the latest attendance date accepted: today
// in the plan's time zone.
func trainingLatestAttendanceDate(plan *coredata.TrainingPlan) time.Time {
	now := time.Now()
	if location, err := time.LoadLocation(plan.TimeZone); err == nil {
		now = now.In(location)
	}

	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func trainingDateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (s TrainingService) Get(
	ctx context.Context,
	scope coredata.Scoper,
	trainingPlanID gid.GID,
) (*coredata.TrainingPlan, error) {
	plan := &coredata.TrainingPlan{}

	err := s.svc.pg.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
		return plan.LoadByID(ctx, conn, scope, trainingPlanID)
	})
	if err != nil {
		return nil, fmt.Errorf("cannot load training plan: %w", err)
	}

	return plan, nil
}

func (s TrainingService) GetParticipant(
	ctx context.Context,
	scope coredata.Scoper,
	trainingParticipantID gid.GID,
) (*coredata.TrainingParticipant, error) {
	participant := &coredata.TrainingParticipant{}

	err := s.svc.pg.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
		return participant.LoadByID(ctx, conn, scope, trainingParticipantID)
	})
	if err != nil {
		return nil, fmt.Errorf("cannot load training participant: %w", err)
	}

	return participant, nil
}

func (s TrainingService) ListForOrganizationID(
	ctx context.Context,
	scope coredata.Scoper,
	organizationID gid.GID,
	fiscalYear *int,
	cursor *page.Cursor[coredata.TrainingPlanOrderField],
) (*page.Page[*coredata.TrainingPlan, coredata.TrainingPlanOrderField], error) {
	var plans coredata.TrainingPlans

	err := s.svc.pg.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
		return plans.LoadByOrganizationID(ctx, conn, scope, organizationID, fiscalYear, cursor)
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list training plans: %w", err)
	}

	return page.NewPage(plans, cursor), nil
}

func (s TrainingService) CountForOrganizationID(
	ctx context.Context,
	scope coredata.Scoper,
	organizationID gid.GID,
	fiscalYear *int,
) (int, error) {
	var count int

	err := s.svc.pg.WithConn(ctx, func(ctx context.Context, conn pg.Querier) (err error) {
		plans := coredata.TrainingPlans{}
		count, err = plans.CountByOrganizationID(ctx, conn, scope, organizationID, fiscalYear)

		return err
	})
	if err != nil {
		return 0, fmt.Errorf("cannot count training plans: %w", err)
	}

	return count, nil
}

func (s TrainingService) Create(
	ctx context.Context,
	scope coredata.Scoper,
	req CreateTrainingPlanRequest,
) (*coredata.TrainingPlan, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	now := time.Now()
	plan := &coredata.TrainingPlan{
		ID:             gid.New(scope.GetTenantID(), coredata.TrainingPlanEntityType),
		OrganizationID: req.OrganizationID,
		FiscalYear:     req.FiscalYear,
		Name:           req.Name,
		Kind:           req.Kind,
		Description:    req.Description,
		Method:         req.Method,
		Instructor:     req.Instructor,
		StartsOn:       trainingDateOnly(req.StartsOn),
		EndsOn:         trainingDateOnly(req.EndsOn),
		TimeZone:       req.TimeZone,
		TestRequired:   req.TestRequired,
		PassScore:      req.PassScore,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := checkTrainingPlan(plan); err != nil {
		return nil, err
	}

	err := s.svc.pg.WithTx(ctx, func(ctx context.Context, tx pg.Tx) error {
		return plan.Insert(ctx, tx, scope)
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create training plan: %w", err)
	}

	return plan, nil
}

func (s TrainingService) Update(
	ctx context.Context,
	scope coredata.Scoper,
	req UpdateTrainingPlanRequest,
) (*coredata.TrainingPlan, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	plan := &coredata.TrainingPlan{}

	err := s.svc.pg.WithTx(ctx, func(ctx context.Context, tx pg.Tx) error {
		if err := plan.LoadByID(ctx, tx, scope, req.ID); err != nil {
			return fmt.Errorf("cannot load training plan: %w", err)
		}

		applyTrainingPlanUpdate(plan, req)

		if err := checkTrainingPlan(plan); err != nil {
			return err
		}

		return plan.Update(ctx, tx, scope)
	})
	if err != nil {
		return nil, err
	}

	return plan, nil
}

func applyTrainingPlanUpdate(plan *coredata.TrainingPlan, req UpdateTrainingPlanRequest) {
	if req.FiscalYear != nil {
		plan.FiscalYear = *req.FiscalYear
	}
	if req.Name != nil {
		plan.Name = *req.Name
	}
	if req.Kind != nil {
		plan.Kind = *req.Kind
	}
	if req.Description != nil {
		plan.Description = *req.Description
	}
	if req.Method != nil {
		plan.Method = *req.Method
	}
	if req.Instructor != nil {
		plan.Instructor = *req.Instructor
	}
	if req.StartsOn != nil {
		plan.StartsOn = trainingDateOnly(*req.StartsOn)
	}
	if req.EndsOn != nil {
		plan.EndsOn = trainingDateOnly(*req.EndsOn)
	}
	if req.TimeZone != nil {
		plan.TimeZone = *req.TimeZone
	}
	if req.TestRequired != nil {
		plan.TestRequired = *req.TestRequired
	}
	if req.PassScoreSet {
		plan.PassScore = req.PassScore
	}

	plan.UpdatedAt = time.Now()
}

// Delete deletes a plan whose participants have no record yet.
func (s TrainingService) Delete(
	ctx context.Context,
	scope coredata.Scoper,
	trainingPlanID gid.GID,
) error {
	return s.svc.pg.WithTx(ctx, func(ctx context.Context, tx pg.Tx) error {
		plan := &coredata.TrainingPlan{}
		if err := plan.LoadByID(ctx, tx, scope, trainingPlanID); err != nil {
			return fmt.Errorf("cannot load training plan: %w", err)
		}

		participants, err := loadTrainingProgress(ctx, tx, scope, plan)
		if err != nil {
			return err
		}
		for _, p := range participants {
			if p.HasRecord() {
				return &ErrTrainingHasRecords{}
			}
		}

		return plan.Delete(ctx, tx, scope)
	})
}

func (s TrainingService) ListMaterials(
	ctx context.Context,
	scope coredata.Scoper,
	trainingPlanID gid.GID,
) (coredata.TrainingPlanMaterials, error) {
	var materials coredata.TrainingPlanMaterials

	err := s.svc.pg.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
		return materials.LoadByTrainingPlanID(ctx, conn, scope, trainingPlanID)
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list training plan materials: %w", err)
	}

	return materials, nil
}

// SetMaterials replaces the materials of a plan. Every document must belong
// to the plan's organization.
func (s TrainingService) SetMaterials(
	ctx context.Context,
	scope coredata.Scoper,
	trainingPlanID gid.GID,
	documentIDs []gid.GID,
) (*coredata.TrainingPlan, error) {
	if err := validateTrainingIDs(documentIDs, "document_ids", coredata.DocumentEntityType, trainingMaxMaterials); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	plan := &coredata.TrainingPlan{}

	err := s.svc.pg.WithTx(ctx, func(ctx context.Context, tx pg.Tx) error {
		if err := plan.LoadByID(ctx, tx, scope, trainingPlanID); err != nil {
			return fmt.Errorf("cannot load training plan: %w", err)
		}

		if len(documentIDs) > 0 {
			documents := coredata.Documents{}
			if err := documents.LoadByIDs(ctx, tx, scope, documentIDs); err != nil {
				return fmt.Errorf("cannot load documents: %w", err)
			}
			if len(documents) != len(documentIDs) {
				return coredata.ErrResourceNotFound
			}
			for _, document := range documents {
				if document.OrganizationID != plan.OrganizationID {
					return coredata.ErrResourceNotFound
				}
			}
		}

		return coredata.ReplaceTrainingPlanMaterials(ctx, tx, scope, plan.OrganizationID, plan.ID, documentIDs)
	})
	if err != nil {
		return nil, err
	}

	return plan, nil
}

// AddParticipants adds profiles of the plan's organization as participants.
// Profiles that already take part are skipped. It returns how many were
// added.
func (s TrainingService) AddParticipants(
	ctx context.Context,
	scope coredata.Scoper,
	trainingPlanID gid.GID,
	profileIDs []gid.GID,
) (int, error) {
	if err := validateTrainingIDs(profileIDs, "profile_ids", coredata.MembershipProfileEntityType, trainingMaxIDs); err != nil {
		return 0, fmt.Errorf("invalid request: %w", err)
	}

	var added int

	err := s.svc.pg.WithTx(ctx, func(ctx context.Context, tx pg.Tx) error {
		plan := &coredata.TrainingPlan{}
		if err := plan.LoadByID(ctx, tx, scope, trainingPlanID); err != nil {
			return fmt.Errorf("cannot load training plan: %w", err)
		}

		profiles := coredata.MembershipProfiles{}
		if err := profiles.LoadByIDs(ctx, tx, scope, profileIDs); err != nil {
			return fmt.Errorf("cannot load profiles: %w", err)
		}
		if len(profiles) != len(profileIDs) {
			return coredata.ErrResourceNotFound
		}
		for _, profile := range profiles {
			if profile.OrganizationID != plan.OrganizationID {
				return coredata.ErrResourceNotFound
			}
		}

		count, err := coredata.CountTrainingParticipants(ctx, tx, scope, plan.ID)
		if err != nil {
			return err
		}
		if count+len(profileIDs) > trainingMaxParticipants {
			return &ErrTrainingTooManyParticipants{}
		}

		participantIDs := make([]gid.GID, len(profileIDs))
		for i := range profileIDs {
			participantIDs[i] = gid.New(scope.GetTenantID(), coredata.TrainingParticipantEntityType)
		}

		added, err = coredata.InsertTrainingParticipants(ctx, tx, scope, plan.OrganizationID, plan.ID, participantIDs, profileIDs)

		return err
	})
	if err != nil {
		return 0, err
	}

	return added, nil
}

// RemoveParticipant removes a participant who has no record yet, such as
// one added by mistake.
func (s TrainingService) RemoveParticipant(
	ctx context.Context,
	scope coredata.Scoper,
	trainingParticipantID gid.GID,
) error {
	return s.svc.pg.WithTx(ctx, func(ctx context.Context, tx pg.Tx) error {
		participant := &coredata.TrainingParticipant{}
		if err := participant.LoadByID(ctx, tx, scope, trainingParticipantID); err != nil {
			return fmt.Errorf("cannot load training participant: %w", err)
		}

		plan := &coredata.TrainingPlan{}
		if err := plan.LoadByID(ctx, tx, scope, participant.TrainingPlanID); err != nil {
			return fmt.Errorf("cannot load training plan: %w", err)
		}

		participants, err := loadTrainingProgress(ctx, tx, scope, plan)
		if err != nil {
			return err
		}
		for _, p := range participants {
			if p.Participant.ID == participant.ID && p.HasRecord() {
				return &ErrTrainingHasRecords{}
			}
		}

		return participant.Delete(ctx, tx, scope)
	})
}

// UpdateParticipant stores what is entered by hand for a participant. Only
// an explicit test result is stored; otherwise it is derived from the score
// when read, so a corrected score or pass score takes effect.
func (s TrainingService) UpdateParticipant(
	ctx context.Context,
	scope coredata.Scoper,
	req UpdateTrainingParticipantRequest,
) (*coredata.TrainingParticipant, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	participant := &coredata.TrainingParticipant{}

	err := s.svc.pg.WithTx(ctx, func(ctx context.Context, tx pg.Tx) error {
		if err := participant.LoadByID(ctx, tx, scope, req.ID); err != nil {
			return fmt.Errorf("cannot load training participant: %w", err)
		}

		plan := &coredata.TrainingPlan{}
		if err := plan.LoadByID(ctx, tx, scope, participant.TrainingPlanID); err != nil {
			return fmt.Errorf("cannot load training plan: %w", err)
		}

		participant.AttendedOn = nil
		if req.AttendedOn != nil {
			attendedOn := trainingDateOnly(*req.AttendedOn)
			if attendedOn.After(trainingLatestAttendanceDate(plan)) {
				return &ErrTrainingAttendedInFuture{}
			}
			participant.AttendedOn = &attendedOn
		}
		participant.TestScore = req.TestScore
		participant.TestPassed = req.TestPassed
		participant.Note = req.Note
		participant.UpdatedAt = time.Now()

		return participant.Update(ctx, tx, scope)
	})
	if err != nil {
		return nil, err
	}

	return participant, nil
}

// ListParticipants returns the participants of a plan with their progress.
func (s TrainingService) ListParticipants(
	ctx context.Context,
	scope coredata.Scoper,
	plan *coredata.TrainingPlan,
) ([]TrainingParticipantProgress, error) {
	var result []TrainingParticipantProgress

	err := s.svc.pg.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
		var err error
		result, err = loadTrainingProgress(ctx, conn, scope, plan)

		return err
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// HasRecord tells whether the participant has a training record to keep:
// an attendance, by signature or entered by hand, or a test result.
func (p TrainingParticipantProgress) HasRecord() bool {
	participant := p.Participant.TrainingParticipant

	return p.Progress.AttendedOn != nil ||
		participant.AttendedOn != nil ||
		participant.TestScore != nil ||
		participant.TestPassed != nil
}

// Summarize counts the participants of a plan by status.
func (s TrainingService) Summarize(
	ctx context.Context,
	scope coredata.Scoper,
	plan *coredata.TrainingPlan,
) (*TrainingPlanSummary, error) {
	participants, err := s.ListParticipants(ctx, scope, plan)
	if err != nil {
		return nil, err
	}

	summary := &TrainingPlanSummary{Total: len(participants)}
	for _, p := range participants {
		if p.HasRecord() {
			summary.Recorded++
		}

		switch p.Progress.Status {
		case coredata.TrainingParticipantStatusCompleted:
			summary.Completed++
		case coredata.TrainingParticipantStatusNotAttended:
			summary.NotAttended++
		}
	}

	return summary, nil
}

func loadTrainingProgress(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	plan *coredata.TrainingPlan,
) ([]TrainingParticipantProgress, error) {
	var participants coredata.TrainingParticipantDetails
	if err := participants.LoadByTrainingPlanIDs(ctx, conn, scope, []gid.GID{plan.ID}); err != nil {
		return nil, fmt.Errorf("cannot load training participants: %w", err)
	}

	var materials coredata.TrainingPlanMaterials
	if err := materials.LoadByTrainingPlanID(ctx, conn, scope, plan.ID); err != nil {
		return nil, fmt.Errorf("cannot load training plan materials: %w", err)
	}

	var signatures coredata.TrainingMaterialSignatures
	if len(materials) > 0 && len(participants) > 0 {
		if err := signatures.LoadByTrainingPlanID(ctx, conn, scope, plan.ID); err != nil {
			return nil, fmt.Errorf("cannot load training material signatures: %w", err)
		}
	}

	materialIDs := materials.DocumentIDs()
	result := make([]TrainingParticipantProgress, len(participants))
	for i, participant := range participants {
		result[i] = TrainingParticipantProgress{
			Participant: participant,
			Progress:    EvaluateTrainingProgress(plan, &participant.TrainingParticipant, materialIDs, signatures),
		}
	}

	return result, nil
}

// RequestSignatures asks every participant who has not attended yet to sign
// the plan's materials, using the document signature requests. Participants
// whose contract has ended or whose profile is deactivated are skipped. It
// returns how many participants were asked.
func (s TrainingService) RequestSignatures(
	ctx context.Context,
	scope coredata.Scoper,
	trainingPlanID gid.GID,
) (int, error) {
	plan, err := s.Get(ctx, scope, trainingPlanID)
	if err != nil {
		return 0, err
	}

	materials, err := s.ListMaterials(ctx, scope, plan.ID)
	if err != nil {
		return 0, err
	}
	if len(materials) == 0 {
		return 0, &ErrTrainingPlanHasNoMaterials{}
	}

	participants, err := s.ListParticipants(ctx, scope, plan)
	if err != nil {
		return 0, err
	}

	if err := s.checkMaterialsPublished(ctx, scope, materials); err != nil {
		return 0, err
	}

	profileIDs, err := s.signableParticipants(ctx, scope, participants)
	if err != nil {
		return 0, err
	}

	for start := 0; start < len(profileIDs); start += trainingSignatureBatch {
		end := min(start+trainingSignatureBatch, len(profileIDs))

		_, err = s.svc.Documents.BulkRequestSignatures(ctx, scope, BulkRequestSignaturesRequest{
			DocumentIDs:  materials.DocumentIDs(),
			SignatoryIDs: profileIDs[start:end],
		})
		if err != nil {
			return start, fmt.Errorf("cannot request signatures: %w", err)
		}
	}

	return len(profileIDs), nil
}

// checkMaterialsPublished tells which material cannot be signed: signature
// requests go to the latest version, which must be published.
func (s TrainingService) checkMaterialsPublished(
	ctx context.Context,
	scope coredata.Scoper,
	materials coredata.TrainingPlanMaterials,
) error {
	return s.svc.pg.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
		for _, material := range materials {
			version := &coredata.DocumentVersion{}
			if err := version.LoadLatestVersion(ctx, conn, scope, material.DocumentID); err != nil {
				return fmt.Errorf("cannot load latest version of %q: %w", material.DocumentID, err)
			}
			if version.Status != coredata.DocumentVersionStatusPublished {
				return &ErrTrainingMaterialNotPublished{Title: material.Title}
			}
		}

		return nil
	})
}

func (s TrainingService) signableParticipants(
	ctx context.Context,
	scope coredata.Scoper,
	participants []TrainingParticipantProgress,
) ([]gid.GID, error) {
	pending := make([]gid.GID, 0, len(participants))
	for _, p := range participants {
		if p.Progress.AttendedOn == nil {
			pending = append(pending, p.Participant.ProfileID)
		}
	}
	if len(pending) == 0 {
		return nil, nil
	}

	profiles := coredata.MembershipProfiles{}
	err := s.svc.pg.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
		return profiles.LoadByIDs(ctx, conn, scope, pending)
	})
	if err != nil {
		return nil, fmt.Errorf("cannot load participant profiles: %w", err)
	}

	now := time.Now()
	profileIDs := make([]gid.GID, 0, len(profiles))
	for _, profile := range profiles {
		if profile.State == coredata.ProfileStateDeactivated {
			continue
		}
		if profile.ContractEndDate != nil && profile.ContractEndDate.Before(now) {
			continue
		}
		profileIDs = append(profileIDs, profile.ID)
	}

	return profileIDs, nil
}

// Export returns the training records (教育実施記録) of the organization,
// optionally of one fiscal year, as an Excel workbook.
func (s TrainingService) Export(
	ctx context.Context,
	scope coredata.Scoper,
	organizationID gid.GID,
	fiscalYear *int,
) ([]byte, error) {
	data := trainingSheetData{}

	err := s.svc.pg.WithConn(ctx, func(ctx context.Context, conn pg.Querier) error {
		organization := &coredata.Organization{}
		if err := organization.LoadByID(ctx, conn, scope, organizationID); err != nil {
			return fmt.Errorf("cannot load organization: %w", err)
		}
		data.OrganizationName = organization.Name

		var plans coredata.TrainingPlans
		if err := plans.LoadAllByOrganizationID(ctx, conn, scope, organizationID, fiscalYear); err != nil {
			return fmt.Errorf("cannot load training plans: %w", err)
		}

		for _, plan := range plans {
			participants, err := loadTrainingProgress(ctx, conn, scope, plan)
			if err != nil {
				return err
			}
			data.Plans = append(data.Plans, trainingSheetPlan{Plan: plan, Participants: participants})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	workbook, err := xlsxexport.Build(buildTrainingSheet(data, time.Now().In(ledgerTimeZone)))
	if err != nil {
		return nil, fmt.Errorf("cannot build training records workbook: %w", err)
	}

	return workbook, nil
}
