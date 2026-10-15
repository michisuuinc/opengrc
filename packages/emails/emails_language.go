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
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"
)

// The emails come in English and Japanese. A Presenter with Language "ja"
// renders the Japanese templates (dist/ja/, built from src/ja/ and
// templates/ja/) and subjects; any other language renders English.

const (
	LanguageEnglish  = "en"
	LanguageJapanese = "ja"
)

type localizedTemplates struct {
	text *texttemplate.Template
	html *htmltemplate.Template
}

// japaneseTemplates maps each English text template to its Japanese pair.
var japaneseTemplates = map[*texttemplate.Template]localizedTemplates{
	confirmEmailTextTemplate:                           japanese("confirm-email"),
	passwordResetTextTemplate:                          japanese("password-reset"),
	invitationTextTemplate:                             japanese("invitation"),
	documentApprovalTextTemplate:                       japanese("document-approval"),
	documentSigningTextTemplate:                        japanese("document-signing"),
	documentExportTextTemplate:                         japanese("document-export"),
	frameworkExportTextTemplate:                        japanese("framework-export"),
	compliancePortalAccessTextTemplate:                 japanese("compliance-portal-access"),
	compliancePortalDocumentAccessRejectedTextTemplate: japanese("compliance-portal-document-access-rejected"),
	magicLinkTextTemplate:                              japanese("magic-link"),
	electronicSignatureCertificateTextTemplate:         japanese("electronic-signature-certificate"),
	mailingListSubscriptionTextTemplate:                japanese("mailing-list-subscription"),
	mailingListUnsubscriptionTextTemplate:              japanese("mailing-list-unsubscription"),
	mailingListUpdatesTextTemplate:                     japanese("mailing-list-updates"),
	logExportTextTemplate:                              japanese("log-export"),
}

// japaneseSubjects maps each English subject format to its Japanese one.
// The format verbs stay in the same order.
var japaneseSubjects = map[string]string{
	subjectConfirmEmail:                           "メールアドレスの確認",
	subjectPasswordReset:                          "パスワードの再設定",
	subjectInvitation:                             "%s から Probo への招待が届いています",
	subjectDocumentApproval:                       "【要対応】%s の文書の確認と承認のお願い",
	subjectDocumentSigning:                        "【要対応】%s の文書の確認と署名のお願い",
	subjectDocumentExport:                         "文書のエクスポートが完了しました",
	subjectFrameworkExport:                        "フレームワークのエクスポートが完了しました",
	subjectCompliancePortalAccess:                 "コンプライアンスポータルへの招待 - %s",
	subjectCompliancePortalDocumentAccessRejected: "コンプライアンスポータルの文書の閲覧申請について - %s",
	subjectMagicLink:                              "%s へのログイン",
	subjectMailingListSubscription:                "%s – アップデートの購読の確認",
	subjectMailingListUnsubscription:              "%s – アップデートの購読を解除しました",
	subjectMailingListUpdates:                     "%s – %s",
	subjectAuditLogExport:                         "監査ログのエクスポートが完了しました",
	subjectSCIMEventExport:                        "SCIM イベントのエクスポートが完了しました",
}

var japaneseLabels = map[string]string{
	"audit log":  "監査ログ",
	"SCIM event": "SCIM イベント",
}

func japanese(name string) localizedTemplates {
	return localizedTemplates{
		text: texttemplate.Must(texttemplate.ParseFS(Templates, "dist/ja/"+name+".txt.tmpl")),
		html: htmltemplate.Must(htmltemplate.ParseFS(Templates, "dist/ja/"+name+".html.tmpl")),
	}
}

// NormalizeLanguage returns the email language for a BCP 47 tag or a SCIM
// language value: "ja" for Japanese ("ja", "ja-JP", "ja_JP"), "en" for any
// other language, and "" when the tag is empty.
func NormalizeLanguage(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	switch {
	case tag == "":
		return ""
	case tag == LanguageJapanese, strings.HasPrefix(tag, "ja-"), strings.HasPrefix(tag, "ja_"):
		return LanguageJapanese
	default:
		return LanguageEnglish
	}
}

// ResolveLanguage returns the email language from the recipient's known
// preferences, most specific first: the first preference set decides, so a
// recipient who chose French gets English, not the organization default.
// With no preference at all, the email is in English.
func ResolveLanguage(preferences ...*string) string {
	for _, p := range preferences {
		if p == nil {
			continue
		}

		if language := NormalizeLanguage(*p); language != "" {
			return language
		}
	}

	return LanguageEnglish
}

// WithLanguage sets the language of the emails the presenter renders and
// returns the presenter.
func (p *Presenter) WithLanguage(language string) *Presenter {
	p.Language = NormalizeLanguage(language)

	return p
}

func (p *Presenter) isJapanese() bool {
	return p.Language == LanguageJapanese
}

// renderLocalized renders the English templates or their translation.
func (p *Presenter) renderLocalized(
	textTemplate *texttemplate.Template,
	htmlTemplate *htmltemplate.Template,
	data any,
) (string, *string, error) {
	if p.isJapanese() {
		if t, ok := japaneseTemplates[textTemplate]; ok {
			return renderEmail(t.text, t.html, data)
		}
	}

	return renderEmail(textTemplate, htmlTemplate, data)
}

// subject returns the subject format in the presenter's language.
func (p *Presenter) subject(english string) string {
	if p.isJapanese() {
		if s, ok := japaneseSubjects[english]; ok {
			return s
		}
	}

	return english
}

// label returns a short text inserted into a template in the presenter's
// language.
func (p *Presenter) label(english string) string {
	if p.isJapanese() {
		if l, ok := japaneseLabels[english]; ok {
			return l
		}
	}

	return english
}

// PreferredLanguage returns the first language of an Accept-Language
// header, or nil when there is none. Pass it to ResolveLanguage after the
// recipient's saved language.
func PreferredLanguage(acceptLanguage string) *string {
	first, _, _ := strings.Cut(acceptLanguage, ",")
	tag, _, _ := strings.Cut(first, ";")
	tag = strings.TrimSpace(tag)

	if tag == "" || tag == "*" {
		return nil
	}

	return &tag
}
