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
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"testing"
)

func RoundtripJWS(sigAlg SignatureAlgorithm, serializer func(*JsonWebSignature) (string, error), corrupter func(*JsonWebSignature), signingKey interface{}, verificationKey interface{}) error {
	signer, err := NewSigner(sigAlg, signingKey)
	if err != nil {
		return fmt.Errorf("error on new signer: %s", err)
	}

	input := []byte("Lorem ipsum dolor sit amet")
	obj, err := signer.Sign(input)
	if err != nil {
		return fmt.Errorf("error on sign: %s", err)
	}

	msg, err := serializer(obj)
	if err != nil {
		return fmt.Errorf("error on serialize: %s", err)
	}

	obj, err = ParseSigned(msg)
	if err != nil {
		return fmt.Errorf("error on parse: %s", err)
	}

	// (Maybe) mangle the object
	corrupter(obj)

	output, err := obj.Verify(verificationKey)
	if err != nil {
		return fmt.Errorf("error on verify: %s", err)
	}

	// Check that verify works with embedded keys (if present)
	for i, sig := range obj.Signatures {
		if sig.Header.JsonWebKey != nil {
			_, err = obj.Verify(sig.Header.JsonWebKey)
			if err != nil {
				return fmt.Errorf("error on verify with embedded key %d: %s", i, err)
			}
		}
	}

	if bytes.Compare(output, input) != 0 {
		return fmt.Errorf("input/output do not match, got '%s', expected '%s'", output, input)
	}

	return nil
}

func TestRoundtripsJWS(t *testing.T) {
	// Test matrix
	sigAlgs := []SignatureAlgorithm{RS256, RS384, RS512, PS256, PS384, PS512, HS256, HS384, HS512, ES256, ES384, ES512}

	serializers := []func(*JsonWebSignature) (string, error){
		func(obj *JsonWebSignature) (string, error) { return obj.CompactSerialize() },
		func(obj *JsonWebSignature) (string, error) { return obj.FullSerialize(), nil },
	}

	corrupter := func(obj *JsonWebSignature) {}

	for _, alg := range sigAlgs {
		signingKey, verificationKey := GenerateSigningTestKey(alg)

		for i, serializer := range serializers {
			err := RoundtripJWS(alg, serializer, corrupter, signingKey, verificationKey)
			if err != nil {
				t.Error(err, alg, i)
			}
		}
	}
}

func TestRoundtripsJWSCorruptSignature(t *testing.T) {
	// Test matrix
	sigAlgs := []SignatureAlgorithm{RS256, RS384, RS512, PS256, PS384, PS512, HS256, HS384, HS512, ES256, ES384, ES512}

	serializers := []func(*JsonWebSignature) (string, error){
		func(obj *JsonWebSignature) (string, error) { return obj.CompactSerialize() },
		func(obj *JsonWebSignature) (string, error) { return obj.FullSerialize(), nil },
	}

	corrupters := []func(*JsonWebSignature){
		func(obj *JsonWebSignature) {
			// Changes bytes in signature
			obj.Signatures[0].signature[10]++
		},
		func(obj *JsonWebSignature) {
			// Set totally invalid signature
			obj.Signatures[0].signature = []byte("###")
		},
	}

	// Test all different configurations
	for _, alg := range sigAlgs {
		signingKey, verificationKey := GenerateSigningTestKey(alg)

		for i, serializer := range serializers {
			for j, corrupter := range corrupters {
				err := RoundtripJWS(alg, serializer, corrupter, signingKey, verificationKey)
				if err == nil {
					t.Error("failed to detect corrupt signature", err, alg, i, j)
				}
			}
		}
	}
}

func TestSignerWithBrokenRand(t *testing.T) {
	sigAlgs := []SignatureAlgorithm{RS256, RS384, RS512, PS256, PS384, PS512}

	serializer := func(obj *JsonWebSignature) (string, error) { return obj.CompactSerialize() }
	corrupter := func(obj *JsonWebSignature) {}

	// Break rand reader
	readers := []func() io.Reader{
		// Totally broken
		func() io.Reader { return bytes.NewReader([]byte{}) },
		// Not enough bytes
		func() io.Reader { return io.LimitReader(rand.Reader, 20) },
	}

	defer resetRandReader()

	for _, alg := range sigAlgs {
		signingKey, verificationKey := GenerateSigningTestKey(alg)
		for i, getReader := range readers {
			randReader = getReader()
			err := RoundtripJWS(alg, serializer, corrupter, signingKey, verificationKey)
			if err == nil {
				t.Error("signer should fail if rand is broken", alg, i)
			}
		}
	}
}

func TestJWSInvalidKey(t *testing.T) {
	signingKey0, verificationKey0 := GenerateSigningTestKey(RS256)
	_, verificationKey1 := GenerateSigningTestKey(ES256)

	signer, err := NewSigner(RS256, signingKey0)
	if err != nil {
		panic(err)
	}

	input := []byte("Lorem ipsum dolor sit amet")
	obj, err := signer.Sign(input)
	if err != nil {
		panic(err)
	}

	// Must work with correct key
	_, err = obj.Verify(verificationKey0)
	if err != nil {
		t.Error("error on verify", err)
	}

	// Must not work with incorrect key
	_, err = obj.Verify(verificationKey1)
	if err == nil {
		t.Error("verification should fail with incorrect key")
	}

	// Must not work with invalid key
	_, err = obj.Verify("")
	if err == nil {
		t.Error("verification should fail with incorrect key")
	}
}

func TestMultiRecipientJWS(t *testing.T) {
	signer := NewMultiSigner()

	sharedKey := []byte{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
	}

	signer.AddRecipient(RS256, rsaTestKey)
	signer.AddRecipient(HS384, sharedKey)

	input := []byte("Lorem ipsum dolor sit amet")
	obj, err := signer.Sign(input)
	if err != nil {
		t.Error("error on sign: ", err)
		return
	}

	_, err = obj.CompactSerialize()
	if err == nil {
		t.Error("message with multiple recipient was compact serialized")
	}

	msg := obj.FullSerialize()

	obj, err = ParseSigned(msg)
	if err != nil {
		t.Error("error on parse: ", err)
		return
	}

	output, err := obj.Verify(&rsaTestKey.PublicKey)
	if err != nil {
		t.Error("error on verify: ", err)
		return
	}

	if bytes.Compare(output, input) != 0 {
		t.Error("input/output do not match", output, input)
		return
	}

	output, err = obj.Verify(sharedKey)
	if err != nil {
		t.Error("error on verify: ", err)
		return
	}

	if bytes.Compare(output, input) != 0 {
		t.Error("input/output do not match", output, input)
		return
	}
}

func GenerateSigningTestKey(sigAlg SignatureAlgorithm) (sig, ver interface{}) {
	switch sigAlg {
	case RS256, RS384, RS512, PS256, PS384, PS512:
		sig = rsaTestKey
		ver = &rsaTestKey.PublicKey
	case HS256, HS384, HS512:
		sig, _, _ = randomKeyGenerator{size: 16}.genKey()
		ver = sig
	case ES256:
		key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		sig = key
		ver = &key.PublicKey
	case ES384:
		key, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		sig = key
		ver = &key.PublicKey
	case ES512:
		key, _ := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		sig = key
		ver = &key.PublicKey
	default:
		panic("Must update test case")
	}

	return
}

func TestInvalidSignerAlg(t *testing.T) {
	_, err := NewSigner("XYZ", nil)
	if err == nil {
		t.Error("should not accept invalid algorithm")
	}

	_, err = NewSigner("XYZ", []byte{})
	if err == nil {
		t.Error("should not accept invalid algorithm")
	}
}

func TestInvalidJWS(t *testing.T) {
	signer, err := NewSigner(PS256, rsaTestKey)
	if err != nil {
		panic(err)
	}

	obj, err := signer.Sign([]byte("Lorem ipsum dolor sit amet"))
	obj.Signatures[0].header = &rawHeader{
		Crit: []string{"TEST"},
	}

	_, err = obj.Verify(&rsaTestKey.PublicKey)
	if err == nil {
		t.Error("should not verify message with unknown crit header")
	}

	// Try without alg header
	obj.Signatures[0].protected = &rawHeader{}
	obj.Signatures[0].header = &rawHeader{}

	_, err = obj.Verify(&rsaTestKey.PublicKey)
	if err == nil {
		t.Error("should not verify message with missing headers")
	}
}

func TestSignerKid(t *testing.T) {
	kid := "DEADBEEF"
	payload := []byte("Lorem ipsum dolor sit amet")

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Error("problem generating test signing key", err)
	}

	basejwk := JsonWebKey{Key: key}
	jsonbar, err := basejwk.MarshalJSON()
	if err != nil {
		t.Error("problem marshalling base JWK", err)
	}

	var jsonmsi map[string]interface{}
	err = json.Unmarshal(jsonbar, &jsonmsi)
	if err != nil {
		t.Error("problem unmarshalling base JWK", err)
	}
	jsonmsi["kid"] = kid
	jsonbar2, err := json.Marshal(jsonmsi)
	if err != nil {
		t.Error("problem marshalling kided JWK", err)
	}

	var jwk JsonWebKey
	err = jwk.UnmarshalJSON(jsonbar2)
	if err != nil {
		t.Error("problem unmarshalling kided JWK", err)
	}

	signer, err := NewSigner(ES256, &jwk)
	if err != nil {
		t.Error("problem creating signer", err)
	}
	signed, err := signer.Sign(payload)

	serialized := signed.FullSerialize()

	parsed, err := ParseSigned(serialized)
	if err != nil {
		t.Error("problem parsing signed object", err)
	}

	if parsed.Signatures[0].Header.KeyID != kid {
		t.Error("KeyID did not survive trip")
	}
}
