// Copyright 2026 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package serviceaccount

import (
	"testing"
	"time"

	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func cxrNewController(b *cxrBase, mode cxrAuthMode) *Controller {
	return NewController(
		check.PrincipalUIDDefault, cxrAuthorizer{mode: mode}, b.principals, b.spaces, b.repos,
		database.NewTokenStore(b.db))
}

func cxrCreateInput(parentID int64) *CreateInput {
	return &CreateInput{
		Email: " sa@example.com ", DisplayName: " Bot ", ParentType: enum.ParentResourceTypeSpace, ParentID: parentID,
	}
}

func TestCxrServiceAccountLifecycle(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	space := b.createSpace(ctx, t)
	c := cxrNewController(b, cxrAllow)

	created, err := c.Create(ctx, b.session(), cxrCreateInput(space.ID))
	require.NoError(t, err)
	require.Contains(t, created.UID, "sa-space-")
	require.Equal(t, "sa@example.com", created.Email)
	require.Equal(t, "Bot", created.DisplayName)

	found, err := c.Find(ctx, b.session(), created.UID)
	require.NoError(t, err)
	require.Equal(t, created.UID, found.UID)
	noAuth, err := cxrNewController(b, cxrDeny).FindNoAuth(ctx, created.UID)
	require.NoError(t, err)
	require.Equal(t, created.UID, noAuth.UID)

	require.NoError(t, c.Delete(ctx, b.session(), created.UID))
	_, err = c.Find(ctx, b.session(), created.UID)
	require.Error(t, err)
}

func TestCxrServiceAccountCreateNoAuth(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	space := b.createSpace(ctx, t)
	c := cxrNewController(b, cxrDeny)

	created, err := c.CreateNoAuth(ctx, cxrCreateInput(space.ID), "fixed-sa")
	require.NoError(t, err)
	require.Equal(t, "fixed-sa", created.UID)

	tests := []struct {
		name string
		uid  string
		in   *CreateInput
	}{
		{"bad uid", "!!", cxrCreateInput(space.ID)},
		{"bad email", "sa1", &CreateInput{ParentType: enum.ParentResourceTypeSpace, ParentID: space.ID, DisplayName: "x"}},
		{"bad display name", "sa2", &CreateInput{
			Email: "a@example.com", ParentType: enum.ParentResourceTypeSpace, ParentID: space.ID,
		}},
		{"bad parent type", "sa3", &CreateInput{Email: "a@example.com", DisplayName: "x", ParentID: space.ID}},
		{"bad parent id", "sa4", &CreateInput{
			Email: "a@example.com", DisplayName: "x", ParentType: enum.ParentResourceTypeSpace,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.CreateNoAuth(ctx, tt.in, tt.uid)
			require.ErrorContains(t, err, "invalid input")
		})
	}

	_, err = c.CreateNoAuth(ctx, cxrCreateInput(space.ID), "fixed-sa")
	require.Error(t, err, "duplicate uid")
}

func TestCxrServiceAccountTokens(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	space := b.createSpace(ctx, t)
	c := cxrNewController(b, cxrAllow)
	sa, err := c.Create(ctx, b.session(), cxrCreateInput(space.ID))
	require.NoError(t, err)

	lifetime := 24 * time.Hour
	resp, err := c.CreateToken(ctx, b.session(), sa.UID, &CreateTokenInput{UID: "tok", Lifetime: &lifetime})
	require.NoError(t, err)
	require.Equal(t, "tok", resp.Token.Identifier)
	require.NotEmpty(t, resp.AccessToken)
	require.Equal(t, enum.TokenTypeSAT, resp.Token.Type)

	tokens, err := c.ListTokens(ctx, b.session(), sa.UID)
	require.NoError(t, err)
	require.Len(t, tokens, 1)

	require.NoError(t, c.DeleteToken(ctx, b.session(), sa.UID, "tok"))
	tokens, err = c.ListTokens(ctx, b.session(), sa.UID)
	require.NoError(t, err)
	require.Empty(t, tokens)

	// A token of a different principal must not be deletable through the service account.
	other := b.createUser(ctx, t, "alice", false)
	tokenStore := database.NewTokenStore(b.db)
	require.NoError(t, tokenStore.Create(ctx, &types.Token{
		Type: enum.TokenTypePAT, Identifier: "pat", PrincipalID: other.ID, CreatedBy: other.ID,
		IssuedAt: time.Now().UnixMilli(),
	}))
	require.Error(t, c.DeleteToken(ctx, b.session(), sa.UID, "pat"))
	require.Error(t, c.DeleteToken(ctx, b.session(), sa.UID, "missing"))
}

func TestCxrServiceAccountTokenErrors(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	space := b.createSpace(ctx, t)
	allow, deny := cxrNewController(b, cxrAllow), cxrNewController(b, cxrDeny)
	sa, err := allow.Create(ctx, b.session(), cxrCreateInput(space.ID))
	require.NoError(t, err)

	tooShort, tooLong := time.Second, 100*365*24*time.Hour
	_, err = allow.CreateToken(ctx, b.session(), sa.UID, &CreateTokenInput{Identifier: "!!"})
	require.ErrorContains(t, err, "failed to sanitize input")
	_, err = allow.CreateToken(ctx, b.session(), sa.UID, &CreateTokenInput{Identifier: "tok", Lifetime: &tooShort})
	require.ErrorContains(t, err, "failed to sanitize input")
	_, err = allow.CreateToken(ctx, b.session(), sa.UID, &CreateTokenInput{Identifier: "tok", Lifetime: &tooLong})
	require.ErrorContains(t, err, "failed to sanitize input")
	_, err = allow.CreateToken(ctx, b.session(), "missing", &CreateTokenInput{Identifier: "tok"})
	require.Error(t, err)
	_, err = deny.CreateToken(ctx, b.session(), sa.UID, &CreateTokenInput{Identifier: "tok"})
	require.Error(t, err)

	// A token without lifetime is allowed, a duplicate identifier is not.
	_, err = allow.CreateToken(ctx, b.session(), sa.UID, &CreateTokenInput{Identifier: "tok"})
	require.NoError(t, err)
	_, err = allow.CreateToken(ctx, b.session(), sa.UID, &CreateTokenInput{Identifier: "tok"})
	require.Error(t, err)

	_, err = deny.ListTokens(ctx, b.session(), sa.UID)
	require.Error(t, err)
	_, err = allow.ListTokens(ctx, b.session(), "missing")
	require.Error(t, err)
	require.Error(t, deny.DeleteToken(ctx, b.session(), sa.UID, "tok"))
	require.Error(t, allow.DeleteToken(ctx, b.session(), "missing", "tok"))

	_, err = deny.Create(ctx, b.session(), cxrCreateInput(space.ID))
	require.Error(t, err)
	_, err = deny.Find(ctx, b.session(), sa.UID)
	require.Error(t, err)
	require.Error(t, deny.Delete(ctx, b.session(), sa.UID))
	require.Error(t, allow.Delete(ctx, b.session(), "missing"))
}
