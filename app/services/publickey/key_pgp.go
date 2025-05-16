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

package publickey

import (
	"bytes"
	"encoding/hex"
	"io"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"golang.org/x/exp/slices"
)

func parsePGP(r io.Reader) (pgpKeyInfo, error) {
	keyRing, err := openpgp.ReadArmoredKeyRing(r)
	if err != nil {
		return pgpKeyInfo{}, errors.InvalidArgument("failed to read PGP key ring: %s", err.Error())
	}

	if len(keyRing) == 0 {
		return pgpKeyInfo{}, errors.InvalidArgument("PGP key ring contains no keys")
	}

	if len(keyRing) > 1 {
		return pgpKeyInfo{}, errors.InvalidArgument("can't accept a PGP key ring with multiple primary keys")
	}

	if keyRing[0] == nil {
		return pgpKeyInfo{}, errors.InvalidArgument("PGP key ring entity is nil")
	}

	key := *keyRing[0]

	identity, err := firstIdentity(key)
	if err != nil {
		return pgpKeyInfo{}, err
	}

	return pgpKeyInfo{Entity: key, Identity: *identity}, nil
}

type pgpKeyInfo struct {
	Entity   openpgp.Entity
	Identity types.Identity
}

func (key pgpKeyInfo) Matches(s string) bool {
	otherKey, err := parsePGP(strings.NewReader(s))
	if err != nil {
		return false
	}

	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	_ = key.Entity.PrimaryKey.Serialize(buf1)
	_ = otherKey.Entity.PrimaryKey.Serialize(buf2)

	return slices.Equal(buf1.Bytes(), buf2.Bytes()) && key.Identity == otherKey.Identity
}

func (key pgpKeyInfo) Fingerprint() string {
	sb := strings.Builder{}

	sb.WriteString(hex.EncodeToString(key.Entity.PrimaryKey.Fingerprint))

	if len(key.Entity.Subkeys) == 0 {
		return sb.String()
	}

	sb.WriteByte(':')
	for i, subkey := range key.Entity.Subkeys {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(hex.EncodeToString(subkey.PublicKey.Fingerprint))
	}

	return sb.String()
}

func (key pgpKeyInfo) Type() string {
	return pgpAlgo(key.Entity.PrimaryKey.PubKeyAlgo)
}

func (key pgpKeyInfo) Scheme() enum.PublicKeyScheme {
	return enum.PublicKeySchemePGP
}

func firstIdentity(entity openpgp.Entity) (*types.Identity, error) {
	if len(entity.Identities) > 1 {
		return nil, errors.InvalidArgument("primary key must contain only one identity")
	}

	for _, ident := range entity.Identities {
		if ident == nil || ident.UserId == nil {
			continue
		}
		return &types.Identity{
			Name:  ident.UserId.Name,
			Email: ident.UserId.Email,
		}, nil
	}

	return nil, errors.InvalidArgument("primary key does not contain any identities")
}

func pgpAlgo(algorithm packet.PublicKeyAlgorithm) string {
	switch algorithm {
	case packet.PubKeyAlgoRSA, packet.PubKeyAlgoRSASignOnly, packet.PubKeyAlgoRSAEncryptOnly:
		return "RSA"
	case packet.PubKeyAlgoElGamal:
		return "ElGamal"
	case packet.PubKeyAlgoDSA:
		return "DSA"
	case packet.PubKeyAlgoECDH:
		return "ECDH"
	case packet.PubKeyAlgoECDSA:
		return "ECDSA"
	case packet.PubKeyAlgoEdDSA:
		return "EdDSA"
	case packet.PubKeyAlgoX25519:
		return "X25519"
	case packet.PubKeyAlgoX448:
		return "X448"
	case packet.PubKeyAlgoEd25519:
		return "Ed25519"
	case packet.PubKeyAlgoEd448:
		return "Ed448"
	}

	return ""
}
