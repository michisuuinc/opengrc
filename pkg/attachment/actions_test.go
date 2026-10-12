// Copyright (c) 2026 Probo Inc <hello@probo.com>.
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

package attachment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
)

func TestParentReadAction(t *testing.T) {
	t.Parallel()

	tenant := gid.NewTenantID()
	actions := ParentReadActions{
		DocumentVersion: "document-version",
		Task:            "task",
		TaskComment:     "task-comment",
		RiskAnalysis:    "risk-analysis",
	}
	svc := NewService(nil, actions)
	cases := []struct {
		name   string
		id     gid.GID
		action string
	}{
		{"document version", gid.New(tenant, coredata.DocumentVersionEntityType), actions.DocumentVersion},
		{"task", gid.New(tenant, coredata.TaskEntityType), actions.Task},
		{"task comment", gid.New(tenant, coredata.TaskCommentEntityType), actions.TaskComment},
		{"risk analysis", gid.New(tenant, coredata.RiskAnalysisEntityType), actions.RiskAnalysis},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			action, ok := svc.ParentReadAction(tc.id)
			require.True(t, ok)
			assert.Equal(t, tc.action, action)
		})
	}

	t.Run("unknown parent", func(t *testing.T) {
		t.Parallel()

		_, ok := svc.ParentReadAction(gid.New(tenant, coredata.OrganizationEntityType))
		assert.False(t, ok)
	})

	t.Run("empty action", func(t *testing.T) {
		t.Parallel()

		empty := NewService(nil, ParentReadActions{DocumentVersion: actions.DocumentVersion})
		_, ok := empty.ParentReadAction(gid.New(tenant, coredata.TaskEntityType))
		assert.False(t, ok)
	})
}
