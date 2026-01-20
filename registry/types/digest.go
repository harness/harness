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
	"crypto/md5"  //nolint:gosec
	"crypto/sha1" //nolint:gosec
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"strings"

	"github.com/opencontainers/go-digest"
)

// Digest is the database representation of a digest, stored in the format `<algorithm prefix><hex>`.
type Digest string

// Algorithm represents a supported hash algorithm.
type Algorithm string

const (
	// Algorithm prefixes are sequences of two digits. These should never change, only additions are allowed.
	sha256DigestAlgorithmPrefix = "01"
	sha512DigestAlgorithmPrefix = "02"
	sha1DigestAlgorithmPrefix   = "03"
	md5DigestAlgorithmPrefix    = "04"

	// Supported algorithms.
	AlgorithmSHA256 Algorithm = "sha256"
	AlgorithmSHA512 Algorithm = "sha512"
	AlgorithmSHA1   Algorithm = "sha1"
	AlgorithmMD5    Algorithm = "md5"
)

// String implements the Stringer interface for Algorithm.
func (a Algorithm) String() string {
	return string(a)
}

// Prefix returns the 2-character prefix used for database storage.
func (a Algorithm) Prefix() string {
	switch a {
	case AlgorithmSHA256:
		return sha256DigestAlgorithmPrefix
	case AlgorithmSHA512:
		return sha512DigestAlgorithmPrefix
	case AlgorithmSHA1:
		return sha1DigestAlgorithmPrefix
	case AlgorithmMD5:
		return md5DigestAlgorithmPrefix
	default:
		return ""
	}
}

// Hash returns a new hash.Hash for this algorithm.
func (a Algorithm) Hash() hash.Hash {
	switch a {
	case AlgorithmSHA256:
		return sha256.New()
	case AlgorithmSHA512:
		return sha512.New()
	case AlgorithmSHA1:
		return sha1.New() //nolint:gosec
	case AlgorithmMD5:
		return md5.New() //nolint:gosec
	default:
		return nil
	}
}

// Size returns the size in bytes of the hash output.
func (a Algorithm) Size() int {
	switch a {
	case AlgorithmSHA256:
		return sha256.Size
	case AlgorithmSHA512:
		return sha512.Size
	case AlgorithmSHA1:
		return sha1.Size
	case AlgorithmMD5:
		return md5.Size
	default:
		return 0
	}
}

// String implements the Stringer interface for Digest.
func (d Digest) String() string {
	return string(d)
}

// Algorithm returns the algorithm of this digest.
func (d Digest) Algorithm() (Algorithm, error) {
	if len(d) < 2 {
		return "", errors.New("invalid digest: too short")
	}
	prefix := string(d)[:2]
	switch prefix {
	case sha256DigestAlgorithmPrefix:
		return AlgorithmSHA256, nil
	case sha512DigestAlgorithmPrefix:
		return AlgorithmSHA512, nil
	case sha1DigestAlgorithmPrefix:
		return AlgorithmSHA1, nil
	case md5DigestAlgorithmPrefix:
		return AlgorithmMD5, nil
	default:
		return "", fmt.Errorf("unknown algorithm prefix: %s", prefix)
	}
}

// Hex returns the hex-encoded hash value (without prefix).
func (d Digest) Hex() string {
	if len(d) < 2 {
		return ""
	}
	return string(d)[2:]
}

// ToBytes extracts the raw hash bytes from a Digest.
// The digest format is "<2-digit algorithm prefix><hex>", this extracts and decodes the hex part.
func (d Digest) ToBytes() ([]byte, error) {
	if d == "" {
		return nil, nil
	}
	str := d.String()
	if len(str) < 2 {
		return nil, errors.New("invalid digest format: too short")
	}
	return hex.DecodeString(str)
}

// HexDecode decodes binary data from a textual representation.
// The output is equivalent to the PostgreSQL binary `decode` function with the hex textual format.
// See https://www.postgresql.org/docs/14/functions-binarystring.html.
func (d Digest) HexDecode() string {
	return fmt.Sprintf("\\x%s", d.String())
}

// IsValid checks if the digest has a valid format.
func (d Digest) IsValid() bool {
	if len(d) < 2 {
		return false
	}
	alg, err := d.Algorithm()
	if err != nil {
		return false
	}
	// Check hex length matches expected size for algorithm.
	expectedHexLen := alg.Size() * 2
	actualHexLen := len(d) - 2
	if actualHexLen != expectedHexLen {
		return false
	}
	// Verify it's valid hex.
	_, err = hex.DecodeString(d.Hex())
	return err == nil
}

// NewDigestFromBytes creates a Digest from raw hash bytes and an algorithm.
func NewDigestFromBytes(algorithm Algorithm, b []byte) (Digest, error) {
	if len(b) == 0 {
		return "", nil
	}
	prefix := algorithm.Prefix()
	if prefix == "" {
		return "", fmt.Errorf("unsupported algorithm: %s", algorithm)
	}
	expectedSize := algorithm.Size()
	if len(b) != expectedSize {
		return "", fmt.Errorf("invalid hash length for %s: expected %d bytes, got %d", algorithm, expectedSize, len(b))
	}
	return Digest(prefix + hex.EncodeToString(b)), nil
}

// NewDigestFromHex creates a Digest from a hex string and an algorithm.
func NewDigestFromHex(algorithm Algorithm, hexStr string) (Digest, error) {
	if hexStr == "" {
		return "", nil
	}
	// Validate hex.
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", fmt.Errorf("invalid hex string: %w", err)
	}
	return NewDigestFromBytes(algorithm, b)
}

// NewSHA256Digest creates a Digest from raw SHA256 hash bytes.
func NewSHA256Digest(b []byte) Digest {
	if len(b) == 0 {
		return ""
	}
	return Digest(sha256DigestAlgorithmPrefix + hex.EncodeToString(b))
}

// NewSHA512Digest creates a Digest from raw SHA512 hash bytes.
func NewSHA512Digest(b []byte) Digest {
	if len(b) == 0 {
		return ""
	}
	return Digest(sha512DigestAlgorithmPrefix + hex.EncodeToString(b))
}

// NewSHA1Digest creates a Digest from raw SHA1 hash bytes.
func NewSHA1Digest(b []byte) Digest {
	if len(b) == 0 {
		return ""
	}
	return Digest(sha1DigestAlgorithmPrefix + hex.EncodeToString(b))
}

// NewMD5Digest creates a Digest from raw MD5 hash bytes.
func NewMD5Digest(b []byte) Digest {
	if len(b) == 0 {
		return ""
	}
	return Digest(md5DigestAlgorithmPrefix + hex.EncodeToString(b))
}

// GetHexDecodedBytes decodes a hex string to bytes.
func GetHexDecodedBytes(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// ====================================================================================
// Compatibility functions for opencontainers/go-digest interoperability.
// These functions allow seamless conversion between types.Digest and digest.Digest.
// ====================================================================================

// OCI algorithm constants for compatibility with opencontainers/go-digest.
const (
	SHA1OCI digest.Algorithm = "sha1"
	MD5OCI  digest.Algorithm = "md5"
)

// NewDigest builds a Digest from an opencontainers/go-digest digest.Digest.
// This is provided for backward compatibility with existing code.
func NewDigest(d digest.Digest) (Digest, error) {
	// Check for valid format first (must contain ':' separator).
	if d == "" || !strings.Contains(string(d), ":") {
		return "", fmt.Errorf("invalid digest format: missing separator")
	}

	// For SHA1 and MD5, we don't call Validate() as opencontainers/go-digest
	// doesn't natively support these algorithms.
	alg := d.Algorithm()
	if alg != SHA1OCI && alg != MD5OCI {
		if err := d.Validate(); err != nil {
			return "", err
		}
	}

	var algPrefix string
	switch alg {
	case digest.SHA256:
		algPrefix = sha256DigestAlgorithmPrefix
	case digest.SHA512:
		algPrefix = sha512DigestAlgorithmPrefix
	case SHA1OCI:
		algPrefix = sha1DigestAlgorithmPrefix
	case MD5OCI:
		algPrefix = md5DigestAlgorithmPrefix
	case digest.SHA384:
		return "", fmt.Errorf("unimplemented algorithm %q", digest.SHA384)
	default:
		return "", fmt.Errorf("unknown algorithm %q", alg)
	}

	return Digest(fmt.Sprintf("%s%s", algPrefix, d.Encoded())), nil
}

// Parse maps a Digest to an opencontainers/go-digest digest.Digest.
// This is provided for backward compatibility with existing code.
func (d Digest) Parse() (digest.Digest, error) {
	if d == "" {
		return "", nil
	}

	str := d.String()
	if len(str) < 2 {
		return "", errors.New("invalid digest length")
	}

	algPrefix := str[:2]
	if len(str) == 2 {
		return "", errors.New("no checksum")
	}

	var alg digest.Algorithm
	switch algPrefix {
	case sha256DigestAlgorithmPrefix:
		alg = digest.SHA256
	case sha512DigestAlgorithmPrefix:
		alg = digest.SHA512
	case sha1DigestAlgorithmPrefix:
		alg = SHA1OCI
	case md5DigestAlgorithmPrefix:
		alg = MD5OCI
	default:
		return "", fmt.Errorf("unknown algorithm prefix %q", algPrefix)
	}

	dgst := digest.NewDigestFromHex(alg.String(), str[2:])

	// For SHA1 and MD5, we don't call Validate() as opencontainers/go-digest
	// doesn't natively support these algorithms.
	if alg != SHA1OCI && alg != MD5OCI {
		if err := dgst.Validate(); err != nil {
			return "", err
		}
	}

	return dgst, nil
}

// GetDigestBytes converts an opencontainers/go-digest digest.Digest to bytes.
// This first converts to our Digest format, then extracts the bytes.
func GetDigestBytes(dgst digest.Digest) ([]byte, error) {
	if dgst == "" {
		return nil, nil
	}

	d, err := NewDigest(dgst)
	if err != nil {
		return nil, err
	}

	return d.ToBytes()
}
