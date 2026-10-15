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

package emails

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type renderedEmail struct {
	name    string
	subject string
	text    string
	html    string
}

// renderAll renders every email with the presenter.
func renderAll(t *testing.T, p *Presenter) []renderedEmail {
	t.Helper()

	ctx := context.Background()
	docs := []DocumentSummary{{Title: "個人情報保護方針", Type: "方針", URL: "https://example.com/d"}}
	out := []renderedEmail{}

	add := func(name, subject, text string, html *string, err error) {
		require.NoError(t, err, name)
		require.NotNil(t, html, name)
		out = append(out, renderedEmail{name, subject, text, *html})
	}

	s, tx, h, err := p.RenderConfirmEmail(ctx, "/confirm", "tok")
	add("confirm-email", s, tx, h, err)
	s, tx, h, err = p.RenderPasswordReset(ctx, "/reset", "tok")
	add("password-reset", s, tx, h, err)
	s, tx, h, err = p.RenderInvitation(ctx, "/invite", "tok", "サンプル株式会社")
	add("invitation", s, tx, h, err)
	s, tx, h, err = p.RenderDocumentApproval(ctx, "https://example.com/a", "サンプル株式会社", docs)
	add("document-approval", s, tx, h, err)
	s, tx, h, err = p.RenderDocumentSigning(ctx, "https://example.com/s", "サンプル株式会社", docs)
	add("document-signing", s, tx, h, err)
	s, tx, h, err = p.RenderDocumentExport(ctx, "https://example.com/x")
	add("document-export", s, tx, h, err)
	s, tx, h, err = p.RenderFrameworkExport(ctx, "https://example.com/x")
	add("framework-export", s, tx, h, err)
	s, tx, h, err = p.RenderLogExport(ctx, "https://example.com/x", false)
	add("log-export", s, tx, h, err)
	s, tx, h, err = p.RenderLogExport(ctx, "https://example.com/x", true)
	add("scim-export", s, tx, h, err)
	s, tx, h, err = p.RenderCompliancePortalAccess(ctx, "サンプル株式会社")
	add("compliance-portal-access", s, tx, h, err)
	s, tx, h, err = p.RenderCompliancePortalDocumentAccessRejected(ctx, []string{"a.pdf"}, "サンプル株式会社")
	add("compliance-portal-document-access-rejected", s, tx, h, err)
	s, tx, h, err = p.RenderMagicLink(ctx, "/magic", "tok", 15*time.Minute, "サンプル株式会社")
	add("magic-link", s, tx, h, err)
	tx, h, err = p.RenderElectronicSignatureCertificate(ctx, "評価 一郎", "個人情報保護方針", "署名済みの文書")
	add("electronic-signature-certificate", "署名済みの文書", tx, h, err)
	s, tx, h, err = p.RenderMailingListSubscription(ctx, "サンプル株式会社", "https://example.com/c", "https://example.com/u")
	add("mailing-list-subscription", s, tx, h, err)
	s, tx, h, err = p.RenderMailingListUnsubscription(ctx, "サンプル株式会社")
	add("mailing-list-unsubscription", s, tx, h, err)
	s, tx, h, err = p.RenderMailingListNews(ctx, "サンプル株式会社", "お知らせ", "本文", "https://example.com/p", "https://example.com/u")
	add("mailing-list-updates", s, tx, h, err)

	return out
}

func hasKana(s string) bool {
	for _, r := range s {
		if r >= 0x3040 && r <= 0x30ff {
			return true
		}
	}

	return false
}

func TestJapaneseEmails(t *testing.T) {
	t.Parallel()

	p := NewPresenter("https://example.com", "山田 太郎").WithLanguage("ja-JP")

	for _, e := range renderAll(t, p) {
		assert.True(t, hasKana(e.subject), "%s subject %q", e.name, e.subject)
		assert.Contains(t, e.text, "山田 太郎 様", e.name)
		assert.Contains(t, e.html, `lang="ja"`, e.name)
		assert.Regexp(t, `山田 太郎\s*様`, e.html, e.name)
		assert.NotContains(t, e.text, "Hi ", e.name)
		assert.NotContains(t, e.text, "Click the", e.name)
		assert.NotContains(t, e.html, "Click the", e.name)
	}
}

func TestEnglishEmailsUnchanged(t *testing.T) {
	t.Parallel()

	for _, language := range []string{"", "en", "fr"} {
		p := NewPresenter("https://example.com", "Taro").WithLanguage(language)

		for _, e := range renderAll(t, p) {
			assert.False(t, strings.Contains(e.subject, "要対応"), e.name)
			assert.Contains(t, e.text, "Hi Taro,", e.name)
			assert.Contains(t, e.html, `lang="en"`, e.name)
		}

		subject, _, _, err := p.RenderInvitation(context.Background(), "/invite", "tok", "Acme")
		require.NoError(t, err)
		assert.Equal(t, "Invitation to join Acme on Probo", subject)
	}
}

func TestResolveLanguage(t *testing.T) {
	t.Parallel()

	ja, jaJP, fr, empty := "ja", "ja-JP", "fr", ""

	assert.Equal(t, "ja", ResolveLanguage(nil, &jaJP))
	assert.Equal(t, "ja", ResolveLanguage(&empty, &ja))
	assert.Equal(t, "en", ResolveLanguage(&fr, &ja), "a chosen language wins over the organization default")
	assert.Equal(t, "en", ResolveLanguage(nil, &empty))
	assert.Equal(t, "en", ResolveLanguage())
}

func TestPreferredLanguage(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ja", *PreferredLanguage("ja,en-US;q=0.9"))
	assert.Equal(t, "en-US", *PreferredLanguage(" en-US;q=0.8, ja"))
	assert.Nil(t, PreferredLanguage(""))
	assert.Nil(t, PreferredLanguage("*"))
}
