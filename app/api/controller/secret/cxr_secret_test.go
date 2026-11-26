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

package secret

import (
	"errors"
	"strings"
	"testing"

	"github.com/harness/gitness/app/store/database"

	"github.com/stretchr/testify/require"
)

var errCxrCrypt = errors.New("crypt failure")

// cxrEncrypter is a reversible fake encrypter.
type cxrEncrypter struct {
	failEncrypt bool
	failDecrypt bool
}

func (e cxrEncrypter) Encrypt(plaintext string) ([]byte, error) {
	if e.failEncrypt {
		return nil, errCxrCrypt
	}
	return []byte("enc:" + plaintext), nil
}

func (e cxrEncrypter) Decrypt(ciphertext []byte) (string, error) {
	if e.failDecrypt {
		return "", errCxrCrypt
	}
	return strings.TrimPrefix(string(ciphertext), "enc:"), nil
}

func cxrNewController(b *cxrBase, mode cxrAuthMode, enc cxrEncrypter) *Controller {
	return NewController(cxrAuthorizer{mode: mode}, enc, database.NewSecretStore(b.db), b.spaceFinder)
}

func TestCxrSecretLifecycle(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	b.createSpace(ctx, t)
	c := cxrNewController(b, cxrAllow, cxrEncrypter{})

	created, err := c.Create(ctx, b.session(), &CreateInput{
		SpaceRef: "root", UID: "db-pass", Description: "  my secret  ", Data: "hunter2",
	})
	require.NoError(t, err)
	require.Equal(t, "db-pass", created.Identifier)
	require.Equal(t, "enc:hunter2", created.Data)

	found, err := c.Find(ctx, b.session(), "root", "db-pass")
	require.NoError(t, err)
	require.Equal(t, "hunter2", found.Data)

	newData, newDesc, newID := "rotated", "rotated desc", "db-pass2"
	updated, err := c.Update(ctx, b.session(), "root", "db-pass", &UpdateInput{
		Identifier: &newID, Description: &newDesc, Data: &newData,
	})
	require.NoError(t, err)
	require.Equal(t, newID, updated.Identifier)
	require.Equal(t, newDesc, updated.Description)
	require.Equal(t, "enc:rotated", updated.Data)

	// update using the deprecated uid field and no data change.
	uid := "db-pass3"
	updated, err = c.Update(ctx, b.session(), "root", newID, &UpdateInput{UID: &uid})
	require.NoError(t, err)
	require.Equal(t, uid, updated.Identifier)
	require.Equal(t, "enc:rotated", updated.Data)

	require.NoError(t, c.Delete(ctx, b.session(), "root", uid))
	_, err = c.Find(ctx, b.session(), "root", uid)
	require.Error(t, err)
}

func TestCxrSecretCreateErrors(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	b.createSpace(ctx, t)

	tests := []struct {
		name string
		mode cxrAuthMode
		enc  cxrEncrypter
		in   *CreateInput
		want string
	}{
		{"empty space ref", cxrAllow, cxrEncrypter{}, &CreateInput{Identifier: "abc"}, "Parent space required"},
		{"negative space id", cxrAllow, cxrEncrypter{}, &CreateInput{SpaceRef: "-1", Identifier: "abc"},
			"Parent space required"},
		{"bad identifier", cxrAllow, cxrEncrypter{}, &CreateInput{SpaceRef: "root", Identifier: "!!"}, ""},
		{"bad description", cxrAllow, cxrEncrypter{},
			&CreateInput{SpaceRef: "root", Identifier: "abc", Description: strings.Repeat("x", 1100)}, ""},
		{"unknown space", cxrAllow, cxrEncrypter{}, &CreateInput{SpaceRef: "nope", Identifier: "abc"},
			"failed to find parent"},
		{"forbidden", cxrDeny, cxrEncrypter{}, &CreateInput{SpaceRef: "root", Identifier: "abc"}, ""},
		{"encrypt failure", cxrAllow, cxrEncrypter{failEncrypt: true},
			&CreateInput{SpaceRef: "root", Identifier: "abc"}, "could not encrypt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cxrNewController(b, tt.mode, tt.enc)
			got, err := c.Create(ctx, b.session(), tt.in)
			require.Error(t, err)
			require.Nil(t, got)
			require.Contains(t, err.Error(), tt.want)
		})
	}

	c := cxrNewController(b, cxrAllow, cxrEncrypter{})
	_, err := c.Create(ctx, b.session(), &CreateInput{SpaceRef: "root", Identifier: "dup", Data: "x"})
	require.NoError(t, err)
	_, err = c.Create(ctx, b.session(), &CreateInput{SpaceRef: "root", Identifier: "dup", Data: "x"})
	require.ErrorContains(t, err, "secret creation failed")
}

func TestCxrSecretFindUpdateDeleteErrors(t *testing.T) {
	ctx := t.Context()
	b := cxrNewBase(ctx, t)
	b.createSpace(ctx, t)
	allow := cxrNewController(b, cxrAllow, cxrEncrypter{})
	_, err := allow.Create(ctx, b.session(), &CreateInput{SpaceRef: "root", Identifier: "s1", Data: "x"})
	require.NoError(t, err)

	deny := cxrNewController(b, cxrDeny, cxrEncrypter{})
	badDec := cxrNewController(b, cxrAllow, cxrEncrypter{failDecrypt: true})
	badEnc := cxrNewController(b, cxrAllow, cxrEncrypter{failEncrypt: true})
	data := "new"
	badID := "!!"
	longDesc := strings.Repeat("x", 1100)

	_, err = allow.Find(ctx, b.session(), "nope", "s1")
	require.ErrorContains(t, err, "failed to find space")
	_, err = deny.Find(ctx, b.session(), "root", "s1")
	require.ErrorContains(t, err, "failed to authorize")
	_, err = allow.Find(ctx, b.session(), "root", "missing")
	require.ErrorContains(t, err, "failed to find secret")
	_, err = badDec.Find(ctx, b.session(), "root", "s1")
	require.ErrorContains(t, err, "could not decrypt")

	_, err = allow.Update(ctx, b.session(), "root", "s1", &UpdateInput{Identifier: &badID})
	require.Error(t, err)
	_, err = allow.Update(ctx, b.session(), "root", "s1", &UpdateInput{Description: &longDesc})
	require.Error(t, err)
	_, err = allow.Update(ctx, b.session(), "nope", "s1", &UpdateInput{})
	require.ErrorContains(t, err, "failed to find space")
	_, err = deny.Update(ctx, b.session(), "root", "s1", &UpdateInput{})
	require.ErrorContains(t, err, "failed to authorize")
	_, err = allow.Update(ctx, b.session(), "root", "missing", &UpdateInput{})
	require.ErrorContains(t, err, "failed to find secret")
	_, err = badEnc.Update(ctx, b.session(), "root", "s1", &UpdateInput{Data: &data})
	require.ErrorContains(t, err, "could not encrypt")

	require.ErrorContains(t, allow.Delete(ctx, b.session(), "nope", "s1"), "failed to find space")
	require.ErrorContains(t, deny.Delete(ctx, b.session(), "root", "s1"), "failed to authorize")
	// Deleting a missing secret is a no-op in the store.
	require.NoError(t, allow.Delete(ctx, b.session(), "root", "missing"))
}

func TestCxrSecretEncDecNil(t *testing.T) {
	_, err := enc(cxrEncrypter{}, nil)
	require.Error(t, err)
	_, err = Dec(cxrEncrypter{}, nil)
	require.Error(t, err)
	_, err = Dec(cxrEncrypter{failDecrypt: true}, nil)
	require.Error(t, err)
}
