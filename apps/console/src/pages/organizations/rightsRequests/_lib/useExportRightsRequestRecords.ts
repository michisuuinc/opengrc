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

import type { useExportRightsRequestRecordsMutation } from "#/__generated__/core/useExportRightsRequestRecordsMutation.graphql";
import { useMutation } from "#/lib/relay/useMutation";

const exportRightsRequestRecordsMutation = graphql`
  mutation useExportRightsRequestRecordsMutation(
    $input: ExportRightsRequestRecordsInput!
  ) {
    exportRightsRequestRecords(input: $input) {
      data
    }
  }
`;

// Downloads the PrivacyMark record of requests from individuals
// (本人からの請求の記録) as an .xlsx file. Empty dates mean no bound.
export function useExportRightsRequestRecords(organizationId: string) {
  const { t } = useTranslation();
  const [exportRecords, isExporting] = useMutation<useExportRightsRequestRecordsMutation>(
    exportRightsRequestRecordsMutation,
    { errorToast: t("rightsRequestRecord.errors.export") },
  );

  async function download(receivedFrom: string, receivedTo: string) {
    const response = await exportRecords({
      variables: {
        input: {
          organizationId,
          receivedFrom: receivedFrom || null,
          receivedTo: receivedTo || null,
        },
      },
    });
    // Local date, so the name matches the export date shown in the sheet.
    const today = new Date().toLocaleDateString("sv-SE");
    downloadFile(
      response.exportRightsRequestRecords.data,
      `${t("rightsRequestRecord.fileName")}_${today}.xlsx`,
    );
  }

  return [download, isExporting] as const;
}
