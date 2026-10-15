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
import { Link } from "react-router";

import type { TrainingMaterialsCardFragment$key } from "#/__generated__/core/TrainingMaterialsCardFragment.graphql";
import { useOrganizationId } from "#/hooks/useOrganizationId";

import { useSetTrainingPlanMaterials } from "../_lib/trainingGraph";

const fragment = graphql`
  fragment TrainingMaterialsCardFragment on TrainingPlan {
    id
    canUpdate: permission(action: "core:training-plan:update")
    materials {
      documentId
      title
    }
    organization {
      documents(first: 500, orderBy: { field: TITLE, direction: ASC }) {
        edges {
          node {
            id
            versions(first: 1, orderBy: { field: CREATED_AT, direction: DESC }) {
              edges {
                node {
                  title
                }
              }
            }
          }
        }
      }
    }
  }
`;

function toggle(values: readonly string[], value: string, checked: boolean): string[] {
  const others = values.filter(v => v !== value);
  return checked ? [...others, value] : others;
}

// The documents participants read and sign to attend the training.
export function TrainingMaterialsCard({ planKey }: { planKey: TrainingMaterialsCardFragment$key }) {
  const plan = useFragment(fragment, planKey);
  const stored = plan.materials.map(m => m.documentId);

  return (
    <TrainingMaterialsForm
      // A new key resets the selection to what was stored after a save.
      key={stored.join(",")}
      planId={plan.id}
      canUpdate={plan.canUpdate}
      stored={stored}
      materials={plan.materials}
      documents={plan.organization.documents.edges.map(e => ({
        id: e.node.id,
        title: e.node.versions.edges[0]?.node.title ?? "",
      }))}
    />
  );
}

interface FormProps {
  planId: string;
  canUpdate: boolean;
  stored: readonly string[];
  materials: readonly { readonly documentId: string; readonly title: string }[];
  documents: readonly { id: string; title: string }[];
}

function TrainingMaterialsForm({ planId, canUpdate, stored, materials, documents }: FormProps) {
  const { t } = useTranslation();
  const organizationId = useOrganizationId();
  const [setMaterials, isSaving] = useSetTrainingPlanMaterials();
  const [editing, setEditing] = useState(false);
  const [selected, setSelected] = useState<string[]>([...stored]);

  const onSave = async () => {
    await setMaterials({
      variables: { input: { trainingPlanId: planId, documentIds: selected } },
    })
      .then(() => setEditing(false))
      .catch(() => {
        // useMutation already shows the error toast.
      });
  };

  return (
    <Card>
      <div className="p-6 space-y-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold">{t("training.materials.title")}</h2>
            <p className="text-sm text-txt-tertiary">{t("training.materials.description")}</p>
          </div>
          {canUpdate && !editing && (
            <Button variant="secondary" onClick={() => setEditing(true)}>
              {t("training.materials.edit")}
            </Button>
          )}
        </div>

        {!editing && (
          materials.length === 0
            ? <p className="text-sm text-txt-secondary">{t("training.materials.empty")}</p>
            : (
                <ul className="list-disc pl-5 space-y-1 text-sm">
                  {materials.map(m => (
                    <li key={m.documentId}>
                      <Link
                        className="underline"
                        to={`/organizations/${organizationId}/governance/documents/${m.documentId}`}
                      >
                        {m.title}
                      </Link>
                    </li>
                  ))}
                </ul>
              )
        )}

        {editing && (
          <>
            <div className="max-h-64 overflow-y-auto space-y-2">
              {documents.map(document => (
                <label key={document.id} className="flex items-center gap-2 text-sm text-txt-primary">
                  <Checkbox
                    checked={selected.includes(document.id)}
                    onChange={checked => setSelected(toggle(selected, document.id, checked))}
                  />
                  {document.title}
                </label>
              ))}
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="secondary" onClick={() => setEditing(false)}>
                {t("training.actions.cancel")}
              </Button>
              <Button disabled={isSaving} onClick={() => void onSave()}>
                {t("training.actions.save")}
              </Button>
            </div>
          </>
        )}
      </div>
    </Card>
  );
}
