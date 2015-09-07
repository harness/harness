/*-
 * Copyright 2014 Square Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jose

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
)

// Signer represents a signer which takes a payload and produces a signed JWS object.
type Signer interface {
	Sign(payload []byte) (*JsonWebSignature, error)
}

// MultiSigner represents a signer which supports multiple recipients.
type MultiSigner interface {
	Sign(payload []byte) (*JsonWebSignature, error)
	AddRecipient(alg SignatureAlgorithm, signingKey interface{}) error
}

type payloadSigner interface {
	signPayload(payload []byte, alg SignatureAlgorithm) (Signature, error)
}

type payloadVerifier interface {
	verifyPayload(payload []byte, signature []byte, alg SignatureAlgorithm) error
}

type genericSigner struct {
	recipients []recipientSigInfo
}

type recipientSigInfo struct {
	sigAlg    SignatureAlgorithm
	publicKey *JsonWebKey
	signer    payloadSigner
}

// NewSigner creates an appropriate signer based on the key type
func NewSigner(alg SignatureAlgorithm, signingKey interface{}) (Signer, error) {
	// NewMultiSigner never fails (currently)
	signer := NewMultiSigner()

	err := signer.AddRecipient(alg, signingKey)
	if err != nil {
		return nil, err
	}

	return signer, nil
}

// NewMultiSigner creates a signer for multiple recipients
func NewMultiSigner() MultiSigner {
	return &genericSigner{
		recipients: []recipientSigInfo{},
	}
}

// newVerifier creates a verifier based on the key type
func newVerifier(verificationKey interface{}) (payloadVerifier, error) {
	switch verificationKey := verificationKey.(type) {
	case *rsa.PublicKey:
		return &rsaEncrypterVerifier{
			publicKey: verificationKey,
		}, nil
	case *ecdsa.PublicKey:
		return &ecEncrypterVerifier{
			publicKey: verificationKey,
		}, nil
	case []byte:
		return &symmetricMac{
			key: verificationKey,
		}, nil
	case *JsonWebKey:
		return newVerifier(verificationKey.Key)
	default:
		return nil, ErrUnsupportedKeyType
	}
}

func (ctx *genericSigner) AddRecipient(alg SignatureAlgorithm, signingKey interface{}) error {
	recipient, err := makeRecipient(alg, signingKey)
	if err != nil {
		return err
	}

	ctx.recipients = append(ctx.recipients, recipient)
	return nil
}

func makeRecipient(alg SignatureAlgorithm, signingKey interface{}) (recipientSigInfo, error) {
	switch signingKey := signingKey.(type) {
	case *rsa.PrivateKey:
		return newRSASigner(alg, signingKey)
	case *ecdsa.PrivateKey:
		return newECDSASigner(alg, signingKey)
	case []byte:
		return newSymmetricSigner(alg, signingKey)
	case *JsonWebKey:
		recipient, err := makeRecipient(alg, signingKey.Key)
		if err != nil {
			return recipientSigInfo{}, err
		}
		recipient.publicKey.KeyID = signingKey.KeyID
		return recipient, nil
	default:
		return recipientSigInfo{}, ErrUnsupportedKeyType
	}
}

func (ctx *genericSigner) Sign(payload []byte) (*JsonWebSignature, error) {
	obj := &JsonWebSignature{}
	obj.payload = payload
	obj.Signatures = make([]Signature, len(ctx.recipients))

	for i, recipient := range ctx.recipients {
		protected := &rawHeader{
			Alg: string(recipient.sigAlg),
		}

		if recipient.publicKey != nil {
			protected.Jwk = recipient.publicKey
			protected.Kid = recipient.publicKey.KeyID
		}

		serializedProtected := mustSerializeJSON(protected)

		input := []byte(fmt.Sprintf("%s.%s",
			base64URLEncode(serializedProtected),
			base64URLEncode(payload)))

		signatureInfo, err := recipient.signer.signPayload(input, recipient.sigAlg)
		if err != nil {
			return nil, err
		}

		signatureInfo.protected = protected
		obj.Signatures[i] = signatureInfo
	}

	return obj, nil
}

// Verify validates the signature on the object and returns the payload.
func (obj JsonWebSignature) Verify(verificationKey interface{}) ([]byte, error) {
	verifier, err := newVerifier(verificationKey)
	if err != nil {
		return nil, err
	}

	for _, signature := range obj.Signatures {
		headers := signature.mergedHeaders()
		if len(headers.Crit) > 0 {
			// Unsupported crit header
			continue
		}

		input := obj.computeAuthData(&signature)
		alg := SignatureAlgorithm(headers.Alg)
		err := verifier.verifyPayload(input, signature.signature, alg)
		if err == nil {
			return obj.payload, nil
		}
	}

	return nil, ErrCryptoFailure
}
