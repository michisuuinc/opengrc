// Copyright (c) 2025-2026 Probo Inc <hello@probo.com>.
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

package probo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.probo.inc/probo/pkg/attachment"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/iam/policy"
	"go.probo.inc/probo/pkg/probo"
)

func TestAuditorPolicy_ProcessingActivityPageReadAccess(t *testing.T) {
	t.Parallel()

	organizationID := gid.New(gid.NewTenantID(), 1)
	evaluator := policy.NewEvaluator()
	conditionContext := policy.ConditionContext{
		Principal: map[string]string{
			"organization_id": organizationID.String(),
		},
		Resource: map[string]string{
			"organization_id": organizationID.String(),
		},
	}

	tests := []struct {
		name   string
		action string
	}{
		{
			name:   "list processing activities",
			action: probo.ActionProcessingActivityList,
		},
		{
			name:   "list data protection impact assessments",
			action: probo.ActionDataProtectionImpactAssessmentList,
		},
		{
			name:   "list transfer impact assessments",
			action: probo.ActionTransferImpactAssessmentList,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := evaluator.Evaluate(
				policy.AuthorizationRequest{
					Principal:        organizationID,
					Resource:         organizationID,
					Action:           tt.action,
					ConditionContext: conditionContext,
				},
				[]*policy.Policy{probo.AuditorPolicy},
			)

			assert.True(t, result.IsAllowed())
		})
	}
}

func TestAuditorPolicy_OrganizationContextReadAccess(t *testing.T) {
	t.Parallel()

	organizationID := gid.New(gid.NewTenantID(), 1)
	evaluator := policy.NewEvaluator()
	conditionContext := policy.ConditionContext{
		Principal: map[string]string{
			"organization_id": organizationID.String(),
		},
		Resource: map[string]string{
			"organization_id": organizationID.String(),
		},
	}

	result := evaluator.Evaluate(
		policy.AuthorizationRequest{
			Principal:        organizationID,
			Resource:         organizationID,
			Action:           probo.ActionOrganizationContextGet,
			ConditionContext: conditionContext,
		},
		[]*policy.Policy{probo.AuditorPolicy},
	)

	assert.True(t, result.IsAllowed())
}

func TestOwnerAndAdminCanOnlyCreateTheirOwnAttachment(t *testing.T) {
	t.Parallel()

	organizationID := gid.New(gid.NewTenantID(), 1)
	uploaderID := gid.New(gid.NewTenantID(), 51)
	otherID := gid.New(gid.NewTenantID(), 51)
	fileID := gid.New(gid.NewTenantID(), 25)
	evaluator := policy.NewEvaluator()

	tests := []struct {
		name      string
		role      *policy.Policy
		principal gid.GID
		uploader  string
		want      bool
	}{
		{
			name:      "owner creates own attachment",
			role:      probo.OwnerPolicy,
			principal: uploaderID,
			uploader:  uploaderID.String(),
			want:      true,
		},
		{
			name:      "owner creates another attachment",
			role:      probo.OwnerPolicy,
			principal: otherID,
			uploader:  uploaderID.String(),
			want:      false,
		},
		{
			name:      "admin creates own attachment",
			role:      probo.AdminPolicy,
			principal: uploaderID,
			uploader:  uploaderID.String(),
			want:      true,
		},
		{
			name:      "admin creates another attachment",
			role:      probo.AdminPolicy,
			principal: otherID,
			uploader:  uploaderID.String(),
			want:      false,
		},
		{
			name:      "owner creates an attachment with no uploader",
			role:      probo.OwnerPolicy,
			principal: uploaderID,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := evaluator.Evaluate(
				policy.AuthorizationRequest{
					Principal: tt.principal,
					Resource:  fileID,
					Action:    attachment.ActionCreate,
					ConditionContext: policy.ConditionContext{
						Principal: map[string]string{
							"id":              tt.principal.String(),
							"organization_id": organizationID.String(),
						},
						Resource: map[string]string{
							"organization_id": organizationID.String(),
							"uploader_id":     tt.uploader,
						},
					},
				},
				[]*policy.Policy{tt.role},
			)

			assert.Equal(t, tt.want, result.IsAllowed())
		})
	}
}
