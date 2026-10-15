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

package docgen

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.probo.inc/probo/pkg/coredata"
)

func labelsTestDocument(title string, text string) DocumentData {
	// 2026-04-01 15:30 UTC is 2026-04-02 in Japan.
	published := time.Date(2026, 4, 1, 15, 30, 0, 0, time.UTC)
	signed := time.Date(2026, 4, 3, 1, 0, 0, 0, time.UTC)

	content, _ := json.Marshal(map[string]any{
		"type": "doc",
		"content": []any{map[string]any{
			"type":    "paragraph",
			"content": []any{map[string]any{"type": "text", "text": text}},
		}},
	})

	return DocumentData{
		Title:          title,
		Content:        content,
		Major:          1,
		Classification: ClassificationInternal,
		Approvers:      []string{"Taro", "Hanako"},
		PublishedAt:    &published,
		Signatures: []SignatureData{
			{SignedBy: "Taro", State: coredata.DocumentVersionSignatureStateSigned, RequestedAt: published, SignedAt: &signed},
			{SignedBy: "Hanako", State: coredata.DocumentVersionSignatureStateRequested, RequestedAt: published},
		},
	}
}

func TestRenderHTMLJapaneseLabels(t *testing.T) {
	t.Parallel()

	out, err := RenderHTML(labelsTestDocument("個人情報保護方針", "当社は、個人情報を適切に取り扱います。"))
	require.NoError(t, err)

	html := string(out)
	assert.Contains(t, html, `<html lang="ja"`)
	for _, want := range []string{"機密区分", "社内限", "承認者", "公開日", "2026年4月2日", "文書の署名", "署名者", "署名済み", "依頼済み", "2026/04/02", "2026/04/03"} {
		assert.Contains(t, html, want)
	}
	for _, unwanted := range []string{"Classification<", "Published<", "Document Signatures", ">SIGNED<", "INTERNAL"} {
		assert.NotContains(t, html, unwanted)
	}
}

func TestRenderHTMLEnglishLabelsUnchanged(t *testing.T) {
	t.Parallel()

	out, err := RenderHTML(labelsTestDocument("Privacy Policy", "We handle personal data with care."))
	require.NoError(t, err)

	html := string(out)
	for _, want := range []string{"Classification", "INTERNAL", "Approvers", "Version:", "Published", "April 1, 2026", "Document Signatures", "Signatory", "SIGNED", "REQUESTED", "Apr 1, 2026", "Apr 3, 2026"} {
		assert.Contains(t, html, want)
	}
}

func TestRenderSignaturePageHTMLLabels(t *testing.T) {
	t.Parallel()

	data := labelsTestDocument("x", "y")

	ja, err := RenderSignaturePageHTML(SignaturePageData{Signatures: data.Signatures, Language: "ja"})
	require.NoError(t, err)
	assert.Contains(t, string(ja), "文書の署名")
	assert.Contains(t, string(ja), "署名済み")
	assert.Contains(t, string(ja), "2026/04/03")

	en, err := RenderSignaturePageHTML(SignaturePageData{Signatures: data.Signatures})
	require.NoError(t, err)
	assert.Contains(t, string(en), "Document Signatures")
	assert.Contains(t, string(en), "Apr 3, 2026")
}

func TestIsJapanese(t *testing.T) {
	t.Parallel()

	for tag, want := range map[string]bool{"ja": true, "ja-JP": true, "en": false, "jam": false, "": false} {
		assert.Equal(t, want, IsJapanese(tag), tag)
	}
}

func TestDetectTitleLanguage(t *testing.T) {
	t.Parallel()

	for title, want := range map[string]string{"個人情報保護方針": "ja", "委託先評価チェックリスト": "ja", "Privacy Policy": "en", "": "en"} {
		assert.Equal(t, want, DetectTitleLanguage(title), title)
	}
}
