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

package types

import (
	"time"

	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/page"
)

type (
	TrainingPlanOrderBy OrderBy[coredata.TrainingPlanOrderField]

	TrainingPlanConnection struct {
		TotalCount int
		Edges      []*TrainingPlanEdge
		PageInfo   PageInfo

		Resolver   any
		ParentID   gid.GID
		FiscalYear *int
	}
)

func NewTrainingPlanConnection(
	p *page.Page[*coredata.TrainingPlan, coredata.TrainingPlanOrderField],
	resolver any,
	parentID gid.GID,
	fiscalYear *int,
) *TrainingPlanConnection {
	edges := make([]*TrainingPlanEdge, len(p.Data))
	for i, plan := range p.Data {
		edges[i] = NewTrainingPlanEdge(plan, p.Cursor.OrderBy.Field)
	}

	return &TrainingPlanConnection{
		Edges:    edges,
		PageInfo: *NewPageInfo(p),

		Resolver:   resolver,
		ParentID:   parentID,
		FiscalYear: fiscalYear,
	}
}

func NewTrainingPlanEdge(plan *coredata.TrainingPlan, orderField coredata.TrainingPlanOrderField) *TrainingPlanEdge {
	return &TrainingPlanEdge{
		Node:   NewTrainingPlan(plan),
		Cursor: plan.CursorKey(orderField),
	}
}

func NewTrainingPlan(plan *coredata.TrainingPlan) *TrainingPlan {
	return &TrainingPlan{
		ID:           plan.ID,
		FiscalYear:   plan.FiscalYear,
		Name:         plan.Name,
		Kind:         plan.Kind,
		Description:  plan.Description,
		Method:       plan.Method,
		Instructor:   plan.Instructor,
		StartsOn:     plan.StartsOn.Format(dateOnly),
		EndsOn:       plan.EndsOn.Format(dateOnly),
		TimeZone:     plan.TimeZone,
		TestRequired: plan.TestRequired,
		PassScore:    plan.PassScore,
		CreatedAt:    plan.CreatedAt,
		UpdatedAt:    plan.UpdatedAt,
	}
}

func NewTrainingPlanMaterials(materials coredata.TrainingPlanMaterials) []*TrainingPlanMaterial {
	result := make([]*TrainingPlanMaterial, len(materials))
	for i, material := range materials {
		result[i] = &TrainingPlanMaterial{
			DocumentID: material.DocumentID,
			Title:      material.Title,
		}
	}

	return result
}

// dateOnly is the format of dates without a time in the training API.
const dateOnly = time.DateOnly

func formatDateOnly(t *time.Time) *string {
	if t == nil {
		return nil
	}

	value := t.Format(dateOnly)

	return &value
}
