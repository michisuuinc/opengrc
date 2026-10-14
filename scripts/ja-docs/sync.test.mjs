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

import { test } from "node:test";
import assert from "node:assert/strict";

import { classification, documentType, expandSections, planDocument } from "./sync.mjs";

const SECTIONS = ["J.1.1", "J.1.2", "J.1.3", "J.2.1", "J.4.4.2", "J.9.2", "J.10.1", "J.10.2"];
const CONTROLS = Object.fromEntries(SECTIONS.map(s => [s, `id-${s}`]));
const INTERNAL_CONTROLS = {
  "組織の状況": { id: "ic-1", sections: ["J.1.1", "J.1.2"] },
  "安全管理": { id: "ic-2", sections: ["J.9.2"] },
  "緊急事態への準備と漏えい等への対応": { id: "ic-3", sections: ["J.4.4.2"] },
  "開示等": { id: "ic-4", sections: ["J.10.1"] },
};

const document = (number, row) => `# 題名\n\n| 対応する J 番号 | ${row} |\n\n本文\n`;

test("expandSections expands ranges in section order", () => {
  assert.deepEqual([...expandSections("J.1.2〜J.2.1、J.10.1", SECTIONS)], ["J.1.2", "J.1.3", "J.2.1", "J.10.1"]);
});

test("expandSections orders J.10 after J.9", () => {
  assert.deepEqual([...expandSections("J.9.2〜J.10.2", SECTIONS)], ["J.9.2", "J.10.1", "J.10.2"]);
});

test("expandSections rejects unknown numbers", () => {
  assert.throws(() => expandSections("J.99", SECTIONS), /unknown J number/);
  assert.throws(() => expandSections("J.2.1〜J.1.1", SECTIONS), /unknown range/);
});

test("documentType and classification follow the template number", () => {
  assert.equal(documentType("01"), "POLICY");
  assert.equal(documentType("21"), "POLICY");
  assert.equal(documentType("12"), "PROCEDURE");
  assert.equal(documentType("33"), "TEMPLATE");
  assert.equal(classification("01"), "PUBLIC");
  assert.equal(classification("02"), "INTERNAL");
  assert.equal(classification("20"), "PUBLIC");
  assert.equal(classification("34"), "INTERNAL");
});

test("planDocument links internal controls sharing a J number", () => {
  const plan = planDocument("12-security-rules.md", document("12", "J.9.2、J.1.1"), CONTROLS, INTERNAL_CONTROLS);

  assert.equal(plan.title, "題名");
  assert.deepEqual(plan.sections, ["J.1.1", "J.9.2"]);
  assert.deepEqual(plan.internalControls, ["安全管理", "組織の状況"]);
});

test("planDocument uses the override for the incident procedure", () => {
  const plan = planDocument("15-incident-procedure.md", document("15", "J.4.4.2、J.9.2"), CONTROLS, INTERNAL_CONTROLS);

  assert.deepEqual(plan.internalControls, ["緊急事態への準備と漏えい等への対応"]);
});

test("planDocument requires a title and a J number row", () => {
  assert.throws(() => planDocument("10-x.md", "本文\n", CONTROLS, INTERNAL_CONTROLS), /title/);
  assert.throws(() => planDocument("10-x.md", "# 題名\n本文\n", CONTROLS, INTERNAL_CONTROLS), /J 番号/);
});
