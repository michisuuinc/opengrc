// Copyright (c) 2025-2026 Probo Inc <hello@probo.com>.
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

import { Button, Section, Text } from 'react-email';
import * as React from 'react';
import EmailLayout, { bodyText, button, buttonContainer, footerText } from '../components/EmailLayout';

const documentList = {
  ...bodyText,
  paddingLeft: '20px',
};

export const DocumentSigningJapanese = () => {
  return (
    <EmailLayout lang="ja" subject={'【要対応】{{.OrganizationName}} の文書の確認と署名のお願い'}>
      <Text style={bodyText}>
        所属する <strong>{'{{.OrganizationName}}'}</strong> から、確認と署名が必要な文書が共有されています。
      </Text>

      <ul style={documentList} dangerouslySetInnerHTML={{ __html: '{{range .Documents}}<li><a href="{{.URL}}">{{.Title}}</a> （{{.Type}}）</li>{{end}}' }} />

      <Text style={bodyText}>
        社内の規程に沿って、次のボタンから文書を確認し、署名してください。
      </Text>

      <Section style={buttonContainer}>
        <Button style={button} href={'{{.SigningUrl}}'}>
          文書を確認して署名する
        </Button>
      </Section>

      <Text style={bodyText}>
        ご不明な点は、社内の情報セキュリティの担当者にお問い合わせください。
      </Text>

      <Text style={footerText}>
        この手続きは、<strong>{'{{.OrganizationName}}'}</strong> に代わって Probo が安全に管理しています。
      </Text>
      <Text style={footerText}>
        よろしくお願いいたします。
      </Text>
    </EmailLayout>
  );
};

export default DocumentSigningJapanese;
