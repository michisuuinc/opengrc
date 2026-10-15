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

package console_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.probo.inc/probo/e2e/internal/factory"
	"go.probo.inc/probo/e2e/internal/testutil"
)

const (
	organizationEmailLanguageQuery = `
		query($id: ID!) {
			node(id: $id) {
				... on Organization { emailLanguage }
			}
		}
	`

	updateOrganizationEmailLanguageMutation = `
		mutation($input: UpdateOrganizationEmailLanguageInput!) {
			updateOrganizationEmailLanguage(input: $input) {
				organization { emailLanguage }
			}
		}
	`

	updateViewerLocaleMutation = `
		mutation($input: UpdateViewerLocaleInput!) {
			updateViewerLocale(input: $input) {
				identity { locale }
			}
		}
	`
)

func setOrganizationEmailLanguage(t *testing.T, c *testutil.Client, language string) {
	t.Helper()

	c.MustExecute(updateOrganizationEmailLanguageMutation, map[string]any{
		"input": map[string]any{"organizationId": c.GetOrganizationID().String(), "language": language},
	}, nil)
}

func inviteProfile(t *testing.T, c *testutil.Client, profileID string) {
	t.Helper()

	err := c.ExecuteConnect(`
		mutation($input: InviteUserInput!) {
			inviteUser(input: $input) { invitationEdge { node { id } } }
		}
	`, map[string]any{
		"input": map[string]any{"organizationId": c.GetOrganizationID().String(), "profileId": profileID},
	}, nil)
	require.NoError(t, err)
}

// waitForMail returns the first email to the address whose subject passes
// accept.
func waitForMail(t *testing.T, c *testutil.Client, address string, accept func(subject string) bool) *testutil.MailpitMessageDetail {
	t.Helper()

	var detail *testutil.MailpitMessageDetail

	ok := testutil.Poll(t, 60*time.Second, 500*time.Millisecond, func() bool {
		found, err := c.FindMailpitMessage(fmt.Sprintf("to:%s", address), func(m *testutil.MailpitMessageDetail) bool {
			return accept(m.Subject)
		})
		detail = found
		return err == nil && found != nil
	})
	require.True(t, ok, "no matching email to %s", address)

	return detail
}

func TestEmailLanguage_OrganizationSetting(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)

	var before struct {
		Node struct {
			EmailLanguage *string `json:"emailLanguage"`
		} `json:"node"`
	}
	owner.MustExecute(organizationEmailLanguageQuery, map[string]any{"id": owner.GetOrganizationID().String()}, &before)
	assert.Nil(t, before.Node.EmailLanguage)

	setOrganizationEmailLanguage(t, owner, "ja")

	var after struct {
		Node struct {
			EmailLanguage *string `json:"emailLanguage"`
		} `json:"node"`
	}
	owner.MustExecute(organizationEmailLanguageQuery, map[string]any{"id": owner.GetOrganizationID().String()}, &after)
	require.NotNil(t, after.Node.EmailLanguage)
	assert.Equal(t, "ja", *after.Node.EmailLanguage)

	_, err := owner.Do(updateOrganizationEmailLanguageMutation, map[string]any{
		"input": map[string]any{"organizationId": owner.GetOrganizationID().String(), "language": "fr"},
	})
	require.Error(t, err, "only English and Japanese are supported")
}

func TestEmailLanguage_RBAC(t *testing.T) {
	t.Parallel()

	org := testutil.NewOrganizationRoles(t)

	for _, tt := range []struct {
		role      testutil.TestRole
		forbidden bool
	}{
		{testutil.RoleOwner, false},
		{testutil.RoleAdmin, false},
		{testutil.RoleViewer, true},
	} {
		t.Run(string(tt.role), func(t *testing.T) {
			t.Parallel()

			client := org.Client(t, tt.role)
			_, err := client.Do(updateOrganizationEmailLanguageMutation, map[string]any{
				"input": map[string]any{"organizationId": client.GetOrganizationID().String(), "language": "ja"},
			})
			if tt.forbidden {
				testutil.RequireForbiddenError(t, err)
			} else {
				require.NoError(t, err)
			}

			_, err = client.Do(organizationEmailLanguageQuery, map[string]any{"id": client.GetOrganizationID().String()})
			require.NoError(t, err)
		})
	}
}

func TestEmailLanguage_Invitation(t *testing.T) {
	t.Parallel()

	t.Run(
		"the organization default makes the invitation Japanese",
		func(t *testing.T) {
			t.Parallel()

			owner := testutil.NewClient(t, testutil.RoleOwner)
			setOrganizationEmailLanguage(t, owner, "ja")

			address := factory.SafeEmail()
			profileID := factory.CreateUser(owner, factory.Attrs{"emailAddress": address, "fullName": "招待 太郎"})
			inviteProfile(t, owner, profileID)

			mail := waitForMail(t, owner, address, func(s string) bool { return strings.Contains(s, "Probo への招待") })
			assert.Contains(t, mail.Text, "招待 太郎 様")
			assert.Contains(t, mail.Text, "アカウントを有効にしてください")
			assert.Contains(t, mail.HTML, `lang="ja"`)
		},
	)

	t.Run(
		"without a default the invitation stays English",
		func(t *testing.T) {
			t.Parallel()

			owner := testutil.NewClient(t, testutil.RoleOwner)

			address := factory.SafeEmail()
			profileID := factory.CreateUser(owner, factory.Attrs{"emailAddress": address})
			inviteProfile(t, owner, profileID)

			mail := waitForMail(t, owner, address, func(s string) bool { return strings.HasPrefix(s, "Invitation to join") })
			assert.Contains(t, mail.Text, "Click the link below to activate your account")
		},
	)
}

func TestEmailLanguage_ViewerLocale(t *testing.T) {
	t.Parallel()

	owner := testutil.NewClient(t, testutil.RoleOwner)

	var result struct {
		UpdateViewerLocale struct {
			Identity struct {
				Locale *string `json:"locale"`
			} `json:"identity"`
		} `json:"updateViewerLocale"`
	}
	require.NoError(t, owner.ExecuteConnect(updateViewerLocaleMutation, map[string]any{"input": map[string]any{"locale": "ja"}}, &result))
	require.NotNil(t, result.UpdateViewerLocale.Identity.Locale)
	assert.Equal(t, "ja", *result.UpdateViewerLocale.Identity.Locale)

	_, err := owner.DoConnect(updateViewerLocaleMutation, map[string]any{"input": map[string]any{"locale": "xx"}})
	require.Error(t, err, "unsupported locales are rejected")

	// The saved language is used for emails to the person, here the
	// password reset.
	_, err = testutil.NewUnauthenticatedClient(t).DoConnect(`
		mutation($input: ForgotPasswordInput!) {
			forgotPassword(input: $input) { success }
		}
	`, map[string]any{"input": map[string]any{"email": owner.GetEmail()}})
	require.NoError(t, err)

	mail := waitForMail(t, owner, owner.GetEmail(), func(s string) bool { return s == "パスワードの再設定" })
	assert.Contains(t, mail.Text, "パスワードを再設定してください")
}
