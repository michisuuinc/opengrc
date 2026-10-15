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
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/docgen"
)

// certificateLabels holds the headings and labels of the certificate of
// completion. The Japanese ones keep the English term in parentheses, since
// the certificate is evidence: the consent text and the ERSD stay in the
// English original that the signer was shown, and OriginalNote says so (it
// is empty in English). The seal verification steps also stay in English,
// since they name the exact fields to hash.
type certificateLabels struct {
	Title                 string
	Status                string
	Completed             string
	Subject               string
	DocumentType          string
	SignatureID           string
	OrganizationID        string
	FileID                string
	FileHash              string
	Signer                string
	Name                  string
	Email                 string
	SignedAt              string
	SecurityLevel         string
	SecurityLevelValue    string
	IPAddress             string
	UserAgent             string
	ERSDConsent           string
	Accepted              string
	ElectronicConsent     string
	SummaryEvents         string
	EventStatus           string
	Timestamps            string
	IntegrityVerification string
	Seal                  string
	TSAAuthority          string
	TSATimestamp          string
	TSASerialNumber       string
	SealVerification      string
	ERSD                  string
	OriginalNote          string
	Footer                string
	EventTypes            map[coredata.ElectronicSignatureEventType]string
	EventSources          map[coredata.ElectronicSignatureEventSource]string
	DocumentTypes         map[coredata.ElectronicSignatureDocumentType]string
}

var (
	englishCertificateLabels = certificateLabels{
		Title:                 "Certificate Of Completion",
		Status:                "Status",
		Completed:             "Completed",
		Subject:               "Subject",
		DocumentType:          "Document Type",
		SignatureID:           "Signature Id",
		OrganizationID:        "Organization Id",
		FileID:                "File Id",
		FileHash:              "File Hash (SHA-256)",
		Signer:                "Signer",
		Name:                  "Name",
		Email:                 "Email",
		SignedAt:              "Signed At",
		SecurityLevel:         "Security Level",
		SecurityLevelValue:    "Email, Account Authentication",
		IPAddress:             "IP Address",
		UserAgent:             "User Agent",
		ERSDConsent:           "ERSD Consent",
		Accepted:              "Accepted",
		ElectronicConsent:     "Electronic Consent",
		SummaryEvents:         "Signing Summary Events",
		EventStatus:           "Status",
		Timestamps:            "Timestamps",
		IntegrityVerification: "Integrity Verification",
		Seal:                  "Seal",
		TSAAuthority:          "TSA Authority",
		TSATimestamp:          "TSA Timestamp",
		TSASerialNumber:       "TSA Serial Number",
		SealVerification:      "Seal Verification",
		ERSD:                  "Electronic Record and Signature Disclosure",
		Footer: "This certificate was generated automatically by Probo. The integrity " +
			"seal and TSA timestamp provide tamper-evident proof of the signing " +
			"event. All information required to independently recompute and verify " +
			"the seal is included in this document.",
		EventTypes:    map[coredata.ElectronicSignatureEventType]string{},
		EventSources:  map[coredata.ElectronicSignatureEventSource]string{},
		DocumentTypes: map[coredata.ElectronicSignatureDocumentType]string{},
	}

	japaneseCertificateLabels = certificateLabels{
		Title:                 "完了証明書（Certificate Of Completion）",
		Status:                "状態（Status）",
		Completed:             "完了（Completed）",
		Subject:               "件名（Subject）",
		DocumentType:          "文書の種類（Document Type）",
		SignatureID:           "署名 ID（Signature Id）",
		OrganizationID:        "組織 ID（Organization Id）",
		FileID:                "ファイル ID（File Id）",
		FileHash:              "ファイルのハッシュ値（File Hash, SHA-256）",
		Signer:                "署名者（Signer）",
		Name:                  "氏名（Name）",
		Email:                 "メールアドレス（Email）",
		SignedAt:              "署名日時・UTC（Signed At）",
		SecurityLevel:         "本人確認の水準（Security Level）",
		SecurityLevelValue:    "メール、アカウント認証（Email, Account Authentication）",
		IPAddress:             "IP アドレス（IP Address）",
		UserAgent:             "ユーザーエージェント（User Agent）",
		ERSDConsent:           "電子記録・電子署名の開示への同意（ERSD Consent）",
		Accepted:              "同意済み（Accepted）",
		ElectronicConsent:     "電子署名への同意（Electronic Consent）",
		SummaryEvents:         "署名の経過（Signing Summary Events）",
		EventStatus:           "記録元（Status）",
		Timestamps:            "日時・UTC（Timestamps）",
		IntegrityVerification: "完全性の検証（Integrity Verification）",
		Seal:                  "封印（Seal）",
		TSAAuthority:          "タイムスタンプ局（TSA Authority）",
		TSATimestamp:          "タイムスタンプ・UTC（TSA Timestamp）",
		TSASerialNumber:       "タイムスタンプのシリアル番号（TSA Serial Number）",
		SealVerification:      "封印の検証方法（Seal Verification）",
		ERSD:                  "電子記録・電子署名の開示（Electronic Record and Signature Disclosure）",
		OriginalNote:          "以下は、署名者に表示された英語の原文です。",
		Footer: "この証明書は Probo が自動で作成しました。封印とタイムスタンプにより、" +
			"署名後に改ざんされていないことを確認できます。封印を検証するための情報は、" +
			"すべてこの証明書に記載しています。",
		EventTypes: map[coredata.ElectronicSignatureEventType]string{
			coredata.ElectronicSignatureEventTypeDocumentViewed:       "文書を表示",
			coredata.ElectronicSignatureEventTypeConsentGiven:         "電子署名に同意",
			coredata.ElectronicSignatureEventTypeFullNameTyped:        "氏名を入力",
			coredata.ElectronicSignatureEventTypeSignatureAccepted:    "署名を受付",
			coredata.ElectronicSignatureEventTypeSignatureCompleted:   "署名を完了",
			coredata.ElectronicSignatureEventTypeSealComputed:         "封印を計算",
			coredata.ElectronicSignatureEventTypeTimestampRequested:   "タイムスタンプを取得",
			coredata.ElectronicSignatureEventTypeCertificateGenerated: "証明書を作成",
			coredata.ElectronicSignatureEventTypeProcessingError:      "処理エラー",
		},
		EventSources: map[coredata.ElectronicSignatureEventSource]string{
			coredata.ElectronicSignatureEventSourceClient: "署名者の端末",
			coredata.ElectronicSignatureEventSourceServer: "サーバー",
		},
		DocumentTypes: map[coredata.ElectronicSignatureDocumentType]string{
			coredata.ElectronicSignatureDocumentTypeNDA:                      "秘密保持契約",
			coredata.ElectronicSignatureDocumentTypeDPA:                      "データ処理契約",
			coredata.ElectronicSignatureDocumentTypeMSA:                      "基本契約",
			coredata.ElectronicSignatureDocumentTypeSOW:                      "作業範囲記述書",
			coredata.ElectronicSignatureDocumentTypeSLA:                      "サービスレベル契約",
			coredata.ElectronicSignatureDocumentTypeTOS:                      "利用規約",
			coredata.ElectronicSignatureDocumentTypePrivacyPolicy:            "プライバシーポリシー",
			coredata.ElectronicSignatureDocumentTypeGovernance:               "ガバナンス",
			coredata.ElectronicSignatureDocumentTypePolicy:                   "方針",
			coredata.ElectronicSignatureDocumentTypeProcedure:                "手順",
			coredata.ElectronicSignatureDocumentTypePlan:                     "計画",
			coredata.ElectronicSignatureDocumentTypeRegister:                 "台帳",
			coredata.ElectronicSignatureDocumentTypeRecord:                   "記録",
			coredata.ElectronicSignatureDocumentTypeReport:                   "報告書",
			coredata.ElectronicSignatureDocumentTypeTemplate:                 "テンプレート",
			coredata.ElectronicSignatureDocumentTypeStatementOfApplicability: "適用宣言書",
			coredata.ElectronicSignatureDocumentTypeOther:                    "その他",
		},
	}
)

// certificateLanguage is the language of the certificate: Japanese when the
// signed document's name is, English otherwise.
func certificateLanguage(signature *coredata.ElectronicSignature) string {
	if signature.DocumentName == nil {
		return "en"
	}

	return docgen.DetectTitleLanguage(*signature.DocumentName)
}

func certificateLabelsFor(language string) certificateLabels {
	if docgen.IsJapanese(language) {
		return japaneseCertificateLabels
	}

	return englishCertificateLabels
}

// labelOr returns the label of a value, or the fallback when there is none.
func labelOr[K comparable](labels map[K]string, value K, fallback string) string {
	if l, ok := labels[value]; ok {
		return l
	}

	return fallback
}
