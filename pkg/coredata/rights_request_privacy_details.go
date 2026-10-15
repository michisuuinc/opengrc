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
package coredata

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/jackc/pgx/v5"
	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/gid"
)

type (
	// RightsRequestPrivacyDetails holds the Japanese PrivacyMark record of a
	// request from an individual (本人からの請求の記録) that RightsRequest
	// does not carry. A request has at most one row. Dates without a time
	// are stored as DATE.
	RightsRequestPrivacyDetails struct {
		RightsRequestID       gid.GID                          `db:"rights_request_id"`
		OrganizationID        gid.GID                          `db:"organization_id"`
		Category              *RightsRequestPrivacyCategory    `db:"category"`
		ReceivedOn            *time.Time                       `db:"received_on"`
		ReceiptChannel        *RightsRequestReceiptChannel     `db:"receipt_channel"`
		ByAgent               bool                             `db:"by_agent"`
		TargetDepartment      string                           `db:"target_department"`
		HandlerProfileID      *gid.GID                         `db:"handler_profile_id"`
		VerificationMethod    *RightsRequestVerificationMethod `db:"verification_method"`
		VerifiedOn            *time.Time                       `db:"verified_on"`
		VerificationNote      string                           `db:"verification_note"`
		SupplementRequestedOn *time.Time                       `db:"supplement_requested_on"`
		SupplementReceivedOn  *time.Time                       `db:"supplement_received_on"`
		Decision              *RightsRequestDecision           `db:"decision"`
		DecisionReason        string                           `db:"decision_reason"`
		AnswerMethod          *RightsRequestAnswerMethod       `db:"answer_method"`
		AnsweredOn            *time.Time                       `db:"answered_on"`
		ApproverProfileID     *gid.GID                         `db:"approver_profile_id"`
		FirstRespondedOn      *time.Time                       `db:"first_responded_on"`
		FindingID             *gid.GID                         `db:"finding_id"`
		CreatedAt             time.Time                        `db:"created_at"`
		UpdatedAt             time.Time                        `db:"updated_at"`
	}

	RightsRequestPrivacyDetailsList []*RightsRequestPrivacyDetails
)

const rightsRequestPrivacyDetailsColumns = `
	rights_request_id,
	organization_id,
	category,
	received_on,
	receipt_channel,
	by_agent,
	target_department,
	handler_profile_id,
	verification_method,
	verified_on,
	verification_note,
	supplement_requested_on,
	supplement_received_on,
	decision,
	decision_reason,
	answer_method,
	answered_on,
	approver_profile_id,
	first_responded_on,
	finding_id,
	created_at,
	updated_at
`

func (d *RightsRequestPrivacyDetails) LoadByRightsRequestID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	rightsRequestID gid.GID,
) error {
	q := `
SELECT
` + rightsRequestPrivacyDetailsColumns + `
FROM
	rights_request_privacy_details
WHERE
	%s
	AND rights_request_id = @rights_request_id
LIMIT 1;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"rights_request_id": rightsRequestID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query rights request privacy details: %w", err)
	}

	details, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[RightsRequestPrivacyDetails])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect rights request privacy details: %w", err)
	}

	*d = details

	return nil
}

// LoadByOrganizationID loads the details of every request of the
// organization.
func (l *RightsRequestPrivacyDetailsList) LoadByOrganizationID(
	ctx context.Context,
	conn pg.Querier,
	scope Scoper,
	organizationID gid.GID,
) error {
	q := `
SELECT
` + rightsRequestPrivacyDetailsColumns + `
FROM
	rights_request_privacy_details
WHERE
	%s
	AND organization_id = @organization_id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"organization_id": organizationID}
	maps.Copy(args, scope.SQLArguments())

	rows, err := conn.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot query rights request privacy details: %w", err)
	}

	list, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[RightsRequestPrivacyDetails])
	if err != nil {
		return fmt.Errorf("cannot collect rights request privacy details: %w", err)
	}

	*l = list

	return nil
}

// LockRightsRequest locks the parent request row for the rest of the
// transaction and returns its organization, so concurrent updates of the
// same request, including the first insert, run one after the other.
func (d *RightsRequestPrivacyDetails) LockRightsRequest(
	ctx context.Context,
	tx pg.Tx,
	scope Scoper,
	rightsRequestID gid.GID,
) (gid.GID, error) {
	q := `
SELECT
	organization_id
FROM
	rights_requests
WHERE
	%s
	AND id = @rights_request_id
FOR UPDATE;
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{"rights_request_id": rightsRequestID}
	maps.Copy(args, scope.SQLArguments())

	var organizationID gid.GID
	if err := tx.QueryRow(ctx, q, args).Scan(&organizationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gid.GID{}, ErrResourceNotFound
		}

		return gid.GID{}, fmt.Errorf("cannot lock rights request: %w", err)
	}

	return organizationID, nil
}

// Upsert inserts the row or replaces its values. The receiver is synced with
// the stored row, so CreatedAt keeps its first value. The handler and
// approver, and the finding, must belong to the organization, or
// ErrResourceNotFound is returned.
func (d *RightsRequestPrivacyDetails) Upsert(
	ctx context.Context,
	tx pg.Tx,
	scope Scoper,
) error {
	profileIDs := make([]gid.GID, 0, 2)
	for _, id := range []*gid.GID{d.HandlerProfileID, d.ApproverProfileID} {
		if id != nil && (len(profileIDs) == 0 || profileIDs[0] != *id) {
			profileIDs = append(profileIDs, *id)
		}
	}

	if err := checkProfilesInOrganization(ctx, tx, scope, d.OrganizationID, gidStrings(profileIDs)); err != nil {
		return err
	}

	if err := checkFindingInOrganization(ctx, tx, scope, d.OrganizationID, d.FindingID); err != nil {
		return err
	}

	q := `
INSERT INTO rights_request_privacy_details (
	rights_request_id,
	tenant_id,
	organization_id,
	category,
	received_on,
	receipt_channel,
	by_agent,
	target_department,
	handler_profile_id,
	verification_method,
	verified_on,
	verification_note,
	supplement_requested_on,
	supplement_received_on,
	decision,
	decision_reason,
	answer_method,
	answered_on,
	approver_profile_id,
	first_responded_on,
	finding_id,
	created_at,
	updated_at
) VALUES (
	@rights_request_id,
	@tenant_id,
	@organization_id,
	@category,
	@received_on,
	@receipt_channel,
	@by_agent,
	@target_department,
	@handler_profile_id,
	@verification_method,
	@verified_on,
	@verification_note,
	@supplement_requested_on,
	@supplement_received_on,
	@decision,
	@decision_reason,
	@answer_method,
	@answered_on,
	@approver_profile_id,
	@first_responded_on,
	@finding_id,
	@created_at,
	@updated_at
)
ON CONFLICT (rights_request_id) DO UPDATE
SET
	category = EXCLUDED.category,
	received_on = EXCLUDED.received_on,
	receipt_channel = EXCLUDED.receipt_channel,
	by_agent = EXCLUDED.by_agent,
	target_department = EXCLUDED.target_department,
	handler_profile_id = EXCLUDED.handler_profile_id,
	verification_method = EXCLUDED.verification_method,
	verified_on = EXCLUDED.verified_on,
	verification_note = EXCLUDED.verification_note,
	supplement_requested_on = EXCLUDED.supplement_requested_on,
	supplement_received_on = EXCLUDED.supplement_received_on,
	decision = EXCLUDED.decision,
	decision_reason = EXCLUDED.decision_reason,
	answer_method = EXCLUDED.answer_method,
	answered_on = EXCLUDED.answered_on,
	approver_profile_id = EXCLUDED.approver_profile_id,
	first_responded_on = EXCLUDED.first_responded_on,
	finding_id = EXCLUDED.finding_id,
	updated_at = EXCLUDED.updated_at
WHERE
	rights_request_privacy_details.tenant_id = EXCLUDED.tenant_id
RETURNING
` + rightsRequestPrivacyDetailsColumns

	args := pgx.StrictNamedArgs{
		"rights_request_id":       d.RightsRequestID,
		"tenant_id":               scope.GetTenantID(),
		"organization_id":         d.OrganizationID,
		"category":                d.Category,
		"received_on":             d.ReceivedOn,
		"receipt_channel":         d.ReceiptChannel,
		"by_agent":                d.ByAgent,
		"target_department":       d.TargetDepartment,
		"handler_profile_id":      d.HandlerProfileID,
		"verification_method":     d.VerificationMethod,
		"verified_on":             d.VerifiedOn,
		"verification_note":       d.VerificationNote,
		"supplement_requested_on": d.SupplementRequestedOn,
		"supplement_received_on":  d.SupplementReceivedOn,
		"decision":                d.Decision,
		"decision_reason":         d.DecisionReason,
		"answer_method":           d.AnswerMethod,
		"answered_on":             d.AnsweredOn,
		"approver_profile_id":     d.ApproverProfileID,
		"first_responded_on":      d.FirstRespondedOn,
		"finding_id":              d.FindingID,
		"created_at":              d.CreatedAt,
		"updated_at":              d.UpdatedAt,
	}

	rows, err := tx.Query(ctx, q, args)
	if err != nil {
		return fmt.Errorf("cannot upsert rights request privacy details: %w", err)
	}

	details, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[RightsRequestPrivacyDetails])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}

		return fmt.Errorf("cannot collect upserted rights request privacy details: %w", err)
	}

	*d = details

	return nil
}

func checkFindingInOrganization(
	ctx context.Context,
	tx pg.Tx,
	scope Scoper,
	organizationID gid.GID,
	findingID *gid.GID,
) error {
	if findingID == nil {
		return nil
	}

	q := `
SELECT
	COUNT(*)
FROM
	findings
WHERE
	%s
	AND organization_id = @organization_id
	AND id = @finding_id
`

	q = fmt.Sprintf(q, scope.SQLFragment())

	args := pgx.StrictNamedArgs{
		"organization_id": organizationID,
		"finding_id":      *findingID,
	}
	maps.Copy(args, scope.SQLArguments())

	var count int
	if err := tx.QueryRow(ctx, q, args).Scan(&count); err != nil {
		return fmt.Errorf("cannot count findings: %w", err)
	}

	if count != 1 {
		return ErrResourceNotFound
	}

	return nil
}
