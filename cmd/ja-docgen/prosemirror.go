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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"go.probo.inc/probo/pkg/prosemirror"
)

// ToProseMirror converts a finished Markdown document to the ProseMirror JSON
// that OpenGRC stores as document content. The leading "# " title line is
// dropped, because OpenGRC keeps the title separately.
func ToProseMirror(markdown string) (string, error) {
	if first, rest, _ := strings.Cut(markdown, "\n"); strings.HasPrefix(first, "# ") {
		markdown = rest
	}

	node, err := prosemirror.ParseMarkdown(markdown)
	if err != nil {
		return "", fmt.Errorf("cannot convert markdown: %w", err)
	}
	data, err := json.Marshal(node)
	if err != nil {
		return "", fmt.Errorf("cannot encode document: %w", err)
	}
	if err := prosemirror.ValidateDocumentContentJSON(string(data)); err != nil {
		return "", fmt.Errorf("invalid document content: %w", err)
	}
	return string(data), nil
}

func runConvert(path string, w io.Writer) int {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("cannot read document: %w", err))
		return 2
	}
	content, err := ToProseMirror(string(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if _, err := io.WriteString(w, content); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("cannot write document: %w", err))
		return 2
	}
	return 0
}
