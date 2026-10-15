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
  Badge,
  Button,
  Card,
  Option,
  Select,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  useConfirm,
} from "@probo/ui";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { graphql, useFragment } from "react-relay";

import type { TrainingParticipantsCardFragment$key } from "#/__generated__/core/TrainingParticipantsCardFragment.graphql";

import {
  useRemoveTrainingParticipant,
  useRequestTrainingSignatures,
} from "../_lib/trainingGraph";
import {
  statusBadgeVariant,
  trainingParticipantStatuses,
} from "../_lib/trainingOptions";

import { AddTrainingParticipantsDialog } from "./AddTrainingParticipantsDialog";
import { EditTrainingParticipantDialog } from "./EditTrainingParticipantDialog";

const fragment = graphql`
  fragment TrainingParticipantsCardFragment on TrainingPlan {
    id
    testRequired
    canUpdate: permission(action: "core:training-plan:update")
    canRequestSignatures: permission(action: "core:document-version-signature:request")
    materials {
      # eslint-disable-next-line relay/unused-fields
      documentId
    }
    participants {
      id
      profileId
      fullName
      department
      status
      attendedOn
      attendanceSource
      signedMaterialCount
      signedOutsidePeriodCount
      # Passed on to EditTrainingParticipantDialog.
      # eslint-disable-next-line relay/unused-fields
      manualAttendedOn
      testScore
      testPassed
      # eslint-disable-next-line relay/unused-fields
      manualTestPassed
      hasRecord
      note
    }
  }
`;

const allStatuses = "ALL";

export function TrainingParticipantsCard({ planKey }: { planKey: TrainingParticipantsCardFragment$key }) {
  const { t } = useTranslation();
  const plan = useFragment(fragment, planKey);
  const [status, setStatus] = useState<string>(allStatuses);
  const [requestSignatures, isRequesting] = useRequestTrainingSignatures();
  const [removeParticipant] = useRemoveTrainingParticipant();
  const confirm = useConfirm();

  const participants = status === allStatuses
    ? plan.participants
    : plan.participants.filter(p => p.status === status);
  const materialCount = plan.materials.length;

  const onRemove = (participant: { id: string; fullName: string }) => {
    confirm(
      () => removeParticipant({ variables: { input: { trainingParticipantId: participant.id } } })
        .then(() => undefined),
      { message: t("training.participants.removeConfirm", { name: participant.fullName }) },
    );
  };

  return (
    <Card>
      <div className="p-6 space-y-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold">{t("training.participants.title")}</h2>
            <p className="text-sm text-txt-tertiary">{t("training.participants.description")}</p>
            <p className="text-xs text-txt-tertiary">{t("training.participants.recordsKept")}</p>
          </div>
          {plan.canUpdate && (
            <div className="flex gap-2">
              {plan.canRequestSignatures && (
                <Button
                  variant="secondary"
                  disabled={isRequesting || materialCount === 0 || plan.participants.length === 0}
                  onClick={() => void requestSignatures({
                    variables: { input: { trainingPlanId: plan.id } },
                  }).catch(() => {
                    // useMutation already shows the error toast.
                  })}
                >
                  {t("training.participants.requestSignatures")}
                </Button>
              )}
              <AddTrainingParticipantsDialog
                trainingPlanId={plan.id}
                participantProfileIds={plan.participants.map(p => p.profileId)}
              >
                <Button>{t("training.participants.add")}</Button>
              </AddTrainingParticipantsDialog>
            </div>
          )}
        </div>

        <div className="w-56">
          <Select value={status} onValueChange={setStatus}>
            <Option value={allStatuses}>{t("training.participants.allStatuses")}</Option>
            {trainingParticipantStatuses.map(s => (
              <Option key={s} value={s}>{t(`training.statuses.${s}`)}</Option>
            ))}
          </Select>
        </div>

        {participants.length === 0
          ? <p className="text-sm text-txt-secondary">{t("training.participants.empty")}</p>
          : (
              <Table>
                <Thead>
                  <Tr>
                    <Th>{t("training.participants.fields.fullName")}</Th>
                    <Th>{t("training.participants.fields.department")}</Th>
                    <Th>{t("training.participants.fields.status")}</Th>
                    <Th>{t("training.participants.fields.attendedOn")}</Th>
                    <Th>{t("training.participants.fields.signatures")}</Th>
                    {plan.testRequired && <Th>{t("training.participants.fields.testScore")}</Th>}
                    <Th>{t("training.participants.fields.note")}</Th>
                    {plan.canUpdate && <Th />}
                  </Tr>
                </Thead>
                <Tbody>
                  {participants.map(p => (
                    <Tr key={p.id}>
                      <Td>{p.fullName}</Td>
                      <Td>{p.department ?? ""}</Td>
                      <Td>
                        <Badge variant={statusBadgeVariant(p.status)}>
                          {t(`training.statuses.${p.status}`)}
                        </Badge>
                      </Td>
                      <Td>
                        {p.attendedOn
                          ? t("training.participants.attendedValue", {
                              date: p.attendedOn,
                              source: t(`training.sources.${p.attendanceSource}`),
                            })
                          : ""}
                      </Td>
                      <Td>
                        {t("training.participants.signaturesValue", {
                          signed: p.signedMaterialCount,
                          total: materialCount,
                        })}
                        {p.signedOutsidePeriodCount > 0 && !p.attendedOn && (
                          <p className="text-xs text-txt-warning">
                            {t("training.participants.signedOutsidePeriod", { count: p.signedOutsidePeriodCount })}
                          </p>
                        )}
                      </Td>
                      {plan.testRequired && (
                        <Td>
                          {p.testScore ?? ""}
                          {p.testPassed != null && ` (${p.testPassed ? t("training.testPassed.passed") : t("training.testPassed.failed")})`}
                        </Td>
                      )}
                      <Td className="max-w-48 truncate">{p.note}</Td>
                      {plan.canUpdate && (
                        <Td>
                          <div className="flex gap-2 justify-end">
                            <EditTrainingParticipantDialog participant={p} testRequired={plan.testRequired}>
                              <Button variant="secondary">{t("training.actions.edit")}</Button>
                            </EditTrainingParticipantDialog>
                            {/* Records are kept for the PrivacyMark audit. */}
                            {!p.hasRecord && (
                              <Button variant="quaternary" onClick={() => onRemove(p)}>
                                {t("training.actions.remove")}
                              </Button>
                            )}
                          </div>
                        </Td>
                      )}
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            )}
      </div>
    </Card>
  );
}
