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

package webhook

import (
	"testing"

	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckExtraHeaders(t *testing.T) {
	tooMany := make([]types.ExtraHeader, webhookMaxExtraHeaders+1)
	for i := range tooMany {
		tooMany[i] = types.ExtraHeader{Key: "X-Key-" + string(rune('a'+i)), Value: "v"}
	}

	tests := []struct {
		name    string
		headers []types.ExtraHeader
		wantErr bool
	}{
		{name: "nil", headers: nil},
		{name: "valid", headers: []types.ExtraHeader{
			{Key: "X-A", Value: "a"},
			{Key: "Authorization", Value: "token", Masked: true},
		}},
		{name: "unmasked_empty_value_allowed", headers: []types.ExtraHeader{{Key: "X-A"}}},
		{name: "empty_key", headers: []types.ExtraHeader{{Key: "", Value: "a"}}, wantErr: true},
		{name: "duplicate_key_case_insensitive", headers: []types.ExtraHeader{
			{Key: "X-A", Value: "a"},
			{Key: "x-a", Value: "b"},
		}, wantErr: true},
		{name: "masked_without_value", headers: []types.ExtraHeader{{Key: "Authorization", Masked: true}}, wantErr: true},
		{name: "too_many", headers: tooMany, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckExtraHeaders(tc.headers)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMergeExtraHeaders(t *testing.T) {
	existing := []types.ExtraHeader{
		{Key: "Authorization", Value: "stored-token", Masked: true},
		{Key: "X-Plain", Value: "plain"},
		{Key: "X-Removed", Value: "gone", Masked: true},
	}

	tests := []struct {
		name string
		in   []types.ExtraHeader
		want []types.ExtraHeader
	}{
		{
			name: "round_trip_of_redacted_headers_keeps_stored_values",
			in:   types.RedactExtraHeaders(existing[:2]),
			want: existing[:2],
		},
		{
			name: "key_match_is_case_insensitive",
			in:   []types.ExtraHeader{{Key: "authorization", Masked: true}},
			want: []types.ExtraHeader{{Key: "authorization", Value: "stored-token", Masked: true}},
		},
		{
			name: "new_value_replaces_stored_value",
			in:   []types.ExtraHeader{{Key: "Authorization", Value: "new-token", Masked: true}},
			want: []types.ExtraHeader{{Key: "Authorization", Value: "new-token", Masked: true}},
		},
		{
			name: "masking_existing_unmasked_header_keeps_value",
			in:   []types.ExtraHeader{{Key: "X-Plain", Masked: true}},
			want: []types.ExtraHeader{{Key: "X-Plain", Value: "plain", Masked: true}},
		},
		{
			name: "unmasking_without_value_does_not_reveal_stored_value",
			in:   []types.ExtraHeader{{Key: "Authorization"}},
			want: []types.ExtraHeader{{Key: "Authorization"}},
		},
		{
			name: "new_masked_header_without_value_stays_empty",
			in:   []types.ExtraHeader{{Key: "X-New", Masked: true}},
			want: []types.ExtraHeader{{Key: "X-New", Masked: true}},
		},
		{
			name: "empty_list_removes_all",
			in:   []types.ExtraHeader{},
			want: []types.ExtraHeader{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, MergeExtraHeaders(existing, tc.in))
		})
	}

	// a new masked header without a value must be rejected after merging
	assert.Error(t, CheckExtraHeaders(MergeExtraHeaders(existing, []types.ExtraHeader{{Key: "X-New", Masked: true}})))
}

func TestEncryptSecret(t *testing.T) {
	encrypter, err := encrypt.New("fb4f3d6b1ec2fe0c5bf3d5bd8f1ac3a9", false)
	require.NoError(t, err)
	s := &Service{encrypter: encrypter}

	// encrypting an empty string with AES-GCM yields a non-empty ciphertext, which must not be stored.
	empty, err := s.encryptSecret("")
	require.NoError(t, err)
	assert.Empty(t, empty)

	encrypted, err := s.encryptSecret("my-secret")
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, "my-secret", encrypted)

	decrypted, err := encrypter.Decrypt([]byte(encrypted))
	require.NoError(t, err)
	assert.Equal(t, "my-secret", decrypted)
}
