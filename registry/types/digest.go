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
	"errors"
	"fmt"

	"github.com/harness/gitness/registry/app/store/database/util"

	"github.com/opencontainers/go-digest"
)

// Digest is the database representation of a digest, stored in the format `<algorithm prefix><hex>`.
type Digest string

const (
	// Algorithm prefixes are sequences of two digits. These should never change, only additions are allowed.
	sha256DigestAlgorithmPrefix = "01"
	sha512DigestAlgorithmPrefix = "02"
)

func GetDigestBytes(dgst digest.Digest) ([]byte, error) {
	if len(dgst.String()) == 0 {
		return nil, nil
	}

	newDigest, err := NewDigest(dgst)
	if err != nil {
		return nil, err
	}

	digestBytes, err := util.GetHexDecodedBytes(string(newDigest))
	if err != nil {
		return nil, err
	}
	return digestBytes, nil
}

// String implements the Stringer interface.
func (d Digest) String() string {
	return string(d)
}

// NewDigest builds a Digest based on a digest.Digest.
func NewDigest(d digest.Digest) (Digest, error) {
	if err := d.Validate(); err != nil {
		return "", err
	}

	var algPrefix string
	switch d.Algorithm() {
	case digest.SHA256:
		algPrefix = sha256DigestAlgorithmPrefix
	case digest.SHA512:
		algPrefix = sha512DigestAlgorithmPrefix
	case digest.SHA384:
		return "", fmt.Errorf("unimplemented algorithm %q", digest.SHA384)
	default:
		return "", fmt.Errorf("unknown algorithm %q", d.Algorithm())
	}

	return Digest(fmt.Sprintf("%s%s", algPrefix, d.Hex())), nil
}

// Parse maps a Digest to a digest.Digest.
func (d Digest) Parse() (digest.Digest, error) {
	str := d.String()
	valid, err := d.validate(str)
	if !valid {
		return "", err
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
	default:
		return "", fmt.Errorf("unknown algorithm prefix %q", algPrefix)
	}

	dgst := digest.NewDigestFromHex(alg.String(), str[2:])
	if err := dgst.Validate(); err != nil {
		return "", err
	}

	return dgst, nil
}

func (d Digest) validate(str string) (bool, error) {
	if len(str) == 0 {
		return false, nil
	}
	if len(str) < 2 {
		return false, errors.New("invalid digest length")
	}
	return true, nil
}

// HexDecode decodes binary data from a textual representation.
// The output is equivalent to the PostgreSQL binary `decode` function with the hex textual format. See
// https://www.postgresql.org/docs/14/functions-binarystring.html.
func (d Digest) HexDecode() string {
	return fmt.Sprintf("\\x%s", d.String())
}

func (d Digest) Validate() string {
	return fmt.Sprintf("\\x%s", d.String())
}
