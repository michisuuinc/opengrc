// Copyright (c) 2026 Probo Inc <hello@probo.com>.
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

import { Button, Hr, Link, Section, Text } from "react-email";
import * as React from "react";
import EmailLayout, {
  bodyText,
  button,
  buttonContainer,
  footerText,
} from "../components/EmailLayout";

export const MailingListSubscriptionJapanese = () => {
  return (
    <EmailLayout lang="ja"
      subject={`${"{{.OrganizationName}}"} – アップデートの購読の確認`}
    >
      <Text style={bodyText}>
        <strong>{"{{.OrganizationName}}"}</strong> のコンプライアンスのアップデートの購読が申し込まれました。
      </Text>

      <Text style={bodyText}>
        次のボタンから、購読を確定してください。
      </Text>

      <Section style={buttonContainer}>
        <Button style={button} href={"{{.ConfirmURL}}"}>
          購読を確定する
        </Button>
      </Section>

      <Hr style={{ borderColor: "#ecefec", margin: "8px 0 20px" }} />

      <Text style={{ ...footerText, textAlign: "center" }}>
        お心当たりのない場合や、今後このメールが不要な場合は、
        <Link
          href={"{{.UnsubscribeURL}}"}
          style={{ color: "#374151", textDecoration: "underline", fontWeight: "500" }}
        >
          こちらから購読を解除できます。
        </Link>
      </Text>
    </EmailLayout>
  );
};

export default MailingListSubscriptionJapanese;
