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

export const InvitationJapanese = () => {
  return (
    <EmailLayout lang="ja" subject={'{{.OrganizationName}} から Probo への招待が届いています'}>
      <Text style={bodyText}>
        <strong>{'{{.OrganizationName}}'}</strong> から、Probo への招待が届いています。次のボタンからアカウントを有効にしてください。
      </Text>

      <Section style={buttonContainer}>
        <Button style={button} href={'{{.InvitationUrl}}'}>
          アカウントを有効にする
        </Button>
      </Section>

      <Text style={footerText}>
        参加しない場合は、このメールを破棄してください。
      </Text>
    </EmailLayout>
  );
};

export default InvitationJapanese;
