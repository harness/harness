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

package secret

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/gitspace/secret/enum"
	gitnesssecret "github.com/harness/gitness/secret"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSecretService struct {
	decryptFn func(ctx context.Context, spacePath, secretIdentifier string) (string, error)
}

func (m *mockSecretService) DecryptSecret(ctx context.Context, spacePath, secretIdentifier string) (string, error) {
	if m.decryptFn != nil {
		return m.decryptFn(ctx, spacePath, secretIdentifier)
	}
	return "", errors.New("secret not found")
}

var _ gitnesssecret.Service = (*mockSecretService)(nil)

func TestPasswordResolver_Resolve_ConfiguredSecret(t *testing.T) {
	ctx := context.Background()
	mockSvc := &mockSecretService{
		decryptFn: func(_ context.Context, spacePath, secretIdentifier string) (string, error) {
			if spacePath == "space1" && secretIdentifier == "my-configured-secret-ref" {
				return "my-secure-password-456", nil
			}
			return "", errors.New("not found")
		},
	}

	resolver := NewPasswordResolver(mockSvc)
	resolved, err := resolver.Resolve(ctx, ResolutionContext{
		UserIdentifier:     "victim",
		GitspaceIdentifier: "gs-1",
		SecretRef:          "my-configured-secret-ref",
		SpaceIdentifier:    "space1",
	})
	require.NoError(t, err)
	assert.Equal(t, "my-secure-password-456", resolved.SecretValue)
	assert.NotEqual(t, "Harness@123", resolved.SecretValue)
}

func TestPasswordResolver_Resolve_SecretNotFound(t *testing.T) {
	ctx := context.Background()
	mockSvc := &mockSecretService{
		decryptFn: func(_ context.Context, _, _ string) (string, error) {
			return "", errors.New("secret not found in store")
		},
	}

	resolver := NewPasswordResolver(mockSvc)
	_, err := resolver.Resolve(ctx, ResolutionContext{
		UserIdentifier:     "user1",
		GitspaceIdentifier: "gs-1",
		SecretRef:          "nonexistent-secret",
		SpaceIdentifier:    "space1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve secret")
}

func TestPasswordResolver_Resolve_DefaultPasswordRef_ExistsInStore(t *testing.T) {
	ctx := context.Background()
	mockSvc := &mockSecretService{
		decryptFn: func(_ context.Context, spacePath, secretIdentifier string) (string, error) {
			if secretIdentifier == "harness_password" {
				return "org-wide-gitspace-password", nil
			}
			return "", errors.New("not found")
		},
	}

	resolver := NewPasswordResolver(mockSvc)
	resolved, err := resolver.Resolve(ctx, ResolutionContext{
		UserIdentifier:     "user1",
		GitspaceIdentifier: "gs-1",
		SecretRef:          "harness_password",
		SpaceIdentifier:    "space1",
	})
	require.NoError(t, err)
	assert.Equal(t, "org-wide-gitspace-password", resolved.SecretValue)
	assert.NotEqual(t, "Harness@123", resolved.SecretValue)
}

func TestPasswordResolver_Resolve_DefaultPasswordRef_GeneratesRandom(t *testing.T) {
	ctx := context.Background()
	mockSvc := &mockSecretService{
		decryptFn: func(_ context.Context, _, _ string) (string, error) {
			return "", errors.New("not found in store")
		},
	}

	resolver := NewPasswordResolver(mockSvc)
	resolved1, err := resolver.Resolve(ctx, ResolutionContext{
		UserIdentifier:     "user1",
		GitspaceIdentifier: "gs-1",
		SecretRef:          "harness_password",
		SpaceIdentifier:    "space1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resolved1.SecretValue)
	assert.NotEqual(t, "Harness@123", resolved1.SecretValue)

	resolved2, err := resolver.Resolve(ctx, ResolutionContext{
		UserIdentifier:     "user2",
		GitspaceIdentifier: "gs-2",
		SecretRef:          "harness_password",
		SpaceIdentifier:    "space1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resolved2.SecretValue)
	assert.NotEqual(t, "Harness@123", resolved2.SecretValue)

	// Ensure random passwords differ between instances
	assert.NotEqual(t, resolved1.SecretValue, resolved2.SecretValue)
}

func TestPasswordResolver_Resolve_EmptySecretRef_GeneratesRandom(t *testing.T) {
	ctx := context.Background()
	resolver := NewPasswordResolver(nil)
	resolved, err := resolver.Resolve(ctx, ResolutionContext{
		UserIdentifier:     "user1",
		GitspaceIdentifier: "gs-1",
		SecretRef:          "",
		SpaceIdentifier:    "space1",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resolved.SecretValue)
	assert.NotEqual(t, "Harness@123", resolved.SecretValue)
	assert.Len(t, resolved.SecretValue, 24)
}

func TestPasswordResolver_Type(t *testing.T) {
	resolver := NewPasswordResolver(nil)
	assert.Equal(t, enum.PasswordSecretType, resolver.Type())
}
