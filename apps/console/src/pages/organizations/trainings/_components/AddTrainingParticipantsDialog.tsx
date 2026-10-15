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
  Checkbox,
  Dialog,
  DialogContent,
  DialogFooter,
  useDialogRef,
} from "@probo/ui";
import { type ReactNode, Suspense, useState } from "react";
import { useTranslation } from "react-i18next";

import { usePeople } from "#/hooks/graph/PeopleGraph";
import { useOrganizationId } from "#/hooks/useOrganizationId";

import { useAddTrainingParticipants } from "../_lib/trainingGraph";

type Props = {
  children: ReactNode;
  trainingPlanId: string;
  participantProfileIds: readonly string[];
};

// Picks members of the organization to take the training. Members whose
// contract has ended are not offered.
export function AddTrainingParticipantsDialog({ children, trainingPlanId, participantProfileIds }: Props) {
  const { t } = useTranslation();
  const ref = useDialogRef();

  return (
    <Dialog
      ref={ref}
      trigger={children}
      title={<Breadcrumb items={[t("training.title"), t("training.participants.add")]} />}
    >
      <Suspense fallback={<DialogContent padded>{t("training.loading")}</DialogContent>}>
        <AddTrainingParticipantsForm
          trainingPlanId={trainingPlanId}
          participantProfileIds={participantProfileIds}
          onDone={() => ref.current?.close()}
        />
      </Suspense>
    </Dialog>
  );
}

function AddTrainingParticipantsForm({
  trainingPlanId,
  participantProfileIds,
  onDone,
}: Omit<Props, "children"> & { onDone: () => void }) {
  const { t } = useTranslation();
  const organizationId = useOrganizationId();
  const people = usePeople(organizationId, { contractEnded: false });
  const candidates = people.filter(p => !participantProfileIds.includes(p.id));
  const [selected, setSelected] = useState<string[]>([]);
  const [addParticipants, isSaving] = useAddTrainingParticipants();

  const onSubmit = async () => {
    await addParticipants({
      variables: { input: { trainingPlanId, profileIds: selected } },
    })
      .then(() => {
        setSelected([]);
        onDone();
      })
      .catch(() => {
        // useMutation already shows the error toast.
      });
  };

  return (
    <>
      <DialogContent padded className="space-y-4">
        {candidates.length === 0
          ? <p className="text-sm text-txt-secondary">{t("training.participants.noCandidates")}</p>
          : (
              <>
                <div className="flex justify-between items-center">
                  <p className="text-sm text-txt-secondary">
                    {t("training.participants.selectedCount", { count: selected.length })}
                  </p>
                  <Button
                    variant="secondary"
                    onClick={() => setSelected(
                      selected.length === candidates.length ? [] : candidates.map(p => p.id),
                    )}
                  >
                    {selected.length === candidates.length
                      ? t("training.participants.clearAll")
                      : t("training.participants.selectAll")}
                  </Button>
                </div>
                <div className="max-h-80 overflow-y-auto space-y-2">
                  {candidates.map(person => (
                    <label key={person.id} className="flex items-center gap-2 text-sm text-txt-primary">
                      <Checkbox
                        checked={selected.includes(person.id)}
                        onChange={checked => setSelected(
                          checked ? [...selected, person.id] : selected.filter(id => id !== person.id),
                        )}
                      />
                      {person.fullName}
                      <span className="text-txt-tertiary">{person.emailAddress}</span>
                    </label>
                  ))}
                </div>
              </>
            )}
      </DialogContent>
      <DialogFooter>
        <Button disabled={isSaving || selected.length === 0} onClick={() => void onSubmit()}>
          {t("training.participants.addSelected")}
        </Button>
      </DialogFooter>
    </>
  );
}
