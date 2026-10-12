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

import { BellIcon } from "@phosphor-icons/react";
import { Button } from "@probo/ui/src/v2/Button/Button";
import { iconButton } from "@probo/ui/src/v2/IconButton/variants";
import { Popover } from "@probo/ui/src/v2/Popover/Popover";
import { PopoverPopup } from "@probo/ui/src/v2/Popover/PopoverPopup";
import { PopoverTrigger } from "@probo/ui/src/v2/Popover/PopoverTrigger";
import { Text } from "@probo/ui/src/v2/typography/Text";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchQuery, graphql, useLazyLoadQuery, useRelayEnvironment } from "react-relay";
import { Link } from "react-router";

import type { NotificationsBellMarkAllReadMutation } from "#/__generated__/core/NotificationsBellMarkAllReadMutation.graphql";
import type { NotificationsBellMarkReadMutation } from "#/__generated__/core/NotificationsBellMarkReadMutation.graphql";
import type {
  NotificationsBellQuery$data,
  NotificationsBellQuery,
} from "#/__generated__/core/NotificationsBellQuery.graphql";
import { useMutation } from "#/lib/relay/useMutation";

// Lives outside pages/iam: that tree compiles against the connect schema, and
// the inbox is on the console API. The top bar wraps this in the core Relay
// environment.
const notificationTypes = [
  "DOCUMENT_SIGNATURE_REQUESTED",
  "DOCUMENT_APPROVAL_REQUESTED",
] as const;

const notificationsQuery = graphql`
  query NotificationsBellQuery(
    $organizationId: ID!
    $types: [NotificationType!]
  ) {
    viewer {
      id
      notificationUnreadCount(organizationId: $organizationId, types: $types)
      notifications(
        organizationId: $organizationId
        types: $types
        first: 20
      ) {
        edges {
          node {
            id
            type
            subjectId
            title
            readAt
            createdAt
          }
        }
      }
    }
  }
`;

const markReadMutation = graphql`
  mutation NotificationsBellMarkReadMutation(
    $input: MarkNotificationReadInput!
    $organizationId: ID!
    $types: [NotificationType!]
  ) {
    markNotificationRead(input: $input) {
      notification {
        id
        readAt
      }
      viewer {
        id
        notificationUnreadCount(organizationId: $organizationId, types: $types)
      }
    }
  }
`;

const markAllReadMutation = graphql`
  mutation NotificationsBellMarkAllReadMutation(
    $input: MarkNotificationsReadInput!
    $organizationId: ID!
    $types: [NotificationType!]
  ) {
    markNotificationsRead(input: $input) {
      viewer {
        id
        notificationUnreadCount(organizationId: $organizationId, types: $types)
      }
    }
  }
`;

type NotificationNode = NotificationsBellQuery$data["viewer"]["notifications"]["edges"][number]["node"];

interface NotificationsBellProps {
  organizationId: string;
  employeePortalId: string;
}

export function NotificationsBell({
  organizationId,
  employeePortalId,
}: NotificationsBellProps) {
  const variables = useMemo(
    () => ({ organizationId, types: notificationTypes }),
    [organizationId],
  );
  const environment = useRelayEnvironment();
  const data = useLazyLoadQuery<NotificationsBellQuery>(
    notificationsQuery,
    variables,
    { fetchPolicy: "store-and-network" },
  );
  const [open, setOpen] = useState(false);
  const { t, i18n } = useTranslation();
  const [markRead] = useMutation<NotificationsBellMarkReadMutation>(markReadMutation);
  const [markAllRead, markingAll] = useMutation<NotificationsBellMarkAllReadMutation>(markAllReadMutation);

  useEffect(() => {
    let subscription: { unsubscribe: () => void } | null = null;

    function refresh() {
      subscription?.unsubscribe();
      subscription = fetchQuery(environment, notificationsQuery, variables).subscribe({});
    }

    window.addEventListener("focus", refresh);
    return () => {
      window.removeEventListener("focus", refresh);
      subscription?.unsubscribe();
    };
  }, [environment, variables]);

  const label = t("notifications.label");
  const unreadCount = data.viewer.notificationUnreadCount;
  const when = new Intl.DateTimeFormat(i18n.language, {
    dateStyle: "medium",
    timeStyle: "short",
  });

  function refreshList() {
    fetchQuery(environment, notificationsQuery, variables).subscribe({});
  }

  function handleOpenChange(next: boolean) {
    setOpen(next);
    if (next) {
      refreshList();
    }
  }

  function handleOpen(notification: NotificationNode) {
    setOpen(false);
    if (notification.readAt != null) {
      return;
    }

    void markRead({
      variables: {
        input: { notificationId: notification.id },
        organizationId,
        types: notificationTypes,
      },
    }).catch(() => undefined);
  }

  function handleMarkAllRead() {
    void markAllRead({
      variables: {
        input: { organizationId, types: notificationTypes },
        organizationId,
        types: notificationTypes,
      },
    }).then(() => {
      refreshList();
    }).catch(() => undefined);
  }

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <span className="relative inline-flex">
        <PopoverTrigger
          className={iconButton({
            variant: "ghost",
            color: "neutral",
            className: "data-popup-open:bg-sand-3",
          })}
          aria-label={unreadCount > 0
            ? `${label}, ${t("notifications.unread", { count: unreadCount })}`
            : label}
        >
          <BellIcon />
        </PopoverTrigger>
        {unreadCount > 0 && (
          <span className="pointer-events-none absolute top-1.5 right-1.5 size-1.5 rounded-full bg-gold-9" />
        )}
      </span>
      <PopoverPopup className="w-80 p-0" align="end" sideOffset={8}>
        <div className="flex items-center justify-between gap-2 px-3 pt-3 pb-2">
          <Text size={2} weight="medium" className="min-w-0 truncate">
            {label}
          </Text>
          {unreadCount > 0 && (
            <Button
              variant="ghost"
              color="neutral"
              size={1}
              loading={markingAll}
              onClick={handleMarkAllRead}
            >
              {t("notifications.markAllRead")}
            </Button>
          )}
        </div>
        {data.viewer.notifications.edges.length === 0
          ? (
              <Text size={2} color="faint" className="block px-3 pt-2 pb-4">
                {t("notifications.empty")}
              </Text>
            )
          : (
              <ul className="flex max-h-80 flex-col overflow-y-auto px-1 pb-1">
                {data.viewer.notifications.edges.map(({ node }) => (
                  <li key={node.id}>
                    <Link
                      to={notificationPath(employeePortalId, node)}
                      onClick={() => handleOpen(node)}
                      className="flex flex-col gap-0.5 rounded-2 px-2 py-2 hover:bg-sand-3"
                    >
                      <Text
                        size={2}
                        weight={node.readAt == null ? "medium" : "regular"}
                        highContrast={node.readAt == null}
                      >
                        {notificationMessage(node, t)}
                      </Text>
                      <Text size={1} color="faint">
                        {when.format(new Date(node.createdAt))}
                      </Text>
                    </Link>
                  </li>
                ))}
              </ul>
            )}
      </PopoverPopup>
    </Popover>
  );
}

function notificationPath(employeePortalId: string, notification: NotificationNode): string {
  if (notification.type === "DOCUMENT_SIGNATURE_REQUESTED") {
    return `/${employeePortalId}/signatures/${notification.subjectId}`;
  }
  if (notification.type === "DOCUMENT_APPROVAL_REQUESTED") {
    return `/${employeePortalId}/approvals/${notification.subjectId}`;
  }

  return `/${employeePortalId}`;
}

function notificationMessage(
  notification: NotificationNode,
  t: (key: string, options?: { title: string }) => string,
): string {
  switch (notification.type) {
    case "DOCUMENT_SIGNATURE_REQUESTED":
      return t("notifications.documentSignatureRequested", { title: notification.title });
    case "DOCUMENT_APPROVAL_REQUESTED":
      return t("notifications.documentApprovalRequested", { title: notification.title });
    case "TASK_ASSIGNED":
      return t("notifications.taskAssigned", { title: notification.title });
  }
}
