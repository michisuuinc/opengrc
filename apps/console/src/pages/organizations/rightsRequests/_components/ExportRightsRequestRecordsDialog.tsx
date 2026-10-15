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
  useDialogRef,
} from "@probo/ui";
import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";

import { type RightsRequestExportKind, useExportRightsRequestRecords } from "../_lib/useExportRightsRequestRecords";

interface Props {
  children: ReactNode;
  organizationId: string;
  kind: RightsRequestExportKind;
}

// Asks for the receipt period, such as the fiscal year under review, and
// downloads the record of requests received in it.
export function ExportRightsRequestRecordsDialog({ children, organizationId, kind }: Props) {
  const { t } = useTranslation();
  const ref = useDialogRef();
  const [receivedFrom, setReceivedFrom] = useState("");
  const [receivedTo, setReceivedTo] = useState("");
  const [exportRecords, isExporting] = useExportRightsRequestRecords(organizationId, kind);

  const invalid = receivedFrom !== "" && receivedTo !== "" && receivedTo < receivedFrom;

  const onExport = async () => {
    await exportRecords(receivedFrom, receivedTo)
      .then(() => ref.current?.close())
      .catch(() => {
        // useMutation already shows the error toast.
      });
  };

  return (
    <Dialog
      ref={ref}
      trigger={children}
      title={<Breadcrumb items={[t("rightsRequestsPage.title"), t(`rightsRequestRecord.export.${kind}.title`)]} />}
    >
      <DialogContent padded className="space-y-4">
        <p className="text-sm text-txt-secondary">{t(`rightsRequestRecord.export.${kind}.description`)}</p>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Field name="rr-export-from" label={t("rightsRequestRecord.export.from")}>
            <Input
              id="rr-export-from"
              type="date"
              value={receivedFrom}
              onChange={e => setReceivedFrom(e.target.value)}
            />
          </Field>
          <Field name="rr-export-to" label={t("rightsRequestRecord.export.to")}>
            <Input
              id="rr-export-to"
              type="date"
              value={receivedTo}
              onChange={e => setReceivedTo(e.target.value)}
            />
          </Field>
        </div>
        {invalid && (
          <p className="text-sm text-txt-danger" role="alert">{t("rightsRequestRecord.export.invalidPeriod")}</p>
        )}
      </DialogContent>
      <DialogFooter>
        <Button
          type="button"
          variant="primary"
          disabled={isExporting || invalid}
          onClick={() => void onExport()}
        >
          {t(`rightsRequestRecord.export.${kind}.action`)}
        </Button>
      </DialogFooter>
    </Dialog>
  );
}
