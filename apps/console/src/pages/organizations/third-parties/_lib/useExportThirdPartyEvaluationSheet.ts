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

import type { useExportThirdPartyEvaluationSheetMutation } from "#/__generated__/core/useExportThirdPartyEvaluationSheetMutation.graphql";
import { useMutation } from "#/lib/relay/useMutation";

const exportThirdPartyEvaluationSheetMutation = graphql`
  mutation useExportThirdPartyEvaluationSheetMutation(
    $input: ExportThirdPartyEvaluationSheetInput!
  ) {
    exportThirdPartyEvaluationSheet(input: $input) {
      data
    }
  }
`;

// Downloads the PrivacyMark outsourcing evaluation (委託先評価表) as an .xlsx
// file.
export function useExportThirdPartyEvaluationSheet(organizationId: string) {
  const { t } = useTranslation();
  const [exportSheet, isExporting] = useMutation<useExportThirdPartyEvaluationSheetMutation>(
    exportThirdPartyEvaluationSheetMutation,
    { errorToast: t("thirdPartyEvaluation.errors.export") },
  );

  async function download() {
    const response = await exportSheet({ variables: { input: { organizationId } } });
    // Local date, so the name matches the export date shown in the sheet.
    const today = new Date().toLocaleDateString("sv-SE");
    downloadFile(
      response.exportThirdPartyEvaluationSheet.data,
      `${t("thirdPartyEvaluation.fileName")}_${today}.xlsx`,
    );
  }

  return [download, isExporting] as const;
}
