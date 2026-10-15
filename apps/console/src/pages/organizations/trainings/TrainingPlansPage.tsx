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
import {
  Badge,
  Button,
  IconArrowDown,
  IconPlusLarge,
  Option,
  PageHeader,
  Select,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
} from "@probo/ui";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { graphql, type PreloadedQuery, useFragment, usePreloadedQuery } from "react-relay";

import type { trainingGraphListQuery } from "#/__generated__/core/trainingGraphListQuery.graphql";
import type { TrainingPlansPageRowFragment$key } from "#/__generated__/core/TrainingPlansPageRowFragment.graphql";
import { useOrganizationId } from "#/hooks/useOrganizationId";

import { TrainingPlanFormDialog } from "./_components/TrainingPlanFormDialog";
import { trainingPlansQuery } from "./_lib/trainingGraph";
import { currentFiscalYear } from "./_lib/trainingOptions";
import { useExportTrainingRecords } from "./_lib/useExportTrainingRecords";

const rowFragment = graphql`
  fragment TrainingPlansPageRowFragment on TrainingPlan {
    id
    fiscalYear
    name
    kind
    startsOn
    endsOn
    summary {
      totalCount
      completedCount
      notAttendedCount
    }
  }
`;

const allYears = "ALL";

type Props = {
  queryRef: PreloadedQuery<trainingGraphListQuery>;
};

export default function TrainingPlansPage({ queryRef }: Props) {
  const { t } = useTranslation();
  const organizationId = useOrganizationId();
  usePageTitle(t("training.title"));

  const data = usePreloadedQuery<trainingGraphListQuery>(trainingPlansQuery, queryRef);
  const connection = data.node.trainingPlans;
  const plans = connection?.edges.map(edge => edge.node) ?? [];
  const [exportRecords, isExporting] = useExportTrainingRecords(organizationId);

  const [year, setYear] = useState<string>(String(currentFiscalYear()));

  return (
    <div className="space-y-6">
      <PageHeader title={t("training.title")} description={t("training.description")}>
        <div className="flex gap-2">
          <Button
            variant="secondary"
            icon={IconArrowDown}
            disabled={isExporting}
            onClick={() => void exportRecords(year === allYears ? null : Number(year)).catch(() => {
              // useMutation already shows the error toast.
            })}
          >
            {t("training.actions.export")}
          </Button>
          {data.node.canCreateTrainingPlan && connection && (
            <TrainingPlanFormDialog
              mode="create"
              organizationId={organizationId}
              connectionId={connection.__id}
            >
              <Button icon={IconPlusLarge}>{t("training.actions.create")}</Button>
            </TrainingPlanFormDialog>
          )}
        </div>
      </PageHeader>

      <YearSelect value={year} onChange={setYear} />

      <Table>
        <Thead>
          <Tr>
            <Th>{t("training.fields.name")}</Th>
            <Th>{t("training.fields.fiscalYear")}</Th>
            <Th>{t("training.fields.kind")}</Th>
            <Th>{t("training.fields.period")}</Th>
            <Th>{t("training.fields.progress")}</Th>
          </Tr>
        </Thead>
        <Tbody>
          {plans.map(plan => (
            <TrainingPlanRow
              key={plan.id}
              planKey={plan}
              organizationId={organizationId}
              year={year}
            />
          ))}
        </Tbody>
      </Table>
      {plans.length === 0 && (
        <p className="text-sm text-txt-secondary">{t("training.empty")}</p>
      )}
    </div>
  );
}

function YearSelect({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  const { t } = useTranslation();
  const years = useMemo(() => {
    const current = currentFiscalYear();
    return [current + 1, current, current - 1, current - 2, current - 3];
  }, []);

  return (
    <div className="w-48">
      <Select value={value} onValueChange={onChange}>
        <Option value={allYears}>{t("training.allYears")}</Option>
        {years.map(y => (
          <Option key={y} value={String(y)}>
            {t("training.fiscalYearLabel", { year: y })}
          </Option>
        ))}
      </Select>
    </div>
  );
}

function TrainingPlanRow({
  planKey,
  organizationId,
  year,
}: {
  planKey: TrainingPlansPageRowFragment$key;
  organizationId: string;
  year: string;
}) {
  const { t } = useTranslation();
  const plan = useFragment(rowFragment, planKey);

  if (year !== allYears && String(plan.fiscalYear) !== year) {
    return null;
  }

  const { totalCount, completedCount, notAttendedCount } = plan.summary;

  return (
    <Tr to={`/organizations/${organizationId}/governance/trainings/${plan.id}`}>
      <Td>{plan.name}</Td>
      <Td>{t("training.fiscalYearLabel", { year: plan.fiscalYear })}</Td>
      <Td>{t(`training.kinds.${plan.kind}`)}</Td>
      <Td>{t("training.periodValue", { startsOn: plan.startsOn, endsOn: plan.endsOn })}</Td>
      <Td>
        <div className="flex items-center gap-2">
          <Badge variant={totalCount > 0 && completedCount === totalCount ? "success" : "neutral"}>
            {t("training.progressValue", { completed: completedCount, total: totalCount })}
          </Badge>
          {notAttendedCount > 0 && (
            <span className="text-xs text-txt-secondary">
              {t("training.notAttendedValue", { count: notAttendedCount })}
            </span>
          )}
        </div>
      </Td>
    </Tr>
  );
}
