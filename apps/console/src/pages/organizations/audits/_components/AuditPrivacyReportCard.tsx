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

import { Button, Card, Field, IconArrowDown, Textarea } from "@probo/ui";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { graphql, useFragment, useLazyLoadQuery } from "react-relay";
import { Link } from "react-router";

import type { AuditPrivacyReportCardFragment$key } from "#/__generated__/core/AuditPrivacyReportCardFragment.graphql";
import type { AuditPrivacyReportCardQuery } from "#/__generated__/core/AuditPrivacyReportCardQuery.graphql";
import { PeopleMultiSelectField } from "#/components/form/PeopleMultiSelectField";
import { useOrganizationId } from "#/hooks/useOrganizationId";

import { useExportAuditReport } from "../_lib/useExportAuditReport";
import { useUpdateAuditPrivacyDetails } from "../_lib/useUpdateAuditPrivacyDetails";

const auditPrivacyReportCardQuery = graphql`
  query AuditPrivacyReportCardQuery($auditId: ID!) {
    node(id: $auditId) {
      ... on Audit {
        ...AuditPrivacyReportCardFragment
      }
    }
  }
`;

const auditPrivacyReportCardFragment = graphql`
  fragment AuditPrivacyReportCardFragment on Audit {
    id
    canUpdate: permission(action: "core:audit:update")
    privacyDetails {
      scope
      criteria
      auditeeDepartments
      externalAuditors
      overallOpinion
      updatedAt
    }
    privacyAuditors {
      id
      fullName
      emailAddress
    }
    privacyIndependenceConflicts {
      auditor {
        id
        fullName
      }
      department
    }
    findings(first: 100, orderBy: { field: REFERENCE_ID, direction: ASC }) {
      totalCount
      edges {
        node {
          id
          referenceId
          description
          status
          kind
          privacyDetails {
            effectiveCategory
          }
        }
      }
    }
  }
`;

// The report needs the member list (auditors) and the findings, so the
// audit page renders this only for callers who may list both, and it loads
// its own data.
export function AuditPrivacyReportSection({ auditId }: { auditId: string }) {
  const data = useLazyLoadQuery<AuditPrivacyReportCardQuery>(
    auditPrivacyReportCardQuery,
    { auditId },
    // No background refetch, which would reset a form being edited.
    { fetchPolicy: "store-or-network" },
  );

  if (!data.node) {
    return null;
  }

  return <AuditPrivacyReportCard auditKey={data.node} />;
}

type FormValues = {
  scope: string;
  criteria: string;
  auditeeDepartments: string;
  externalAuditors: string;
  overallOpinion: string;
  auditorIds: string[];
};

// One name per line in the text areas.
function splitLines(value: string): string[] {
  const names = value.split("\n").map(line => line.trim()).filter(Boolean);
  return [...new Set(names)];
}

function AuditPrivacyReportCard({ auditKey }: { auditKey: AuditPrivacyReportCardFragment$key }) {
  const audit = useFragment(auditPrivacyReportCardFragment, auditKey);
  const { t } = useTranslation();
  const details = audit.privacyDetails;

  const stored: FormValues = {
    scope: details.scope,
    // Suggest the usual criteria until the report items are first saved.
    criteria: details.updatedAt ? details.criteria : t("auditPrivacy.defaults.criteria"),
    auditeeDepartments: details.auditeeDepartments.join("\n"),
    externalAuditors: details.externalAuditors.join("\n"),
    overallOpinion: details.overallOpinion,
    auditorIds: audit.privacyAuditors.map(a => a.id),
  };

  return (
    <Card padded>
      <div className="space-y-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold">
              {t("auditPrivacy.title")}
            </h2>
            <p className="text-sm text-txt-tertiary">
              {t("auditPrivacy.description")}
            </p>
          </div>
          <ExportAuditReportButton auditId={audit.id} />
        </div>

        <AuditPrivacyForm
          // A new key resets the form to what was stored after a save.
          key={JSON.stringify(stored)}
          auditId={audit.id}
          canUpdate={audit.canUpdate}
          stored={stored}
          auditors={audit.privacyAuditors.map(a => ({
            id: a.id,
            fullName: a.fullName,
            emailAddress: a.emailAddress,
          }))}
        />

        {audit.privacyIndependenceConflicts.length > 0 && (
          <div className="rounded-lg border border-warning bg-warning-subtle p-4 text-sm space-y-1" role="alert">
            <p className="font-medium">{t("auditPrivacy.independence.title")}</p>
            <ul className="list-disc pl-5">
              {audit.privacyIndependenceConflicts.map(c => (
                <li key={`${c.auditor.id}-${c.department}`}>
                  {t("auditPrivacy.independence.item", {
                    name: c.auditor.fullName,
                    department: c.department,
                  })}
                </li>
              ))}
            </ul>
          </div>
        )}

        <AuditFindingList
          findings={audit.findings?.edges.map(e => e.node) ?? []}
          totalCount={audit.findings?.totalCount ?? 0}
        />
      </div>
    </Card>
  );
}

function ExportAuditReportButton({ auditId }: { auditId: string }) {
  const { t } = useTranslation();
  const [exportReport, isExporting] = useExportAuditReport(auditId);

  return (
    <Button
      type="button"
      variant="secondary"
      icon={IconArrowDown}
      disabled={isExporting}
      onClick={() => {
        // useMutation already shows the error toast.
        void exportReport().catch(() => {});
      }}
    >
      {t("auditPrivacy.actions.export")}
    </Button>
  );
}

interface AuditPrivacyFormProps {
  auditId: string;
  canUpdate: boolean;
  stored: FormValues;
  auditors: { id: string; fullName: string; emailAddress: string }[];
}

function AuditPrivacyForm({ auditId, canUpdate, stored, auditors }: AuditPrivacyFormProps) {
  const { t } = useTranslation();
  const organizationId = useOrganizationId();
  const [updateAuditPrivacy, isSaving] = useUpdateAuditPrivacyDetails();
  const { control, register, handleSubmit } = useForm<FormValues>({ defaultValues: stored });

  const disabled = !canUpdate || isSaving;

  const onSubmit = handleSubmit(async (values) => {
    await updateAuditPrivacy({
      variables: {
        input: {
          auditId,
          scope: values.scope.trim(),
          criteria: values.criteria.trim(),
          auditeeDepartments: splitLines(values.auditeeDepartments),
          externalAuditors: splitLines(values.externalAuditors),
          overallOpinion: values.overallOpinion.trim(),
          auditorIds: values.auditorIds,
        },
      },
    }).catch(() => {
      // useMutation already shows the error toast.
    });
  });

  return (
    <form onSubmit={e => void onSubmit(e)} className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Field name="audit-privacy-scope" label={t("auditPrivacy.fields.scope")}>
          <Textarea
            id="audit-privacy-scope"
            {...register("scope")}
            rows={3}
            placeholder={t("auditPrivacy.placeholders.scope")}
            disabled={disabled}
          />
        </Field>
        <Field name="audit-privacy-criteria" label={t("auditPrivacy.fields.criteria")}>
          <Textarea
            id="audit-privacy-criteria"
            {...register("criteria")}
            rows={3}
            disabled={disabled}
          />
        </Field>
      </div>

      <PeopleMultiSelectField
        organizationId={organizationId}
        control={control}
        name="auditorIds"
        label={t("auditPrivacy.fields.auditors")}
        disabled={disabled}
        selectedPeople={auditors}
        placeholder={t("auditPrivacy.placeholders.auditors")}
      />

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Field
          name="audit-privacy-externalAuditors"
          label={t("auditPrivacy.fields.externalAuditors")}
          help={t("auditPrivacy.help.onePerLine")}
        >
          <Textarea
            id="audit-privacy-externalAuditors"
            {...register("externalAuditors")}
            rows={2}
            disabled={disabled}
          />
        </Field>
        <Field
          name="audit-privacy-auditeeDepartments"
          label={t("auditPrivacy.fields.auditeeDepartments")}
          help={t("auditPrivacy.help.onePerLine")}
        >
          <Textarea
            id="audit-privacy-auditeeDepartments"
            {...register("auditeeDepartments")}
            rows={2}
            placeholder={t("auditPrivacy.placeholders.auditeeDepartments")}
            disabled={disabled}
          />
        </Field>
      </div>

      <Field name="audit-privacy-overallOpinion" label={t("auditPrivacy.fields.overallOpinion")}>
        <Textarea
          id="audit-privacy-overallOpinion"
          {...register("overallOpinion")}
          rows={4}
          placeholder={t("auditPrivacy.placeholders.overallOpinion")}
          disabled={disabled}
        />
      </Field>

      {canUpdate && (
        <div className="flex justify-end">
          <Button type="submit" variant="primary" disabled={isSaving}>
            {isSaving ? t("auditPrivacy.actions.saving") : t("auditPrivacy.actions.save")}
          </Button>
        </div>
      )}
    </form>
  );
}

interface AuditFinding {
  readonly id: string;
  readonly referenceId: string;
  readonly description: string | null | undefined;
  readonly status: string;
  readonly kind: string;
  readonly privacyDetails: { readonly effectiveCategory: string | null | undefined };
}

function AuditFindingList({ findings, totalCount }: { findings: readonly AuditFinding[]; totalCount: number }) {
  const { t } = useTranslation();
  const organizationId = useOrganizationId();

  // Same label as the exported report: a major nonconformity is marked.
  const categoryLabel = (f: AuditFinding) => {
    const category = f.privacyDetails.effectiveCategory;
    if (!category) {
      return t("auditPrivacy.findings.noCategory");
    }

    const label = t(`findingPrivacy.categories.${category}`);
    return category === "NONCONFORMITY" && f.kind === "MAJOR_NONCONFORMITY"
      ? t("auditPrivacy.findings.major", { category: label })
      : label;
  };

  return (
    <div className="space-y-2">
      <h3 className="text-sm font-medium text-txt-primary">
        {t("auditPrivacy.findings.title")}
      </h3>
      {findings.length === 0
        ? (
            <p className="text-sm text-txt-secondary">{t("auditPrivacy.findings.empty")}</p>
          )
        : (
            <ul className="divide-y divide-border-low text-sm">
              {findings.map(f => (
                <li key={f.id} className="flex gap-4 py-2">
                  <Link
                    to={`/organizations/${organizationId}/governance/findings/${f.id}`}
                    className="shrink-0 font-medium underline"
                  >
                    {f.referenceId}
                  </Link>
                  <span className="shrink-0 w-28 text-txt-secondary">
                    {categoryLabel(f)}
                  </span>
                  <span className="grow line-clamp-2">{f.description}</span>
                  <span className="shrink-0 text-txt-secondary">
                    {t(`findingsPage.status.${f.status.toLowerCase()}`)}
                  </span>
                </li>
              ))}
            </ul>
          )}
      {totalCount > findings.length && (
        <p className="text-xs text-txt-secondary">
          {t("auditPrivacy.findings.truncated", { shown: findings.length, total: totalCount })}
        </p>
      )}
      <p className="text-xs text-txt-tertiary">{t("auditPrivacy.findings.help")}</p>
    </div>
  );
}
