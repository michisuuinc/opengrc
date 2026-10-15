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

package coredata

import (
	"context"
	"errors"
	"fmt"

	"go.gearno.de/kit/pg"
	"go.probo.inc/probo/pkg/gid"
	"go.probo.inc/probo/pkg/mail"
)

// MembershipProfileLanguagePreferences returns the languages known for a
// member, most specific first: the language the person saved
// (identities.locale), then the SCIM preferred language and locale, then
// the organization's default email language. Pass them to
// emails.ResolveLanguage. Unknown values are nil.
func MembershipProfileLanguagePreferences(
	ctx context.Context,
	conn pg.Querier,
	profile *MembershipProfile,
) ([]*string, error) {
	var identityLocale *string

	if profile.IdentityID != (gid.GID{}) {
		identity := &Identity{}
		err := identity.LoadByID(ctx, conn, profile.IdentityID)
		switch {
		case err == nil:
			identityLocale = identity.Locale
		case !errors.Is(err, ErrResourceNotFound):
			return nil, fmt.Errorf("cannot load identity: %w", err)
		}
	}

	organizationLanguage, err := LoadOrganizationDefaultLanguage(ctx, conn, profile.OrganizationID)
	if err != nil {
		return nil, err
	}

	return []*string{identityLocale, profile.PreferredLanguage, profile.Locale, organizationLanguage}, nil
}

// IdentityLanguagePreferences returns the languages known for a person in
// an organization: the language they saved, then the organization's
// default email language. identity may be nil when the person has no
// account.
func IdentityLanguagePreferences(
	ctx context.Context,
	conn pg.Querier,
	identity *Identity,
	organizationID gid.GID,
) ([]*string, error) {
	organizationLanguage, err := LoadOrganizationDefaultLanguage(ctx, conn, organizationID)
	if err != nil {
		return nil, err
	}

	var identityLocale *string
	if identity != nil {
		identityLocale = identity.Locale
	}

	return []*string{identityLocale, organizationLanguage}, nil
}

// EmailAddressLanguagePreferences returns the languages known for an email
// address in an organization: the language saved by the account with that
// address, if any, then the organization's default email language.
func EmailAddressLanguagePreferences(
	ctx context.Context,
	conn pg.Querier,
	address mail.Addr,
	organizationID gid.GID,
) ([]*string, error) {
	identity := &Identity{}
	err := identity.LoadByEmail(ctx, conn, address)
	switch {
	case errors.Is(err, ErrResourceNotFound):
		identity = nil
	case err != nil:
		return nil, fmt.Errorf("cannot load identity: %w", err)
	}

	return IdentityLanguagePreferences(ctx, conn, identity, organizationID)
}
