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

import { Button, Card, Field, Label, Textarea } from "@probo/ui";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { graphql, useFragment } from "react-relay";

import type { ProcessingActivityLedgerCardFragment$key } from "#/__generated__/core/ProcessingActivityLedgerCardFragment.graphql";

import { useUpdateProcessingActivityLedgerDetails } from "../_lib/useUpdateProcessingActivityLedgerDetails";

const processingActivityLedgerCardFragment = graphql`
  fragment ProcessingActivityLedgerCardFragment on ProcessingActivity {
    id
    canUpdate: permission(action: "core:processing-activity:update")
    ledgerDetails {
      department
      approximateRecordCount
      collectionMethod
      storageLocation
      accessScope
      disposalMethod
    }
  }
`;

// Same limit as the server (TitleMaxLength).
const maxLength = 1000;

const textFields = [
  "collectionMethod",
  "storageLocation",
  "accessScope",
  "disposalMethod",
] as const;

type LedgerFormValues = {
  department: string;
  approximateRecordCount: string;
  collectionMethod: string;
  storageLocation: string;
  accessScope: string;
  disposalMethod: string;
};

type LedgerDetails = {
  readonly department: string | null | undefined;
  readonly approximateRecordCount: string | null | undefined;
  readonly collectionMethod: string | null | undefined;
  readonly storageLocation: string | null | undefined;
  readonly accessScope: string | null | undefined;
  readonly disposalMethod: string | null | undefined;
};

function toFormValues(details: LedgerDetails): LedgerFormValues {
  return {
    department: details.department ?? "",
    approximateRecordCount: details.approximateRecordCount ?? "",
    collectionMethod: details.collectionMethod ?? "",
    storageLocation: details.storageLocation ?? "",
    accessScope: details.accessScope ?? "",
    disposalMethod: details.disposalMethod ?? "",
  };
}

interface ProcessingActivityLedgerCardProps {
  processingActivityKey: ProcessingActivityLedgerCardFragment$key;
}

// Register items of the Japanese PrivacyMark register (個人情報管理台帳)
// that the processing activity form does not cover.
export function ProcessingActivityLedgerCard({
  processingActivityKey,
}: ProcessingActivityLedgerCardProps) {
  const { t } = useTranslation();
  const activity = useFragment(
    processingActivityLedgerCardFragment,
    processingActivityKey,
  );
  const [updateLedgerDetails, isSaving] = useUpdateProcessingActivityLedgerDetails();

  const details = activity.ledgerDetails;
  const { register, handleSubmit, reset } = useForm<LedgerFormValues>({
    defaultValues: toFormValues(details),
  });

  // Show what was stored (trimmed, cleared) after a save or a refetch.
  useEffect(() => {
    reset(toFormValues(details));
  }, [details, reset]);

  const onSubmit = handleSubmit(async (values) => {
    const orNull = (value: string) => value.trim() || null;

    await updateLedgerDetails({
      variables: {
        input: {
          processingActivityId: activity.id,
          department: orNull(values.department),
          approximateRecordCount: orNull(values.approximateRecordCount),
          collectionMethod: orNull(values.collectionMethod),
          storageLocation: orNull(values.storageLocation),
          accessScope: orNull(values.accessScope),
          disposalMethod: orNull(values.disposalMethod),
        },
      },
    }).catch(() => {
      // useMutation already shows the error toast.
    });
  });

  return (
    <Card>
      <div className="p-6 space-y-6">
        <div>
          <h2 className="text-lg font-semibold">
            {t("processingActivityLedger.title")}
          </h2>
          <p className="text-sm text-txt-tertiary">
            {t("processingActivityLedger.description")}
          </p>
        </div>

        <form onSubmit={e => void onSubmit(e)} className="space-y-6">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <Field
              label={t("processingActivityLedger.fields.department")}
              placeholder={t("processingActivityLedger.placeholders.department")}
              {...register("department")}
              maxLength={maxLength}
              disabled={!activity.canUpdate}
            />
            <Field
              label={t("processingActivityLedger.fields.approximateRecordCount")}
              placeholder={t("processingActivityLedger.placeholders.approximateRecordCount")}
              {...register("approximateRecordCount")}
              maxLength={maxLength}
              disabled={!activity.canUpdate}
            />
            {textFields.map(name => (
              <div key={name}>
                <Label htmlFor={`ledger-${name}`}>
                  {t(`processingActivityLedger.fields.${name}`)}
                </Label>
                <Textarea
                  id={`ledger-${name}`}
                  {...register(name)}
                  maxLength={maxLength}
                  placeholder={t(`processingActivityLedger.placeholders.${name}`)}
                  rows={2}
                  disabled={!activity.canUpdate}
                />
              </div>
            ))}
          </div>

          {activity.canUpdate && (
            <div className="flex justify-end">
              <Button type="submit" variant="primary" disabled={isSaving}>
                {isSaving
                  ? t("processingActivityLedger.actions.saving")
                  : t("processingActivityLedger.actions.save")}
              </Button>
            </div>
          )}
        </form>
      </div>
    </Card>
  );
}
