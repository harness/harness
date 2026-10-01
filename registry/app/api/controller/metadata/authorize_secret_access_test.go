//  Copyright 2023 Harness, Inc.
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

package metadata

import (
	"testing"

	registrytypes "github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/assert"
)

func TestUpstreamSecretReferenceChanged(t *testing.T) {
	baseExisting := func() *registrytypes.UpstreamProxy {
		return &registrytypes.UpstreamProxy{
			SecretIdentifier:         "secret-a",
			SecretSpaceID:            10,
			UserNameSecretIdentifier: "user-secret-a",
			UserNameSecretSpaceID:    20,
			RepoURL:                  "https://upstream.example/repo",
			Source:                   "Custom",
		}
	}
	baseUpdated := func() *registrytypes.UpstreamProxyConfig {
		return &registrytypes.UpstreamProxyConfig{
			SecretIdentifier:         "secret-a",
			SecretSpaceID:            10,
			UserNameSecretIdentifier: "user-secret-a",
			UserNameSecretSpaceID:    20,
			URL:                      "https://upstream.example/repo",
			Source:                   "Custom",
		}
	}

	tests := []struct {
		name     string
		existing *registrytypes.UpstreamProxy
		updated  *registrytypes.UpstreamProxyConfig
		want     bool
	}{
		{
			name:     "nil_existing",
			existing: nil,
			updated:  baseUpdated(),
			want:     true,
		},
		{
			name:     "nil_updated",
			existing: baseExisting(),
			updated:  nil,
			want:     true,
		},
		{
			name:     "unchanged",
			existing: baseExisting(),
			updated:  baseUpdated(),
			want:     false,
		},
		{
			name:     "secret_identifier_changed",
			existing: baseExisting(),
			updated: func() *registrytypes.UpstreamProxyConfig {
				u := baseUpdated()
				u.SecretIdentifier = "secret-b"
				return u
			}(),
			want: true,
		},
		{
			name:     "secret_space_id_changed",
			existing: baseExisting(),
			updated: func() *registrytypes.UpstreamProxyConfig {
				u := baseUpdated()
				u.SecretSpaceID = 11
				return u
			}(),
			want: true,
		},
		{
			name: "secret_space_id_unset_minus_one_vs_zero_unchanged",
			existing: func() *registrytypes.UpstreamProxy {
				e := baseExisting()
				e.SecretSpaceID = -1
				return e
			}(),
			updated: func() *registrytypes.UpstreamProxyConfig {
				u := baseUpdated()
				u.SecretSpaceID = 0
				return u
			}(),
			want: false,
		},
		{
			name: "secret_space_id_unset_zero_vs_minus_one_unchanged",
			existing: func() *registrytypes.UpstreamProxy {
				e := baseExisting()
				e.SecretSpaceID = 0
				return e
			}(),
			updated: func() *registrytypes.UpstreamProxyConfig {
				u := baseUpdated()
				u.SecretSpaceID = -1
				return u
			}(),
			want: false,
		},
		{
			name:     "username_secret_identifier_changed",
			existing: baseExisting(),
			updated: func() *registrytypes.UpstreamProxyConfig {
				u := baseUpdated()
				u.UserNameSecretIdentifier = "user-secret-b"
				return u
			}(),
			want: true,
		},
		{
			name:     "username_secret_space_id_changed",
			existing: baseExisting(),
			updated: func() *registrytypes.UpstreamProxyConfig {
				u := baseUpdated()
				u.UserNameSecretSpaceID = 21
				return u
			}(),
			want: true,
		},
		{
			name: "username_secret_space_id_unset_minus_one_vs_zero_unchanged",
			existing: func() *registrytypes.UpstreamProxy {
				e := baseExisting()
				e.UserNameSecretSpaceID = -1
				return e
			}(),
			updated: func() *registrytypes.UpstreamProxyConfig {
				u := baseUpdated()
				u.UserNameSecretSpaceID = 0
				return u
			}(),
			want: false,
		},
		{
			name:     "url_changed",
			existing: baseExisting(),
			updated: func() *registrytypes.UpstreamProxyConfig {
				u := baseUpdated()
				u.URL = "https://attacker.example/exfil"
				return u
			}(),
			want: true,
		},
		{
			name:     "url_empty_on_update_does_not_trigger",
			existing: baseExisting(),
			updated: func() *registrytypes.UpstreamProxyConfig {
				u := baseUpdated()
				u.URL = ""
				return u
			}(),
			want: false,
		},
		{
			name:     "source_changed",
			existing: baseExisting(),
			updated: func() *registrytypes.UpstreamProxyConfig {
				u := baseUpdated()
				u.Source = "Dockerhub"
				return u
			}(),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, upstreamSecretReferenceChanged(tt.existing, tt.updated))
		})
	}
}

func TestNormalizeSecretSpaceID(t *testing.T) {
	assert.Equal(t, int64(0), normalizeSecretSpaceID(-1))
	assert.Equal(t, int64(0), normalizeSecretSpaceID(0))
	assert.Equal(t, int64(42), normalizeSecretSpaceID(42))
}
