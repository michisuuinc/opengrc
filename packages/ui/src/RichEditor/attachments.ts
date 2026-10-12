// Copyright (c) 2026 Probo Inc <hello@probo.com>.
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
  acceptData,
  acceptDocument,
  acceptImage,
  acceptPresentation,
  acceptSpreadsheet,
  acceptText,
} from "@probo/helpers";
import { useEffect } from "react";

export const maxAttachmentSize = 50 * 1024 * 1024;

type AcceptMap = Record<string, readonly string[]>;

const attachmentAccept: AcceptMap = {
  ...acceptDocument,
  ...acceptSpreadsheet,
  ...acceptPresentation,
  ...acceptText,
  ...acceptImage,
  ...acceptData,
};

const imageExtensions = extensionsOf(acceptImage);

export const imageAccept = acceptAttribute(acceptImage);

const fileAcceptMap = Object.fromEntries(
  Object.entries(attachmentAccept).filter(
    ([mimeType]) => !(mimeType in acceptImage),
  ),
);

export const fileAccept = acceptAttribute(fileAcceptMap);

export type AttachmentKind = "image" | "file";

type AttachmentPreview = {
  url: string;
  users: number;
};

const attachmentPreviews = new Map<string, AttachmentPreview>();

export function rememberAttachmentPreview(fileId: string, file: Blob) {
  const current = attachmentPreviews.get(fileId);
  if (current) {
    current.users += 1;

    return current.url;
  }

  const url = URL.createObjectURL(file);
  attachmentPreviews.set(fileId, { url, users: 1 });

  return url;
}

export function releaseAttachmentPreview(fileId: string) {
  const current = attachmentPreviews.get(fileId);
  if (!current) {
    return;
  }

  current.users -= 1;
  if (current.users > 0) {
    return;
  }

  URL.revokeObjectURL(current.url);
  attachmentPreviews.delete(fileId);
}

export function releaseAbsentAttachmentPreviews(
  ownedIds: readonly string[],
  presentIds: ReadonlySet<string>,
) {
  const retained: string[] = [];

  for (const fileId of ownedIds) {
    if (presentIds.has(fileId)) {
      retained.push(fileId);
      continue;
    }

    releaseAttachmentPreview(fileId);
  }

  return retained;
}

export function attachmentPreviewURL(fileId: string) {
  return attachmentPreviews.get(fileId)?.url;
}

export function useAttachmentPreview(fileId: string) {
  const url = attachmentPreviewURL(fileId);

  useEffect(() => {
    if (!fileId || !url) {
      return;
    }

    const current = attachmentPreviews.get(fileId);
    if (!current) {
      return;
    }

    current.users += 1;

    return () => {
      releaseAttachmentPreview(fileId);
    };
  }, [fileId, url]);

  return url;
}

const attachmentPathPrefix = "/api/files/v1/attachments/";

export function attachmentPath(fileId: string) {
  return `${attachmentPathPrefix}${encodeURIComponent(fileId)}`;
}

export function fileIDFromAttachmentURL(src: string | null | undefined) {
  const raw = src?.trim() ?? "";
  if (raw === "") {
    return null;
  }

  let pathname = raw;
  if (raw.includes("://") || raw.startsWith("//")) {
    try {
      pathname = new URL(raw).pathname;
    } catch {
      return null;
    }
  } else {
    const query = pathname.indexOf("?");
    const hash = pathname.indexOf("#");
    let end = pathname.length;
    if (query >= 0) {
      end = query;
    }
    if (hash >= 0 && hash < end) {
      end = hash;
    }
    pathname = pathname.slice(0, end);
  }

  if (!pathname.startsWith(attachmentPathPrefix)) {
    return null;
  }

  let id = pathname.slice(attachmentPathPrefix.length);
  try {
    id = decodeURIComponent(id);
  } catch {
    return null;
  }

  id = id.trim();
  if (id === "" || id.includes("/")) {
    return null;
  }

  return id;
}

export function prepareAttachment(file: File) {
  const extension = fileExtension(file.name);
  const mimeType = file.type.toLowerCase();
  const nextExtension
    = extension || extensionForMime(attachmentAccept, mimeType);
  const allowed = mimesForExtension(attachmentAccept, nextExtension);
  const nextMimeType = allowed.includes(mimeType) ? mimeType : allowed[0] ?? "";
  if (
    !nextExtension
    || !nextMimeType
    || (extension === nextExtension && mimeType === nextMimeType)
  ) {
    return file;
  }

  const name = extension ? file.name : `pasted${nextExtension}`;

  return new File([file], name, { type: nextMimeType });
}

export function attachmentKind(file: File): AttachmentKind | null {
  if (file.size <= 0 || file.size > maxAttachmentSize) {
    return null;
  }

  const extension = fileExtension(file.name);
  const allowed = mimesForExtension(attachmentAccept, extension);
  if (allowed.length === 0) {
    return null;
  }

  const mimeType = file.type.toLowerCase();
  if (mimeType !== "" && !allowed.includes(mimeType)) {
    return null;
  }

  return imageExtensions.has(extension) ? "image" : "file";
}

function extensionsOf(accept: AcceptMap) {
  return new Set(Object.values(accept).flat());
}

function acceptAttribute(accept: AcceptMap) {
  return [...extensionsOf(accept), ...Object.keys(accept)].join(",");
}

function mimesForExtension(accept: AcceptMap, extension: string) {
  return Object.entries(accept)
    .filter(([, extensions]) => extensions.includes(extension))
    .map(([mimeType]) => mimeType);
}

function extensionForMime(accept: AcceptMap, mimeType: string) {
  for (const [mime, extensions] of Object.entries(accept)) {
    if (mime === mimeType && extensions[0]) {
      return extensions[0];
    }
  }

  return "";
}

function fileExtension(name: string) {
  const dot = name.lastIndexOf(".");
  if (dot < 0) {
    return "";
  }

  return name.slice(dot).toLowerCase();
}
