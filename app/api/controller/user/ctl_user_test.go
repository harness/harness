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

package user

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/system"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestCtlUserController_CreateAndFind(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)

	first, err := e.ctrl.Create(ctx, &auth.Session{}, &CreateInput{
		UID: "alice", Email: " alice@example.com ", DisplayName: " Alice ", Password: "password123",
	})
	require.NoError(t, err)
	require.True(t, first.Admin, "first user must become admin")
	require.Equal(t, "alice@example.com", first.Email)
	require.Equal(t, "Alice", first.DisplayName)

	second, err := e.ctrl.Create(ctx, ctlSession(first), &CreateInput{
		UID: "bob", Email: "bob@example.com", DisplayName: "Bob", Password: "password123",
	})
	require.NoError(t, err)
	require.False(t, second.Admin)

	found, err := e.ctrl.Find(ctx, ctlSession(first), "bob")
	require.NoError(t, err)
	require.Equal(t, second.ID, found.ID)

	foundByEmail, err := e.ctrl.FindEmail(ctx, ctlSession(first), "bob@example.com")
	require.NoError(t, err)
	require.Equal(t, second.ID, foundByEmail.ID)

	noAuth, err := e.ctrl.FindNoAuth(ctx, "alice")
	require.NoError(t, err)
	require.Equal(t, first.ID, noAuth.ID)

	_, err = e.ctrl.Find(ctx, ctlSession(first), "nobody")
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	_, err = e.ctrl.FindEmail(ctx, ctlSession(first), "nobody@example.com")
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
}

func TestCtlUserController_CreateInvalidInput(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)

	valid := func() *CreateInput {
		return &CreateInput{UID: "carol", Email: "carol@example.com", DisplayName: "Carol", Password: "password123"}
	}

	tests := []struct {
		name   string
		mutate func(in *CreateInput)
	}{
		{"bad uid", func(in *CreateInput) { in.UID = "" }},
		{"anonymous uid", func(in *CreateInput) { in.UID = types.AnonymousPrincipalUID }},
		{"bad email", func(in *CreateInput) { in.Email = "" }},
		{"bad display name", func(in *CreateInput) { in.DisplayName = "" }},
		{"bad password", func(in *CreateInput) { in.Password = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid()
			tt.mutate(in)
			_, err := e.ctrl.CreateNoAuth(ctx, in, false)
			require.Error(t, err)
		})
	}
}

func TestCtlUserController_CreateDuplicate(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	u := ctlCreateUser(t, e, "dave")

	_, err := e.ctrl.Create(ctx, ctlSession(u), &CreateInput{
		UID: "dave", Email: "other@example.com", DisplayName: "Dave", Password: "password123",
	})
	require.Error(t, err)
}

func TestCtlUserController_HashError(t *testing.T) {
	e := ctlNewEnv(t)
	hashErr := errors.New("hash failure")
	hashPassword = func([]byte, int) ([]byte, error) { return nil, hashErr }

	_, err := e.ctrl.CreateNoAuth(context.Background(), &CreateInput{
		UID: "erin", Email: "erin@example.com", DisplayName: "Erin", Password: "password123",
	}, false)
	require.ErrorIs(t, err, hashErr)

	u := ctlCreateUserDirect(t, e, "frank")
	pw := "newpassword123"
	_, err = e.ctrl.Update(context.Background(), ctlSession(u), "frank", &UpdateInput{Password: &pw})
	require.ErrorIs(t, err, hashErr)
}

func ctlCreateUserDirect(t *testing.T, e *ctlEnv, uid string) *types.User {
	t.Helper()
	u := &types.User{UID: uid, Email: uid + "@example.com", DisplayName: uid, Salt: "salt", Admin: false}
	require.NoError(t, e.principal.CreateUser(context.Background(), u))
	return u
}

func TestCtlUserController_AuthFailures(t *testing.T) {
	ctx := context.Background()
	allow := ctlNewEnv(t)
	admin := ctlCreateUser(t, allow, "admin")
	target := ctlCreateUser(t, allow, "target")

	for _, mode := range []ctlAuthMode{ctlDeny, ctlError} {
		// Reuse the stores of the allow environment but with a different authorizer.
		c := *allow.ctrl
		c.authorizer = ctlAuthorizer{mode: mode}
		session := ctlSession(admin)
		anon := &auth.Session{Principal: types.Principal{UID: types.AnonymousPrincipalUID}}

		check := func(err error) {
			t.Helper()
			require.Error(t, err)
			if mode == ctlError {
				require.ErrorIs(t, err, errCtlAuthz)
			} else {
				require.ErrorIs(t, err, apiauth.ErrForbidden)
			}
		}

		_, err := c.Create(ctx, session, &CreateInput{})
		check(err)
		_, _, err = c.List(ctx, session, &types.UserFilter{})
		check(err)
		_, err = c.Find(ctx, session, target.UID)
		check(err)
		_, err = c.FindEmail(ctx, session, target.Email)
		check(err)
		_, err = c.Update(ctx, session, target.UID, &UpdateInput{})
		check(err)
		_, err = c.UpdateAdmin(ctx, session, target.UID, &UpdateAdminInput{Admin: true})
		check(err)
		check(c.Delete(ctx, session, target.UID))
		check(c.DeleteToken(ctx, session, target.UID, enum.TokenTypePAT, "tok"))
		_, err = c.ListTokens(ctx, session, target.UID, enum.TokenTypePAT)
		check(err)
		_, err = c.CreateAccessToken(ctx, session, target.UID, &CreateTokenInput{Identifier: "tok"})
		check(err)
		_, _, err = c.MembershipSpaces(ctx, session, target.UID, types.MembershipSpaceFilter{})
		check(err)
		_, err = c.CreatePublicKey(ctx, session, target.UID, &CreatePublicKeyInput{})
		check(err)
		_, _, err = c.ListPublicKeys(ctx, session, target.UID, &types.PublicKeyFilter{})
		check(err)
		check(c.DeletePublicKey(ctx, session, target.UID, "k"))
		_, err = c.UpdatePublicKey(ctx, session, target.UID, "k", &UpdatePublicKeyInput{})
		check(err)

		if mode == ctlDeny {
			_, err = c.Find(ctx, anon, target.UID)
			require.ErrorIs(t, err, apiauth.ErrUnauthorized)
		}
	}
}

func TestCtlUserController_NotFound(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	admin := ctlCreateUser(t, e, "admin")
	s := ctlSession(admin)

	_, err := e.ctrl.Update(ctx, s, "ghost", &UpdateInput{})
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	_, err = e.ctrl.UpdateAdmin(ctx, s, "ghost", &UpdateAdminInput{})
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	require.ErrorIs(t, e.ctrl.Delete(ctx, s, "ghost"), gitnessstore.ErrResourceNotFound)
	require.ErrorIs(t, e.ctrl.DeleteToken(ctx, s, "ghost", enum.TokenTypePAT, "t"), gitnessstore.ErrResourceNotFound)
	_, err = e.ctrl.ListTokens(ctx, s, "ghost", enum.TokenTypePAT)
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	_, err = e.ctrl.CreateAccessToken(ctx, s, "ghost", &CreateTokenInput{Identifier: "tok"})
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	_, _, err = e.ctrl.MembershipSpaces(ctx, s, "ghost", types.MembershipSpaceFilter{})
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	_, err = e.ctrl.CreatePublicKey(ctx, s, "ghost", &CreatePublicKeyInput{})
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	_, _, err = e.ctrl.ListPublicKeys(ctx, s, "ghost", &types.PublicKeyFilter{})
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	require.ErrorIs(t, e.ctrl.DeletePublicKey(ctx, s, "ghost", "k"), gitnessstore.ErrResourceNotFound)
	_, err = e.ctrl.UpdatePublicKey(ctx, s, "ghost", "k", &UpdatePublicKeyInput{})
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
}

func TestCtlUserController_ListAndUpdate(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	admin := ctlCreateUser(t, e, "admin")
	user := ctlCreateUser(t, e, "user1")
	s := ctlSession(admin)

	users, count, err := e.ctrl.List(ctx, s, &types.UserFilter{Page: 1, Size: 10})
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.Len(t, users, 2)

	email, name, pw := " new@example.com ", " New Name ", "anotherpassword"
	updated, err := e.ctrl.Update(ctx, s, user.UID, &UpdateInput{Email: &email, DisplayName: &name, Password: &pw})
	require.NoError(t, err)
	require.Equal(t, "new@example.com", updated.Email)
	require.Equal(t, "New Name", updated.DisplayName)
	require.NotEqual(t, user.Password, updated.Password)

	badEmail, badName, badPw := "", "", ""
	for _, in := range []*UpdateInput{{Email: &badEmail}, {DisplayName: &badName}, {Password: &badPw}} {
		_, err = e.ctrl.Update(ctx, s, user.UID, in)
		require.Error(t, err)
	}
}

func TestCtlUserController_UpdateAdminAndDelete(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	admin := ctlCreateUser(t, e, "admin")
	user := ctlCreateUser(t, e, "user1")
	s := ctlSession(admin)

	// The only admin can neither be demoted nor deleted.
	_, err := e.ctrl.UpdateAdmin(ctx, s, admin.UID, &UpdateAdminInput{Admin: false})
	var userErr *usererror.Error
	require.ErrorAs(t, err, &userErr)
	require.Equal(t, 400, userErr.Status)
	err = e.ctrl.Delete(ctx, s, admin.UID)
	require.ErrorAs(t, err, &userErr)
	require.Equal(t, 400, userErr.Status)

	promoted, err := e.ctrl.UpdateAdmin(ctx, s, user.UID, &UpdateAdminInput{Admin: true})
	require.NoError(t, err)
	require.True(t, promoted.Admin)

	// Now that there are two admins the first one may be demoted again.
	demoted, err := e.ctrl.UpdateAdmin(ctx, s, admin.UID, &UpdateAdminInput{Admin: false})
	require.NoError(t, err)
	require.False(t, demoted.Admin)

	// Deleting an admin is allowed while another admin exists... but not once it is the last one.
	require.NoError(t, e.ctrl.Delete(ctx, s, admin.UID))
	require.ErrorAs(t, e.ctrl.Delete(ctx, s, user.UID), &userErr)

	_, err = e.ctrl.FindNoAuth(ctx, admin.UID)
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
}

func TestCtlUserController_LoginLogoutRegister(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	user := ctlCreateUser(t, e, "grace")

	// Login by UID and by email.
	resp, err := e.ctrl.Login(ctx, &LoginInput{LoginIdentifier: "grace", Password: "password123"})
	require.NoError(t, err)
	require.NotEmpty(t, resp.AccessToken)
	require.Equal(t, enum.TokenTypeSession, resp.Token.Type)

	resp2, err := e.ctrl.Login(ctx, &LoginInput{LoginIdentifier: user.Email, Password: "password123"})
	require.NoError(t, err)
	require.NotEqual(t, resp.Token.ID, resp2.Token.ID)

	// Failures never leak whether the user exists.
	_, err = e.ctrl.Login(ctx, &LoginInput{LoginIdentifier: "grace", Password: "wrongpassword"})
	require.ErrorIs(t, err, usererror.ErrNotFound)
	_, err = e.ctrl.Login(ctx, &LoginInput{LoginIdentifier: "ghost", Password: "password123"})
	require.ErrorIs(t, err, usererror.ErrNotFound)

	// Logout.
	require.ErrorIs(t, e.ctrl.Logout(ctx, nil), usererror.ErrUnauthorized)
	require.Error(t, e.ctrl.Logout(ctx, &auth.Session{}))
	pat := &auth.Session{Metadata: &auth.TokenMetadata{TokenType: enum.TokenTypePAT, TokenID: 1}}
	require.Error(t, e.ctrl.Logout(ctx, pat))

	sess := &auth.Session{
		Principal: *user.ToPrincipal(),
		Metadata:  &auth.TokenMetadata{TokenType: enum.TokenTypeSession, TokenID: resp.Token.ID},
	}
	require.NoError(t, e.ctrl.Logout(ctx, sess))
	tokens, err := e.ctrl.ListTokens(ctx, ctlSession(user), user.UID, enum.TokenTypeSession)
	require.NoError(t, err)
	require.Len(t, tokens, 1)

	// Register: the first user is always allowed, afterwards it depends on the config.
	cfg := &types.Config{UserSignupEnabled: false}
	sysCtrl := system.NewController(e.principal, cfg)
	_, err = e.ctrl.Register(ctx, sysCtrl, &RegisterInput{
		UID: "heidi", Email: "heidi@example.com", DisplayName: "Heidi", Password: "password123",
	})
	var userErr *usererror.Error
	require.ErrorAs(t, err, &userErr)
	require.Equal(t, 403, userErr.Status)

	cfg.UserSignupEnabled = true
	reg, err := e.ctrl.Register(ctx, sysCtrl, &RegisterInput{
		UID: "heidi", Email: "heidi@example.com", DisplayName: "Heidi", Password: "password123",
	})
	require.NoError(t, err)
	require.NotEmpty(t, reg.AccessToken)

	_, err = e.ctrl.Register(ctx, sysCtrl, &RegisterInput{UID: "", Email: "x@example.com", DisplayName: "X"})
	require.Error(t, err)
}

func TestCtlUserController_Tokens(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	user := ctlCreateUser(t, e, "ivan")
	s := ctlSession(user)

	lifetime := 24 * time.Hour
	resp, err := e.ctrl.CreateAccessToken(ctx, s, user.UID, &CreateTokenInput{Identifier: "my-token", Lifetime: &lifetime})
	require.NoError(t, err)
	require.NotEmpty(t, resp.AccessToken)

	// The deprecated UID field is used if the identifier is missing.
	_, err = e.ctrl.CreateAccessToken(ctx, s, user.UID, &CreateTokenInput{UID: "legacy-token"})
	require.NoError(t, err)

	tooLong := 100 * 365 * 24 * time.Hour
	_, err = e.ctrl.CreateAccessToken(ctx, s, user.UID, &CreateTokenInput{Identifier: "long", Lifetime: &tooLong})
	require.Error(t, err)
	_, err = e.ctrl.CreateAccessToken(ctx, s, user.UID, &CreateTokenInput{Identifier: ""})
	require.Error(t, err)

	tokens, err := e.ctrl.ListTokens(ctx, s, user.UID, enum.TokenTypePAT)
	require.NoError(t, err)
	require.Len(t, tokens, 2)

	_, err = e.ctrl.ListTokens(ctx, s, user.UID, enum.TokenTypeSAT)
	require.ErrorIs(t, err, usererror.ErrBadRequest)

	// Token type mismatch is reported as not found.
	require.ErrorIs(t, e.ctrl.DeleteToken(ctx, s, user.UID, enum.TokenTypeSession, "my-token"), usererror.ErrNotFound)
	err = e.ctrl.DeleteToken(ctx, s, user.UID, enum.TokenTypePAT, "unknown")
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	require.NoError(t, e.ctrl.DeleteToken(ctx, s, user.UID, enum.TokenTypePAT, "my-token"))

	tokens, err = e.ctrl.ListTokens(ctx, s, user.UID, enum.TokenTypePAT)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
}

func ctlSSHKey(t *testing.T) string {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sshPub, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)
	return string(ssh.MarshalAuthorizedKey(sshPub))
}

func TestCtlUserController_PublicKeys(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	user := ctlCreateUser(t, e, "judy")
	s := ctlSession(user)

	// Invalid inputs.
	for name, in := range map[string]*CreatePublicKeyInput{
		"bad identifier": {Identifier: "", Content: "x"},
		"bad usage":      {Identifier: "k", Usage: "nope", Content: "x"},
		"bad scheme":     {Identifier: "k", Scheme: "nope", Content: "x"},
		"no content":     {Identifier: "k", Content: "  "},
		"garbage":        {Identifier: "k", Content: "not a key"},
		"scheme mismatch": {
			Identifier: "k", Scheme: enum.PublicKeySchemePGP, Content: ctlSSHKey(t),
		},
	} {
		_, err := e.ctrl.CreatePublicKey(ctx, s, user.UID, in)
		require.Error(t, err, name)
	}

	content := ctlSSHKey(t)
	key, err := e.ctrl.CreatePublicKey(ctx, s, user.UID, &CreatePublicKeyInput{Identifier: "laptop", Content: content})
	require.NoError(t, err)
	require.Equal(t, enum.PublicKeySchemeSSH, key.Scheme)
	require.Equal(t, enum.PublicKeyUsageAuth, key.Usage)

	// Same key cannot be registered twice.
	_, err = e.ctrl.CreatePublicKey(ctx, s, user.UID, &CreatePublicKeyInput{Identifier: "dup", Content: content})
	require.Error(t, err)

	list, count, err := e.ctrl.ListPublicKeys(ctx, s, user.UID, &types.PublicKeyFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Len(t, list, 1)

	// Count query path when the first page is full.
	list, count, err = e.ctrl.ListPublicKeys(ctx, s, user.UID, &types.PublicKeyFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 1}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Len(t, list, 1)

	// Update validity.
	from, to := int64(1000), int64(2000)
	upd, err := e.ctrl.UpdatePublicKey(ctx, s, user.UID, "laptop", &UpdatePublicKeyInput{ValidFrom: &from, ValidTo: &to})
	require.NoError(t, err)
	require.Equal(t, from, *upd.ValidFrom)
	require.Equal(t, to, *upd.ValidTo)

	// No change returns the key as is.
	_, err = e.ctrl.UpdatePublicKey(ctx, s, user.UID, "laptop", &UpdatePublicKeyInput{})
	require.NoError(t, err)

	// Zero clears the timestamps.
	zero := int64(0)
	upd, err = e.ctrl.UpdatePublicKey(ctx, s, user.UID, "laptop", &UpdatePublicKeyInput{ValidFrom: &zero, ValidTo: &zero})
	require.NoError(t, err)
	require.Nil(t, upd.ValidFrom)
	require.Nil(t, upd.ValidTo)

	// Invalid update inputs.
	reason := enum.RevocationReasonRetired
	badReason := enum.RevocationReason("nope")
	later, earlier := int64(3000), int64(1000)
	for _, in := range []*UpdatePublicKeyInput{
		{RevocationReason: &badReason},
		{RevocationReason: &reason, ValidFrom: &from},
		{ValidFrom: &later, ValidTo: &earlier},
	} {
		_, err = e.ctrl.UpdatePublicKey(ctx, s, user.UID, "laptop", in)
		require.Error(t, err)
	}
	_, err = e.ctrl.UpdatePublicKey(ctx, s, user.UID, "missing", &UpdatePublicKeyInput{})
	require.Error(t, err)

	// Revoke (retired), then validity changes are rejected.
	upd, err = e.ctrl.UpdatePublicKey(ctx, s, user.UID, "laptop", &UpdatePublicKeyInput{RevocationReason: &reason})
	require.NoError(t, err)
	require.Equal(t, reason, *upd.RevocationReason)
	_, err = e.ctrl.UpdatePublicKey(ctx, s, user.UID, "laptop", &UpdatePublicKeyInput{ValidFrom: &from})
	require.Error(t, err)

	// Escalate to compromised; afterwards the reason is frozen.
	compromised := enum.RevocationReasonCompromised
	upd, err = e.ctrl.UpdatePublicKey(ctx, s, user.UID, "laptop", &UpdatePublicKeyInput{RevocationReason: &compromised})
	require.NoError(t, err)
	require.Equal(t, compromised, *upd.RevocationReason)
	_, err = e.ctrl.UpdatePublicKey(ctx, s, user.UID, "laptop", &UpdatePublicKeyInput{RevocationReason: &reason})
	require.Error(t, err)

	require.NoError(t, e.ctrl.DeletePublicKey(ctx, s, user.UID, "laptop"))
	require.Error(t, e.ctrl.DeletePublicKey(ctx, s, user.UID, "laptop"))
}

func TestCtlUserController_Favorites(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	user := ctlCreateUser(t, e, "kate")
	s := ctlSession(user)
	repo := ctlCreateRepo(t, e, user, "repo1")

	fav := &types.FavoriteResource{ID: repo.ID, Type: enum.ResourceTypeRepo}
	out, err := e.ctrl.CreateFavorite(ctx, s, fav)
	require.NoError(t, err)
	require.Equal(t, repo.ID, out.ID)

	// Creating twice must fail on the unique constraint.
	_, err = e.ctrl.CreateFavorite(ctx, s, &types.FavoriteResource{ID: repo.ID, Type: enum.ResourceTypeRepo})
	require.Error(t, err)

	_, err = e.ctrl.CreateFavorite(ctx, s, &types.FavoriteResource{ID: repo.ID, Type: enum.ResourceTypeSpace})
	require.Error(t, err)
	_, err = e.ctrl.CreateFavorite(ctx, s, &types.FavoriteResource{ID: 9999, Type: enum.ResourceTypeRepo})
	require.Error(t, err)

	require.NoError(t, e.ctrl.DeleteFavorite(ctx, s, &types.FavoriteResource{ID: repo.ID, Type: enum.ResourceTypeRepo}))
	require.Error(t, e.ctrl.DeleteFavorite(ctx, s, &types.FavoriteResource{ID: repo.ID, Type: enum.ResourceTypeSpace}))
	require.Error(t, e.ctrl.DeleteFavorite(ctx, s, &types.FavoriteResource{ID: 9999, Type: enum.ResourceTypeRepo}))

	// Denied access to the repo.
	c := *e.ctrl
	c.authorizer = ctlAuthorizer{mode: ctlDeny}
	_, err = c.CreateFavorite(ctx, s, &types.FavoriteResource{ID: repo.ID, Type: enum.ResourceTypeRepo})
	require.ErrorIs(t, err, apiauth.ErrForbidden)
	err = c.DeleteFavorite(ctx, s, &types.FavoriteResource{ID: repo.ID, Type: enum.ResourceTypeRepo})
	require.ErrorIs(t, err, apiauth.ErrForbidden)
}

func TestCtlUserController_MembershipSpaces(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	user := ctlCreateUser(t, e, "leo")
	s := ctlSession(user)

	spaces, count, err := e.ctrl.MembershipSpaces(ctx, s, user.UID, types.MembershipSpaceFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
	})
	require.NoError(t, err)
	require.Empty(t, spaces)
	require.EqualValues(t, 0, count)

	// A filter with a page > 1 forces the count query.
	_, count, err = e.ctrl.MembershipSpaces(ctx, s, user.UID, types.MembershipSpaceFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 10}},
	})
	require.NoError(t, err)
	require.EqualValues(t, 0, count)
}
