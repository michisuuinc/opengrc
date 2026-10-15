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

import { Button, Card, Field, Input, Option, Select, Textarea } from "@probo/ui";
import { Controller, useForm, useWatch } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { graphql, useFragment, useLazyLoadQuery } from "react-relay";

import type {
  ThirdPartyEvaluationItem,
  ThirdPartyEvaluationJudgement,
  ThirdPartyEvaluationKind,
  ThirdPartyEvaluationResult,
  ThirdPartyPrivacyEvaluationCardFragment$key,
} from "#/__generated__/core/ThirdPartyPrivacyEvaluationCardFragment.graphql";
import type { ThirdPartyPrivacyEvaluationCardQuery } from "#/__generated__/core/ThirdPartyPrivacyEvaluationCardQuery.graphql";
import { PeopleSelectField } from "#/components/form/PeopleSelectField";
import { useOrganizationId } from "#/hooks/useOrganizationId";

import { useUpdateThirdPartyPrivacyEvaluation } from "../../_lib/useUpdateThirdPartyPrivacyEvaluation";

const thirdPartyPrivacyEvaluationCardQuery = graphql`
  query ThirdPartyPrivacyEvaluationCardQuery($riskAssessmentId: ID!) {
    node(id: $riskAssessmentId) {
      ... on ThirdPartyRiskAssessment {
        ...ThirdPartyPrivacyEvaluationCardFragment
      }
    }
  }
`;

const thirdPartyPrivacyEvaluationCardFragment = graphql`
  fragment ThirdPartyPrivacyEvaluationCardFragment on ThirdPartyRiskAssessment {
    id
    canUpdate: permission(action: "core:thirdParty-risk-assessment:create")
    privacyEvaluation {
      kind
      adHocReason
      entrustedWork
      personalData
      storageCountries
      evaluator {
        id
      }
      evaluatedOn
      answers {
        item
        result
        note
      }
      judgement
      conditions
      approver {
        id
      }
      approvedOn
    }
  }
`;

const kinds: readonly ThirdPartyEvaluationKind[] = ["NEW", "PERIODIC", "AD_HOC"];
const results: readonly ThirdPartyEvaluationResult[] = ["SUITABLE", "UNSUITABLE", "NOT_APPLICABLE"];
const judgements: readonly ThirdPartyEvaluationJudgement[] = [
  "APPROVED",
  "CONDITIONALLY_APPROVED",
  "REJECTED",
];

// The selects cannot hold an empty value, so "not chosen" gets its own.
const none = "__NONE__";

type AnswerValues = {
  item: ThirdPartyEvaluationItem;
  result: string;
  note: string;
};

type FormValues = {
  kind: string;
  adHocReason: string;
  entrustedWork: string;
  personalData: string;
  storageCountries: string;
  evaluatorId: string | null;
  evaluatedOn: string;
  answers: AnswerValues[];
  judgement: string;
  conditions: string;
  approverId: string | null;
  approvedOn: string;
};

// The checklist needs the member list (evaluator and approver), so the
// assessment row renders this only for callers who may list members, and it
// loads its own data.
export function ThirdPartyPrivacyEvaluationSection({ riskAssessmentId }: { riskAssessmentId: string }) {
  const data = useLazyLoadQuery<ThirdPartyPrivacyEvaluationCardQuery>(
    thirdPartyPrivacyEvaluationCardQuery,
    { riskAssessmentId },
    // No background refetch, which would reset a form being edited.
    { fetchPolicy: "store-or-network" },
  );

  if (!data.node) {
    return null;
  }

  return <ThirdPartyPrivacyEvaluationCard assessmentKey={data.node} />;
}

function ThirdPartyPrivacyEvaluationCard(
  { assessmentKey }: { assessmentKey: ThirdPartyPrivacyEvaluationCardFragment$key },
) {
  const assessment = useFragment(thirdPartyPrivacyEvaluationCardFragment, assessmentKey);
  const { t } = useTranslation();
  const evaluation = assessment.privacyEvaluation;

  const stored: FormValues = {
    kind: evaluation.kind ?? none,
    adHocReason: evaluation.adHocReason,
    entrustedWork: evaluation.entrustedWork,
    personalData: evaluation.personalData,
    storageCountries: evaluation.storageCountries,
    evaluatorId: evaluation.evaluator?.id ?? null,
    evaluatedOn: evaluation.evaluatedOn ?? "",
    answers: evaluation.answers.map(a => ({
      item: a.item,
      result: a.result ?? none,
      note: a.note,
    })),
    judgement: evaluation.judgement ?? none,
    conditions: evaluation.conditions,
    approverId: evaluation.approver?.id ?? null,
    approvedOn: evaluation.approvedOn ?? "",
  };

  return (
    <Card padded>
      <div className="space-y-6">
        <div>
          <h2 className="text-lg font-semibold">
            {t("thirdPartyEvaluation.title")}
          </h2>
          <p className="text-sm text-txt-tertiary">
            {t("thirdPartyEvaluation.description")}
          </p>
        </div>

        <ThirdPartyPrivacyEvaluationForm
          // A new key resets the form to what was stored after a save.
          key={JSON.stringify(stored)}
          riskAssessmentId={assessment.id}
          canUpdate={assessment.canUpdate}
          stored={stored}
        />
      </div>
    </Card>
  );
}

function orNull<T extends string>(value: string): T | null {
  return value === none ? null : value as T;
}

interface ThirdPartyPrivacyEvaluationFormProps {
  riskAssessmentId: string;
  canUpdate: boolean;
  stored: FormValues;
}

function ThirdPartyPrivacyEvaluationForm(
  { riskAssessmentId, canUpdate, stored }: ThirdPartyPrivacyEvaluationFormProps,
) {
  const { t } = useTranslation();
  const organizationId = useOrganizationId();
  const [updateEvaluation, isSaving] = useUpdateThirdPartyPrivacyEvaluation();
  const { control, register, handleSubmit } = useForm<FormValues>({ defaultValues: stored });

  const disabled = !canUpdate || isSaving;

  const onSubmit = handleSubmit(async (values) => {
    await updateEvaluation({
      variables: {
        input: {
          riskAssessmentId,
          kind: orNull<ThirdPartyEvaluationKind>(values.kind),
          adHocReason: values.kind === "AD_HOC" ? values.adHocReason.trim() : "",
          entrustedWork: values.entrustedWork.trim(),
          personalData: values.personalData.trim(),
          storageCountries: values.storageCountries.trim(),
          evaluatorId: values.evaluatorId,
          evaluatedOn: values.evaluatedOn || null,
          answers: values.answers.map(a => ({
            item: a.item,
            result: orNull<ThirdPartyEvaluationResult>(a.result),
            note: a.note.trim(),
          })),
          judgement: orNull<ThirdPartyEvaluationJudgement>(values.judgement),
          conditions: values.conditions.trim(),
          approverId: values.approverId,
          approvedOn: values.approvedOn || null,
        },
      },
    }).catch(() => {
      // useMutation already shows the error toast.
    });
  });

  return (
    <form onSubmit={e => void onSubmit(e)} className="space-y-6">
      <section className="space-y-4">
        <h3 className="text-sm font-medium text-txt-primary">
          {t("thirdPartyEvaluation.sections.outsourcing")}
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Field label={t("thirdPartyEvaluation.fields.kind")}>
            <Controller
              control={control}
              name="kind"
              render={({ field }) => (
                <Select value={field.value} onValueChange={field.onChange} disabled={disabled}>
                  <Option value={none}>{t("thirdPartyEvaluation.notChosen")}</Option>
                  {kinds.map(k => (
                    <Option key={k} value={k}>{t(`thirdPartyEvaluation.kinds.${k}`)}</Option>
                  ))}
                </Select>
              )}
            />
          </Field>
          <AdHocReasonField control={control} register={register} disabled={disabled} />
          <Field name="tpe-entrustedWork" label={t("thirdPartyEvaluation.fields.entrustedWork")}>
            <Textarea
              id="tpe-entrustedWork"
              {...register("entrustedWork")}
              rows={2}
              placeholder={t("thirdPartyEvaluation.placeholders.entrustedWork")}
              disabled={disabled}
            />
          </Field>
          <Field name="tpe-personalData" label={t("thirdPartyEvaluation.fields.personalData")}>
            <Textarea
              id="tpe-personalData"
              {...register("personalData")}
              rows={2}
              placeholder={t("thirdPartyEvaluation.placeholders.personalData")}
              disabled={disabled}
            />
          </Field>
        </div>
      </section>

      <ChecklistSection control={control} register={register} disabled={disabled} />

      <section className="space-y-4">
        <h3 className="text-sm font-medium text-txt-primary">
          {t("thirdPartyEvaluation.sections.judgement")}
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <PeopleSelectField
            organizationId={organizationId}
            control={control}
            name="evaluatorId"
            label={t("thirdPartyEvaluation.fields.evaluator")}
            disabled={disabled}
            optional
          />
          <Field name="tpe-evaluatedOn" label={t("thirdPartyEvaluation.fields.evaluatedOn")}>
            <Input id="tpe-evaluatedOn" {...register("evaluatedOn")} type="date" disabled={disabled} />
          </Field>
          <Field label={t("thirdPartyEvaluation.fields.judgement")}>
            <Controller
              control={control}
              name="judgement"
              render={({ field }) => (
                <Select value={field.value} onValueChange={field.onChange} disabled={disabled}>
                  <Option value={none}>{t("thirdPartyEvaluation.notChosen")}</Option>
                  {judgements.map(j => (
                    <Option key={j} value={j}>{t(`thirdPartyEvaluation.judgements.${j}`)}</Option>
                  ))}
                </Select>
              )}
            />
          </Field>
          <Field
            name="tpe-conditions"
            label={t("thirdPartyEvaluation.fields.conditions")}
            help={t("thirdPartyEvaluation.help.conditions")}
          >
            <Textarea id="tpe-conditions" {...register("conditions")} rows={2} disabled={disabled} />
          </Field>
          <PeopleSelectField
            organizationId={organizationId}
            control={control}
            name="approverId"
            label={t("thirdPartyEvaluation.fields.approver")}
            help={t("thirdPartyEvaluation.help.approver")}
            disabled={disabled}
            optional
          />
          <Field name="tpe-approvedOn" label={t("thirdPartyEvaluation.fields.approvedOn")}>
            <Input id="tpe-approvedOn" {...register("approvedOn")} type="date" disabled={disabled} />
          </Field>
        </div>
        <JudgementHints control={control} />
      </section>

      {canUpdate && (
        <div className="flex justify-end">
          <Button type="submit" variant="primary" disabled={isSaving}>
            {isSaving ? t("thirdPartyEvaluation.actions.saving") : t("thirdPartyEvaluation.actions.save")}
          </Button>
        </div>
      )}
    </form>
  );
}

type FormControlProps = {
  control: ReturnType<typeof useForm<FormValues>>["control"];
  register: ReturnType<typeof useForm<FormValues>>["register"];
  disabled: boolean;
};

function AdHocReasonField({ control, register, disabled }: FormControlProps) {
  const { t } = useTranslation();
  const kind = useWatch({ control, name: "kind" });

  if (kind !== "AD_HOC") {
    return <div className="hidden md:block" />;
  }

  return (
    <Field name="tpe-adHocReason" label={t("thirdPartyEvaluation.fields.adHocReason")}>
      <Input
        id="tpe-adHocReason"
        {...register("adHocReason")}
        placeholder={t("thirdPartyEvaluation.placeholders.adHocReason")}
        disabled={disabled}
      />
    </Field>
  );
}

function ChecklistSection({ control, register, disabled }: FormControlProps) {
  const { t } = useTranslation();
  const answers = useWatch({ control, name: "answers" });

  return (
    <section className="space-y-3">
      <div>
        <h3 className="text-sm font-medium text-txt-primary">
          {t("thirdPartyEvaluation.sections.checklist")}
        </h3>
        <p className="text-xs text-txt-tertiary">{t("thirdPartyEvaluation.help.checklist")}</p>
      </div>
      <ol className="divide-y divide-border-low">
        {answers.map((answer, index) => (
          <li key={answer.item} className="grid grid-cols-1 md:grid-cols-[minmax(0,1fr)_10rem_minmax(0,1fr)] gap-3 py-3">
            <div className="text-sm">
              <p className="font-medium">
                {index + 1}
                .
                {" "}
                {t(`thirdPartyEvaluation.items.${answer.item}.label`)}
              </p>
              <p className="text-xs text-txt-secondary">
                {t(`thirdPartyEvaluation.items.${answer.item}.criterion`)}
              </p>
            </div>
            <Controller
              control={control}
              name={`answers.${index}.result`}
              render={({ field }) => (
                <Select
                  value={field.value}
                  onValueChange={field.onChange}
                  disabled={disabled}
                  aria-label={t(`thirdPartyEvaluation.items.${answer.item}.label`)}
                >
                  <Option value={none}>{t("thirdPartyEvaluation.notEvaluated")}</Option>
                  {results.map(r => (
                    <Option key={r} value={r}>{t(`thirdPartyEvaluation.results.${r}`)}</Option>
                  ))}
                </Select>
              )}
            />
            <div className="space-y-2">
              <Input
                {...register(`answers.${index}.note`)}
                placeholder={t("thirdPartyEvaluation.placeholders.note")}
                aria-label={t("thirdPartyEvaluation.fields.note")}
                disabled={disabled}
              />
              {answer.item === "FOREIGN_STORAGE" && (
                <Input
                  {...register("storageCountries")}
                  placeholder={t("thirdPartyEvaluation.placeholders.storageCountries")}
                  aria-label={t("thirdPartyEvaluation.fields.storageCountries")}
                  disabled={disabled}
                />
              )}
            </div>
          </li>
        ))}
      </ol>
    </section>
  );
}

// Shows how the checklist came out and what the save would reject, so the
// judgement can be chosen the way the checklist (35) says.
function JudgementHints({ control }: { control: FormControlProps["control"] }) {
  const { t } = useTranslation();
  const [answers, judgement, conditions, evaluatorId, approverId] = useWatch({
    control,
    name: ["answers", "judgement", "conditions", "evaluatorId", "approverId"],
  });

  const unsuitable = answers.filter(a => a.result === "UNSUITABLE").length;
  const notEvaluated = answers.filter(a => a.result === none).length;

  const warnings: string[] = [];
  if (judgement === "APPROVED" && unsuitable > 0) {
    warnings.push(t("thirdPartyEvaluation.warnings.approvedWithUnsuitable"));
  }
  if (judgement === "CONDITIONALLY_APPROVED" && conditions.trim() === "") {
    warnings.push(t("thirdPartyEvaluation.warnings.conditionsMissing"));
  }
  if (evaluatorId && evaluatorId === approverId) {
    warnings.push(t("thirdPartyEvaluation.warnings.sameApprover"));
  }

  return (
    <div className="space-y-2 text-sm">
      <p className="text-txt-secondary">
        {t("thirdPartyEvaluation.summary", {
          unsuitable,
          notEvaluated,
          suggestion: unsuitable > 0
            ? t("thirdPartyEvaluation.suggestions.unsuitable")
            : notEvaluated > 0
              ? t("thirdPartyEvaluation.suggestions.incomplete")
              : t("thirdPartyEvaluation.suggestions.approved"),
        })}
      </p>
      {warnings.length > 0 && (
        <div className="rounded-lg border border-warning bg-warning-subtle p-4 space-y-1" role="alert">
          {warnings.map(w => <p key={w}>{w}</p>)}
        </div>
      )}
    </div>
  );
}
