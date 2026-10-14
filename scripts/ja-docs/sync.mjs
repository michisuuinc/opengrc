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

// Registers the finished PrivacyMark documents made by ja-docgen in OpenGRC
// as drafts, and links each one to the PrivacyMark framework controls listed
// in its "対応する J 番号" row and to the internal controls that share those
// J numbers.
//
// It is safe to run again: a document whose title already exists is not
// created twice, and existing links are kept. The content of an existing
// document is never updated; edit it in OpenGRC instead.
//
// Usage (after `bin/prb auth login --hostname <url>`):
//   node scripts/ja-docs/sync.mjs --org <organization-id> --docs <dir>          # dry run
//   node scripts/ja-docs/sync.mjs --org <organization-id> --docs <dir> --apply

import { execFileSync } from "node:child_process";
import { mkdtempSync, readdirSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const PMARK_FRAMEWORK = "プライバシーマーク（JIS Q 15001:2023）";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const DOCUMENT_FILE = /^(\d\d)-.*\.md$/;
const J_ROW = /対応する J 番号 \| (.*?) \|/;

// Internal controls to link instead of every one sharing a J number, where
// sharing a J number is too broad.
export const INTERNAL_CONTROL_OVERRIDES = {
  15: ["緊急事態への準備と漏えい等への対応"],
};

const LIST_QUERY = `
query($id: ID!) { node(id: $id) { ... on Organization {
  documents(first: 200) { edges { node { id versions(first: 1) { edges { node { title } } } } } }
  frameworks(first: 50) { edges { node { name controls(first: 500) { edges { node { id sectionTitle } } } } } }
  internalControls(first: 500) { edges { node { id name
    controls(first: 100) { edges { node { sectionTitle } } } } } }
} } }`;

function compareSections(a, b) {
  const x = a.slice(2).split(".").map(Number);
  const y = b.slice(2).split(".").map(Number);
  for (let i = 0; i < Math.max(x.length, y.length); i++) {
    const d = (x[i] ?? -1) - (y[i] ?? -1);
    if (d !== 0) return d;
  }
  return 0;
}

// expandSections turns "J.1.1〜J.1.5、J.2.2" into the set of J numbers it
// covers, using the order of the framework's sections for ranges.
export function expandSections(spec, sections) {
  const ordered = [...sections].sort(compareSections);
  const result = new Set();
  for (const part of spec.split("、").map(p => p.trim())) {
    if (part.includes("〜")) {
      const [first, last] = part.split("〜");
      const from = ordered.indexOf(first);
      const to = ordered.indexOf(last);
      if (from < 0 || to < from) throw new Error(`unknown range: ${part}`);
      ordered.slice(from, to + 1).forEach(s => result.add(s));
    } else {
      if (!ordered.includes(part)) throw new Error(`unknown J number: ${part}`);
      result.add(part);
    }
  }
  return result;
}

export function documentType(number) {
  if (["01", "02", "20", "21"].includes(number)) return "POLICY";
  return number.startsWith("1") ? "PROCEDURE" : "TEMPLATE";
}

export function classification(number) {
  return ["01", "20", "21"].includes(number) ? "PUBLIC" : "INTERNAL";
}

// planDocument decides what to register and link for one finished document.
export function planDocument(file, text, controls, internalControls) {
  const number = file.slice(0, 2);
  const [first] = text.split("\n", 1);
  if (!first.startsWith("# ")) throw new Error(`${file}: the first line must be the title`);
  const row = text.match(J_ROW);
  if (!row) throw new Error(`${file}: no 対応する J 番号 row`);

  const sections = expandSections(row[1], Object.keys(controls));
  const internal = INTERNAL_CONTROL_OVERRIDES[Number(number)]
    ?? Object.entries(internalControls)
      .filter(([, ic]) => ic.sections.some(s => sections.has(s)))
      .map(([name]) => name)
      .sort();

  return {
    file,
    title: first.slice(2).trim(),
    documentType: documentType(number),
    classification: classification(number),
    sections: [...sections].sort(compareSections),
    internalControls: internal,
  };
}

function prbApi(prb, query, fields) {
  const args = ["api", query];
  for (const [key, value] of Object.entries(fields)) args.push("-f", `${key}=${value}`);
  return JSON.parse(execFileSync(prb, args, { encoding: "utf8", maxBuffer: 64 << 20 }));
}

function loadOrganization(prb, org) {
  const node = prbApi(prb, LIST_QUERY, { id: org }).data?.node;
  if (!node) throw new Error(`organization not found: ${org}`);

  const documents = Object.fromEntries(node.documents.edges
    .filter(e => e.node.versions.edges.length > 0)
    .map(e => [e.node.versions.edges[0].node.title, e.node.id]));

  const framework = node.frameworks.edges.find(e => e.node.name === PMARK_FRAMEWORK);
  if (!framework) throw new Error(`framework not found: ${PMARK_FRAMEWORK}`);
  const controls = Object.fromEntries(framework.node.controls.edges
    .map(e => [e.node.sectionTitle, e.node.id]));

  const internalControls = Object.fromEntries(node.internalControls.edges.map(e => [
    e.node.name,
    {
      id: e.node.id,
      sections: e.node.controls.edges.map(c => c.node.sectionTitle).filter(s => s in controls),
    },
  ]));

  return { documents, controls, internalControls };
}

function createDocument(prb, org, plan, content) {
  const result = prbApi(
    prb,
    "mutation($input: CreateDocumentInput!) { createDocument(input: $input) { documentEdge { node { id } } } }",
    {
      input: JSON.stringify({
        organizationId: org,
        title: plan.title,
        documentType: plan.documentType,
        classification: plan.classification,
        content,
      }),
    },
  );
  if (result.errors?.length) throw new Error(`${plan.file}: ${result.errors[0].message}`);
  return result.data.createDocument.documentEdge.node.id;
}

// link returns true when a link was created and false when it already existed.
function link(prb, mutation, inputType, input) {
  const result = prbApi(
    prb,
    `mutation($input: ${inputType}!) { ${mutation}(input: $input) { __typename } }`,
    { input: JSON.stringify(input) },
  );
  const error = result.errors?.[0];
  if (!error) return true;
  if (error.extensions?.code === "CONFLICT") return false;
  throw new Error(error.message);
}

function parseArguments(argv) {
  const options = { apply: false, prb: join(ROOT, "bin", "prb") };
  for (let i = 0; i < argv.length; i++) {
    switch (argv[i]) {
      case "--org": options.org = argv[++i]; break;
      case "--docs": options.docs = argv[++i]; break;
      case "--prb": options.prb = argv[++i]; break;
      case "--apply": options.apply = true; break;
      default: throw new Error(`unknown argument: ${argv[i]}`);
    }
  }
  if (!options.org || !options.docs) {
    throw new Error("usage: node scripts/ja-docs/sync.mjs --org <organization-id> --docs <dir> [--apply] [--prb path]");
  }
  return options;
}

function main(argv) {
  let options;
  try {
    options = parseArguments(argv);
  } catch (error) {
    console.error(error.message);
    return 2;
  }

  const files = readdirSync(options.docs).filter(f => DOCUMENT_FILE.test(f)).sort();
  if (files.length === 0) {
    console.error(`no documents in ${options.docs}`);
    return 2;
  }

  const work = mkdtempSync(join(tmpdir(), "ja-docs-"));
  try {
    const org = loadOrganization(options.prb, options.org);
    const docgen = join(work, "ja-docgen");
    if (options.apply) {
      execFileSync("go", ["build", "-o", docgen, "./cmd/ja-docgen"], { cwd: ROOT, stdio: "inherit" });
    }

    let created = 0;
    let linked = 0;
    let existing = 0;
    for (const file of files) {
      const path = join(options.docs, file);
      const plan = planDocument(file, readFileSync(path, "utf8"), org.controls, org.internalControls);
      let id = org.documents[plan.title];
      console.log(`${file}: ${plan.title} [${id ? "exists" : "create"}] ${plan.documentType}/${plan.classification}, `
        + `${plan.sections.length} controls, ${plan.internalControls.length} internal controls`);
      if (!options.apply) continue;

      if (!id) {
        const content = execFileSync(docgen, ["-prosemirror", path], { encoding: "utf8", maxBuffer: 64 << 20 });
        id = createDocument(options.prb, options.org, plan, content);
        created++;
      }
      const links = [
        ...plan.sections.map(s => ["createControlDocumentMapping", "CreateControlDocumentMappingInput",
          { controlId: org.controls[s], documentId: id }]),
        ...plan.internalControls.map(n => ["createInternalControlDocumentMapping",
          "CreateInternalControlDocumentMappingInput", { internalControlId: org.internalControls[n].id, documentId: id }]),
      ];
      for (const [mutation, inputType, input] of links) {
        if (link(options.prb, mutation, inputType, input)) linked++;
        else existing++;
      }
    }

    console.log(options.apply
      ? `created ${created} document(s), ${linked} new link(s), ${existing} link(s) already existed`
      : "dry run; add --apply to make the changes");
    return 0;
  } catch (error) {
    console.error(error.message);
    return 1;
  } finally {
    rmSync(work, { recursive: true, force: true });
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  process.exitCode = main(process.argv.slice(2));
}
