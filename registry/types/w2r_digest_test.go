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

package types

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

const (
	w2rHex256 = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	w2rHex1   = "a9993e364706816aba3e25717850c26c9cd0d89d"
)

func TestW2rAlgorithmHelpers(t *testing.T) {
	tests := []struct {
		alg    Algorithm
		prefix string
		size   int
	}{
		{AlgorithmSHA256, "01", 32},
		{AlgorithmSHA512, "02", 64},
		{AlgorithmSHA1, "03", 20},
		{AlgorithmMD5, "04", 16},
	}
	for _, tt := range tests {
		require.Equal(t, tt.prefix, tt.alg.Prefix(), tt.alg)
		require.Equal(t, tt.size, tt.alg.Size(), tt.alg)
		require.Equal(t, tt.size, tt.alg.Hash().Size(), tt.alg)
		require.Equal(t, string(tt.alg), tt.alg.String())
	}
	require.Empty(t, Algorithm("crc32").Prefix())
	require.Zero(t, Algorithm("crc32").Size())
	require.Nil(t, Algorithm("crc32").Hash())
}

func TestW2rDigestAccessors(t *testing.T) {
	d := Digest("01" + w2rHex256)

	alg, err := d.Algorithm()
	require.NoError(t, err)
	require.Equal(t, AlgorithmSHA256, alg)
	require.Equal(t, w2rHex256, d.Hex())
	require.Equal(t, `\x01`+w2rHex256, d.HexDecode())
	require.True(t, d.IsValid())

	b, err := d.ToBytes()
	require.NoError(t, err)
	require.Len(t, b, 33)

	for prefix, want := range map[string]Algorithm{"02": AlgorithmSHA512, "03": AlgorithmSHA1, "04": AlgorithmMD5} {
		got, aErr := Digest(prefix + "00").Algorithm()
		require.NoError(t, aErr)
		require.Equal(t, want, got)
	}

	_, err = Digest("9").Algorithm()
	require.Error(t, err)
	_, err = Digest("99aa").Algorithm()
	require.Error(t, err)

	require.Empty(t, Digest("1").Hex())
	b, err = Digest("").ToBytes()
	require.NoError(t, err)
	require.Nil(t, b)
	_, err = Digest("1").ToBytes()
	require.Error(t, err)
}

func TestW2rDigestIsValid(t *testing.T) {
	tests := map[string]struct {
		d    Digest
		want bool
	}{
		"valid sha256":   {Digest("01" + w2rHex256), true},
		"valid sha1":     {Digest("03" + w2rHex1), true},
		"too short":      {Digest("0"), false},
		"unknown prefix": {Digest("99" + w2rHex256), false},
		"wrong length":   {Digest("01" + w2rHex1), false},
		"not hex":        {Digest("03" + strings.Repeat("z", 40)), false},
	}
	for name, tt := range tests {
		require.Equal(t, tt.want, tt.d.IsValid(), name)
	}
}

func TestW2rDigestConstructors(t *testing.T) {
	raw256, err := hex.DecodeString(w2rHex256)
	require.NoError(t, err)

	d, err := NewDigestFromBytes(AlgorithmSHA256, raw256)
	require.NoError(t, err)
	require.Equal(t, Digest("01"+w2rHex256), d)

	empty, err := NewDigestFromBytes(AlgorithmSHA256, nil)
	require.NoError(t, err)
	require.Empty(t, empty)

	_, err = NewDigestFromBytes(Algorithm("crc32"), raw256)
	require.Error(t, err)
	_, err = NewDigestFromBytes(AlgorithmSHA1, raw256)
	require.Error(t, err)

	fromHex, err := NewDigestFromHex(AlgorithmSHA256, w2rHex256)
	require.NoError(t, err)
	require.Equal(t, d, fromHex)
	emptyHex, err := NewDigestFromHex(AlgorithmSHA256, "")
	require.NoError(t, err)
	require.Empty(t, emptyHex)
	_, err = NewDigestFromHex(AlgorithmSHA256, "zz")
	require.Error(t, err)

	require.Equal(t, Digest("01"+w2rHex256), NewSHA256Digest(raw256))
	require.Equal(t, Digest("02aa"), NewSHA512Digest([]byte{0xaa}))
	require.Equal(t, Digest("03aa"), NewSHA1Digest([]byte{0xaa}))
	require.Equal(t, Digest("04aa"), NewMD5Digest([]byte{0xaa}))
	for _, empty := range []Digest{NewSHA256Digest(nil), NewSHA512Digest(nil), NewSHA1Digest(nil), NewMD5Digest(nil)} {
		require.Empty(t, empty)
	}

	got, err := GetHexDecodedBytes("0aff")
	require.NoError(t, err)
	require.Equal(t, []byte{0x0a, 0xff}, got)
	_, err = GetHexDecodedBytes("0g")
	require.Error(t, err)
}

func TestW2rDigestParseAndNewDigest(t *testing.T) {
	ociDigest := digest.NewDigestFromHex("sha256", w2rHex256)

	d, err := NewDigest(ociDigest)
	require.NoError(t, err)
	require.Equal(t, Digest("01"+w2rHex256), d)

	parsed, err := d.Parse()
	require.NoError(t, err)
	require.Equal(t, ociDigest, parsed)

	sha1D, err := NewDigest(digest.NewDigestFromHex("sha1", w2rHex1))
	if err == nil {
		back, pErr := sha1D.Parse()
		require.NoError(t, pErr)
		require.Equal(t, w2rHex1, back.Encoded())
	}

	_, err = NewDigest("")
	require.Error(t, err)
	_, err = NewDigest("nocolon")
	require.Error(t, err)
	_, err = NewDigest("sha256:short")
	require.Error(t, err)
	_, err = NewDigest(digest.NewDigestFromHex("sha384", strings.Repeat("a", 96)))
	require.Error(t, err)

	empty, err := Digest("").Parse()
	require.NoError(t, err)
	require.Empty(t, empty)
	_, err = Digest("0").Parse()
	require.Error(t, err)
	_, err = Digest("01").Parse()
	require.Error(t, err)
	_, err = Digest("99aa").Parse()
	require.Error(t, err)
	_, err = Digest("01aa").Parse()
	require.Error(t, err)
}

func TestW2rGetDigestBytes(t *testing.T) {
	b, err := GetDigestBytes(digest.NewDigestFromHex("sha256", w2rHex256))
	require.NoError(t, err)
	require.Len(t, b, 33, "the algorithm prefix byte is part of the stored bytes")
}
