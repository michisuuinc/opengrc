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

import { render } from "react-email";
import { copyFile, mkdir, writeFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import * as React from "react";

import MailingListUpdates from "../src/MailingListUpdates";
import ConfirmEmail from "../src/ConfirmEmail";
import DocumentExport from "../src/DocumentExport";
import DocumentApproval from "../src/DocumentApproval";
import DocumentSigning from "../src/DocumentSigning";
import FrameworkExport from "../src/FrameworkExport";
import Invitation from "../src/Invitation";
import PasswordReset from "../src/PasswordReset";
import CompliancePortalAccess from "../src/CompliancePortalAccess";
import CompliancePortalDocumentAccessRejected from "../src/CompliancePortalDocumentAccessRejected";
import ElectronicSignatureCertificate from "../src/ElectronicSignatureCertificate";
import MailingListSubscription from "../src/MailingListSubscription";
import MailingListUnsubscription from "../src/MailingListUnsubscription";
import LogExport from "../src/LogExport";
import MagicLink from "../src/MagicLink";
// Fork: Japanese emails (src/ja/, templates/ja/), built into dist/ja/.
import MailingListUpdatesJapanese from "../src/ja/MailingListUpdates";
import ConfirmEmailJapanese from "../src/ja/ConfirmEmail";
import DocumentExportJapanese from "../src/ja/DocumentExport";
import DocumentApprovalJapanese from "../src/ja/DocumentApproval";
import DocumentSigningJapanese from "../src/ja/DocumentSigning";
import FrameworkExportJapanese from "../src/ja/FrameworkExport";
import InvitationJapanese from "../src/ja/Invitation";
import PasswordResetJapanese from "../src/ja/PasswordReset";
import CompliancePortalAccessJapanese from "../src/ja/CompliancePortalAccess";
import CompliancePortalDocumentAccessRejectedJapanese from "../src/ja/CompliancePortalDocumentAccessRejected";
import ElectronicSignatureCertificateJapanese from "../src/ja/ElectronicSignatureCertificate";
import MailingListSubscriptionJapanese from "../src/ja/MailingListSubscription";
import MailingListUnsubscriptionJapanese from "../src/ja/MailingListUnsubscription";
import LogExportJapanese from "../src/ja/LogExport";
import MagicLinkJapanese from "../src/ja/MagicLink";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

type TemplateConfig = {
  name: string;
  render: () => React.ReactElement;
};

const templates: TemplateConfig[] = [
  {
    name: "confirm-email",
    render: () => ConfirmEmail(),
  },
  {
    name: "password-reset",
    render: () => PasswordReset(),
  },
  {
    name: "invitation",
    render: () => Invitation(),
  },
  {
    name: "document-approval",
    render: () => DocumentApproval(),
  },
  {
    name: "document-signing",
    render: () => DocumentSigning(),
  },
  {
    name: "document-export",
    render: () => DocumentExport(),
  },
  {
    name: "framework-export",
    render: () => FrameworkExport(),
  },
  {
    name: "compliance-portal-access",
    render: () => CompliancePortalAccess(),
  },
  {
    name: "compliance-portal-document-access-rejected",
    render: () => CompliancePortalDocumentAccessRejected(),
  },
  {
    name: "magic-link",
    render: () => MagicLink(),
  },
  {
    name: "electronic-signature-certificate",
    render: () => ElectronicSignatureCertificate(),
  },
  {
    name: "mailing-list-subscription",
    render: () => MailingListSubscription(),
  },
  {
    name: "mailing-list-unsubscription",
    render: () => MailingListUnsubscription(),
  },
  {
    name: "mailing-list-updates",
    render: () => MailingListUpdates(),
  },
  {
    name: "log-export",
    render: () => LogExport(),
  },
];

const japaneseTemplates: TemplateConfig[] = [
  {
    name: "mailing-list-updates",
    render: () => MailingListUpdatesJapanese(),
  },
  {
    name: "confirm-email",
    render: () => ConfirmEmailJapanese(),
  },
  {
    name: "document-export",
    render: () => DocumentExportJapanese(),
  },
  {
    name: "document-approval",
    render: () => DocumentApprovalJapanese(),
  },
  {
    name: "document-signing",
    render: () => DocumentSigningJapanese(),
  },
  {
    name: "framework-export",
    render: () => FrameworkExportJapanese(),
  },
  {
    name: "invitation",
    render: () => InvitationJapanese(),
  },
  {
    name: "password-reset",
    render: () => PasswordResetJapanese(),
  },
  {
    name: "compliance-portal-access",
    render: () => CompliancePortalAccessJapanese(),
  },
  {
    name: "compliance-portal-document-access-rejected",
    render: () => CompliancePortalDocumentAccessRejectedJapanese(),
  },
  {
    name: "electronic-signature-certificate",
    render: () => ElectronicSignatureCertificateJapanese(),
  },
  {
    name: "mailing-list-subscription",
    render: () => MailingListSubscriptionJapanese(),
  },
  {
    name: "mailing-list-unsubscription",
    render: () => MailingListUnsubscriptionJapanese(),
  },
  {
    name: "log-export",
    render: () => LogExportJapanese(),
  },
  {
    name: "magic-link",
    render: () => MagicLinkJapanese(),
  },
];

async function buildLanguage(
  list: TemplateConfig[],
  outputDir: string,
  templatesDir: string,
) {
  await mkdir(outputDir, { recursive: true });

  for (const template of list) {
    const html = await render(template.render(), { pretty: true });

    await writeFile(join(outputDir, `${template.name}.html.tmpl`), html);
    await copyFile(
      join(templatesDir, `${template.name}.txt`),
      join(outputDir, `${template.name}.txt.tmpl`),
    );
  }
}

async function build() {
  const outputDir = join(__dirname, "..", "dist");
  const templatesDir = join(__dirname, "..", "templates");
  await mkdir(outputDir, { recursive: true });

  for (const template of templates) {
    const html = await render(template.render(), { pretty: true });

    const htmlPath = join(outputDir, `${template.name}.html.tmpl`);
    const txtSrcPath = join(templatesDir, `${template.name}.txt`);
    const txtDstPath = join(outputDir, `${template.name}.txt.tmpl`);

    await writeFile(htmlPath, html);
    await copyFile(txtSrcPath, txtDstPath);
  }

  await buildLanguage(
    japaneseTemplates,
    join(outputDir, "ja"),
    join(templatesDir, "ja"),
  );
}

build().catch((err) => {
  console.error("Failed to build email templates:", err);
  process.exit(1);
});
