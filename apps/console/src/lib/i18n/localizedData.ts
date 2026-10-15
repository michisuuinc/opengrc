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

// Picks the static catalog files under /data that match the UI language.
// Translated copies live in a subdirectory named after the locale
// (e.g. /data/frameworks/ja-JP/ISO27001-2022.json). The lists below must
// match the files on disk: the console is served with an SPA fallback, so a
// missing file would come back as index.html instead of a 404.

const LOCALIZED_FRAMEWORKS: Readonly<Record<string, readonly string[]>> = {
  "ja-JP": ["ISO27001-2022", "ISO27701-2025"],
};

const LOCALIZED_RISK_TEMPLATES: readonly string[] = ["ja-JP"];

// Frameworks that exist only for one jurisdiction and have no English
// source, keyed by the id the framework selector emits.
const JURISDICTION_FRAMEWORKS: Readonly<Record<string, string>> = {
  "PMARK-JISQ15001-2023": "jp/PMARK-JISQ15001-2023.json",
};

export function frameworkDataPath(frameworkId: string, language: string): string {
  const jurisdictionFile = JURISDICTION_FRAMEWORKS[frameworkId];
  if (jurisdictionFile) {
    return `/data/frameworks/${jurisdictionFile}`;
  }
  if (LOCALIZED_FRAMEWORKS[language]?.includes(frameworkId)) {
    return `/data/frameworks/${language}/${frameworkId}.json`;
  }
  return `/data/frameworks/${frameworkId}.json`;
}

export function riskTemplatesPath(language: string): string {
  return LOCALIZED_RISK_TEMPLATES.includes(language)
    ? `/data/risks/${language}/risks.json`
    : "/data/risks/risks.json";
}
