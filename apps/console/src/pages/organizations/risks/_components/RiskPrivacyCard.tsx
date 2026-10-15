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

import { Button, Card, Checkbox } from "@probo/ui";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { graphql, useFragment } from "react-relay";

import type {
  PersonalDataLifecycleStage,
  RiskPrivacyCardFragment$key,
} from "#/__generated__/core/RiskPrivacyCardFragment.graphql";

import { useUpdateRiskPrivacyDetails } from "../_lib/useUpdateRiskPrivacyDetails";

const riskPrivacyCardFragment = graphql`
  fragment RiskPrivacyCardFragment on Risk {
    id
    canUpdate: permission(action: "risk-management:risk:update")
    privacyDetails {
      lifecycleStages
    }
    privacyProcessingActivities {
      id
    }
    organization {
      processingActivities(
        first: 1000
        orderBy: { field: NAME, direction: ASC }
      ) {
        edges {
          node {
            id
            name
          }
        }
      }
    }
  }
`;

// In lifecycle order, as in the PMS rules template (article 16).
const stages: readonly PersonalDataLifecycleStage[] = [
  "COLLECTION",
  "TRANSFER",
  "USE",
  "STORAGE",
  "DISPOSAL",
];

function toggle<T>(values: readonly T[], value: T, checked: boolean): T[] {
  const others = values.filter(v => v !== value);
  return checked ? [...others, value] : others;
}

interface RiskPrivacyCardProps {
  riskKey: RiskPrivacyCardFragment$key;
}

// Items of the Japanese PrivacyMark risk analysis (リスク分析表) that the
// risk form does not cover: the lifecycle stages and the register entries.
export function RiskPrivacyCard({ riskKey }: RiskPrivacyCardProps) {
  const risk = useFragment(riskPrivacyCardFragment, riskKey);

  const storedStages = risk.privacyDetails.lifecycleStages;
  const storedActivityIds = risk.privacyProcessingActivities.map(a => a.id);

  // A new key resets the form to what was stored after a save or a refetch.
  const storedKey = [...storedStages, "|", ...storedActivityIds].join(",");

  return (
    <RiskPrivacyForm
      key={storedKey}
      riskId={risk.id}
      canUpdate={risk.canUpdate}
      storedStages={storedStages}
      storedActivityIds={storedActivityIds}
      activities={risk.organization.processingActivities.edges.map(e => e.node)}
    />
  );
}

interface RiskPrivacyFormProps {
  riskId: string;
  canUpdate: boolean;
  storedStages: readonly PersonalDataLifecycleStage[];
  storedActivityIds: readonly string[];
  activities: readonly { readonly id: string; readonly name: string }[];
}

function RiskPrivacyForm({
  riskId,
  canUpdate,
  storedStages,
  storedActivityIds,
  activities,
}: RiskPrivacyFormProps) {
  const { t } = useTranslation();
  const [updateRiskPrivacy, isSaving] = useUpdateRiskPrivacyDetails();
  const [selectedStages, setSelectedStages] = useState<PersonalDataLifecycleStage[]>([...storedStages]);
  const [selectedActivityIds, setSelectedActivityIds] = useState<string[]>([...storedActivityIds]);

  const disabled = !canUpdate;

  const onSave = async () => {
    await updateRiskPrivacy({
      variables: {
        input: {
          riskId,
          // Store the stages in lifecycle order.
          lifecycleStages: stages.filter(s => selectedStages.includes(s)),
          processingActivityIds: selectedActivityIds,
        },
      },
    }).catch(() => {
      // useMutation already shows the error toast.
    });
  };

  return (
    <Card>
      <div className="p-6 space-y-6">
        <div>
          <h2 className="text-lg font-semibold">
            {t("riskPrivacy.title")}
          </h2>
          <p className="text-sm text-txt-tertiary">
            {t("riskPrivacy.description")}
          </p>
        </div>

        <fieldset className="space-y-2">
          <legend className="text-sm font-medium text-txt-primary mb-2">
            {t("riskPrivacy.fields.lifecycleStages")}
          </legend>
          <div className="flex flex-wrap gap-x-6 gap-y-2">
            {stages.map(stage => (
              <label key={stage} className="flex items-center gap-2 text-sm text-txt-primary">
                <Checkbox
                  checked={selectedStages.includes(stage)}
                  disabled={disabled}
                  onChange={checked => setSelectedStages(toggle(selectedStages, stage, checked))}
                />
                {t(`riskPrivacy.stages.${stage}`)}
              </label>
            ))}
          </div>
        </fieldset>

        <fieldset className="space-y-2">
          <legend className="text-sm font-medium text-txt-primary mb-2">
            {t("riskPrivacy.fields.processingActivities")}
          </legend>
          {activities.length === 0
            ? (
                <p className="text-sm text-txt-secondary">
                  {t("riskPrivacy.emptyProcessingActivities")}
                </p>
              )
            : (
                <div className="max-h-64 overflow-y-auto space-y-2">
                  {activities.map(activity => (
                    <label key={activity.id} className="flex items-center gap-2 text-sm text-txt-primary">
                      <Checkbox
                        checked={selectedActivityIds.includes(activity.id)}
                        disabled={disabled}
                        onChange={checked => setSelectedActivityIds(
                          toggle(selectedActivityIds, activity.id, checked),
                        )}
                      />
                      {activity.name}
                    </label>
                  ))}
                </div>
              )}
        </fieldset>

        {canUpdate && (
          <div className="flex justify-end">
            <Button
              type="button"
              variant="primary"
              disabled={isSaving}
              onClick={() => void onSave()}
            >
              {isSaving
                ? t("riskPrivacy.actions.saving")
                : t("riskPrivacy.actions.save")}
            </Button>
          </div>
        )}
      </div>
    </Card>
  );
}
