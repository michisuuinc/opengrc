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

import { lazy } from "@probo/react-lazy";
import {
  type AppRoute,
  loaderFromQueryLoader,
  withQueryRef,
} from "@probo/routes";
import { loadQuery } from "react-relay";

import type { trainingGraphListQuery } from "#/__generated__/core/trainingGraphListQuery.graphql";
import type { trainingGraphNodeQuery } from "#/__generated__/core/trainingGraphNodeQuery.graphql";
import { PageSkeleton } from "#/components/skeletons/PageSkeleton";
import { coreEnvironment } from "#/environments";
import {
  trainingPlanNodeQuery,
  trainingPlansQuery,
} from "#/pages/organizations/trainings/_lib/trainingGraph";

// Training (教育) for the Japanese PrivacyMark (fork: michisuuinc/opengrc).
export const trainingRoutes = [
  {
    path: "trainings",
    Fallback: PageSkeleton,
    loader: loaderFromQueryLoader(({ organizationId }) =>
      loadQuery<trainingGraphListQuery>(
        coreEnvironment,
        trainingPlansQuery,
        { organizationId },
        { fetchPolicy: "network-only" },
      ),
    ),
    Component: withQueryRef(
      lazy(() => import("#/pages/organizations/trainings/TrainingPlansPage")),
    ),
  },
  {
    path: "trainings/:trainingPlanId",
    Fallback: PageSkeleton,
    loader: loaderFromQueryLoader(({ trainingPlanId }) =>
      loadQuery<trainingGraphNodeQuery>(
        coreEnvironment,
        trainingPlanNodeQuery,
        { trainingPlanId },
        { fetchPolicy: "network-only" },
      ),
    ),
    Component: withQueryRef(
      lazy(() => import("#/pages/organizations/trainings/TrainingPlanDetailsPage")),
    ),
  },
] satisfies AppRoute[];
