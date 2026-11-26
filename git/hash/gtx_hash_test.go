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

package hash

import (
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGtxNew(t *testing.T) {
	tests := []struct {
		name    string
		t       Type
		at      AggregationType
		wantErr string
	}{
		{name: "sha256 xor", t: TypeSHA256, at: AggregationTypeXOR},
		{name: "unknown hash type", t: "md5", at: AggregationTypeXOR, wantErr: "unknown hash type 'md5'"},
		{name: "empty hash type", t: "", at: AggregationTypeXOR, wantErr: "unknown hash type ''"},
		{name: "unknown aggregation type", t: TypeSHA256, at: "sum", wantErr: "unknown aggregation type 'sum'"},
		{name: "empty aggregation type", t: TypeSHA256, at: "", wantErr: "unknown aggregation type ''"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			agg, err := New(test.t, test.at)
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				assert.Nil(t, agg)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, agg)
			assert.Len(t, agg.Empty(), sha256.Size)
		})
	}
}

func TestGtxSerialize(t *testing.T) {
	assert.Equal(t, []byte("refs/heads/main:abc123"), SerializeReference("refs/heads/main", "abc123"))
	assert.Equal(t, []byte(":"), SerializeReference("", ""))
	assert.Equal(t, []byte("HEAD:refs/heads/main"), SerializeHead("refs/heads/main"))
	assert.Equal(t, []byte("HEAD:"), SerializeHead(""))
}

func TestGtxXORAggregator(t *testing.T) {
	agg, err := New(TypeSHA256, AggregationTypeXOR)
	require.NoError(t, err)

	slice := func(vals ...string) Source {
		data := make([][]byte, len(vals))
		for i := range vals {
			data[i] = []byte(vals[i])
		}
		return SourceFromSlice(data)
	}

	t.Run("hash of empty source is the empty hash", func(t *testing.T) {
		got, err := agg.Hash(slice())
		require.NoError(t, err)
		assert.Equal(t, agg.Empty(), got)
	})

	t.Run("order doesn't matter", func(t *testing.T) {
		first, err := agg.Hash(slice("a", "b", "c"))
		require.NoError(t, err)
		second, err := agg.Hash(slice("c", "a", "b"))
		require.NoError(t, err)

		assert.Equal(t, first, second)
		assert.NotEqual(t, agg.Empty(), first)
	})

	t.Run("equal elements cancel each other out", func(t *testing.T) {
		got, err := agg.Hash(slice("a", "b", "a"))
		require.NoError(t, err)
		want, err := agg.Hash(slice("b"))
		require.NoError(t, err)

		assert.Equal(t, want, got)

		none, err := agg.Hash(slice("x", "x"))
		require.NoError(t, err)
		assert.Equal(t, agg.Empty(), none)
	})

	t.Run("append is equal to hashing everything", func(t *testing.T) {
		base, err := agg.Hash(slice("a", "b"))
		require.NoError(t, err)
		baseCopy := append([]byte(nil), base...)

		appended, err := agg.Append(base, slice("c"))
		require.NoError(t, err)
		all, err := agg.Hash(slice("a", "b", "c"))
		require.NoError(t, err)

		assert.Equal(t, all, appended)
		assert.Equal(t, baseCopy, base, "the original hash must not be modified")
	})

	t.Run("hash matches the single sha256", func(t *testing.T) {
		got, err := agg.Hash(slice("value"))
		require.NoError(t, err)

		want := sha256.Sum256([]byte("value"))
		assert.Equal(t, want[:], got)
	})

	t.Run("append with an invalid hash length", func(t *testing.T) {
		_, err := agg.Append([]byte{1, 2, 3}, slice("a"))
		require.ErrorContains(t, err, "invalid length")
	})

	t.Run("source error", func(t *testing.T) {
		sourceErr := errors.New("gtx source failure")
		calls := 0
		src := SourceFunc(func() ([]byte, error) {
			calls++
			if calls == 1 {
				return []byte("a"), nil
			}
			return nil, sourceErr
		})

		got, err := agg.Hash(src)
		require.ErrorIs(t, err, sourceErr)
		assert.Nil(t, got)
	})
}

func TestGtxXorInPlace(t *testing.T) {
	tests := []struct {
		name string
		a    []byte
		b    []byte
		want []byte
	}{
		{name: "equal length", a: []byte{0b1100, 0b1010}, b: []byte{0b1010, 0b1010}, want: []byte{0b0110, 0}},
		{name: "second is shorter", a: []byte{1, 2, 3}, b: []byte{1}, want: []byte{0, 2, 3}},
		{name: "first is shorter", a: []byte{1}, b: []byte{1, 2, 3}, want: []byte{0, 2, 3}},
		{name: "both empty", a: []byte{}, b: []byte{}, want: []byte{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, xorInPlace(test.a, test.b))
		})
	}
}
