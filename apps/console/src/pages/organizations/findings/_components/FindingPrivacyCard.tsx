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

import { Button, Card, Field, Input, Option, Select } from "@probo/ui";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { graphql, useFragment } from "react-relay";

import type {
  FindingPrivacyCardFragment$key,
  FindingPrivacyCategory,
} from "#/__generated__/core/FindingPrivacyCardFragment.graphql";

import { useUpdateFindingPrivacyDetails } from "../_lib/useUpdateFindingPrivacyDetails";

const findingPrivacyCardFragment = graphql`
  fragment FindingPrivacyCardFragment on Finding {
    id
    canUpdate: permission(action: "core:finding:update")
    privacyDetails {
      category
      effectiveCategory
      basis
    }
  }
`;

// In the order of the internal audit procedure (template 17, article 7).
const categories: readonly FindingPrivacyCategory[] = [
  "NONCONFORMITY",
  "OBSERVATION",
  "IMPROVEMENT_OPPORTUNITY",
];

// The select cannot hold an empty value, so "follow the kind" gets its own.
const followKind = "FOLLOW_KIND";

interface FindingPrivacyCardProps {
  findingKey: FindingPrivacyCardFragment$key;
}

// Items of the Japanese PrivacyMark internal audit report (内部監査報告書)
// that the finding form does not cover: the category and the basis.
export function FindingPrivacyCard({ findingKey }: FindingPrivacyCardProps) {
  const finding = useFragment(findingPrivacyCardFragment, findingKey);
  const { category, effectiveCategory, basis } = finding.privacyDetails;

  // A new key resets the form to what was stored after a save or a refetch.
  return (
    <FindingPrivacyForm
      key={`${category ?? ""}|${basis}`}
      findingId={finding.id}
      canUpdate={finding.canUpdate}
      storedCategory={category ?? null}
      effectiveCategory={effectiveCategory ?? null}
      storedBasis={basis}
    />
  );
}

interface FindingPrivacyFormProps {
  findingId: string;
  canUpdate: boolean;
  storedCategory: FindingPrivacyCategory | null;
  effectiveCategory: FindingPrivacyCategory | null;
  storedBasis: string;
}

function FindingPrivacyForm({
  findingId,
  canUpdate,
  storedCategory,
  effectiveCategory,
  storedBasis,
}: FindingPrivacyFormProps) {
  const { t } = useTranslation();
  const [updateFindingPrivacy, isSaving] = useUpdateFindingPrivacyDetails();
  const [category, setCategory] = useState<string>(storedCategory ?? followKind);
  const [basis, setBasis] = useState(storedBasis);

  const followKindLabel = effectiveCategory
    ? t("findingPrivacy.followKindWith", {
        category: t(`findingPrivacy.categories.${effectiveCategory}`),
      })
    : t("findingPrivacy.followKindNone");

  const onSave = async () => {
    await updateFindingPrivacy({
      variables: {
        input: {
          findingId,
          category: category === followKind ? null : category as FindingPrivacyCategory,
          basis: basis.trim(),
        },
      },
    }).catch(() => {
      // useMutation already shows the error toast.
    });
  };

  return (
    <Card padded>
      <div className="space-y-4">
        <div>
          <h2 className="text-lg font-semibold">
            {t("findingPrivacy.title")}
          </h2>
          <p className="text-sm text-txt-tertiary">
            {t("findingPrivacy.description")}
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Field label={t("findingPrivacy.fields.category")}>
            <Select
              value={category}
              onValueChange={setCategory}
              disabled={!canUpdate}
            >
              <Option value={followKind}>{followKindLabel}</Option>
              {categories.map(c => (
                <Option key={c} value={c}>
                  {t(`findingPrivacy.categories.${c}`)}
                </Option>
              ))}
            </Select>
          </Field>

          <Field name="finding-privacy-basis" label={t("findingPrivacy.fields.basis")}>
            <Input
              id="finding-privacy-basis"
              value={basis}
              onChange={e => setBasis(e.target.value)}
              placeholder={t("findingPrivacy.placeholders.basis")}
              disabled={!canUpdate}
            />
          </Field>
        </div>

        {canUpdate && (
          <div className="flex justify-end">
            <Button
              type="button"
              variant="primary"
              disabled={isSaving}
              onClick={() => void onSave()}
            >
              {isSaving
                ? t("findingPrivacy.actions.saving")
                : t("findingPrivacy.actions.save")}
            </Button>
          </div>
        )}
      </div>
    </Card>
  );
}
