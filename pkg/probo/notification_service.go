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

package probo

import (
	"context"
	"fmt"
	"time"

	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/coredata"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/page"
)

func (s *Service) ListNotifications(
	ctx context.Context,
	scope coredata.Scoper,
	organizationID gid.GID,
	identityID gid.GID,
	cursor *page.Cursor[coredata.NotificationOrderField],
	types []coredata.NotificationType,
) (*page.Page[*coredata.Notification, coredata.NotificationOrderField], error) {
	var notifications coredata.Notifications

	err := s.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			profile, err := loadNotificationRecipient(ctx, conn, scope, organizationID, identityID)
			if err != nil {
				return fmt.Errorf("cannot load notification recipient: %w", err)
			}

			if err := notifications.LoadByRecipientProfileID(
				ctx,
				conn,
				scope,
				organizationID,
				profile.ID,
				cursor,
				coredata.NewNotificationFilter(types),
			); err != nil {
				return fmt.Errorf("cannot load notifications: %w", err)
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return page.NewPage(notifications, cursor), nil
}

func (s *Service) CountUnreadNotifications(
	ctx context.Context,
	scope coredata.Scoper,
	organizationID gid.GID,
	identityID gid.GID,
	types []coredata.NotificationType,
) (int, error) {
	var count int

	err := s.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			profile, err := loadNotificationRecipient(ctx, conn, scope, organizationID, identityID)
			if err != nil {
				return fmt.Errorf("cannot load notification recipient: %w", err)
			}

			count, err = (&coredata.Notifications{}).CountUnreadByRecipientProfileID(
				ctx,
				conn,
				scope,
				organizationID,
				profile.ID,
				coredata.NewNotificationFilter(types),
			)
			if err != nil {
				return fmt.Errorf("cannot count unread notifications: %w", err)
			}

			return nil
		},
	)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Service) GetNotification(
	ctx context.Context,
	scope coredata.Scoper,
	notificationID gid.GID,
	identityID gid.GID,
) (*coredata.Notification, error) {
	notification := &coredata.Notification{}

	err := s.pg.WithConn(
		ctx,
		func(ctx context.Context, conn pg.Querier) error {
			if err := notification.LoadByID(ctx, conn, scope, notificationID); err != nil {
				return fmt.Errorf("cannot load notification: %w", err)
			}

			profile, err := loadNotificationRecipient(
				ctx,
				conn,
				scope,
				notification.OrganizationID,
				identityID,
			)
			if err != nil {
				return fmt.Errorf("cannot load notification recipient: %w", err)
			}

			if notification.RecipientProfileID != profile.ID {
				return coredata.ErrResourceNotFound
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return notification, nil
}

func (s *Service) MarkNotificationRead(
	ctx context.Context,
	scope coredata.Scoper,
	notificationID gid.GID,
	identityID gid.GID,
) (*coredata.Notification, error) {
	notification := &coredata.Notification{}

	err := s.pg.WithTx(
		ctx,
		func(ctx context.Context, tx pg.Tx) error {
			if err := notification.LoadByID(ctx, tx, scope, notificationID); err != nil {
				return fmt.Errorf("cannot load notification: %w", err)
			}

			profile, err := loadNotificationRecipient(
				ctx,
				tx,
				scope,
				notification.OrganizationID,
				identityID,
			)
			if err != nil {
				return fmt.Errorf("cannot load notification recipient: %w", err)
			}

			if notification.RecipientProfileID != profile.ID {
				return coredata.ErrResourceNotFound
			}

			if notification.ReadAt != nil {
				return nil
			}

			if err := notification.MarkRead(ctx, tx, scope, profile.ID, time.Now()); err != nil {
				return fmt.Errorf("cannot mark notification read: %w", err)
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return notification, nil
}

func (s *Service) MarkNotificationsRead(
	ctx context.Context,
	scope coredata.Scoper,
	organizationID gid.GID,
	identityID gid.GID,
	types []coredata.NotificationType,
) error {
	return s.pg.WithTx(
		ctx,
		func(ctx context.Context, tx pg.Tx) error {
			profile, err := loadNotificationRecipient(ctx, tx, scope, organizationID, identityID)
			if err != nil {
				return fmt.Errorf("cannot load notification recipient: %w", err)
			}

			if err := (&coredata.Notifications{}).MarkReadByRecipientProfileID(
				ctx,
				tx,
				scope,
				organizationID,
				profile.ID,
				coredata.NewNotificationFilter(types),
				time.Now(),
			); err != nil {
				return fmt.Errorf("cannot mark notifications read: %w", err)
			}

			return nil
		},
	)
}

func loadNotificationRecipient(
	ctx context.Context,
	conn pg.Querier,
	scope coredata.Scoper,
	organizationID gid.GID,
	identityID gid.GID,
) (*coredata.MembershipProfile, error) {
	profile := &coredata.MembershipProfile{}
	if err := profile.LoadByIdentityIDAndOrganizationID(
		ctx,
		conn,
		scope,
		identityID,
		organizationID,
	); err != nil {
		return nil, fmt.Errorf("cannot load membership profile: %w", err)
	}

	return profile, nil
}
