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

import { useUpdateTrainingParticipant } from "../_lib/trainingGraph";

export interface TrainingParticipantRecord {
  id: string;
  fullName: string;
  manualAttendedOn: string | null | undefined;
  testScore: number | null | undefined;
  manualTestPassed: boolean | null | undefined;
  note: string;
}

type Props = {
  children: ReactNode;
  participant: TrainingParticipantRecord;
  testRequired: boolean;
};

const automatic = "AUTO";

// Edits what is entered by hand for one participant: the attendance date of
// training without signatures, the test result and a note.
export function EditTrainingParticipantDialog({ children, participant, testRequired }: Props) {
  const { t } = useTranslation();
  const ref = useDialogRef();
  const [updateParticipant] = useUpdateTrainingParticipant();

  const schema = z.object({
    attendedOn: z.string().regex(/^(\d{4}-\d{2}-\d{2})?$/, t("training.validation.dateRequired")),
    testScore: z.union([z.nan(), z.number().int().min(0).max(100)]).optional(),
    testPassed: z.enum([automatic, "true", "false"]),
    note: z.string(),
  });

  const { control, handleSubmit, register, formState } = useFormWithSchema(schema, {
    defaultValues: {
      attendedOn: participant.manualAttendedOn ?? "",
      testScore: participant.testScore ?? undefined,
      testPassed: participant.manualTestPassed == null
        ? automatic
        : participant.manualTestPassed ? "true" : "false",
      note: participant.note,
    },
  });

  const onSubmit = async (data: z.infer<typeof schema>) => {
    const score = data.testScore === undefined || Number.isNaN(data.testScore) ? null : data.testScore;
    await updateParticipant({
      variables: {
        input: {
          trainingParticipantId: participant.id,
          attendedOn: data.attendedOn === "" ? null : data.attendedOn,
          testScore: score,
          testPassed: data.testPassed === automatic ? null : data.testPassed === "true",
          note: data.note,
        },
      },
    })
      .then(() => ref.current?.close())
      .catch(() => {
        // useMutation already shows the error toast.
      });
  };

  return (
    <Dialog
      ref={ref}
      trigger={children}
      title={<Breadcrumb items={[t("training.title"), participant.fullName]} />}
    >
      <form onSubmit={e => void handleSubmit(onSubmit)(e)}>
        <DialogContent padded className="space-y-4">
          <Field
            label={t("training.participants.fields.manualAttendedOn")}
            help={t("training.help.manualAttendedOn")}
            error={formState.errors.attendedOn?.message}
          >
            <Input {...register("attendedOn")} type="date" />
          </Field>
          {testRequired && (
            <div className="grid grid-cols-2 gap-4">
              <Field
                label={t("training.participants.fields.testScore")}
                {...register("testScore", { valueAsNumber: true })}
                type="number"
                error={formState.errors.testScore?.message}
              />
              <ControlledField
                control={control}
                name="testPassed"
                type="select"
                label={t("training.participants.fields.testPassed")}
              >
                <Option value={automatic}>{t("training.testPassed.auto")}</Option>
                <Option value="true">{t("training.testPassed.passed")}</Option>
                <Option value="false">{t("training.testPassed.failed")}</Option>
              </ControlledField>
            </div>
          )}
          <Field
            label={t("training.participants.fields.note")}
            {...register("note")}
            type="textarea"
          />
        </DialogContent>
        <DialogFooter>
          <Button disabled={formState.isSubmitting} type="submit">
            {t("training.actions.save")}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  );
}
