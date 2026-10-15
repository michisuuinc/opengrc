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

import { usePageTitle } from "@probo/hooks";
import { Breadcrumb, Button, Card, PageHeader, useConfirm } from "@probo/ui";
import { useTranslation } from "react-i18next";
import { graphql, type PreloadedQuery, useFragment, usePreloadedQuery } from "react-relay";
import { useNavigate } from "react-router";
import { ConnectionHandler } from "relay-runtime";

import type { trainingGraphNodeQuery } from "#/__generated__/core/trainingGraphNodeQuery.graphql";
import type { TrainingPlanDetailsPageFragment$key } from "#/__generated__/core/TrainingPlanDetailsPageFragment.graphql";
import { useOrganizationId } from "#/hooks/useOrganizationId";

import { TrainingMaterialsCard } from "./_components/TrainingMaterialsCard";
import { TrainingParticipantsCard } from "./_components/TrainingParticipantsCard";
import { TrainingPlanFormDialog } from "./_components/TrainingPlanFormDialog";
import { trainingPlanNodeQuery, useDeleteTrainingPlan } from "./_lib/trainingGraph";

const fragment = graphql`
  fragment TrainingPlanDetailsPageFragment on TrainingPlan {
    id
    fiscalYear
    name
    kind
    description
    method
    instructor
    startsOn
    endsOn
    timeZone
    testRequired
    passScore
    canUpdate: permission(action: "core:training-plan:update")
    canDelete: permission(action: "core:training-plan:delete")
    summary {
      totalCount
      completedCount
    }
    ...TrainingMaterialsCardFragment
    ...TrainingParticipantsCardFragment
  }
`;

type Props = {
  queryRef: PreloadedQuery<trainingGraphNodeQuery>;
};

export default function TrainingPlanDetailsPage({ queryRef }: Props) {
  const data = usePreloadedQuery<trainingGraphNodeQuery>(trainingPlanNodeQuery, queryRef);

  return <TrainingPlanDetails planKey={data.node} />;
}

function TrainingPlanDetails({ planKey }: { planKey: TrainingPlanDetailsPageFragment$key }) {
  const { t } = useTranslation();
  const organizationId = useOrganizationId();
  const navigate = useNavigate();
  const confirm = useConfirm();
  const plan = useFragment(fragment, planKey);
  const [deletePlan] = useDeleteTrainingPlan();
  usePageTitle(plan.name);

  const listPath = `/organizations/${organizationId}/governance/trainings`;

  const onDelete = () => {
    const connectionId = ConnectionHandler.getConnectionID(
      organizationId,
      "TrainingPlansPage_trainingPlans",
      { orderBy: { field: "STARTS_ON", direction: "DESC" } },
    );
    confirm(
      () => deletePlan({
        variables: { input: { trainingPlanId: plan.id }, connections: [connectionId] },
      }).then(() => void navigate(listPath)),
      { message: t("training.deleteConfirm", { name: plan.name }) },
    );
  };

  const details: [string, string][] = [
    [t("training.fields.fiscalYear"), t("training.fiscalYearLabel", { year: plan.fiscalYear })],
    [t("training.fields.kind"), t(`training.kinds.${plan.kind}`)],
    [t("training.fields.method"), t(`training.methods.${plan.method}`)],
    [t("training.fields.period"), t("training.periodValue", { startsOn: plan.startsOn, endsOn: plan.endsOn })],
    [t("training.fields.instructor"), plan.instructor],
    [
      t("training.fields.testRequired"),
      plan.testRequired
        ? t("training.testRequiredValue", { passScore: plan.passScore ?? "-" })
        : t("training.testRequired.no"),
    ],
    [
      t("training.fields.progress"),
      t("training.progressValue", { completed: plan.summary.completedCount, total: plan.summary.totalCount }),
    ],
  ];

  return (
    <div className="space-y-6">
      <Breadcrumb items={[{ label: t("training.title"), to: listPath }, { label: plan.name }]} />
      <PageHeader title={plan.name} description={plan.description}>
        <div className="flex gap-2">
          {plan.canUpdate && (
            <TrainingPlanFormDialog
              mode="edit"
              organizationId={organizationId}
              trainingPlanId={plan.id}
              initialValues={{
                fiscalYear: plan.fiscalYear,
                name: plan.name,
                kind: plan.kind,
                description: plan.description,
                method: plan.method,
                instructor: plan.instructor,
                startsOn: plan.startsOn,
                endsOn: plan.endsOn,
                testRequired: plan.testRequired,
                passScore: plan.passScore ?? null,
              }}
            >
              <Button variant="secondary">{t("training.actions.edit")}</Button>
            </TrainingPlanFormDialog>
          )}
          {plan.canDelete && (
            <Button variant="danger" onClick={onDelete}>
              {t("training.actions.delete")}
            </Button>
          )}
        </div>
      </PageHeader>

      <Card>
        <dl className="p-6 grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-3 text-sm">
          {details.map(([label, value]) => (
            <div key={label} className="flex gap-4">
              <dt className="w-32 shrink-0 text-txt-secondary">{label}</dt>
              <dd className="text-txt-primary">{value}</dd>
            </div>
          ))}
        </dl>
        <p className="px-6 pb-6 text-xs text-txt-tertiary">
          {t("training.timeZoneNote", { timeZone: plan.timeZone })}
        </p>
      </Card>

      <TrainingMaterialsCard planKey={plan} />
      <TrainingParticipantsCard planKey={plan} />
    </div>
  );
}
