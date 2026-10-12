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
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
)

const (
	ActionUpload = "core:attachment:upload"
	ActionCreate = "core:attachment:create"
)

// ParentReadActions holds the get action for each parent.
type ParentReadActions struct {
	DocumentVersion string
	Task            string
	TaskComment     string
	RiskAnalysis    string
}

func parentReadAction(parentID gid.GID, actions ParentReadActions) (string, bool) {
	var action string

	switch parentID.EntityType() {
	case coredata.DocumentVersionEntityType:
		action = actions.DocumentVersion
	case coredata.TaskEntityType:
		action = actions.Task
	case coredata.TaskCommentEntityType:
		action = actions.TaskComment
	case coredata.RiskAnalysisEntityType:
		action = actions.RiskAnalysis
	default:
		return "", false
	}

	if action == "" {
		return "", false
	}

	return action, true
}
