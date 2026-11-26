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

package principal

import (
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func cxrUserFilter() *types.PrincipalFilter {
	return &types.PrincipalFilter{Types: []enum.PrincipalType{enum.PrincipalTypeUser}, Size: 10, Page: 1}
}

func TestCxrPrincipalList(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	b.createUser(ctx, t, "alice", false)
	allow := newController(b.principals, cxrAuthorizer{mode: cxrAllow})

	got, err := allow.List(ctx, b.session(), cxrUserFilter())
	require.NoError(t, err)
	require.Len(t, got, 2) // admin and alice
	require.Equal(t, enum.PrincipalTypeUser, got[0].Type)

	filter := cxrUserFilter()
	filter.Query = "alice"
	got, err = allow.List(ctx, b.session(), filter)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "alice", got[0].UID)

	for _, typs := range [][]enum.PrincipalType{
		nil,
		{enum.PrincipalTypeService},
		{enum.PrincipalTypeUser, enum.PrincipalTypeService},
	} {
		_, err = allow.List(ctx, b.session(), &types.PrincipalFilter{Types: typs})
		require.ErrorContains(t, err, "Only listing of users is supported")
	}

	_, err = newController(b.principals, cxrAuthorizer{mode: cxrDeny}).List(ctx, b.session(), cxrUserFilter())
	require.Error(t, err)
}

func TestCxrPrincipalFind(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	alice := b.createUser(ctx, t, "alice", false)
	svc := &types.Service{UID: "svc", Email: "svc@example.com", DisplayName: "svc"}
	require.NoError(t, b.principals.CreateService(ctx, svc))
	svcPrincipal, err := b.principals.FindByUID(ctx, "svc")
	require.NoError(t, err)

	allow := newController(b.principals, cxrAuthorizer{mode: cxrAllow})
	got, err := allow.Find(ctx, b.session(), alice.ID)
	require.NoError(t, err)
	require.Equal(t, "alice", got.UID)

	_, err = allow.Find(ctx, b.session(), svcPrincipal.ID)
	require.ErrorContains(t, err, "only user principals are supported")
	_, err = allow.Find(ctx, b.session(), 99999)
	require.Error(t, err)
	_, err = newController(b.principals, cxrAuthorizer{mode: cxrDeny}).Find(ctx, b.session(), alice.ID)
	require.Error(t, err)
}

func TestCxrPrincipalCheckExistenceByEmails(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	b.createUser(ctx, t, "alice", false)
	allow := newController(b.principals, cxrAuthorizer{mode: cxrAllow})

	out, err := allow.CheckExistenceByEmails(ctx, b.session(), &CheckUsersInput{
		Emails: []string{"alice@example.com", "ghost@example.com", "nobody@example.com"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"ghost@example.com", "nobody@example.com"}, out.UnknownEmails)

	out, err = allow.CheckExistenceByEmails(ctx, b.session(), &CheckUsersInput{})
	require.NoError(t, err)
	require.Empty(t, out.UnknownEmails)

	_, err = newController(b.principals, cxrAuthorizer{mode: cxrDeny}).CheckExistenceByEmails(
		ctx, b.session(), &CheckUsersInput{Emails: []string{"alice@example.com"}})
	require.Error(t, err)
}
