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
	"encoding/json"
	"fmt"
	"text/template"

	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/docgen"
	"go.probo.inc/probo/pkg/gid"
)

// soaTemplateJapanese is the Japanese statement of applicability (適用宣言書).
// It has the same structure and data as soaTemplate.
var soaTemplateJapanese = template.Must(
	template.New("statement_of_applicability.ja.json.tmpl").
		Funcs(template.FuncMap{
			"json": func(v any) (string, error) {
				b, err := json.Marshal(v)
				if err != nil {
					return "", err
				}

				return string(b), nil
			},
			"printf": fmt.Sprintf,
		}).
		ParseFS(Templates, "templates/statement_of_applicability.ja.json.tmpl"),
)

// statementOfApplicabilityLanguage returns "ja" when the statement's name or
// the frameworks and controls it lists are in Japanese, as for the
// Japanese PrivacyMark or ISO 27001 frameworks, and "en" otherwise.
// Documents have no language of their own.
func statementOfApplicabilityLanguage(
	name string,
	statements []*coredata.ApplicabilityStatement,
	controls map[gid.GID]*coredata.Control,
	frameworks map[gid.GID]*coredata.Framework,
) string {
	texts := make([]string, 0, 2*len(statements)+1)
	texts = append(texts, name)

	for _, stmt := range statements {
		control := controls[stmt.ControlID]
		if control == nil {
			continue
		}

		texts = append(texts, control.Name)
		if framework := frameworks[control.FrameworkID]; framework != nil {
			texts = append(texts, framework.Name)
		}
	}

	return docgen.DetectLanguage(texts...)
}
