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

import { Button, Card, Checkbox, Field, Input, Option, Select, Textarea } from "@probo/ui";
import { type Control, Controller, type FieldPath, useForm, useWatch } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { graphql, useFragment, useLazyLoadQuery } from "react-relay";
import { Link } from "react-router";

import type {
  RightsRequestAnswerMethod,
  RightsRequestDecision,
  RightsRequestPrivacyCategory,
  RightsRequestReceiptChannel,
  RightsRequestRecordCardFragment$key,
  RightsRequestVerificationMethod,
} from "#/__generated__/core/RightsRequestRecordCardFragment.graphql";
import type { RightsRequestRecordCardQuery } from "#/__generated__/core/RightsRequestRecordCardQuery.graphql";
import { PeopleSelectField } from "#/components/form/PeopleSelectField";
import { useOrganizationId } from "#/hooks/useOrganizationId";

import { useUpdateRightsRequestPrivacyDetails } from "../_lib/useUpdateRightsRequestPrivacyDetails";

const rightsRequestRecordCardQuery = graphql`
  query RightsRequestRecordCardQuery(
    $rightsRequestId: ID!
    $withFindings: Boolean!
  ) {
    node(id: $rightsRequestId) {
      ... on RightsRequest {
        ...RightsRequestRecordCardFragment
        organization @include(if: $withFindings) {
          findings(
            first: 200
            orderBy: { field: REFERENCE_ID, direction: ASC }
          ) {
            edges {
              node {
                id
                referenceId
                description
              }
            }
          }
        }
      }
    }
  }
`;

const rightsRequestRecordCardFragment = graphql`
  fragment RightsRequestRecordCardFragment on RightsRequest {
    id
    canUpdate: permission(action: "core:rights-request:update")
    privacyDetails {
      category
      effectiveCategory
      receivedOn
      effectiveReceivedOn
      receiptChannel
      effectiveReceiptChannel
      byAgent
      targetDepartment
      handler {
        id
      }
      verificationMethod
      verifiedOn
      verificationNote
      supplementRequestedOn
      supplementReceivedOn
      decision
      decisionReason
      answerMethod
      answeredOn
      approver {
        id
      }
      firstRespondedOn
      finding {
        id
        referenceId
      }
      handlingDays
    }
  }
`;

// In the order of the rights request procedure (template 14).
const categories: readonly RightsRequestPrivacyCategory[] = [
  "PURPOSE_NOTIFICATION",
  "DISCLOSURE",
  "CORRECTION",
  "SUSPENSION",
  "THIRD_PARTY_PROVISION_STOP",
  "THIRD_PARTY_RECORD_DISCLOSURE",
  "COMPLAINT",
  "CONSULTATION",
];
const channels: readonly RightsRequestReceiptChannel[] = ["POSTAL", "EMAIL", "WEB_FORM", "PHONE", "IN_PERSON", "OTHER"];
const verificationMethods: readonly RightsRequestVerificationMethod[] = ["ID_DOCUMENT", "ACCOUNT", "OTHER", "NOT_REQUIRED"];
const decisions: readonly RightsRequestDecision[] = ["FULFILLED", "PARTIALLY_FULFILLED", "NOT_FULFILLED"];
const answerMethods: readonly RightsRequestAnswerMethod[] = ["POSTAL", "EMAIL", "ELECTRONIC_RECORD", "PHONE", "OTHER"];

// The selects cannot hold an empty value, so "not chosen" gets its own.
const none = "__NONE__";

type FormValues = {
  category: string;
  receivedOn: string;
  receiptChannel: string;
  byAgent: boolean;
  targetDepartment: string;
  handlerId: string | null;
  verificationMethod: string;
  verifiedOn: string;
  verificationNote: string;
  supplementRequestedOn: string;
  supplementReceivedOn: string;
  decision: string;
  decisionReason: string;
  answerMethod: string;
  answeredOn: string;
  approverId: string | null;
  firstRespondedOn: string;
  findingId: string;
};

type FindingOption = { id: string; label: string };

// The record needs the member list (handler and approver), so the request
// page renders this only for callers who may list members, and it loads its
// own data. The findings to link are loaded only for callers who may list
// them.
export function RightsRequestRecordSection(
  { rightsRequestId, canListFindings }: { rightsRequestId: string; canListFindings: boolean },
) {
  const data = useLazyLoadQuery<RightsRequestRecordCardQuery>(
    rightsRequestRecordCardQuery,
    { rightsRequestId, withFindings: canListFindings },
    // No background refetch, which would reset a form being edited.
    { fetchPolicy: "store-or-network" },
  );

  if (!data.node) {
    return null;
  }

  const findings = data.node.organization?.findings?.edges.map(e => ({
    id: e.node.id,
    label: e.node.description ? `${e.node.referenceId} ${e.node.description}` : e.node.referenceId,
  }));

  return <RightsRequestRecordCard requestKey={data.node} findings={findings ?? null} />;
}

type RightsRequestRecordCardProps = {
  requestKey: RightsRequestRecordCardFragment$key;
  // null when the caller may not list findings.
  findings: FindingOption[] | null;
};

function RightsRequestRecordCard({ requestKey, findings }: RightsRequestRecordCardProps) {
  const request = useFragment(rightsRequestRecordCardFragment, requestKey);
  const { t } = useTranslation();
  const d = request.privacyDetails;

  const stored: FormValues = {
    category: d.category ?? none,
    receivedOn: d.receivedOn ?? "",
    receiptChannel: d.receiptChannel ?? none,
    byAgent: d.byAgent,
    targetDepartment: d.targetDepartment,
    handlerId: d.handler?.id ?? null,
    verificationMethod: d.verificationMethod ?? none,
    verifiedOn: d.verifiedOn ?? "",
    verificationNote: d.verificationNote,
    supplementRequestedOn: d.supplementRequestedOn ?? "",
    supplementReceivedOn: d.supplementReceivedOn ?? "",
    decision: d.decision ?? none,
    decisionReason: d.decisionReason,
    answerMethod: d.answerMethod ?? none,
    answeredOn: d.answeredOn ?? "",
    approverId: d.approver?.id ?? null,
    firstRespondedOn: d.firstRespondedOn ?? "",
    findingId: d.finding?.id ?? none,
  };

  // Keep the linked finding selectable even when it is not in the loaded
  // page of findings.
  const findingOptions = findings && d.finding && !findings.some(f => f.id === d.finding?.id)
    ? [{ id: d.finding.id, label: d.finding.referenceId }, ...findings]
    : findings;

  return (
    <Card padded>
      <div className="space-y-6">
        <div>
          <h2 className="text-lg font-semibold">{t("rightsRequestRecord.title")}</h2>
          <p className="text-sm text-txt-tertiary">{t("rightsRequestRecord.description")}</p>
        </div>

        <RightsRequestRecordForm
          // A new key resets the form to what was stored after a save.
          key={JSON.stringify(stored)}
          rightsRequestId={request.id}
          canUpdate={request.canUpdate}
          stored={stored}
          effective={{
            category: d.effectiveCategory,
            receivedOn: d.effectiveReceivedOn,
            receiptChannel: d.effectiveReceiptChannel ?? null,
          }}
          handlingDays={d.handlingDays ?? null}
          findings={findingOptions}
          linkedFindingReference={d.finding?.referenceId ?? null}
        />
      </div>
    </Card>
  );
}

function orNull<T extends string>(value: string): T | null {
  return value === none ? null : value as T;
}

interface RightsRequestRecordFormProps {
  rightsRequestId: string;
  canUpdate: boolean;
  stored: FormValues;
  effective: {
    category: RightsRequestPrivacyCategory;
    receivedOn: string;
    receiptChannel: RightsRequestReceiptChannel | null;
  };
  handlingDays: number | null;
  findings: FindingOption[] | null;
  linkedFindingReference: string | null;
}

function RightsRequestRecordForm({
  rightsRequestId,
  canUpdate,
  stored,
  effective,
  handlingDays,
  findings,
  linkedFindingReference,
}: RightsRequestRecordFormProps) {
  const { t } = useTranslation();
  const organizationId = useOrganizationId();
  const [updateRecord, isSaving] = useUpdateRightsRequestPrivacyDetails();
  const { control, register, handleSubmit } = useForm<FormValues>({ defaultValues: stored });

  const disabled = !canUpdate || isSaving;

  const onSubmit = handleSubmit(async (v) => {
    await updateRecord({
      variables: {
        input: {
          rightsRequestId,
          category: orNull<RightsRequestPrivacyCategory>(v.category),
          receivedOn: v.receivedOn || null,
          receiptChannel: orNull<RightsRequestReceiptChannel>(v.receiptChannel),
          byAgent: v.byAgent,
          targetDepartment: v.targetDepartment.trim(),
          handlerId: v.handlerId,
          verificationMethod: orNull<RightsRequestVerificationMethod>(v.verificationMethod),
          verifiedOn: v.verifiedOn || null,
          verificationNote: v.verificationNote.trim(),
          supplementRequestedOn: v.supplementRequestedOn || null,
          supplementReceivedOn: v.supplementReceivedOn || null,
          decision: orNull<RightsRequestDecision>(v.decision),
          decisionReason: v.decisionReason.trim(),
          answerMethod: orNull<RightsRequestAnswerMethod>(v.answerMethod),
          answeredOn: v.answeredOn || null,
          approverId: v.approverId,
          firstRespondedOn: v.firstRespondedOn || null,
          // Without the finding list the link cannot be changed, so keep it.
          findingId: findings ? orNull<string>(v.findingId) : orNull<string>(stored.findingId),
        },
      },
    }).catch(() => {
      // useMutation already shows the error toast.
    });
  });

  const dateField = (name: FieldPath<FormValues>, label: string, help?: string) => (
    <Field name={`rr-${name}`} label={label} help={help}>
      <Input id={`rr-${name}`} {...register(name)} type="date" disabled={disabled} />
    </Field>
  );

  return (
    <form onSubmit={e => void onSubmit(e)} className="space-y-6">
      <section className="space-y-4">
        <h3 className="text-sm font-medium text-txt-primary">{t("rightsRequestRecord.sections.receipt")}</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {dateField("receivedOn", t("rightsRequestRecord.fields.receivedOn"), t("rightsRequestRecord.help.receivedOn", { date: effective.receivedOn }))}
          <EnumSelect
            control={control}
            name="receiptChannel"
            label={t("rightsRequestRecord.fields.receiptChannel")}
            values={channels}
            labelOf={c => t(`rightsRequestRecord.channels.${c}`)}
            noneLabel={effective.receiptChannel
              ? t("rightsRequestRecord.followWith", { value: t(`rightsRequestRecord.channels.${effective.receiptChannel}`) })
              : t("rightsRequestRecord.notChosen")}
            disabled={disabled}
          />
          <EnumSelect
            control={control}
            name="category"
            label={t("rightsRequestRecord.fields.category")}
            values={categories}
            labelOf={c => t(`rightsRequestRecord.categories.${c}`)}
            noneLabel={t("rightsRequestRecord.followType", { value: t(`rightsRequestRecord.categories.${effective.category}`) })}
            disabled={disabled}
          />
          <PeopleSelectField
            organizationId={organizationId}
            control={control}
            name="handlerId"
            label={t("rightsRequestRecord.fields.handler")}
            disabled={disabled}
            optional
          />
          <Field name="rr-targetDepartment" label={t("rightsRequestRecord.fields.targetDepartment")}>
            <Input id="rr-targetDepartment" {...register("targetDepartment")} disabled={disabled} />
          </Field>
          <Controller
            control={control}
            name="byAgent"
            render={({ field }) => (
              <label className="flex items-center gap-2 text-sm text-txt-primary md:pt-7">
                <Checkbox checked={field.value} onChange={field.onChange} disabled={disabled} />
                {t("rightsRequestRecord.fields.byAgent")}
              </label>
            )}
          />
        </div>
      </section>

      <section className="space-y-4">
        <h3 className="text-sm font-medium text-txt-primary">{t("rightsRequestRecord.sections.verification")}</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <EnumSelect
            control={control}
            name="verificationMethod"
            label={t("rightsRequestRecord.fields.verificationMethod")}
            values={verificationMethods}
            labelOf={m => t(`rightsRequestRecord.verificationMethods.${m}`)}
            noneLabel={t("rightsRequestRecord.notChosen")}
            disabled={disabled}
          />
          {dateField("verifiedOn", t("rightsRequestRecord.fields.verifiedOn"))}
          <Field name="rr-verificationNote" label={t("rightsRequestRecord.fields.verificationNote")}>
            <Input
              id="rr-verificationNote"
              {...register("verificationNote")}
              placeholder={t("rightsRequestRecord.placeholders.verificationNote")}
              disabled={disabled}
            />
          </Field>
          {dateField("supplementRequestedOn", t("rightsRequestRecord.fields.supplementRequestedOn"), t("rightsRequestRecord.help.supplement"))}
          {dateField("supplementReceivedOn", t("rightsRequestRecord.fields.supplementReceivedOn"))}
        </div>
      </section>

      <section className="space-y-4">
        <h3 className="text-sm font-medium text-txt-primary">{t("rightsRequestRecord.sections.answer")}</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <EnumSelect
            control={control}
            name="decision"
            label={t("rightsRequestRecord.fields.decision")}
            values={decisions}
            labelOf={x => t(`rightsRequestRecord.decisions.${x}`)}
            noneLabel={t("rightsRequestRecord.notChosen")}
            disabled={disabled}
          />
          <EnumSelect
            control={control}
            name="answerMethod"
            label={t("rightsRequestRecord.fields.answerMethod")}
            values={answerMethods}
            labelOf={m => t(`rightsRequestRecord.answerMethods.${m}`)}
            noneLabel={t("rightsRequestRecord.notChosen")}
            disabled={disabled}
          />
          {dateField("answeredOn", t("rightsRequestRecord.fields.answeredOn"))}
        </div>
        <DecisionReasonField control={control} register={register} disabled={disabled} />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <PeopleSelectField
            organizationId={organizationId}
            control={control}
            name="approverId"
            label={t("rightsRequestRecord.fields.approver")}
            help={t("rightsRequestRecord.help.approver")}
            disabled={disabled}
            optional
          />
        </div>
        <RecordHints control={control} effectiveReceivedOn={effective.receivedOn} handlingDays={handlingDays} />
      </section>

      <ComplaintSection
        control={control}
        register={register}
        disabled={disabled}
        effectiveCategory={effective.category}
        findings={findings}
        linkedFindingReference={linkedFindingReference}
      />

      {canUpdate && (
        <div className="flex justify-end">
          <Button type="submit" variant="primary" disabled={isSaving}>
            {isSaving ? t("rightsRequestRecord.actions.saving") : t("rightsRequestRecord.actions.save")}
          </Button>
        </div>
      )}
    </form>
  );
}

interface EnumSelectProps<T extends string> {
  control: Control<FormValues>;
  name: FieldPath<FormValues>;
  label: string;
  values: readonly T[];
  labelOf: (value: T) => string;
  noneLabel: string;
  disabled: boolean;
}

function EnumSelect<T extends string>(props: EnumSelectProps<T>) {
  const { control, name, label, values, labelOf, noneLabel, disabled } = props;

  return (
    <Field label={label}>
      <Controller
        control={control}
        name={name}
        render={({ field }) => (
          <Select value={String(field.value)} onValueChange={field.onChange} disabled={disabled} aria-label={label}>
            <Option value={none}>{noneLabel}</Option>
            {values.map(v => <Option key={v} value={v}>{labelOf(v)}</Option>)}
          </Select>
        )}
      />
    </Field>
  );
}

type DecisionReasonFieldProps = {
  control: Control<FormValues>;
  register: ReturnType<typeof useForm<FormValues>>["register"];
  disabled: boolean;
};

function DecisionReasonField({ control, register, disabled }: DecisionReasonFieldProps) {
  const { t } = useTranslation();
  const decision = useWatch({ control, name: "decision" });

  if (decision !== "PARTIALLY_FULFILLED" && decision !== "NOT_FULFILLED") {
    return null;
  }

  return (
    <Field
      name="rr-decisionReason"
      label={t("rightsRequestRecord.fields.decisionReason")}
      help={t("rightsRequestRecord.help.decisionReason")}
    >
      <Textarea id="rr-decisionReason" {...register("decisionReason")} rows={3} disabled={disabled} />
    </Field>
  );
}

// Shows the handling days of the stored record, what the save would reject
// and what an auditor would ask about (procedure 14).
type RecordHintsProps = {
  control: Control<FormValues>;
  effectiveReceivedOn: string;
  handlingDays: number | null;
};

function RecordHints({ control, effectiveReceivedOn, handlingDays }: RecordHintsProps) {
  const { t } = useTranslation();
  const [receivedOn, verifiedOn, requestedOn, receivedDocsOn, answeredOn, decision, reason, handlerId, approverId]
    = useWatch({
      control,
      name: [
        "receivedOn",
        "verifiedOn",
        "supplementRequestedOn",
        "supplementReceivedOn",
        "answeredOn",
        "decision",
        "decisionReason",
        "handlerId",
        "approverId",
      ],
    });

  const received = receivedOn || effectiveReceivedOn;
  const warnings: string[] = [];
  if ((decision === "PARTIALLY_FULFILLED" || decision === "NOT_FULFILLED") && reason.trim() === "") {
    warnings.push(t("rightsRequestRecord.warnings.reasonMissing"));
  }
  if (receivedDocsOn && (!requestedOn || receivedDocsOn < requestedOn)) {
    warnings.push(t("rightsRequestRecord.warnings.supplementOrder"));
  }
  if (receivedOn && [verifiedOn, requestedOn, answeredOn].some(date => date && date < receivedOn)) {
    warnings.push(t("rightsRequestRecord.warnings.beforeReceived"));
  }
  if (answeredOn && !approverId) {
    warnings.push(t("rightsRequestRecord.warnings.approverMissing"));
  }
  if (handlerId && handlerId === approverId) {
    warnings.push(t("rightsRequestRecord.warnings.sameApprover"));
  }

  return (
    <div className="space-y-2 text-sm">
      <p className="text-txt-secondary">
        {handlingDays === null
          ? t("rightsRequestRecord.days.notAnswered", { date: received })
          : t("rightsRequestRecord.days.answered", { days: handlingDays })}
      </p>
      {warnings.length > 0 && (
        <div className="rounded-lg border border-warning bg-warning-subtle p-4 space-y-1" role="alert">
          {warnings.map(w => <p key={w}>{w}</p>)}
        </div>
      )}
    </div>
  );
}

type ComplaintSectionProps = DecisionReasonFieldProps & {
  effectiveCategory: RightsRequestPrivacyCategory;
  findings: FindingOption[] | null;
  linkedFindingReference: string | null;
};

// The first response and the finding opened for a cause in the company's
// own handling (procedure 14, sections 7 and 12). Shown for complaints and
// consultations only.
function ComplaintSection(props: ComplaintSectionProps) {
  const { control, register, disabled, effectiveCategory, findings, linkedFindingReference } = props;
  const { t } = useTranslation();
  const organizationId = useOrganizationId();
  const [category, findingId] = useWatch({ control, name: ["category", "findingId"] });

  const current = category === none ? effectiveCategory : category;
  if (current !== "COMPLAINT" && current !== "CONSULTATION") {
    return null;
  }

  return (
    <section className="space-y-4">
      <h3 className="text-sm font-medium text-txt-primary">{t("rightsRequestRecord.sections.complaint")}</h3>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Field
          name="rr-firstRespondedOn"
          label={t("rightsRequestRecord.fields.firstRespondedOn")}
          help={t("rightsRequestRecord.help.firstRespondedOn")}
        >
          <Input id="rr-firstRespondedOn" {...register("firstRespondedOn")} type="date" disabled={disabled} />
        </Field>
        <div className="md:col-span-2">
          {findings
            ? (
                <EnumSelect
                  control={control}
                  name="findingId"
                  label={t("rightsRequestRecord.fields.finding")}
                  values={findings.map(f => f.id)}
                  labelOf={id => findings.find(f => f.id === id)?.label ?? id}
                  noneLabel={t("rightsRequestRecord.noFinding")}
                  disabled={disabled}
                />
              )
            : (
                <Field label={t("rightsRequestRecord.fields.finding")}>
                  <p className="text-sm text-txt-secondary">
                    {linkedFindingReference ?? t("rightsRequestRecord.noFinding")}
                  </p>
                </Field>
              )}
          <p className="mt-1 text-xs text-txt-tertiary">
            {t("rightsRequestRecord.help.finding")}
            {findingId && findingId !== none && (
              <>
                {" "}
                <Link
                  to={`/organizations/${organizationId}/governance/findings/${findingId}`}
                  className="underline"
                >
                  {t("rightsRequestRecord.openFinding")}
                </Link>
              </>
            )}
          </p>
        </div>
      </div>
    </section>
  );
}
