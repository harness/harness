// Copyright 2023 Harness, Inc.
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
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	appstore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"golang.org/x/crypto/bcrypt"
)

// stubPrincipalStore embeds the interface so only the two lookups the login
// flow uses need to be implemented; any other call would panic (and none are
// expected on these paths).
type stubPrincipalStore struct {
	appstore.PrincipalStore
	findByUID   func(ctx context.Context, uid string) (*types.User, error)
	findByEmail func(ctx context.Context, email string) (*types.User, error)
}

func (s *stubPrincipalStore) FindUserByUID(ctx context.Context, uid string) (*types.User, error) {
	return s.findByUID(ctx, uid)
}

func (s *stubPrincipalStore) FindUserByEmail(ctx context.Context, email string) (*types.User, error) {
	return s.findByEmail(ctx, email)
}

// recordComparisons swaps comparePassword for a spy that records the hash passed
// to each call while still performing the real comparison, and restores it on
// cleanup.
func recordComparisons(t *testing.T) *[][]byte {
	t.Helper()
	var hashes [][]byte
	original := comparePassword
	comparePassword = func(hashedPassword, password []byte) error {
		hashes = append(hashes, hashedPassword)
		return original(hashedPassword, password)
	}
	t.Cleanup(func() { comparePassword = original })
	return &hashes
}

// A login attempt for an identifier that resolves to no user must still spend
// time in bcrypt, otherwise its faster response reveals which identifiers exist
// (username enumeration via timing).
func TestLogin_UnknownIdentifier_RunsDummyCompare(t *testing.T) {
	notFound := func(context.Context, string) (*types.User, error) {
		return nil, store.ErrResourceNotFound
	}
	c := &Controller{principalStore: &stubPrincipalStore{findByUID: notFound, findByEmail: notFound}}

	hashes := recordComparisons(t)

	_, err := c.Login(context.Background(), &LoginInput{LoginIdentifier: "does-not-exist", Password: "whatever"})

	if !errors.Is(err, usererror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if len(*hashes) != 1 {
		t.Fatalf("expected exactly one password comparison, got %d", len(*hashes))
	}
	if !bytes.Equal((*hashes)[0], dummyPasswordHash) {
		t.Fatal("unknown-identifier path must compare against the dummy hash to equalize timing")
	}
}

// A wrong password for an existing user must be checked against that user's real
// hash (not the dummy) and still yield the same generic error as an unknown user.
func TestLogin_WrongPassword_ComparesRealHash(t *testing.T) {
	realHash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate hash: %v", err)
	}
	user := &types.User{UID: "alice", Password: string(realHash)}

	c := &Controller{principalStore: &stubPrincipalStore{
		findByUID: func(_ context.Context, uid string) (*types.User, error) {
			if uid == "alice" {
				return user, nil
			}
			return nil, store.ErrResourceNotFound
		},
		findByEmail: func(context.Context, string) (*types.User, error) {
			return nil, store.ErrResourceNotFound
		},
	}}

	hashes := recordComparisons(t)

	_, err = c.Login(context.Background(), &LoginInput{LoginIdentifier: "alice", Password: "wrong-password"})

	if !errors.Is(err, usererror.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if len(*hashes) != 1 {
		t.Fatalf("expected exactly one password comparison, got %d", len(*hashes))
	}
	if !bytes.Equal((*hashes)[0], realHash) {
		t.Fatal("wrong-password path must compare against the user's real hash")
	}
	if bytes.Equal((*hashes)[0], dummyPasswordHash) {
		t.Fatal("wrong-password path must not use the dummy hash")
	}
}
