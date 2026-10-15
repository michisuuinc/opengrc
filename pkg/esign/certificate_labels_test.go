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

package esign

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.probo.inc/probo/pkg/coredata"
)

func renderTestCertificate(t *testing.T, language string) string {
	t.Helper()

	labels := certificateLabelsFor(language)
	data := certificateData{
		Language:         language,
		L:                labels,
		SignatureID:      "sig",
		DocumentType:     "POLICY",
		DocumentTypeName: labelOr(labels.DocumentTypes, coredata.ElectronicSignatureDocumentTypePolicy, "Policy"),
		ConsentText:      "I consent to sign this document electronically.",
		SealVersion:      1,
		TSAAuthority:     "TSA",
		TSATime:          "2026-04-01T00:00:00Z",
		Events: []certificateEvent{{
			EventType:  labelOr(labels.EventTypes, coredata.ElectronicSignatureEventTypeConsentGiven, "CONSENT_GIVEN"),
			Source:     labelOr(labels.EventSources, coredata.ElectronicSignatureEventSourceClient, "CLIENT"),
			OccurredAt: "2026-04-01T00:00:00Z",
		}},
	}

	var buf bytes.Buffer
	require.NoError(t, certificateTemplate.Execute(&buf, data))

	return buf.String()
}

func TestCertificateJapanese(t *testing.T) {
	t.Parallel()

	html := renderTestCertificate(t, "ja")

	assert.Contains(t, html, `<html lang="ja">`)
	for _, want := range []string{"完了証明書", "署名者（Signer）", "方針", "電子署名に同意", "署名者の端末", "封印の検証方法", "以下は、署名者に表示された英語の原文です。", "Probo が自動で作成しました"} {
		assert.Contains(t, html, want)
	}
	// The legal texts stay in the English original.
	assert.Contains(t, html, "I consent to sign this document electronically.")
	assert.Contains(t, html, "Withdrawing your consent")
}

func TestCertificateEnglishUnchanged(t *testing.T) {
	t.Parallel()

	html := renderTestCertificate(t, "en")

	assert.Contains(t, html, `<html lang="en">`)
	for _, want := range []string{"Certificate Of Completion", "<td>Signer</td>", "CONSENT_GIVEN", "CLIENT", "This certificate was generated automatically by Probo."} {
		assert.Contains(t, html, want)
	}
	assert.NotContains(t, html, "original-note\">")
}

func TestCertificateLanguage(t *testing.T) {
	t.Parallel()

	name := "個人情報保護方針"
	assert.Equal(t, "ja", certificateLanguage(&coredata.ElectronicSignature{DocumentName: &name}))

	english := "Privacy Policy"
	assert.Equal(t, "en", certificateLanguage(&coredata.ElectronicSignature{DocumentName: &english}))
	assert.Equal(t, "en", certificateLanguage(&coredata.ElectronicSignature{}))
}
