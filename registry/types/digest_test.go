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
	"bytes"
	"crypto/md5"  //nolint:gosec // tests only
	"crypto/sha1" //nolint:gosec // tests only
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
)

func TestDigest_String(t *testing.T) {
	t.Parallel()

	d := Digest("01abcd")
	if d.String() != "01abcd" {
		t.Fatalf("String() mismatch: got %q", d.String())
	}
}

func TestDigest_HexDecodeAndValidate_Format(t *testing.T) {
	t.Parallel()

	d := Digest("01abcd")
	want := `\x01abcd`

	if got := d.HexDecode(); got != want {
		t.Fatalf("HexDecode() = %q, want %q", got, want)
	}
}

func TestGetHexDecodedBytes(t *testing.T) {
	t.Parallel()

	b, err := GetHexDecodedBytes("01ff10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(b, []byte{0x01, 0xff, 0x10}) {
		t.Fatalf("decoded bytes mismatch: got %v", b)
	}

	_, err = GetHexDecodedBytes("zz")
	if err == nil {
		t.Fatalf("expected error for invalid hex")
	}
}

func TestNewDigest_SHA256_FormatAndRoundTrip(t *testing.T) {
	t.Parallel()

	sum := sha256.Sum256([]byte("hello"))
	hex := fmt.Sprintf("%x", sum[:])
	oci := digest.Digest("sha256:" + hex)

	d, err := NewDigest(oci)
	if err != nil {
		t.Fatalf("NewDigest() error: %v", err)
	}

	if !strings.HasPrefix(d.String(), "01") {
		t.Fatalf("expected sha256 prefix 01, got %q", d.String()[:2])
	}
	if gotHex := d.String()[2:]; gotHex != hex {
		t.Fatalf("hex mismatch: got %q want %q", gotHex, hex)
	}
	if len(d.String()) != 2+64 {
		t.Fatalf("stored sha256 length mismatch: got %d want %d", len(d.String()), 66)
	}

	parsed, err := d.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if parsed.String() != oci.String() {
		t.Fatalf("round-trip mismatch: got %q want %q", parsed.String(), oci.String())
	}
}

func TestNewDigest_SHA512_FormatAndRoundTrip(t *testing.T) {
	t.Parallel()

	sum := sha512.Sum512([]byte("hello"))
	hex := fmt.Sprintf("%x", sum[:])
	oci := digest.Digest("sha512:" + hex)

	d, err := NewDigest(oci)
	if err != nil {
		t.Fatalf("NewDigest() error: %v", err)
	}

	if !strings.HasPrefix(d.String(), "02") {
		t.Fatalf("expected sha512 prefix 02, got %q", d.String()[:2])
	}
	if gotHex := d.String()[2:]; gotHex != hex {
		t.Fatalf("hex mismatch: got %q want %q", gotHex, hex)
	}
	if len(d.String()) != 2+128 {
		t.Fatalf("stored sha512 length mismatch: got %d want %d", len(d.String()), 130)
	}

	parsed, err := d.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if parsed.String() != oci.String() {
		t.Fatalf("round-trip mismatch: got %q want %q", parsed.String(), oci.String())
	}
}

func TestNewDigest_SHA1_FormatAndRoundTrip(t *testing.T) {
	t.Parallel()

	sum := sha1.Sum([]byte("hello")) //nolint:gosec // tests only
	hex := fmt.Sprintf("%x", sum[:])
	oci := digest.Digest("sha1:" + hex)

	d, err := NewDigest(oci)
	if err != nil {
		t.Fatalf("NewDigest() error: %v", err)
	}

	if !strings.HasPrefix(d.String(), "03") {
		t.Fatalf("expected sha1 prefix 03, got %q", d.String()[:2])
	}
	if gotHex := d.String()[2:]; gotHex != hex {
		t.Fatalf("hex mismatch: got %q want %q", gotHex, hex)
	}
	if len(d.String()) != 2+40 {
		t.Fatalf("stored sha1 length mismatch: got %d want %d", len(d.String()), 130)
	}

	parsed, err := d.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if parsed.String() != oci.String() {
		t.Fatalf("round-trip mismatch: got %q want %q", parsed.String(), oci.String())
	}
}

func TestNewDigest_MD5_FormatAndRoundTrip(t *testing.T) {
	t.Parallel()

	sum := md5.Sum([]byte("hello")) //nolint:gosec // tests only
	hex := fmt.Sprintf("%x", sum[:])
	oci := digest.Digest("md5:" + hex)

	d, err := NewDigest(oci)
	if err != nil {
		t.Fatalf("NewDigest() error: %v", err)
	}

	if !strings.HasPrefix(d.String(), "04") {
		t.Fatalf("expected md5 prefix 04, got %q", d.String()[:2])
	}
	if gotHex := d.String()[2:]; gotHex != hex {
		t.Fatalf("hex mismatch: got %q want %q", gotHex, hex)
	}
	if len(d.String()) != 2+32 {
		t.Fatalf("stored md5 length mismatch: got %d want %d", len(d.String()), 130)
	}

	parsed, err := d.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if parsed.String() != oci.String() {
		t.Fatalf("round-trip mismatch: got %q want %q", parsed.String(), oci.String())
	}
}

func TestGetDigestBytes_EmptyAndKnownAlgorithms(t *testing.T) {
	t.Parallel()

	// Empty -> (nil, nil) is the legacy behavior.
	b, err := GetDigestBytes(digest.Digest(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b != nil {
		t.Fatalf("expected nil bytes for empty digest, got %v", b)
	}

	// SHA256 bytes: first byte should be 0x01 (hex prefix "01"), followed by raw hash bytes.
	sha256Sum := sha256.Sum256([]byte("hello"))
	sha256Hex := fmt.Sprintf("%x", sha256Sum[:])
	sha256OCI := digest.Digest("sha256:" + sha256Hex)

	got, err := GetDigestBytes(sha256OCI)
	if err != nil {
		t.Fatalf("GetDigestBytes(sha256) error: %v", err)
	}
	if len(got) != 1+sha256.Size {
		t.Fatalf("sha256 bytes length = %d, want %d", len(got), 1+sha256.Size)
	}
	if got[0] != 0x01 {
		t.Fatalf("sha256 prefix byte = 0x%02x, want 0x01", got[0])
	}
	if !bytes.Equal(got[1:], sha256Sum[:]) {
		t.Fatalf("sha256 raw bytes mismatch")
	}

	// SHA512 bytes: first byte should be 0x02, followed by raw hash bytes.
	sha512Sum := sha512.Sum512([]byte("hello"))
	sha512Hex := fmt.Sprintf("%x", sha512Sum[:])
	sha512OCI := digest.Digest("sha512:" + sha512Hex)

	got, err = GetDigestBytes(sha512OCI)
	if err != nil {
		t.Fatalf("GetDigestBytes(sha512) error: %v", err)
	}
	if len(got) != 1+sha512.Size {
		t.Fatalf("sha512 bytes length = %d, want %d", len(got), 1+sha512.Size)
	}
	if got[0] != 0x02 {
		t.Fatalf("sha512 prefix byte = 0x%02x, want 0x02", got[0])
	}
	if !bytes.Equal(got[1:], sha512Sum[:]) {
		t.Fatalf("sha512 raw bytes mismatch")
	}
}

func TestParse_EmptyDigest_ReturnsEmptyNoError(t *testing.T) {
	t.Parallel()

	parsed, err := Digest("").Parse()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if parsed != "" {
		t.Fatalf("expected empty parsed digest, got %q", parsed)
	}
}

func TestParse_InvalidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Digest
		want string // substring expected in error (if any)
	}{
		{
			name: "too_short_len1",
			in:   Digest("0"),
			want: "invalid digest",
		},
		{
			name: "no_checksum_len2",
			in:   Digest("01"),
			want: "no checksum",
		},
		{
			name: "unknown_prefix",
			in:   Digest("99" + strings.Repeat("a", 64)),
			want: "unknown algorithm prefix",
		},
		{
			name: "sha256_wrong_len",
			in:   Digest("01" + "aa"),
			want: "invalid",
		},
		{
			name: "sha256_invalid_hex",
			in:   Digest("01" + strings.Repeat("z", 64)),
			want: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.in.Parse()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if tt.want != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.want)) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestNewDigest_InvalidInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   digest.Digest
	}{
		{
			name: "missing_separator",
			in:   digest.Digest("sha256"),
		},
		{
			name: "unknown_algorithm",
			in:   digest.Digest("nope:" + strings.Repeat("a", 64)),
		},
		{
			name: "sha256_bad_hex",
			in:   digest.Digest("sha256:" + strings.Repeat("z", 64)),
		},
		{
			name: "sha256_wrong_len",
			in:   digest.Digest("sha256:" + strings.Repeat("a", 2)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewDigest(tt.in)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func Test_SHA1_MD5_Support(t *testing.T) {
	t.Parallel()

	t.Run("sha1", func(t *testing.T) {
		t.Parallel()
		sum := sha1.Sum([]byte("hello")) //nolint:gosec // tests only
		hex := fmt.Sprintf("%x", sum[:])
		oci := digest.Digest("sha1:" + hex)

		d, err := NewDigest(oci)
		if err != nil {
			t.Skipf("SHA1 not supported by this implementation: %v", err)
		}
		if !strings.HasPrefix(d.String(), "03") {
			t.Fatalf("expected sha1 prefix 03, got %q", d.String()[:2])
		}

		parsed, err := d.Parse()
		if err != nil {
			t.Fatalf("Parse() error: %v", err)
		}
		if parsed.String() != oci.String() {
			t.Fatalf("round-trip mismatch: got %q want %q", parsed.String(), oci.String())
		}
	})

	t.Run("md5", func(t *testing.T) {
		t.Parallel()
		sum := md5.Sum([]byte("hello")) //nolint:gosec // tests only
		hex := fmt.Sprintf("%x", sum[:])
		oci := digest.Digest("md5:" + hex)

		d, err := NewDigest(oci)
		if err != nil {
			t.Skipf("MD5 not supported by this implementation: %v", err)
		}
		if !strings.HasPrefix(d.String(), "04") {
			t.Fatalf("expected md5 prefix 04, got %q", d.String()[:2])
		}

		parsed, err := d.Parse()
		if err != nil {
			t.Fatalf("Parse() error: %v", err)
		}
		if parsed.String() != oci.String() {
			t.Fatalf("round-trip mismatch: got %q want %q", parsed.String(), oci.String())
		}
	})
}

func TestCompatibility_NewDigest_EmptyDigest(t *testing.T) {
	t.Parallel()

	d, err := NewDigest(digest.Digest(""))
	if err == nil && d == "" {
		// New behavior (allowed).
		return
	}
	if err != nil {
		// Old behavior (also allowed for compatibility test suite).
		return
	}
	t.Fatalf("unexpected result: d=%q err=%v", d, err)
}

/*
Fuzz tests (optional). Run explicitly with:
  go test -fuzz=Fuzz -run=^$
They help catch panics and weird edge-cases.
*/

func FuzzDigestRoundTripSHA256(f *testing.F) {
	f.Add([]byte("hello"))
	f.Add([]byte(""))
	f.Add([]byte("some longer payload ...."))

	f.Fuzz(func(t *testing.T, b []byte) {
		sum := sha256.Sum256(b)
		hex := fmt.Sprintf("%x", sum[:])
		oci := digest.Digest("sha256:" + hex)

		stored, err := NewDigest(oci)
		if err != nil {
			t.Fatalf("NewDigest error: %v", err)
		}

		parsed, err := stored.Parse()
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if parsed.String() != oci.String() {
			t.Fatalf("round-trip mismatch: got %q want %q", parsed.String(), oci.String())
		}

		got, err := GetDigestBytes(oci)
		if err != nil {
			t.Fatalf("GetDigestBytes error: %v", err)
		}
		if len(got) != 1+sha256.Size {
			t.Fatalf("bytes length mismatch: got %d want %d", len(got), 1+sha256.Size)
		}
		if got[0] != 0x01 {
			t.Fatalf("prefix byte mismatch: got 0x%02x want 0x01", got[0])
		}
		if !bytes.Equal(got[1:], sum[:]) {
			t.Fatalf("raw bytes mismatch")
		}
	})
}

func FuzzDigestRoundTripSHA512(f *testing.F) {
	f.Add([]byte("hello"))
	f.Add([]byte(""))
	f.Add([]byte("some longer payload ...."))

	f.Fuzz(func(t *testing.T, b []byte) {
		sum := sha512.Sum512(b)
		hex := fmt.Sprintf("%x", sum[:])
		oci := digest.Digest("sha512:" + hex)

		stored, err := NewDigest(oci)
		if err != nil {
			t.Fatalf("NewDigest error: %v", err)
		}

		parsed, err := stored.Parse()
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if parsed.String() != oci.String() {
			t.Fatalf("round-trip mismatch: got %q want %q", parsed.String(), oci.String())
		}

		got, err := GetDigestBytes(oci)
		if err != nil {
			t.Fatalf("GetDigestBytes error: %v", err)
		}
		if len(got) != 1+sha512.Size {
			t.Fatalf("bytes length mismatch: got %d want %d", len(got), 1+sha512.Size)
		}
		if got[0] != 0x02 {
			t.Fatalf("prefix byte mismatch: got 0x%02x want 0x02", got[0])
		}
		if !bytes.Equal(got[1:], sum[:]) {
			t.Fatalf("raw bytes mismatch")
		}
	})
}
