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

package keypgp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/harness/gitness/app/services/publickey/validity"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"golang.org/x/exp/slices"
)

type KeyMetadata struct {
	// ID of the key.
	ID string `json:"id"`

	// Fingerprint is a hash of the key data.
	Fingerprint string `json:"fingerprint"`

	// RevocationReason is there only for information and displaying purposes.
	// To determine if the key can be used or not ValidFrom and ValidTo should be used.
	RevocationReason *enum.RevocationReason `json:"revocation_reason,omitempty"`

	// ValidFrom is timestamp (unix time millis) from which the key can be used.
	ValidFrom int64 `json:"valid_from,omitempty"`

	// ValidTo is timestamp (unix time millis) until the key can be used.
	// After this, the key should be considered expired or revoked (if RevocationReason has a value).
	// Nil value means that the key doesn't expire.
	ValidTo *int64 `json:"valid_to,omitempty"`

	Algorithm string `json:"algorithm"`
	BitLength uint16 `json:"bit_length"`
}

type EntityMetadata struct {
	PrimaryIdentity *types.Identity  `json:"primary_identity,omitempty"`
	Identities      []types.Identity `json:"identities,omitempty"`
	PrimaryKey      KeyMetadata      `json:"primary_key"`
	SubKeys         []KeyMetadata    `json:"sub_keys,omitempty"`
}

func Parse(r io.Reader, principal *types.Principal) (KeyInfo, error) {
	keyRing, err := openpgp.ReadArmoredKeyRing(r)
	if err != nil {
		return KeyInfo{}, errors.InvalidArgumentf("failed to read PGP key ring: %s", err.Error())
	}

	if len(keyRing) == 0 {
		return KeyInfo{}, errors.InvalidArgument("PGP key ring contains no keys")
	}

	if len(keyRing) > 1 {
		return KeyInfo{}, errors.InvalidArgument("can't accept a PGP key ring with multiple primary keys")
	}

	keyEntity := keyRing[0]

	if keyEntity == nil || keyEntity.PrimaryKey == nil {
		// Should not happen.
		return KeyInfo{}, errors.InvalidArgument("PGP key ring entity is nil")
	}

	if keyEntity.PrivateKey != nil {
		return KeyInfo{}, errors.InvalidArgument("refusing to accept private key: please upload a public key")
	}

	primarySignature, primaryIdentity := keyEntity.PrimarySelfSignature()

	if primarySignature == nil {
		// Should not happen.
		return KeyInfo{}, errors.InvalidArgument("PGP key entity is missing primary signature")
	}

	// Extract the validity period from the key's primary signature.
	validityKey := validity.FromPublicKey(keyEntity.PrimaryKey, primarySignature)
	validityKey.Revoke(keyEntity.Revocations)

	var identity *types.Identity
	var comment string

	// If principal is nil, it means that no particular principal is needed.
	// By `foundPrincipal = true` we declare that we have "found" it.
	foundPrincipal := principal == nil

	// Process the primary identity (name and email address for the key) if it exists.
	// The identity can also have revocations. We ignore the revocation reason, but honor
	// the validity period. The final validity period for the key is intersection between
	// the validity period of the key and the validity period of the identity.
	if primaryIdentity != nil {
		validityIdent := validity.FromSignature(primarySignature)
		validityIdent.Revoke(primaryIdentity.Revocations)
		validityKey.Intersect(validityIdent)

		identity = &types.Identity{
			Name:  primaryIdentity.UserId.Name,
			Email: primaryIdentity.UserId.Email,
		}

		foundPrincipal = foundPrincipal || strings.EqualFold(identity.Email, principal.Email)

		comment = primaryIdentity.UserId.Comment
	}

	var identities []types.Identity
	for _, ident := range keyEntity.Identities {
		identities = append(identities, types.Identity{
			Name:  ident.UserId.Name,
			Email: ident.UserId.Email,
		})

		foundPrincipal = foundPrincipal || strings.EqualFold(ident.UserId.Email, principal.Email)
	}

	// PGP keys can have multiple identities and one of those must match the current user's.
	// The email address must match, the name can be different.
	if !foundPrincipal {
		return KeyInfo{}, errors.InvalidArgument("key identities don't contain the user's email address")
	}

	var subKeys []KeyMetadata
	for _, subKey := range keyEntity.Subkeys {
		if subKey.PublicKey == nil || subKey.Sig == nil {
			return KeyInfo{}, errors.InvalidArgument("found a subkey without public key")
		}

		// We'll only consider keys than can be used for signing
		if !subKey.PublicKey.CanSign() {
			continue
		}

		validitySubkey := validity.FromSignature(subKey.Sig)
		validitySubkey.Revoke(subKey.Revocations)
		validitySubkey.Intersect(validityKey)

		subKeyValidFrom, subKeyValidTo := validitySubkey.Milliseconds()
		bits, _ := subKey.PublicKey.BitLength()
		subKeys = append(subKeys, KeyMetadata{
			ID:               subKey.PublicKey.KeyIdString(),
			Fingerprint:      fmt.Sprintf("%X", subKey.PublicKey.Fingerprint),
			RevocationReason: getRevocationReason(subKey.Revocations),
			ValidFrom:        subKeyValidFrom,
			ValidTo:          subKeyValidTo,
			Algorithm:        pgpAlgo(subKey.PublicKey.PubKeyAlgo),
			BitLength:        bits,
		})
	}

	keyValidFrom, keyValidTo := validityKey.Milliseconds()
	bits, _ := keyEntity.PrimaryKey.BitLength()
	metadata := EntityMetadata{
		PrimaryIdentity: identity,
		Identities:      identities,
		PrimaryKey: KeyMetadata{
			ID:               keyEntity.PrimaryKey.KeyIdString(),
			Fingerprint:      fmt.Sprintf("%X", keyEntity.PrimaryKey.Fingerprint),
			RevocationReason: getRevocationReason(keyEntity.Revocations),
			ValidFrom:        keyValidFrom,
			ValidTo:          keyValidTo,
			Algorithm:        pgpAlgo(keyEntity.PrimaryKey.PubKeyAlgo),
			BitLength:        bits,
		},
		SubKeys: subKeys,
	}

	keyInfo := KeyInfo{
		entity:    keyEntity,
		metadata:  metadata,
		validFrom: keyValidFrom,
		validTo:   keyValidTo,
		comment:   comment,
	}

	return keyInfo, nil
}

type KeyInfo struct {
	// entity holds the original PGP key
	entity *openpgp.Entity

	// metadata holds additional key info
	metadata EntityMetadata

	validFrom int64
	validTo   *int64
	comment   string
}

func (key KeyInfo) Matches(s string) bool {
	otherKey, err := Parse(strings.NewReader(s), nil)
	if err != nil {
		return false
	}

	if key.entity.PrimaryKey.KeyId != otherKey.entity.PrimaryKey.KeyId {
		return false
	}

	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	_ = key.entity.Serialize(buf1)
	_ = otherKey.entity.Serialize(buf2)

	return slices.Equal(buf1.Bytes(), buf2.Bytes())
}

func (key KeyInfo) Fingerprint() string {
	return key.metadata.PrimaryKey.Fingerprint
}

func (key KeyInfo) Type() string {
	return pgpAlgo(key.entity.PrimaryKey.PubKeyAlgo)
}

func (key KeyInfo) Scheme() enum.PublicKeyScheme {
	return enum.PublicKeySchemePGP
}

func (key KeyInfo) Comment() string {
	return key.comment
}

func (key KeyInfo) ValidFrom() *int64 {
	return &key.validFrom
}

func (key KeyInfo) ValidTo() *int64 {
	return key.validTo
}

func (key KeyInfo) Identities() []types.Identity {
	return key.metadata.Identities
}

func (key KeyInfo) RevocationReason() *enum.RevocationReason {
	return key.metadata.PrimaryKey.RevocationReason
}

func (key KeyInfo) Metadata() json.RawMessage {
	data, _ := json.Marshal(key.metadata)
	return data
}

func (key KeyInfo) KeyIDs() []string {
	subKeyIDs := make([]string, 0)
	subKeyIDs = append(subKeyIDs, key.entity.PrimaryKey.KeyIdString())
	for i := range key.entity.Subkeys {
		if key.entity.Subkeys[i].PublicKey.CanSign() {
			subKeyIDs = append(subKeyIDs, key.entity.Subkeys[i].PublicKey.KeyIdString())
		}
	}
	return subKeyIDs
}

func (key KeyInfo) CompromisedIDs() []string {
	var revokedIDs []string
	var primaryRevoked bool

	revocationReason := getRevocationReason(key.entity.Revocations)
	if revocationReason != nil && *revocationReason == enum.RevocationReasonCompromised {
		revokedIDs = append(revokedIDs, key.entity.PrimaryKey.KeyIdString())
		primaryRevoked = true
	}

	for i := range key.entity.Subkeys {
		if !key.entity.Subkeys[i].PublicKey.CanSign() {
			continue
		}

		if primaryRevoked {
			revokedIDs = append(revokedIDs, key.entity.Subkeys[i].PublicKey.KeyIdString())
			continue
		}

		revocationReason = getRevocationReason(key.entity.Subkeys[i].Revocations)
		if revocationReason != nil && *revocationReason == enum.RevocationReasonCompromised {
			revokedIDs = append(revokedIDs, key.entity.Subkeys[i].PublicKey.KeyIdString())
		}
	}

	return revokedIDs
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

func getRevocationReason(revocations []*packet.Signature) *enum.RevocationReason {
	if len(revocations) == 0 {
		return nil
	}

	reason := enum.RevocationReasonUnknown
	for _, revocation := range revocations {
		if revocation == nil || revocation.RevocationReason == nil {
			continue
		}

		if *revocation.RevocationReason == packet.KeyCompromised {
			reason = enum.RevocationReasonCompromised
			return &reason
		}

		if *revocation.RevocationReason == packet.KeyRetired {
			reason = enum.RevocationReasonRetired
			continue
		}

		if *revocation.RevocationReason == packet.KeySuperseded {
			reason = enum.RevocationReasonSuperseded
			continue
		}
	}

	return &reason
}
