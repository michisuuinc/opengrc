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

import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { graphql } from "react-relay";

import type { ViewerLocaleSyncMutation } from "#/__generated__/iam/ViewerLocaleSyncMutation.graphql";
import { useMutation } from "#/lib/relay/useMutation";

const updateViewerLocaleMutation = graphql`
  mutation ViewerLocaleSyncMutation($input: UpdateViewerLocaleInput!) {
    updateViewerLocale(input: $input) {
      identity {
        id
        locale
      }
    }
  }
`;

// Short tags accepted for identities.locale (iam.SupportedIdentityLocales).
const SUPPORTED_LOCALES = new Set([
  "en", "fr", "de", "es", "id", "it", "ja", "ko", "nl", "pl", "pt", "tr", "uk", "zh",
]);

// Fork: saves the language the console is shown in as the person's
// language (identities.locale), so that emails reach them in it. It runs
// once per page load and only when the saved language differs. Failures
// are ignored: the console works the same without it.
export function ViewerLocaleSync({ savedLocale }: { savedLocale: string | null | undefined }) {
  const { i18n } = useTranslation();
  const [commit] = useMutation<ViewerLocaleSyncMutation>(updateViewerLocaleMutation, { errorToast: false });
  const done = useRef(false);

  const locale = i18n.language.toLowerCase().split("-")[0];

  useEffect(() => {
    if (done.current || !SUPPORTED_LOCALES.has(locale) || locale === savedLocale) {
      return;
    }

    done.current = true;
    void commit({ variables: { input: { locale } } }).catch(() => {});
  }, [commit, locale, savedLocale]);

  return null;
}
