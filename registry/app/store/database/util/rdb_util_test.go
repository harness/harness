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

package util

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type rdbTagged struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
	Skip string
	Note string `db:"note"`
}

func TestRdbGetEmptySQLValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"empty", "", false},
		{"value", "abc", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetEmptySQLString(tc.input)
			require.Equal(t, tc.valid, got.Valid)
			require.Equal(t, tc.input, got.String)
		})
	}

	zero := GetEmptySQLInt64(0)
	require.False(t, zero.Valid)
	set := GetEmptySQLInt64(42)
	require.True(t, set.Valid)
	require.Equal(t, int64(42), set.Int64)
}

func TestRdbConstructQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		args  []any
		want  string
	}{
		{"string is quoted and escaped", "a = ?", []any{"it's"}, "a = 'it''s'"},
		{"numbers", "a = ? AND b = ? AND c = ?", []any{1, int64(2), 1.5}, "a = 1 AND b = 2 AND c = 1.5"},
		{"bool", "a = ?", []any{true}, "a = true"},
		{"other types are quoted", "a = ?", []any{[]int{1, 2}}, "a = '[1 2]'"},
		{"missing args keep the placeholder", "a = ? AND b = ?", []any{"x"}, "a = 'x' AND b = ?"},
		{"no args", "SELECT 1", nil, "SELECT 1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, ConstructQuery(tc.query, tc.args))
		})
	}
}

func TestRdbFormatQuery(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		params []any
		want   string
	}{
		{"string", "a = $1", []any{"x'y"}, "a = 'x''y'"},
		{"string slice", "a = ANY($1)", []any{[]string{"a", "b'c"}}, "a = ANY(ARRAY['a', 'b''c'])"},
		{"default", "a = $1 AND b = $2", []any{7, true}, "a = 7 AND b = true"},
		{"no params", "SELECT 1", nil, "SELECT 1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, FormatQuery(tc.query, tc.params))
		})
	}
}

func TestRdbIntHelpers(t *testing.T) {
	require.Equal(t, uint64(0), SafeIntToUInt64(-5))
	require.Equal(t, uint64(0), SafeIntToUInt64(0))
	require.Equal(t, uint64(9), SafeIntToUInt64(9))

	require.Equal(t, 1, MinInt(1, 2))
	require.Equal(t, 1, MinInt(2, 1))
	require.Equal(t, 3, MinInt(3, 3))
}

func TestRdbArrayConversions(t *testing.T) {
	require.Nil(t, StringToArr(""))
	require.Equal(t, []string{"a", "b"}, StringToArr("a^_b"))
	require.Equal(t, "a^_b", ArrToString([]string{"a", "b"}))
	require.Empty(t, ArrToString(nil))

	require.Equal(t, "1^_2", Int64ArrToString([]int64{1, 2}))
	require.Empty(t, Int64ArrToString(nil))
	require.Nil(t, StringToInt64Arr(""))
	require.Equal(t, []int64{1, 2, 30}, StringToInt64Arr("1^_2^_30"))
	// Unparsable parts decode as zero.
	require.Equal(t, []int64{1, 0}, StringToInt64Arr("1^_x"))

	require.Equal(t, []string{"a", "b"}, StringToArrByDelimiter("a,b", ","))
	require.Equal(t, "a,b", ArrToStringByDelimiter([]string{"a", "b"}, ","))
	require.Equal(t, "1,2", Int64ArrToStringByDelimiter([]int64{1, 2}, ","))
	require.Equal(t, []int64{4, 5}, StringToInt64ArrByDelimiter("4,5", ","))
}

func TestRdbStructTagHelpers(t *testing.T) {
	require.Equal(t, []string{"id", "name", "note"}, GetDBTagsFromStruct(rdbTagged{}))

	require.Equal(t, "id = :id, name = :name, note = :note", GetSetDBKeys(rdbTagged{}))
	require.Equal(t, "name = :name", GetSetDBKeys(rdbTagged{}, "id", "note"))
	require.Empty(t, GetSetDBKeys(rdbTagged{}, "id", "name", "note"))
}

func TestRdbHexHelpers(t *testing.T) {
	b, err := GetHexDecodedBytes("0aff")
	require.NoError(t, err)
	require.Equal(t, []byte{0x0a, 0xff}, b)
	require.Equal(t, "0aff", GetHexEncodedString(b))

	_, err = GetHexDecodedBytes("zz")
	require.Error(t, err)
	_, err = GetHexDecodedBytes("abc")
	require.Error(t, err)
}

func TestRdbStoreErrors(t *testing.T) {
	require.ErrorIs(t, ErrManifestNotFound, ErrNotFound)
	require.ErrorIs(t, ErrRefManifestNotFound, ErrManifestNotFound)
	require.ErrorIs(t, ErrRefManifestNotFound, ErrNotFound)
	require.False(t, errors.Is(ErrManifestReferencedInList, ErrNotFound))

	err := UnknownMediaTypeError{MediaType: "application/x-unknown"}
	require.Equal(t, "unknown media type: application/x-unknown", err.Error())
	var target UnknownMediaTypeError
	require.ErrorAs(t, error(err), &target)
	require.Equal(t, "application/x-unknown", target.MediaType)
}
