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

// Command ja-docgen fills the Japanese PrivacyMark document templates in
// docs/ja/templates with a company's values from a YAML file.
//
//	go run ./cmd/ja-docgen -config ../private/company.yaml -out ../private/out
//	go run ./cmd/ja-docgen -check
//	go run ./cmd/ja-docgen -prosemirror ../private/out/10-pms-rules.md
package main

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var templateNamePattern = regexp.MustCompile(`^(\d\d)-.*\.md$`)

func main() {
	var (
		templatesDir = flag.String("templates", "docs/ja/templates", "directory containing the templates")
		configPath   = flag.String("config", "", "YAML file with the company's values")
		outDir       = flag.String("out", "", "directory to write the finished documents to")
		checkOnly    = flag.Bool("check", false, "only validate the templates")
		prosemirror  = flag.String("prosemirror", "", "convert a finished document to the OpenGRC document format and print it")
	)
	flag.Parse()

	if *prosemirror != "" {
		os.Exit(runConvert(*prosemirror, os.Stdout))
	}
	os.Exit(run(*templatesDir, *configPath, *outDir, *checkOnly))
}

func run(templatesDir, configPath, outDir string, checkOnly bool) int {
	templates, err := loadTemplates(templatesDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	if checkOnly {
		var problems []Problem
		for _, t := range templates {
			problems = append(problems, Check(t.file, t.content)...)
		}
		return report(problems, fmt.Sprintf("%d templates OK", len(templates)))
	}

	if configPath == "" || outDir == "" {
		fmt.Fprintln(os.Stderr, "usage: ja-docgen -config company.yaml -out dir  |  ja-docgen -check")
		return 2
	}

	if err := checkPaths(templatesDir, configPath, outDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("cannot read configuration: %w", err))
		return 2
	}
	cfg, err := ParseConfig(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	var (
		problems []Problem
		rendered = map[string]string{}
	)
	for _, t := range templates {
		out, p := Render(t.file, t.number, t.content, cfg)
		problems = append(problems, p...)
		rendered[t.file] = out
	}
	if len(problems) > 0 {
		return report(problems, "")
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("cannot create output directory: %w", err))
		return 2
	}
	for _, file := range slices.Sorted(maps.Keys(rendered)) {
		if err := os.WriteFile(filepath.Join(outDir, file), []byte(rendered[file]), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, fmt.Errorf("cannot write %s: %w", file, err))
			return 2
		}
	}
	fmt.Printf("%d documents written to %s\n", len(rendered), outDir)
	return 0
}

type template struct {
	file    string
	number  string
	content string
}

func loadTemplates(dir string) ([]template, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read templates: %w", err)
	}

	var templates []template
	for _, e := range entries {
		m := templateNamePattern.FindStringSubmatch(e.Name())
		if e.IsDir() || m == nil {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %w", e.Name(), err)
		}
		templates = append(templates, template{file: e.Name(), number: m[1], content: string(content)})
	}
	slices.SortFunc(
		templates,
		func(a, b template) int {
			return strings.Compare(a.file, b.file)
		},
	)

	if len(templates) == 0 {
		return nil, fmt.Errorf("no templates found in %s", dir)
	}
	return templates, nil
}

// ExampleConfigFile is the only configuration that may live inside the
// repository; it holds fictional values.
const ExampleConfigFile = "config.example.yaml"

// checkPaths refuses a configuration file or an output directory inside the
// repository that holds the templates. A company's own values and finished
// documents are confidential and must not end up in a commit by mistake.
func checkPaths(templatesDir, configPath, outDir string) error {
	templatesAbs, err := filepath.Abs(templatesDir)
	if err != nil {
		return fmt.Errorf("cannot resolve templates directory: %w", err)
	}
	configAbs, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("cannot resolve configuration path: %w", err)
	}
	outAbs, err := filepath.Abs(outDir)
	if err != nil {
		return fmt.Errorf("cannot resolve output directory: %w", err)
	}

	root := repositoryRoot(templatesAbs)
	if isWithin(root, configAbs) && configAbs != filepath.Join(templatesAbs, ExampleConfigFile) {
		return fmt.Errorf("設定ファイルはリポジトリの外に置いてください（%s）: %s", root, configPath)
	}
	if isWithin(root, outAbs) {
		return fmt.Errorf("出力先はリポジトリの外にしてください（%s）: %s", root, outDir)
	}
	return nil
}

// repositoryRoot returns the nearest ancestor of dir containing .git, or dir
// itself when there is none.
func repositoryRoot(dir string) string {
	for d := dir; ; {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return dir
		}
		d = parent
	}
}

func isWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func report(problems []Problem, success string) int {
	if len(problems) == 0 {
		fmt.Println(success)
		return 0
	}
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, p)
	}
	fmt.Fprintf(os.Stderr, "%d problem(s)\n", len(problems))
	return 1
}
