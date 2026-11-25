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
	"fmt"
	"testing"

	"github.com/harness/gitness/app/gitspace/secret/enum"
	gitnesssecret "github.com/harness/gitness/secret"
	"github.com/harness/gitness/store"

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
			return "", fmt.Errorf("failed to find secret: %w", store.ErrResourceNotFound)
		},
	}

	resolver := NewPasswordResolver(mockSvc)
	_, err := resolver.Resolve(ctx, ResolutionContext{
		UserIdentifier:     "user1",
		GitspaceIdentifier: "gs-1",
		SecretRef:          "harness_password",
		SpaceIdentifier:    "space1",
	})
	require.ErrorIs(t, err, store.ErrResourceNotFound)
	assert.Contains(t, err.Error(), "SSH key")
}

func TestPasswordResolver_Resolve_LookupError(t *testing.T) {
	ctx := context.Background()
	mockSvc := &mockSecretService{
		decryptFn: func(_ context.Context, _, _ string) (string, error) {
			return "", errors.New("failed to decrypt secret")
		},
	}

	resolver := NewPasswordResolver(mockSvc)
	_, err := resolver.Resolve(ctx, ResolutionContext{
		UserIdentifier:     "user1",
		GitspaceIdentifier: "gs-1",
		SecretRef:          "harness_password",
		SpaceIdentifier:    "space1",
	})
	require.Error(t, err)
	assert.NotErrorIs(t, err, store.ErrResourceNotFound)
	assert.Contains(t, err.Error(), "failed to decrypt secret")
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

func TestPasswordResolver_Resolve_EmptySecretValue(t *testing.T) {
	ctx := context.Background()
	mockSvc := &mockSecretService{
		decryptFn: func(_ context.Context, _, _ string) (string, error) {
			return "", nil
		},
	}

	resolver := NewPasswordResolver(mockSvc)
	_, err := resolver.Resolve(ctx, ResolutionContext{
		UserIdentifier:     "user1",
		GitspaceIdentifier: "gs-1",
		SecretRef:          "harness_password",
		SpaceIdentifier:    "space1",
	})
	require.Error(t, err)
}

func TestPasswordResolver_Type(t *testing.T) {
	resolver := NewPasswordResolver(nil)
	assert.Equal(t, enum.PasswordSecretType, resolver.Type())
}
