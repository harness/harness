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

package keyssh

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/pem"
	"fmt"
	"hash"

	"github.com/harness/gitness/app/services/keyfetcher"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

const SignatureType = "SSH SIGNATURE"

// signatureBlob describes a lightweight SSH signature format.
// https://github.com/openssh/openssh-portable/blob/V_9_9_P2/PROTOCOL.sshsig#L34
type signatureBlob struct {
	MagicPreamble [6]byte
	Version       uint32
	PublicKey     []byte
	Namespace     string
	Reserved      string
	HashAlgorithm string
	Signature     []byte
}

// messageWrapper represents SSH signed data.
// https://github.com/openssh/openssh-portable/blob/V_9_9_P2/PROTOCOL.sshsig#L81
type messageWrapper struct {
	Namespace     string
	Reserved      string
	HashAlgorithm string
	Hash          []byte
}

// hashFunc returns hash function used for SSH signature verification.
// Data to be signed is first hashed with the specified hash_algorithm.
// This is done to limit the amount of data presented to the signature
// operation, which may be of concern if the signing key is held in limited
// or slow hardware or on a remote ssh-agent. The supported hash algorithms
// are "sha256" and "sha512".
// https://github.com/openssh/openssh-portable/blob/V_9_9_P2/PROTOCOL.sshsig#L63
func hashFunc(hashAlgorithm string) hash.Hash {
	switch hashAlgorithm {
	case "sha256":
		return sha256.New()
	case "sha512":
		return sha512.New()
	}

	return nil
}

const (
	sshMagicPreamble = "SSHSIG"
	sshNamespace     = "git"
)

type Verify struct {
	hashAlgorithm  string
	signatureBytes []byte
	publicKey      []byte
	keyFingerprint string
}

// Parse parses the provided ASCII-armored signature and returns fingerprint of the key used to sign it.
// Also, it updates the internal object fields required for key validation.
func (v *Verify) Parse(
	ctx context.Context,
	signature []byte,
	objectSHA sha.SHA,
) enum.GitSignatureResult {
	block, _ := pem.Decode(signature)
	if block == nil {
		log.Ctx(ctx).Warn().
			Str("object_sha", objectSHA.String()).
			Msg("failed to decode signature")
		return enum.GitSignatureInvalid
	}
	if block.Type != SignatureType {
		log.Ctx(ctx).Warn().
			Str("signature_type", block.Type).
			Str("object_sha", objectSHA.String()).
			Msg("unexpected SSH signature block type")
		return enum.GitSignatureInvalid
	}

	var blob signatureBlob
	if err := ssh.Unmarshal(block.Bytes, &blob); err != nil {
		log.Ctx(ctx).Warn().
			Err(err).
			Str("object_sha", objectSHA.String()).
			Msg("failed to unmarshal SSH signature")
		return enum.GitSignatureInvalid
	}

	// The preamble is the six-byte sequence "SSHSIG". It is included to
	// ensure that manual signatures can never be confused with any message
	// signed during SSH user or host authentication.
	// https://github.com/openssh/openssh-portable/blob/V_9_9_P2/PROTOCOL.sshsig#L89
	if !bytes.Equal(blob.MagicPreamble[:], []byte(sshMagicPreamble)) {
		log.Ctx(ctx).Warn().
			Str("object_sha", objectSHA.String()).
			Msg("invalid SSH signature magic preamble")
		return enum.GitSignatureInvalid
	}

	// Verifiers MUST reject signatures with versions greater than those they support.
	// https://github.com/openssh/openssh-portable/blob/V_9_9_P2/PROTOCOL.sshsig#L50
	if blob.Version > 1 {
		return enum.GitSignatureUnsupported
	}

	// The purpose of the namespace value is to specify a unambiguous
	// interpretation domain for the signature, e.g. file signing.
	// This prevents cross-protocol attacks caused by signatures
	// intended for one intended domain being accepted in another.
	// https://github.com/openssh/openssh-portable/blob/V_9_9_P2/PROTOCOL.sshsig#L53
	if blob.Namespace != sshNamespace {
		log.Ctx(ctx).Warn().
			Str("namespace", blob.Namespace).
			Str("object_sha", objectSHA.String()).
			Msg("SSH signature namespace mismatch")
		return enum.GitSignatureInvalid
	}

	publicKey, err := ssh.ParsePublicKey(blob.PublicKey)
	if err != nil {
		log.Ctx(ctx).Warn().
			Err(err).
			Str("object_sha", objectSHA.String()).
			Msg("invalid SSH signature public key")
		return enum.GitSignatureInvalid
	}

	v.hashAlgorithm = blob.HashAlgorithm
	v.signatureBytes = blob.Signature
	v.publicKey = blob.PublicKey
	v.keyFingerprint = ssh.FingerprintSHA256(publicKey)

	return ""
}

func (v *Verify) Key(
	ctx context.Context,
	keyFetcher keyfetcher.Service,
	principalID int64,
) (*types.PublicKey, error) {
	schemes := []enum.PublicKeyScheme{enum.PublicKeySchemeSSH}
	usages := []enum.PublicKeyUsage{enum.PublicKeyUsageSign}
	keys, err := keyFetcher.FetchByFingerprint(ctx, v.KeyFingerprint(), principalID, usages, schemes)
	if err != nil {
		return nil, fmt.Errorf("failed to list SSH public keys by fingerprint: %w", err)
	}

	if len(keys) == 0 {
		//nolint:nilnil
		return nil, nil // No key is available and there is no error.
	}

	return &keys[0], nil
}

func (v *Verify) Verify(
	ctx context.Context,
	publicKeyRaw []byte,
	signedContent []byte,
	objectSHA sha.SHA,
	_ types.Signature,
) enum.GitSignatureResult {
	publicKey, _, _, _, err := ssh.ParseAuthorizedKey(publicKeyRaw)
	if err != nil {
		log.Ctx(ctx).Warn().
			Err(err).
			Str("object_sha", objectSHA.String()).
			Msg("failed to parse SSH key")
		return enum.GitSignatureUnverified
	}

	hashAlgorithm := v.hashAlgorithm
	signatureBytes := v.signatureBytes

	h := hashFunc(hashAlgorithm)
	if h == nil {
		log.Ctx(ctx).Warn().
			Str("hash_algorithm", v.hashAlgorithm).
			Str("object_sha", objectSHA.String()).
			Msg("unrecognized SSH signature algorithm")
		return enum.GitSignatureInvalid
	}
	h.Write(signedContent)

	hashSum := h.Sum(nil)

	sig := ssh.Signature{}
	if err := ssh.Unmarshal(signatureBytes, &sig); err != nil {
		log.Ctx(ctx).Warn().
			Err(err).
			Str("object_sha", objectSHA.String()).
			Msg("failed to unmarshal SSH signature")
		return enum.GitSignatureInvalid
	}

	signedMessage := ssh.Marshal(messageWrapper{
		Namespace:     sshNamespace,
		HashAlgorithm: hashAlgorithm,
		Hash:          hashSum,
	})
	buf := bytes.NewBuffer(nil)
	_, _ = buf.WriteString(sshMagicPreamble)
	_, _ = buf.Write(signedMessage)
	err = publicKey.Verify(buf.Bytes(), &sig)
	if err != nil {
		return enum.GitSignatureBad
	}

	return enum.GitSignatureGood
}

func (v *Verify) KeyScheme() enum.PublicKeyScheme {
	return enum.PublicKeySchemeSSH
}

func (v *Verify) KeyID() string {
	return ""
}

func (v *Verify) KeyFingerprint() string {
	return v.keyFingerprint
}

func (v *Verify) SignaturePublicKey() []byte {
	return v.publicKey
}
