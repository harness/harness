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

package enum

type PublicKeyScheme string

const (
	PublicKeySchemeSSH PublicKeyScheme = "ssh"
	PublicKeySchemePGP PublicKeyScheme = "pgp"
)

var publicKeySchemes = sortEnum([]PublicKeyScheme{
	PublicKeySchemeSSH, PublicKeySchemePGP,
})

func (PublicKeyScheme) Enum() []interface{} { return toInterfaceSlice(publicKeySchemes) }
func (s PublicKeyScheme) Sanitize() (PublicKeyScheme, bool) {
	return Sanitize(s, GetAllPublicKeySchemes)
}
func GetAllPublicKeySchemes() ([]PublicKeyScheme, PublicKeyScheme) {
	return publicKeySchemes, PublicKeySchemeSSH
}

// PublicKeyUsage represents usage type of public key.
type PublicKeyUsage string

// PublicKeyUsage enumeration.
const (
	PublicKeyUsageAuth     PublicKeyUsage = "auth"
	PublicKeyUsageSign     PublicKeyUsage = "sign"
	PublicKeyUsageAuthSign PublicKeyUsage = "auth_or_sign"
)

var publicKeyTypes = sortEnum([]PublicKeyUsage{
	PublicKeyUsageAuth, PublicKeyUsageSign, PublicKeyUsageAuthSign,
})

func (PublicKeyUsage) Enum() []interface{} { return toInterfaceSlice(publicKeyTypes) }
func (s PublicKeyUsage) Sanitize() (PublicKeyUsage, bool) {
	return Sanitize(s, GetAllPublicKeyUsages)
}
func GetAllPublicKeyUsages() ([]PublicKeyUsage, PublicKeyUsage) {
	return publicKeyTypes, PublicKeyUsageAuth
}

// PublicKeySort is used to specify sorting of public keys.
type PublicKeySort string

// PublicKeySort enumeration.
const (
	PublicKeySortCreated    PublicKeySort = "created"
	PublicKeySortIdentifier PublicKeySort = "identifier"
)

var publicKeySorts = sortEnum([]PublicKeySort{
	PublicKeySortCreated,
	PublicKeySortIdentifier,
})

func (PublicKeySort) Enum() []interface{}               { return toInterfaceSlice(publicKeySorts) }
func (s PublicKeySort) Sanitize() (PublicKeySort, bool) { return Sanitize(s, GetAllPublicKeySorts) }
func GetAllPublicKeySorts() ([]PublicKeySort, PublicKeySort) {
	return publicKeySorts, PublicKeySortCreated
}

// RevocationReason is the reason why a public key has been revoked.
type RevocationReason string

// RevocationReason enumeration.
const (
	RevocationReasonUnknown     RevocationReason = "unknown"
	RevocationReasonSuperseded  RevocationReason = "superseded"
	RevocationReasonRetired     RevocationReason = "retired"
	RevocationReasonCompromised RevocationReason = "compromised"
)

var revocationReasons = sortEnum([]RevocationReason{
	RevocationReasonUnknown,
	RevocationReasonSuperseded,
	RevocationReasonRetired,
	RevocationReasonCompromised,
})

func (RevocationReason) Enum() []interface{} { return toInterfaceSlice(revocationReasons) }
func (s RevocationReason) Sanitize() (RevocationReason, bool) {
	return Sanitize(s, GetAllRevocationReasons)
}
func GetAllRevocationReasons() ([]RevocationReason, RevocationReason) {
	return revocationReasons, ""
}

// GitSignatureResult is outcome of a git object's signature verification.
type GitSignatureResult string

const (
	// GitSignatureInvalid is used when the signature itself is malformed.
	// Shouldn't be stored to the DB.
	GitSignatureInvalid GitSignatureResult = "invalid"

	// GitSignatureUnsupported is used when the system is unable to verify the signature
	// because the signature version is not supported by the system.
	// Shouldn't be stored to the DB.
	GitSignatureUnsupported GitSignatureResult = "unsupported"

	// GitSignatureUnverified is used when the signer is not in the DB or
	// when the key to verify the signature is missing.
	// Shouldn't be stored to the DB.
	GitSignatureUnverified GitSignatureResult = "unverified"

	// GitSignatureGood is used when the signature is cryptographically valid for the signed object.
	GitSignatureGood GitSignatureResult = "good"

	// GitSignatureBad is used when the signature is bad (doesn’t match the object contents).
	GitSignatureBad GitSignatureResult = "bad"

	// GitSignatureKeyExpired is used when the content's timestamp is not within the key's validity period.
	GitSignatureKeyExpired GitSignatureResult = "key_expired"

	// GitSignatureRevoked is used when the signature is valid, but is signed with a revoked key.
	GitSignatureRevoked GitSignatureResult = "revoked"
)

var gitSignatureResults = sortEnum([]GitSignatureResult{
	GitSignatureInvalid,
	GitSignatureUnsupported,
	GitSignatureUnverified,
	GitSignatureGood,
	GitSignatureBad,
	GitSignatureKeyExpired,
	GitSignatureRevoked,
})

func (GitSignatureResult) Enum() []interface{} {
	return toInterfaceSlice(gitSignatureResults)
}
