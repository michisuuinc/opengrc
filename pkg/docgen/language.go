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

import "unicode"

// DetectLanguage returns the BCP 47 tag for the <html lang> attribute of a
// generated PDF. Headless Chrome picks CJK glyphs from the document language,
// so Japanese text in a page tagged "en" can fall back to Chinese glyph
// variants. Documents have no language field, so any kana in the given texts
// means Japanese; kanji alone is not enough because it is shared with Chinese.
func DetectLanguage(texts ...string) string {
	for _, text := range texts {
		for _, r := range text {
			if unicode.In(r, unicode.Hiragana, unicode.Katakana) {
				return "ja"
			}
		}
	}

	return "en"
}

// DetectTitleLanguage is DetectLanguage for a short title, the only text
// some outputs have. Japanese titles are often kanji only, such as
// 個人情報保護方針, so kanji counts as Japanese here too. Chinese titles are
// then taken as Japanese, which only changes the labels around them.
func DetectTitleLanguage(title string) string {
	for _, r := range title {
		if unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han) {
			return "ja"
		}
	}

	return "en"
}
