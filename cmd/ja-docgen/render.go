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

package main

import (
	"bytes"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type (
	// Config is the company-specific input, read from a YAML file.
	Config struct {
		Values        map[string]string
		Choices       map[string]string
		Purposes      []PurposeRow
		Transmissions []TransmissionRow
	}

	// Problem is something that prevents a template from rendering.
	Problem struct {
		File    string
		Line    int
		Message string
	}

	rawConfig struct {
		Sections      map[string]map[string]any `yaml:",inline"`
		Purposes      []PurposeRow              `yaml:"利用目的"`
		Transmissions []TransmissionRow         `yaml:"外部送信"`
	}
)

var (
	placeholderPattern = regexp.MustCompile(`【([^【】]+)】`)
	directivePattern   = regexp.MustCompile(`^\s*<!--\s*(if|else|endif|list)\b\s*(.*?)\s*-->\s*$`)
	headingPattern     = regexp.MustCompile(`^## \d+\. `)
	licensePattern     = regexp.MustCompile(`(?s)\A<!--\s*Copyright.*?-->\s*`)
)

const fillInBlank = "＿＿＿＿＿＿"

func (p Problem) String() string {
	if p.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", p.File, p.Line, p.Message)
	}
	return fmt.Sprintf("%s: %s", p.File, p.Message)
}

// ParseConfig reads the YAML configuration and applies defaults.
func ParseConfig(data []byte) (*Config, error) {
	var raw rawConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("cannot parse configuration: %w", err)
	}

	cfg := &Config{
		Values:        map[string]string{},
		Choices:       map[string]string{},
		Purposes:      slices.Clone(raw.Purposes),
		Transmissions: raw.Transmissions,
	}

	for _, section := range slices.Sorted(maps.Keys(raw.Sections)) {
		values := raw.Sections[section]
		for _, name := range slices.Sorted(maps.Keys(values)) {
			text, err := scalarText(values[name])
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", section, name, err)
			}
			if strings.ContainsAny(text, "\r\n") {
				return nil, fmt.Errorf("%s.%s: 改行を含めずに 1 行で書いてください", section, name)
			}
			if text == "" {
				// A blank value is treated as not written, so defaults apply.
				continue
			}

			if c, ok := choiceByName(name); ok && section == SectionChoice {
				if !slices.Contains(c.Values, text) {
					return nil, fmt.Errorf("選択.%s: %q は選べません（%s）", name, text, strings.Join(c.Values, "／"))
				}
				cfg.Choices[name] = text
				continue
			}

			v, ok := variableByName(name)
			if !ok {
				return nil, fmt.Errorf("%s.%s: 定義されていない項目です", section, name)
			}
			if v.Section != section {
				return nil, fmt.Errorf("%s.%s: この項目は「%s」に書いてください", section, name, v.Section)
			}
			cfg.Values[name] = text
		}
	}

	for _, v := range Variables {
		if _, ok := cfg.Values[v.Name]; !ok && v.Default != "" {
			cfg.Values[v.Name] = v.Default
		}
	}
	if _, ok := cfg.Values[VariableRevisedOn]; !ok {
		if enacted, ok := cfg.Values[VariableEnactedOn]; ok {
			cfg.Values[VariableRevisedOn] = enacted
		}
	}
	for _, name := range []string{VariableEnactedOn, VariableRevisedOn} {
		if value, ok := cfg.Values[name]; ok {
			formatted, err := formatDate(value)
			if err != nil {
				return nil, fmt.Errorf("文書.%s: %w", name, err)
			}
			cfg.Values[name] = formatted
		}
	}

	for _, c := range Choices {
		if _, ok := cfg.Choices[c.Name]; !ok {
			cfg.Choices[c.Name] = c.Default
		}
	}

	if len(cfg.Purposes) == 0 {
		cfg.Purposes = slices.Clone(DefaultPurposes)
	}

	return cfg, nil
}

func scalarText(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v), nil
	case int:
		return strconv.Itoa(v), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	case time.Time:
		return v.Format(time.DateOnly), nil
	case nil:
		return "", nil
	default:
		return "", fmt.Errorf("文字・数字・日付のいずれかで書いてください")
	}
}

func formatDate(value string) (string, error) {
	t, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return "", fmt.Errorf("日付は 2026-11-01 の形式で書いてください: %q", value)
	}
	return fmt.Sprintf("%d年%d月%d日", t.Year(), int(t.Month()), t.Day()), nil
}

// Render produces the finished document for one template. number is the
// two-digit prefix of the template file name (for example "10").
func Render(file, number, template string, cfg *Config) (string, []Problem) {
	var problems []Problem
	lines := strings.Split(licensePattern.ReplaceAllString(template, ""), "\n")

	var (
		out   []string
		stack []bool // whether each open if-block is currently active
	)
	active := func() bool {
		for _, a := range stack {
			if !a {
				return false
			}
		}
		return true
	}

	for i, line := range lines {
		lineNo := i + 1
		if m := directivePattern.FindStringSubmatch(line); m != nil {
			switch m[1] {
			case "if":
				name, value, ok := strings.Cut(m[2], "=")
				if !ok {
					problems = append(problems, Problem{File: file, Line: lineNo, Message: "if には「名前=値」を書いてください"})
					stack = append(stack, false)
					continue
				}
				stack = append(stack, cfg.Choices[strings.TrimSpace(name)] == strings.TrimSpace(value))
			case "else":
				if len(stack) == 0 {
					problems = append(problems, Problem{File: file, Line: lineNo, Message: "対応する if がありません"})
					continue
				}
				stack[len(stack)-1] = !stack[len(stack)-1]
			case "endif":
				if len(stack) == 0 {
					problems = append(problems, Problem{File: file, Line: lineNo, Message: "対応する if がありません"})
					continue
				}
				stack = stack[:len(stack)-1]
			case "list":
				if active() {
					rows, err := renderList(m[2], cfg)
					if err != nil {
						problems = append(problems, Problem{File: file, Line: lineNo, Message: err.Error()})
						continue
					}
					out = append(out, rows...)
				}
			}
			continue
		}

		if !active() || isNote(line) {
			continue
		}

		rendered := placeholderPattern.ReplaceAllStringFunc(
			line,
			func(token string) string {
				name := token[len("【") : len(token)-len("】")]
				if label, ok := strings.CutPrefix(name, FillInPrefix); ok {
					return fillInBlank + "（" + label + "）"
				}
				if name == VariableDocumentNumber {
					return cfg.Values[VariableNumberPrefix] + "-" + number
				}
				if value := cfg.Values[name]; value != "" {
					return value
				}
				problems = append(problems, Problem{File: file, Line: lineNo, Message: fmt.Sprintf("「%s」が入力されていません", name)})
				return token
			},
		)
		out = append(out, rendered)
	}

	if len(stack) != 0 {
		problems = append(problems, Problem{File: file, Message: "endif が足りません"})
	}

	return renumberHeadings(collapseBlankLines(out)), problems
}

func isNote(line string) bool {
	trimmed := strings.TrimLeft(line, " \t>")
	return strings.HasPrefix(trimmed, "※")
}

// listNames are the lists usable in <!-- list 名前 --> directives.
var listNames = []string{ListPurposesPublic, ListPurposesRetained, ListTransmissions}

// tableRow joins cells into a Markdown table row, escaping characters that
// would break the table.
func tableRow(cells ...string) string {
	escaped := make([]string, len(cells))
	for i, c := range cells {
		c = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ", "|", "\\|").Replace(c)
		escaped[i] = strings.TrimSpace(c)
	}
	return "| " + strings.Join(escaped, " | ") + " |"
}

func renderList(name string, cfg *Config) ([]string, error) {
	switch name {
	case ListPurposesPublic:
		var rows []string
		for _, p := range cfg.Purposes {
			rows = append(rows, tableRow(p.Category, p.Items, strings.Join(p.Purposes, "、")))
		}
		return rows, nil
	case ListPurposesRetained:
		var rows []string
		for _, p := range cfg.Purposes {
			if p.Retained != nil && !*p.Retained {
				continue
			}
			rows = append(rows, tableRow(p.Category+"の情報", strings.Join(p.Purposes, "、")))
		}
		return rows, nil
	case ListTransmissions:
		if len(cfg.Transmissions) == 0 {
			return nil, fmt.Errorf("「外部送信」の一覧が入力されていません（Cookieなどの外部送信が「あり」の場合は必須）")
		}
		var rows []string
		for _, t := range cfg.Transmissions {
			rows = append(rows, tableRow(t.Recipient, t.Information, t.Purpose))
		}
		return rows, nil
	default:
		return nil, fmt.Errorf("定義されていない一覧です: %s", name)
	}
}

// collapseBlankLines removes runs of blank lines left by removed notes and
// conditional blocks.
func collapseBlankLines(lines []string) []string {
	var out []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" && (len(out) == 0 || strings.TrimSpace(out[len(out)-1]) == "") {
			continue
		}
		out = append(out, line)
	}
	return out
}

// renumberHeadings rewrites "## N. " headings in order, so removing an
// optional section does not leave a gap.
func renumberHeadings(lines []string) string {
	n := 0
	for i, line := range lines {
		if headingPattern.MatchString(line) {
			n++
			lines[i] = headingPattern.ReplaceAllString(line, fmt.Sprintf("## %d. ", n))
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n")) + "\n"
}

// Check validates a template without a configuration: every 【name】 must be
// a known variable, a 記入欄 or the document number, and every directive
// must refer to a known choice or list.
func Check(file, template string) []Problem {
	var problems []Problem
	depth := 0

	for i, line := range strings.Split(template, "\n") {
		lineNo := i + 1
		if m := directivePattern.FindStringSubmatch(line); m != nil {
			switch m[1] {
			case "if":
				depth++
				name, value, ok := strings.Cut(m[2], "=")
				c, known := choiceByName(strings.TrimSpace(name))
				switch {
				case !ok:
					problems = append(problems, Problem{File: file, Line: lineNo, Message: "if には「名前=値」を書いてください"})
				case !known:
					problems = append(problems, Problem{File: file, Line: lineNo, Message: fmt.Sprintf("定義されていない選択です: %s", name)})
				case !slices.Contains(c.Values, strings.TrimSpace(value)):
					problems = append(problems, Problem{File: file, Line: lineNo, Message: fmt.Sprintf("「%s」に「%s」はありません", name, value)})
				}
			case "else":
				if depth == 0 {
					problems = append(problems, Problem{File: file, Line: lineNo, Message: "対応する if がありません"})
				}
			case "endif":
				if depth == 0 {
					problems = append(problems, Problem{File: file, Line: lineNo, Message: "対応する if がありません"})
				} else {
					depth--
				}
			case "list":
				if !slices.Contains(listNames, m[2]) {
					problems = append(problems, Problem{File: file, Line: lineNo, Message: fmt.Sprintf("定義されていない一覧です: %s", m[2])})
				}
			}
			continue
		}

		for _, m := range placeholderPattern.FindAllStringSubmatch(line, -1) {
			name := m[1]
			if strings.HasPrefix(name, FillInPrefix) || name == VariableDocumentNumber {
				continue
			}
			if _, ok := variableByName(name); !ok {
				problems = append(problems, Problem{File: file, Line: lineNo, Message: fmt.Sprintf("定義されていない変数です: 【%s】", name)})
			}
		}
	}

	if depth != 0 {
		problems = append(problems, Problem{File: file, Message: "endif が足りません"})
	}
	return problems
}
