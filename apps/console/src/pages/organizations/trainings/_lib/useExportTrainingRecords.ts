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

import { downloadFile } from "@probo/helpers";
import { useTranslation } from "react-i18next";
import { graphql } from "relay-runtime";

import type { useExportTrainingRecordsMutation } from "#/__generated__/core/useExportTrainingRecordsMutation.graphql";
import { useMutation } from "#/lib/relay/useMutation";

const exportTrainingRecordsMutation = graphql`
  mutation useExportTrainingRecordsMutation(
    $input: ExportTrainingRecordsInput!
  ) {
    exportTrainingRecords(input: $input) {
      data
    }
  }
`;

// Downloads the training records (教育実施記録) as an .xlsx file, for one
// fiscal year or for every year when fiscalYear is null.
export function useExportTrainingRecords(organizationId: string) {
  const { t } = useTranslation();
  const [exportRecords, isExporting] = useMutation<useExportTrainingRecordsMutation>(
    exportTrainingRecordsMutation,
    { errorToast: t("training.errors.export") },
  );

  async function download(fiscalYear: number | null) {
    const response = await exportRecords({
      variables: { input: { organizationId, fiscalYear } },
    });
    // Local date, so the name matches the export date shown in the sheet.
    const today = new Date().toLocaleDateString("sv-SE");
    const suffix = fiscalYear === null ? today : `${fiscalYear}_${today}`;
    downloadFile(
      response.exportTrainingRecords.data,
      `${t("training.fileName")}_${suffix}.xlsx`,
    );
  }

  return [download, isExporting] as const;
}
