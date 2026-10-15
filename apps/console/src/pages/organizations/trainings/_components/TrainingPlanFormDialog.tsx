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

import {
  Breadcrumb,
  Button,
  Dialog,
  DialogContent,
  DialogFooter,
  Field,
  Input,
  Option,
  useDialogRef,
} from "@probo/ui";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { ControlledField } from "#/components/form/ControlledField";
import { useFormWithSchema } from "#/hooks/useFormWithSchema";
import { z } from "#/lib/zod";

import { useCreateTrainingPlan, useUpdateTrainingPlan } from "../_lib/trainingGraph";
import {
  currentFiscalYear,
  defaultTrainingTimeZone,
  type TrainingPlanKind,
  trainingPlanKinds,
  type TrainingPlanMethod,
  trainingPlanMethods,
} from "../_lib/trainingOptions";

export interface TrainingPlanFormValues {
  fiscalYear: number;
  name: string;
  kind: TrainingPlanKind;
  description: string;
  method: TrainingPlanMethod;
  instructor: string;
  startsOn: string;
  endsOn: string;
  testRequired: boolean;
  passScore: number | null;
}

type Props = {
  children: ReactNode;
  organizationId: string;
} & (
  | { mode: "create"; connectionId: string }
  | { mode: "edit"; trainingPlanId: string; initialValues: TrainingPlanFormValues }
);

const datePattern = /^\d{4}-\d{2}-\d{2}$/;

export function TrainingPlanFormDialog(props: Props) {
  const { t } = useTranslation();
  const ref = useDialogRef();
  const [createPlan] = useCreateTrainingPlan();
  const [updatePlan] = useUpdateTrainingPlan();

  const schema = z
    .object({
      fiscalYear: z.number().int().min(2000).max(2100),
      name: z.string().trim().min(1, t("training.validation.nameRequired")),
      kind: z.enum(trainingPlanKinds as [TrainingPlanKind, ...TrainingPlanKind[]]),
      description: z.string(),
      method: z.enum(trainingPlanMethods as [TrainingPlanMethod, ...TrainingPlanMethod[]]),
      instructor: z.string(),
      startsOn: z.string().regex(datePattern, t("training.validation.dateRequired")),
      endsOn: z.string().regex(datePattern, t("training.validation.dateRequired")),
      testRequired: z.enum(["true", "false"]),
      passScore: z.union([z.nan(), z.number().int().min(0).max(100)]).optional(),
    })
    .refine(v => v.startsOn <= v.endsOn, {
      message: t("training.validation.period"),
      path: ["endsOn"],
    });

  const initial = props.mode === "edit"
    ? props.initialValues
    : null;

  const { control, handleSubmit, register, formState, reset } = useFormWithSchema(schema, {
    defaultValues: {
      fiscalYear: initial?.fiscalYear ?? currentFiscalYear(),
      name: initial?.name ?? "",
      kind: initial?.kind ?? "PERIODIC",
      description: initial?.description ?? "",
      method: initial?.method ?? "E_LEARNING",
      instructor: initial?.instructor ?? "",
      startsOn: initial?.startsOn ?? "",
      endsOn: initial?.endsOn ?? "",
      testRequired: initial?.testRequired ? "true" : "false",
      passScore: initial?.passScore ?? undefined,
    },
  });

  const onSubmit = async (data: z.infer<typeof schema>) => {
    const values = {
      fiscalYear: data.fiscalYear,
      name: data.name,
      kind: data.kind,
      description: data.description,
      method: data.method,
      instructor: data.instructor,
      startsOn: data.startsOn,
      endsOn: data.endsOn,
      testRequired: data.testRequired === "true",
      passScore: data.passScore === undefined || Number.isNaN(data.passScore) ? null : data.passScore,
    };

    try {
      if (props.mode === "create") {
        await createPlan({
          variables: {
            input: {
              ...values,
              organizationId: props.organizationId,
              timeZone: defaultTrainingTimeZone(),
            },
            connections: [props.connectionId],
          },
        });
        reset();
      } else {
        await updatePlan({
          variables: { input: { ...values, trainingPlanId: props.trainingPlanId } },
        });
      }
      ref.current?.close();
    } catch {
      // useMutation already shows the error toast.
    }
  };

  return (
    <Dialog
      ref={ref}
      trigger={props.children}
      title={(
        <Breadcrumb
          items={[
            t("training.title"),
            props.mode === "create" ? t("training.actions.create") : t("training.actions.edit"),
          ]}
        />
      )}
    >
      <form onSubmit={e => void handleSubmit(onSubmit)(e)}>
        <DialogContent padded className="space-y-4">
          <Field
            label={t("training.fields.name")}
            {...register("name")}
            type="text"
            error={formState.errors.name?.message}
          />
          <div className="grid grid-cols-3 gap-4">
            <Field
              label={t("training.fields.fiscalYear")}
              {...register("fiscalYear", { valueAsNumber: true })}
              type="number"
              error={formState.errors.fiscalYear?.message}
            />
            <ControlledField
              control={control}
              name="kind"
              type="select"
              label={t("training.fields.kind")}
            >
              {trainingPlanKinds.map(kind => (
                <Option key={kind} value={kind}>
                  {t(`training.kinds.${kind}`)}
                </Option>
              ))}
            </ControlledField>
            <ControlledField
              control={control}
              name="method"
              type="select"
              label={t("training.fields.method")}
            >
              {trainingPlanMethods.map(method => (
                <Option key={method} value={method}>
                  {t(`training.methods.${method}`)}
                </Option>
              ))}
            </ControlledField>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <Field label={t("training.fields.startsOn")} error={formState.errors.startsOn?.message}>
              <Input {...register("startsOn")} type="date" />
            </Field>
            <Field label={t("training.fields.endsOn")} error={formState.errors.endsOn?.message}>
              <Input {...register("endsOn")} type="date" />
            </Field>
          </div>
          <Field
            label={t("training.fields.instructor")}
            {...register("instructor")}
            type="text"
          />
          <Field
            label={t("training.fields.description")}
            help={t("training.help.description")}
            {...register("description")}
            type="textarea"
          />
          <div className="grid grid-cols-2 gap-4">
            <ControlledField
              control={control}
              name="testRequired"
              type="select"
              label={t("training.fields.testRequired")}
            >
              <Option value="true">{t("training.testRequired.yes")}</Option>
              <Option value="false">{t("training.testRequired.no")}</Option>
            </ControlledField>
            <Field
              label={t("training.fields.passScore")}
              help={t("training.help.passScore")}
              {...register("passScore", { valueAsNumber: true })}
              type="number"
              error={formState.errors.passScore?.message}
            />
          </div>
        </DialogContent>
        <DialogFooter>
          <Button disabled={formState.isSubmitting} type="submit">
            {props.mode === "create" ? t("training.actions.create") : t("training.actions.save")}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  );
}
