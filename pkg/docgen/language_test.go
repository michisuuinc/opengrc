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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		texts []string
		want  string
	}{
		{name: "no text", texts: nil, want: "en"},
		{name: "english", texts: []string{"Information Security Policy"}, want: "en"},
		{name: "hiragana", texts: []string{"Policy", "個人情報の取扱い"}, want: "ja"},
		{name: "katakana", texts: []string{"プライバシー"}, want: "ja"},
		{name: "han only stays english", texts: []string{"信息安全"}, want: "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, DetectLanguage(tt.texts...))
		})
	}
}

func TestRenderHTMLLanguage(t *testing.T) {
	t.Parallel()

	// The title alone has no kana; the body decides.
	t.Run("detects japanese from content", func(t *testing.T) {
		t.Parallel()

		out, err := RenderHTML(DocumentData{
			Title:   "個人情報保護方針",
			Content: json.RawMessage(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"当社は個人情報を適切に取り扱います。"}]}]}`),
		})
		require.NoError(t, err)
		assert.Contains(t, string(out), `<html lang="ja">`)
	})

	t.Run("defaults to english", func(t *testing.T) {
		t.Parallel()

		out, err := RenderHTML(DocumentData{Title: "Policy"})
		require.NoError(t, err)
		assert.Contains(t, string(out), `<html lang="en">`)
	})

	t.Run("explicit language wins", func(t *testing.T) {
		t.Parallel()

		out, err := RenderHTML(DocumentData{Title: "Policy", Language: "ja"})
		require.NoError(t, err)
		assert.Contains(t, string(out), `<html lang="ja">`)
	})
}

func TestRenderSignaturePageHTMLLanguage(t *testing.T) {
	t.Parallel()

	t.Run("defaults to english", func(t *testing.T) {
		t.Parallel()

		out, err := RenderSignaturePageHTML(SignaturePageData{})
		require.NoError(t, err)
		assert.Contains(t, string(out), `<html lang="en">`)
	})

	t.Run("uses the given language", func(t *testing.T) {
		t.Parallel()

		out, err := RenderSignaturePageHTML(SignaturePageData{
			Signatures: []SignatureData{{SignedBy: "山田 太郎"}},
			Language:   "ja",
		})
		require.NoError(t, err)
		assert.Contains(t, string(out), `<html lang="ja">`)
	})
}
