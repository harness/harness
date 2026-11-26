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

package service

import (
	"testing"

	"github.com/harness/gitness/types/check"

	"github.com/stretchr/testify/require"
)

func cxrNewController(b *cxrBase, mode cxrAuthMode) *Controller {
	return NewController(check.PrincipalUIDDefault, cxrAuthorizer{mode: mode}, b.principals)
}

func TestCxrServiceLifecycle(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	c := cxrNewController(b, cxrAllow)

	created, err := c.Create(ctx, b.session(), &CreateInput{
		UID: "my-svc", Email: " svc@example.com ", DisplayName: " My Service ",
	})
	require.NoError(t, err)
	require.Equal(t, "svc@example.com", created.Email)
	require.Equal(t, "My Service", created.DisplayName)
	require.False(t, created.Admin)
	require.NotEmpty(t, created.Salt)

	found, err := c.Find(ctx, b.session(), "my-svc")
	require.NoError(t, err)
	require.Equal(t, "My Service", found.DisplayName)
	noAuth, err := cxrNewController(b, cxrDeny).FindNoAuth(ctx, "my-svc")
	require.NoError(t, err)
	require.Equal(t, found.UID, noAuth.UID)

	count, list, err := c.List(ctx, b.session())
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, list, 1)

	email, name := "other@example.com", " Renamed "
	updated, err := c.Update(ctx, b.session(), "my-svc", &UpdateInput{Email: &email, DisplayName: &name})
	require.NoError(t, err)
	require.Equal(t, "Renamed", updated.DisplayName)

	updated, err = c.UpdateAdmin(ctx, b.session(), "my-svc", true)
	require.NoError(t, err)
	require.True(t, updated.Admin)

	require.NoError(t, c.Delete(ctx, b.session(), "my-svc"))
	_, err = c.Find(ctx, b.session(), "my-svc")
	require.Error(t, err)
}

func TestCxrServiceCreateNoAuth(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	c := cxrNewController(b, cxrDeny)

	created, err := c.CreateNoAuth(ctx, &CreateInput{UID: "admin-svc", Email: "a@example.com", DisplayName: "A"}, true)
	require.NoError(t, err)
	require.True(t, created.Admin)

	_, err = c.CreateNoAuth(ctx, &CreateInput{UID: "admin-svc", Email: "a@example.com", DisplayName: "A"}, true)
	require.Error(t, err, "duplicate uid")
}

func TestCxrServiceInputValidation(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	c := cxrNewController(b, cxrAllow)

	tests := []struct {
		name string
		in   *CreateInput
	}{
		{"bad uid", &CreateInput{UID: "!!", Email: "a@example.com", DisplayName: "A"}},
		{"bad email", &CreateInput{UID: "svc", Email: "", DisplayName: "A"}},
		{"empty display name", &CreateInput{UID: "svc", Email: "a@example.com", DisplayName: " "}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Create(ctx, b.session(), tt.in)
			require.ErrorContains(t, err, "invalid input")
		})
	}

	_, err := c.Create(ctx, b.session(), &CreateInput{UID: "svc", Email: "a@example.com", DisplayName: "A"})
	require.NoError(t, err)
	badEmail, empty := "", " "
	_, err = c.Update(ctx, b.session(), "svc", &UpdateInput{Email: &badEmail})
	require.ErrorContains(t, err, "invalid input")
	_, err = c.Update(ctx, b.session(), "svc", &UpdateInput{DisplayName: &empty})
	require.ErrorContains(t, err, "invalid input")
}

func TestCxrServiceAuthAndLookupErrors(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	allow, deny := cxrNewController(b, cxrAllow), cxrNewController(b, cxrDeny)
	_, err := allow.Create(ctx, b.session(), &CreateInput{UID: "svc", Email: "a@example.com", DisplayName: "A"})
	require.NoError(t, err)

	_, err = deny.Create(ctx, b.session(), &CreateInput{UID: "svc2", Email: "b@example.com", DisplayName: "B"})
	require.Error(t, err)
	_, err = deny.Find(ctx, b.session(), "svc")
	require.Error(t, err)
	_, _, err = deny.List(ctx, b.session())
	require.Error(t, err)
	_, err = deny.Update(ctx, b.session(), "svc", &UpdateInput{})
	require.Error(t, err)
	_, err = deny.UpdateAdmin(ctx, b.session(), "svc", true)
	require.Error(t, err)
	require.Error(t, deny.Delete(ctx, b.session(), "svc"))

	_, err = allow.Find(ctx, b.session(), "missing")
	require.Error(t, err)
	_, err = allow.Update(ctx, b.session(), "missing", &UpdateInput{})
	require.Error(t, err)
	_, err = allow.UpdateAdmin(ctx, b.session(), "missing", true)
	require.Error(t, err)
	require.Error(t, allow.Delete(ctx, b.session(), "missing"))
}
