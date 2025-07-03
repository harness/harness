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
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/app/services/keyfetcher"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	pgperrors "github.com/ProtonMail/go-crypto/openpgp/errors"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/rs/zerolog/log"
)

const (
	SignatureType = "PGP SIGNATURE"
)

type Verify struct {
	signature      []byte
	keyID          string
	keyFingerprint string
}

func (v *Verify) Parse(
	ctx context.Context,
	signature []byte,
	objectSHA sha.SHA,
) enum.GitSignatureResult {
	block, err := armor.Decode(bytes.NewReader(signature))
	if err != nil || block == nil {
		log.Ctx(ctx).Warn().
			Err(err).
			Str("object_sha", objectSHA.String()).
			Msg("failed to decode signature")
		return enum.GitSignatureInvalid
	}
	if block.Type != openpgp.SignatureType {
		log.Ctx(ctx).Warn().
			Str("signature_type", block.Type).
			Str("object_sha", objectSHA.String()).
			Msg("unexpected PGP signature block type")
		return enum.GitSignatureInvalid
	}

	reader := packet.NewReader(block.Body)
	sig, err := reader.Next()
	if err != nil {
		log.Ctx(ctx).Warn().
			Err(err).
			Str("object_sha", objectSHA.String()).
			Msg("failed to read PGP signature")
		return enum.GitSignatureInvalid
	}

	p, ok := sig.(*packet.Signature)
	if !ok {
		log.Ctx(ctx).Warn().
			Str("signature_type", fmt.Sprintf("%T", sig)).
			Str("object_sha", objectSHA.String()).
			Msg("signature type mismatch")
		return enum.GitSignatureInvalid
	}

	if p.IssuerKeyId == nil {
		log.Ctx(ctx).Warn().
			Str("object_sha", objectSHA.String()).
			Msg("no public key ID in PGP signature")
		return enum.GitSignatureInvalid
	}

	v.signature = signature
	v.keyID = fmt.Sprintf("%016X", *p.IssuerKeyId)
	v.keyFingerprint = fmt.Sprintf("%X", p.IssuerFingerprint)

	return ""
}

func (v *Verify) Key(
	ctx context.Context,
	keyFetcher keyfetcher.Service,
	principalID int64,
) (*types.PublicKey, error) {
	schemes := []enum.PublicKeyScheme{enum.PublicKeySchemePGP}
	usages := []enum.PublicKeyUsage{enum.PublicKeyUsageSign}
	keys, err := keyFetcher.FetchBySubKeyID(ctx, v.KeyID(), principalID, usages, schemes)
	if err != nil {
		return nil, fmt.Errorf("failed to list PGP public keys by subkey ID: %w", err)
	}

	if len(keys) == 0 {
		//nolint:nilnil
		return nil, nil // No key is available and there is no error.
	}

	return &keys[0], nil
}

func (v *Verify) Verify(
	ctx context.Context,
	armoredPublicKey []byte,
	signedContent []byte,
	objectSHA sha.SHA,
	committer types.Signature,
) enum.GitSignatureResult {
	keyRingReader := bytes.NewReader(armoredPublicKey)
	keyRing, err := openpgp.ReadArmoredKeyRing(keyRingReader)
	if err != nil {
		log.Ctx(ctx).Warn().
			Err(err).
			Str("object_sha", objectSHA.String()).
			Msg("failed to read key ring")
		return enum.GitSignatureUnverified
	}

	// CheckArmoredDetachedSignature returns an error if:
	//   - The signature (or one of the binding signatures mentioned below)
	//     has a unknown critical notation data subpacket
	//   - The primary key of the signing entity is revoked
	//   - The primary identity is revoked
	//   - The signature is expired
	//   - The primary key of the signing entity is expired according to the
	//     primary identity binding signature
	//
	// ... or, if the signature was signed by a subkey and:
	//   - The signing subkey is revoked
	//   - The signing subkey is expired according to the subkey binding signature
	//   - The signing subkey binding signature is expired
	//   - The signing subkey cross-signature is expired
	//
	// NOTE: The order of these checks is important, as the caller may choose to
	// ignore ErrSignatureExpired or ErrKeyExpired errors, but should never
	// ignore any other errors.
	// NOTE 2: The comment above is copied from the openpgp library.
	signer, err := openpgp.CheckArmoredDetachedSignature(
		keyRing,
		bytes.NewReader(signedContent),
		bytes.NewReader(v.signature),
		&packet.Config{
			Time: func() time.Time {
				return committer.When
			},
		},
	)
	// If error happened, try to convert it to one of the enum values.
	//nolint:nestif
	if err != nil {
		var errUnsupported pgperrors.UnsupportedError
		if errors.As(err, &errUnsupported); errUnsupported != "" {
			return enum.GitSignatureUnsupported
		}

		if errors.Is(err, pgperrors.ErrKeyRevoked) {
			return enum.GitSignatureRevoked
		}

		if errors.Is(err, pgperrors.ErrUnknownIssuer) {
			// This shouldn't happen because we fetched the key by ID,
			// so we are using the correct key with the correct identity.
			return enum.GitSignatureBad
		}

		if errors.Is(err, pgperrors.ErrKeyExpired) {
			return enum.GitSignatureKeyExpired
		}

		if errors.Is(err, pgperrors.ErrSignatureExpired) {
			return enum.GitSignatureBad
		}

		log.Ctx(ctx).Warn().
			Err(err).
			Str("error_type", fmt.Sprintf("%T", err)).
			Str("object_sha", objectSHA.String()).
			Msg("unrecognized error")

		return enum.GitSignatureInvalid
	}

	var signatureIdentity *openpgp.Identity
	for _, identity := range signer.Identities {
		if strings.EqualFold(committer.Identity.Email, identity.UserId.Email) {
			signatureIdentity = identity
		}
	}
	if signatureIdentity == nil {
		return enum.GitSignatureBad
	}

	if signatureIdentity.Revoked(committer.When) {
		return enum.GitSignatureRevoked
	}

	return enum.GitSignatureGood
}

func (v *Verify) KeyScheme() enum.PublicKeyScheme {
	return enum.PublicKeySchemePGP
}

func (v *Verify) KeyID() string {
	return v.keyID
}

func (v *Verify) KeyFingerprint() string {
	return v.keyFingerprint
}
