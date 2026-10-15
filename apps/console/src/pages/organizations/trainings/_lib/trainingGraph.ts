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

import { useTranslation } from "react-i18next";
import { graphql } from "relay-runtime";

import type { trainingGraphAddParticipantsMutation } from "#/__generated__/core/trainingGraphAddParticipantsMutation.graphql";
import type { trainingGraphCreateMutation } from "#/__generated__/core/trainingGraphCreateMutation.graphql";
import type { trainingGraphDeleteMutation } from "#/__generated__/core/trainingGraphDeleteMutation.graphql";
import type { trainingGraphRemoveParticipantMutation } from "#/__generated__/core/trainingGraphRemoveParticipantMutation.graphql";
import type { trainingGraphRequestSignaturesMutation } from "#/__generated__/core/trainingGraphRequestSignaturesMutation.graphql";
import type { trainingGraphSetMaterialsMutation } from "#/__generated__/core/trainingGraphSetMaterialsMutation.graphql";
import type { trainingGraphUpdateMutation } from "#/__generated__/core/trainingGraphUpdateMutation.graphql";
import type { trainingGraphUpdateParticipantMutation } from "#/__generated__/core/trainingGraphUpdateParticipantMutation.graphql";
import { useMutation } from "#/lib/relay/useMutation";

/* eslint-disable relay/unused-fields, relay/must-colocate-fragment-spreads */

export const trainingPlansQuery = graphql`
  query trainingGraphListQuery($organizationId: ID!) {
    node(id: $organizationId) @required(action: THROW) {
      ... on Organization {
        id
        canCreateTrainingPlan: permission(action: "core:training-plan:create")
        trainingPlans(
          first: 500
          orderBy: { field: STARTS_ON, direction: DESC }
        ) @connection(key: "TrainingPlansPage_trainingPlans") {
          __id
          edges {
            node {
              id
              ...TrainingPlansPageRowFragment
            }
          }
        }
      }
    }
  }
`;

export const trainingPlanNodeQuery = graphql`
  query trainingGraphNodeQuery($trainingPlanId: ID!) {
    node(id: $trainingPlanId) @required(action: THROW) {
      ... on TrainingPlan {
        id
        ...TrainingPlanDetailsPageFragment
      }
    }
  }
`;

const createTrainingPlanMutation = graphql`
  mutation trainingGraphCreateMutation(
    $input: CreateTrainingPlanInput!
    $connections: [ID!]!
  ) {
    createTrainingPlan(input: $input) {
      trainingPlanEdge @prependEdge(connections: $connections) {
        node {
          id
          ...TrainingPlansPageRowFragment
        }
      }
    }
  }
`;

const updateTrainingPlanMutation = graphql`
  mutation trainingGraphUpdateMutation($input: UpdateTrainingPlanInput!) {
    updateTrainingPlan(input: $input) {
      trainingPlan {
        ...TrainingPlanDetailsPageFragment
      }
    }
  }
`;

const deleteTrainingPlanMutation = graphql`
  mutation trainingGraphDeleteMutation(
    $input: DeleteTrainingPlanInput!
    $connections: [ID!]!
  ) {
    deleteTrainingPlan(input: $input) {
      deletedTrainingPlanId @deleteEdge(connections: $connections)
    }
  }
`;

const setTrainingPlanMaterialsMutation = graphql`
  mutation trainingGraphSetMaterialsMutation(
    $input: SetTrainingPlanMaterialsInput!
  ) {
    setTrainingPlanMaterials(input: $input) {
      trainingPlan {
        ...TrainingPlanDetailsPageFragment
      }
    }
  }
`;

const addTrainingParticipantsMutation = graphql`
  mutation trainingGraphAddParticipantsMutation(
    $input: AddTrainingParticipantsInput!
  ) {
    addTrainingParticipants(input: $input) {
      addedCount
      trainingPlan {
        ...TrainingPlanDetailsPageFragment
      }
    }
  }
`;

const removeTrainingParticipantMutation = graphql`
  mutation trainingGraphRemoveParticipantMutation(
    $input: RemoveTrainingParticipantInput!
  ) {
    removeTrainingParticipant(input: $input) {
      trainingPlan {
        ...TrainingPlanDetailsPageFragment
      }
    }
  }
`;

const updateTrainingParticipantMutation = graphql`
  mutation trainingGraphUpdateParticipantMutation(
    $input: UpdateTrainingParticipantInput!
  ) {
    updateTrainingParticipant(input: $input) {
      trainingPlan {
        ...TrainingPlanDetailsPageFragment
      }
    }
  }
`;

const requestTrainingSignaturesMutation = graphql`
  mutation trainingGraphRequestSignaturesMutation(
    $input: RequestTrainingSignaturesInput!
  ) {
    requestTrainingSignatures(input: $input) {
      requestedCount
      trainingPlan {
        ...TrainingPlanDetailsPageFragment
      }
    }
  }
`;

export function useCreateTrainingPlan() {
  const { t } = useTranslation();

  return useMutation<trainingGraphCreateMutation>(createTrainingPlanMutation, {
    successMessage: t("training.messages.created"),
    errorToast: t("training.errors.save"),
  });
}

export function useUpdateTrainingPlan() {
  const { t } = useTranslation();

  return useMutation<trainingGraphUpdateMutation>(updateTrainingPlanMutation, {
    successMessage: t("training.messages.saved"),
    errorToast: t("training.errors.save"),
  });
}

export function useDeleteTrainingPlan() {
  const { t } = useTranslation();

  return useMutation<trainingGraphDeleteMutation>(deleteTrainingPlanMutation, {
    successMessage: t("training.messages.deleted"),
    errorToast: t("training.errors.delete"),
  });
}

export function useSetTrainingPlanMaterials() {
  const { t } = useTranslation();

  return useMutation<trainingGraphSetMaterialsMutation>(
    setTrainingPlanMaterialsMutation,
    {
      successMessage: t("training.messages.saved"),
      errorToast: t("training.errors.save"),
    },
  );
}

export function useAddTrainingParticipants() {
  const { t } = useTranslation();

  return useMutation<trainingGraphAddParticipantsMutation>(
    addTrainingParticipantsMutation,
    { errorToast: t("training.errors.save") },
  );
}

export function useRemoveTrainingParticipant() {
  const { t } = useTranslation();

  return useMutation<trainingGraphRemoveParticipantMutation>(
    removeTrainingParticipantMutation,
    {
      successMessage: t("training.messages.participantRemoved"),
      errorToast: t("training.errors.save"),
    },
  );
}

export function useUpdateTrainingParticipant() {
  const { t } = useTranslation();

  return useMutation<trainingGraphUpdateParticipantMutation>(
    updateTrainingParticipantMutation,
    {
      successMessage: t("training.messages.saved"),
      errorToast: t("training.errors.save"),
    },
  );
}

export function useRequestTrainingSignatures() {
  const { t } = useTranslation();

  return useMutation<trainingGraphRequestSignaturesMutation>(
    requestTrainingSignaturesMutation,
    {
      successMessage: t("training.messages.signaturesRequested"),
      errorToast: t("training.errors.requestSignatures"),
    },
  );
}
