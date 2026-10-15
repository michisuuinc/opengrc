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

import { Button, Card, Field, Option, Select } from "@probo/ui";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { graphql, useLazyLoadQuery } from "react-relay";

import type { OrganizationEmailLanguageSectionMutation } from "#/__generated__/core/OrganizationEmailLanguageSectionMutation.graphql";
import type { OrganizationEmailLanguageSectionQuery } from "#/__generated__/core/OrganizationEmailLanguageSectionQuery.graphql";
import { useMutation } from "#/lib/relay/useMutation";

const organizationEmailLanguageSectionQuery = graphql`
  query OrganizationEmailLanguageSectionQuery($organizationId: ID!) {
    node(id: $organizationId) {
      ... on Organization {
        id
        emailLanguage
        canUpdate: permission(action: "core:organization:update")
      }
    }
  }
`;

const updateOrganizationEmailLanguageMutation = graphql`
  mutation OrganizationEmailLanguageSectionMutation(
    $input: UpdateOrganizationEmailLanguageInput!
  ) {
    updateOrganizationEmailLanguage(input: $input) {
      organization {
        id
        emailLanguage
      }
    }
  }
`;

const languages = ["ja", "en"] as const;

// The language of emails to people whose own language is unknown, such as
// invited members and external signers. Members who saved a language get
// theirs.
export function OrganizationEmailLanguageSection({ organizationId }: { organizationId: string }) {
  const { t } = useTranslation();
  const data = useLazyLoadQuery<OrganizationEmailLanguageSectionQuery>(
    organizationEmailLanguageSectionQuery,
    { organizationId },
  );
  const organization = data.node;
  const [language, setLanguage] = useState<string>(organization?.emailLanguage ?? "en");
  const [update, isSaving] = useMutation<OrganizationEmailLanguageSectionMutation>(
    updateOrganizationEmailLanguageMutation,
    {
      successMessage: t("organizationEmailLanguage.messages.saved"),
      errorToast: t("organizationEmailLanguage.errors.save"),
    },
  );

  if (!organization?.id) {
    return null;
  }

  const stored = organization.emailLanguage ?? "en";

  return (
    <Card padded>
      <div className="space-y-4">
        <div>
          <h2 className="text-lg font-semibold">{t("organizationEmailLanguage.title")}</h2>
          <p className="text-sm text-txt-tertiary">{t("organizationEmailLanguage.description")}</p>
        </div>
        <div className="flex flex-wrap items-end gap-4">
          <Field label={t("organizationEmailLanguage.fields.language")}>
            <Select
              value={language}
              onValueChange={setLanguage}
              disabled={!organization.canUpdate}
              aria-label={t("organizationEmailLanguage.fields.language")}
            >
              {languages.map(l => (
                <Option key={l} value={l}>{t(`organizationEmailLanguage.languages.${l}`)}</Option>
              ))}
            </Select>
          </Field>
          {organization.canUpdate && (
            <Button
              type="button"
              variant="primary"
              disabled={isSaving || language === stored}
              onClick={() => {
                void update({
                  variables: { input: { organizationId: organization.id!, language } },
                }).catch(() => {
                  // useMutation already shows the error toast.
                });
              }}
            >
              {t("organizationEmailLanguage.actions.save")}
            </Button>
          )}
        </div>
      </div>
    </Card>
  );
}
