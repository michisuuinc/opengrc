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

import type { TrainingParticipantStatus } from "#/__generated__/core/TrainingParticipantsCardFragment.graphql";
import type {
  TrainingPlanKind,
  TrainingPlanMethod,
} from "#/__generated__/core/TrainingPlanDetailsPageFragment.graphql";

export const trainingPlanKinds: readonly TrainingPlanKind[] = [
  "PERIODIC",
  "ONBOARDING",
  "ROLE_BASED",
  "AD_HOC",
];

export const trainingPlanMethods: readonly TrainingPlanMethod[] = [
  "CLASSROOM",
  "ONLINE",
  "E_LEARNING",
  "MATERIALS",
];

export const trainingParticipantStatuses: readonly TrainingParticipantStatus[] = [
  "NOT_ATTENDED",
  "TEST_PENDING",
  "FAILED",
  "COMPLETED",
];

export type { TrainingParticipantStatus, TrainingPlanKind, TrainingPlanMethod };

export function statusBadgeVariant(status: TrainingParticipantStatus) {
  switch (status) {
    case "COMPLETED":
      return "success";
    case "FAILED":
      return "danger";
    case "TEST_PENDING":
      return "warning";
    default:
      return "neutral";
  }
}

// The time zone a new plan's period is read in: the browser's, so that a
// period entered as dates means those days where the training happens.
export function defaultTrainingTimeZone(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
}

// The Japanese fiscal year (April to March) that contains the given date.
export function currentFiscalYear(date = new Date()): number {
  return date.getMonth() < 3 ? date.getFullYear() - 1 : date.getFullYear();
}
