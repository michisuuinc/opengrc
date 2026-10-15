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

package console_v1

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.gearno.de/kit/log"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/iam"
	"go.probo.inc/probo/pkg/page"
	"go.probo.inc/probo/pkg/probo"
	"go.probo.inc/probo/pkg/server/api/console/v1/types"
	"go.probo.inc/probo/pkg/server/gqlutils"
	"go.probo.inc/probo/pkg/validator"
)

func NewTrainingPlanSummary(summary *probo.TrainingPlanSummary) *types.TrainingPlanSummary {
	return &types.TrainingPlanSummary{
		TotalCount:       summary.Total,
		CompletedCount:   summary.Completed,
		NotAttendedCount: summary.NotAttended,
	}
}

func NewTrainingParticipant(p probo.TrainingParticipantProgress) *types.TrainingParticipant {
	participant := p.Participant

	return &types.TrainingParticipant{
		ID:                       participant.ID,
		ProfileID:                participant.ProfileID,
		FullName:                 participant.FullName,
		EmailAddress:             participant.EmailAddress,
		Department:               participant.Department,
		Status:                   p.Progress.Status,
		AttendedOn:               formatTrainingDate(p.Progress.AttendedOn),
		AttendanceSource:         p.Progress.Source,
		SignedMaterialCount:      p.Progress.SignedMaterials,
		SignedOutsidePeriodCount: p.Progress.SignedOutsidePeriod,
		ManualAttendedOn:         formatTrainingDate(participant.AttendedOn),
		TestScore:                participant.TestScore,
		TestPassed:               p.Progress.TestPassed,
		ManualTestPassed:         participant.TestPassed,
		Note:                     participant.Note,
	}
}

func formatTrainingDate(t *time.Time) *string {
	if t == nil {
		return nil
	}

	value := t.Format(time.DateOnly)

	return &value
}

// parseTrainingDate parses a "YYYY-MM-DD" date of the training API.
func parseTrainingDate(field string, value string) (time.Time, error) {
	date, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, &errInvalidTrainingDate{message: fmt.Sprintf("%s must be a date as YYYY-MM-DD", field)}
	}

	return date, nil
}

func parseOptionalTrainingDate(field string, value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}

	date, err := parseTrainingDate(field, *value)
	if err != nil {
		return nil, err
	}

	return &date, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

// errInvalidTrainingDate marks a date the API could not parse.
type errInvalidTrainingDate struct{ message string }

func (e *errInvalidTrainingDate) Error() string { return e.message }

// trainingError maps service errors of the training API to GraphQL errors.
// msg is logged for unexpected errors.
func (r *Resolver) trainingError(ctx context.Context, err error, msg string) error {
	if validationErrors, ok := errors.AsType[validator.ValidationErrors](err); ok {
		return gqlutils.InvalidValidationErrors(ctx, validationErrors)
	}
	if errDate, ok := errors.AsType[*errInvalidTrainingDate](err); ok {
		return gqlutils.Invalid(ctx, errDate)
	}
	if errPeriod, ok := errors.AsType[*probo.ErrTrainingInvalidPeriod](err); ok {
		return gqlutils.Invalid(ctx, errPeriod)
	}
	if errTimeZone, ok := errors.AsType[*probo.ErrTrainingInvalidTimeZone](err); ok {
		return gqlutils.Invalid(ctx, errTimeZone)
	}
	if errNoMaterials, ok := errors.AsType[*probo.ErrTrainingPlanHasNoMaterials](err); ok {
		return gqlutils.Conflict(ctx, errNoMaterials)
	}
	if errNotPublished, ok := errors.AsType[*probo.ErrTrainingMaterialNotPublished](err); ok {
		return gqlutils.Conflict(ctx, errNotPublished)
	}
	if errTooMany, ok := errors.AsType[*probo.ErrTrainingTooManyParticipants](err); ok {
		return gqlutils.Conflict(ctx, errTooMany)
	}
	if errFuture, ok := errors.AsType[*probo.ErrTrainingAttendedInFuture](err); ok {
		return gqlutils.Invalid(ctx, errFuture)
	}
	if errNotPublished, ok := errors.AsType[*probo.ErrDocumentVersionNotPublished](err); ok {
		return gqlutils.Conflict(ctx, errNotPublished)
	}
	if errContractEnded, ok := errors.AsType[*probo.ErrProfileContractEnded](err); ok {
		return gqlutils.Conflict(ctx, errContractEnded)
	}
	if errors.Is(err, coredata.ErrResourceNotFound) {
		return gqlutils.NotFound(ctx, err)
	}

	r.logger.ErrorCtx(ctx, msg, log.Error(err))

	return gqlutils.Internal(ctx)
}

func (r *Resolver) listTrainingParticipants(
	ctx context.Context,
	scope coredata.Scoper,
	trainingPlanID gid.GID,
) ([]probo.TrainingParticipantProgress, error) {
	plan, err := r.probo.Trainings.Get(ctx, scope, trainingPlanID)
	if err != nil {
		return nil, r.trainingError(ctx, err, "cannot load training plan")
	}

	participants, err := r.probo.Trainings.ListParticipants(ctx, scope, plan)
	if err != nil {
		return nil, r.trainingError(ctx, err, "cannot list training participants")
	}

	return participants, nil
}

// trainingPlanPayload reloads the plan after a mutation and wraps it in the
// mutation's payload.
func trainingPlanPayload[T any](
	ctx context.Context,
	r *Resolver,
	scope coredata.Scoper,
	trainingPlanID gid.GID,
	build func(*types.TrainingPlan) *T,
) (*T, error) {
	plan, err := r.probo.Trainings.Get(ctx, scope, trainingPlanID)
	if err != nil {
		return nil, r.trainingError(ctx, err, "cannot load training plan")
	}

	return build(types.NewTrainingPlan(plan)), nil
}

func (r *Resolver) trainingPlans(
	ctx context.Context,
	obj *types.Organization,
	first *int,
	after *page.CursorKey,
	last *int,
	before *page.CursorKey,
	orderBy *types.TrainingPlanOrderBy,
	filter *types.TrainingPlanFilter,
	parent any,
) (*types.TrainingPlanConnection, error) {
	scope, err := r.authorize(ctx, obj.ID, probo.ActionTrainingPlanList)
	if err != nil {
		return nil, err
	}

	pageOrderBy := page.OrderBy[coredata.TrainingPlanOrderField]{
		Field:     coredata.TrainingPlanOrderFieldStartsOn,
		Direction: page.OrderDirectionDesc,
	}
	if orderBy != nil {
		pageOrderBy = page.OrderBy[coredata.TrainingPlanOrderField]{
			Field:     orderBy.Field,
			Direction: orderBy.Direction,
		}
	}

	var fiscalYear *int
	if filter != nil {
		fiscalYear = filter.FiscalYear
	}

	cursor := types.NewCursor(first, after, last, before, pageOrderBy)

	plans, err := r.probo.Trainings.ListForOrganizationID(ctx, scope, obj.ID, fiscalYear, cursor)
	if err != nil {
		return nil, r.trainingError(ctx, err, "cannot list training plans")
	}

	return types.NewTrainingPlanConnection(plans, parent, obj.ID, fiscalYear), nil
}

// trainingParticipantNode loads a training participant with its progress
// in the plan, for node(id:).
func (r *Resolver) trainingParticipantNode(ctx context.Context, scope coredata.Scoper, id gid.GID) (types.Node, error) {
	// Participants carry profile details (email, department).
	if _, err := r.authorize(ctx, id, iam.ActionMembershipProfileList); err != nil {
		return nil, err
	}

	participant, err := r.probo.Trainings.GetParticipant(ctx, scope, id)
	if err != nil {
		return nil, err
	}

	plan, err := r.probo.Trainings.Get(ctx, scope, participant.TrainingPlanID)
	if err != nil {
		return nil, err
	}

	participants, err := r.probo.Trainings.ListParticipants(ctx, scope, plan)
	if err != nil {
		return nil, err
	}

	for _, p := range participants {
		if p.Participant.ID == id {
			return NewTrainingParticipant(p), nil
		}
	}

	return nil, coredata.ErrResourceNotFound
}

func (r *Resolver) trainingPlanNode(ctx context.Context, scope coredata.Scoper, id gid.GID) (types.Node, error) {
	plan, err := r.probo.Trainings.Get(ctx, scope, id)
	if err != nil {
		return nil, err
	}

	return types.NewTrainingPlan(plan), nil
}
