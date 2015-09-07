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
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"testing"
)

func TestCurveSize(t *testing.T) {
	size256 := curveSize(elliptic.P256())
	size384 := curveSize(elliptic.P384())
	size521 := curveSize(elliptic.P521())
	if size256 != 32 {
		t.Error("P-256 have 32 bytes")
	}
	if size384 != 48 {
		t.Error("P-384 have 48 bytes")
	}
	if size521 != 66 {
		t.Error("P-521 have 66 bytes")
	}
}

func TestRoundtripRsaPrivate(t *testing.T) {
	jwk, err := fromRsaPrivateKey(rsaTestKey)
	if err != nil {
		t.Error("problem constructing JWK from rsa key", err)
	}

	rsa2, err := jwk.rsaPrivateKey()
	if err != nil {
		t.Error("problem converting RSA private -> JWK", err)
	}

	if rsa2.N.Cmp(rsaTestKey.N) != 0 {
		t.Error("RSA private N mismatch")
	}
	if rsa2.E != rsaTestKey.E {
		t.Error("RSA private E mismatch")
	}
	if rsa2.D.Cmp(rsaTestKey.D) != 0 {
		t.Error("RSA private D mismatch")
	}
	if len(rsa2.Primes) != 2 {
		t.Error("RSA private roundtrip expected two primes")
	}
	if rsa2.Primes[0].Cmp(rsaTestKey.Primes[0]) != 0 {
		t.Error("RSA private P mismatch")
	}
	if rsa2.Primes[1].Cmp(rsaTestKey.Primes[1]) != 0 {
		t.Error("RSA private Q mismatch")
	}
}

func TestRsaPrivateInsufficientPrimes(t *testing.T) {
	brokenRsaPrivateKey := rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: rsaTestKey.N,
			E: rsaTestKey.E,
		},
		D:      rsaTestKey.D,
		Primes: []*big.Int{rsaTestKey.Primes[0]},
	}

	_, err := fromRsaPrivateKey(&brokenRsaPrivateKey)
	if err != ErrUnsupportedKeyType {
		t.Error("expected unsupported key type error, got", err)
	}
}

func TestRsaPrivateExcessPrimes(t *testing.T) {
	brokenRsaPrivateKey := rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: rsaTestKey.N,
			E: rsaTestKey.E,
		},
		D: rsaTestKey.D,
		Primes: []*big.Int{
			rsaTestKey.Primes[0],
			rsaTestKey.Primes[1],
			big.NewInt(3),
		},
	}

	_, err := fromRsaPrivateKey(&brokenRsaPrivateKey)
	if err != ErrUnsupportedKeyType {
		t.Error("expected unsupported key type error, got", err)
	}
}

func TestRoundtripEcPublic(t *testing.T) {
	for i, ecTestKey := range []*ecdsa.PrivateKey{ecTestKey256, ecTestKey384, ecTestKey521} {
		jwk, err := fromEcPublicKey(&ecTestKey.PublicKey)

		ec2, err := jwk.ecPublicKey()
		if err != nil {
			t.Error("problem converting ECDSA private -> JWK", i, err)
		}

		if !reflect.DeepEqual(ec2.Curve, ecTestKey.Curve) {
			t.Error("ECDSA private curve mismatch", i)
		}
		if ec2.X.Cmp(ecTestKey.X) != 0 {
			t.Error("ECDSA X mismatch", i)
		}
		if ec2.Y.Cmp(ecTestKey.Y) != 0 {
			t.Error("ECDSA Y mismatch", i)
		}
	}
}

func TestRoundtripEcPrivate(t *testing.T) {
	for i, ecTestKey := range []*ecdsa.PrivateKey{ecTestKey256, ecTestKey384, ecTestKey521} {
		jwk, err := fromEcPrivateKey(ecTestKey)

		ec2, err := jwk.ecPrivateKey()
		if err != nil {
			t.Error("problem converting ECDSA private -> JWK", i, err)
		}

		if !reflect.DeepEqual(ec2.Curve, ecTestKey.Curve) {
			t.Error("ECDSA private curve mismatch", i)
		}
		if ec2.X.Cmp(ecTestKey.X) != 0 {
			t.Error("ECDSA X mismatch", i)
		}
		if ec2.Y.Cmp(ecTestKey.Y) != 0 {
			t.Error("ECDSA Y mismatch", i)
		}
		if ec2.D.Cmp(ecTestKey.D) != 0 {
			t.Error("ECDSA D mismatch", i)
		}
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	kid := "DEADBEEF"

	for i, key := range []interface{}{ecTestKey256, ecTestKey384, ecTestKey521, rsaTestKey} {
		jwk := JsonWebKey{Key: key, KeyID: kid, Algorithm: "foo"}
		jsonbar, err := jwk.MarshalJSON()
		if err != nil {
			t.Error("problem marshaling", i, err)
		}

		var jwk2 JsonWebKey
		err = jwk2.UnmarshalJSON(jsonbar)
		if err != nil {
			t.Error("problem unmarshalling", i, err)
		}

		jsonbar2, err := jwk2.MarshalJSON()
		if err != nil {
			t.Error("problem marshaling", i, err)
		}

		if !bytes.Equal(jsonbar, jsonbar2) {
			t.Error("roundtrip should not lose information", i)
		}

		if jwk2.KeyID != kid {
			t.Error("kid did not roundtrip JSON marshalling", i)
		}

		if jwk2.Algorithm != "foo" {
			t.Error("alg did not roundtrip JSON marshalling", i)
		}
	}
}

func TestMarshalNonPointer(t *testing.T) {
	type EmbedsKey struct {
		Key JsonWebKey
	}

	keyJson := []byte(`{
		"e": "AQAB",
		"kty": "RSA",
		"n": "vd7rZIoTLEe-z1_8G1FcXSw9CQFEJgV4g9V277sER7yx5Qjz_Pkf2YVth6wwwFJEmzc0hoKY-MMYFNwBE4hQHw"
	}`)
	var parsedKey JsonWebKey
	err := json.Unmarshal(keyJson, &parsedKey)
	if err != nil {
		t.Error(fmt.Sprintf("Error unmarshalling key: %v", err))
		return
	}
	ek := EmbedsKey{
		Key: parsedKey,
	}
	out, err := json.Marshal(ek)
	if err != nil {
		t.Error(fmt.Sprintf("Error marshalling JSON: %v", err))
		return
	}
	expected := "{\"Key\":{\"kty\":\"RSA\",\"n\":\"vd7rZIoTLEe-z1_8G1FcXSw9CQFEJgV4g9V277sER7yx5Qjz_Pkf2YVth6wwwFJEmzc0hoKY-MMYFNwBE4hQHw\",\"e\":\"AQAB\"}}"
	if string(out) != expected {
		t.Error("Failed to marshal embedded non-pointer JWK properly:", string(out))
	}
}

func TestMarshalUnmarshalInvalid(t *testing.T) {
	// Make an invalid curve coordinate by creating a byte array that is one
	// byte too large, and setting the first byte to 1 (otherwise it's just zero).
	invalidCoord := make([]byte, curveSize(ecTestKey256.Curve)+1)
	invalidCoord[0] = 1

	keys := []interface{}{
		// Empty keys
		&rsa.PrivateKey{},
		&ecdsa.PrivateKey{},
		// Invalid keys
		&ecdsa.PrivateKey{
			PublicKey: ecdsa.PublicKey{
				// Missing values in pub key
				Curve: elliptic.P256(),
			},
		},
		&ecdsa.PrivateKey{
			PublicKey: ecdsa.PublicKey{
				// Invalid curve
				Curve: nil,
				X:     ecTestKey256.X,
				Y:     ecTestKey256.Y,
			},
		},
		&ecdsa.PrivateKey{
			// Valid pub key, but missing priv key values
			PublicKey: ecTestKey256.PublicKey,
		},
		&ecdsa.PrivateKey{
			// Invalid pub key, values too large
			PublicKey: ecdsa.PublicKey{
				Curve: ecTestKey256.Curve,
				X:     big.NewInt(0).SetBytes(invalidCoord),
				Y:     big.NewInt(0).SetBytes(invalidCoord),
			},
			D: ecTestKey256.D,
		},
		nil,
	}

	for i, key := range keys {
		jwk := JsonWebKey{Key: key}
		_, err := jwk.MarshalJSON()
		if err == nil {
			t.Error("managed to serialize invalid key", i)
		}
	}
}

func TestWebKeyVectorsInvalid(t *testing.T) {
	keys := []string{
		// Invalid JSON
		"{X",
		// Empty key
		"{}",
		// Invalid RSA keys
		`{"kty":"RSA"}`,
		`{"kty":"RSA","e":""}`,
		`{"kty":"RSA","e":"XXXX"}`,
		`{"kty":"RSA","d":"XXXX"}`,
		// Invalid EC keys
		`{"kty":"EC","crv":"ABC"}`,
		`{"kty":"EC","crv":"P-256"}`,
		`{"kty":"EC","crv":"P-256","d":"XXX"}`,
		`{"kty":"EC","crv":"ABC","d":"dGVzdA","x":"dGVzdA"}`,
		`{"kty":"EC","crv":"P-256","d":"dGVzdA","x":"dGVzdA"}`,
	}

	for _, key := range keys {
		var jwk2 JsonWebKey
		err := jwk2.UnmarshalJSON([]byte(key))
		if err == nil {
			t.Error("managed to parse invalid key:", key)
		}
	}
}

func TestWebKeyVectorsValid(t *testing.T) {
	keys := []string{
		stripWhitespace(`{"kty":"RSA",
      "kid":"juliet@capulet.lit",
      "use":"enc",
      "n":"t6Q8PWSi1dkJj9hTP8hNYFlvadM7DflW9mWepOJhJ66w7nyoK1gPNqFMSQRy
           O125Gp-TEkodhWr0iujjHVx7BcV0llS4w5ACGgPrcAd6ZcSR0-Iqom-QFcNP
           8Sjg086MwoqQU_LYywlAGZ21WSdS_PERyGFiNnj3QQlO8Yns5jCtLCRwLHL0
           Pb1fEv45AuRIuUfVcPySBWYnDyGxvjYGDSM-AqWS9zIQ2ZilgT-GqUmipg0X
           OC0Cc20rgLe2ymLHjpHciCKVAbY5-L32-lSeZO-Os6U15_aXrk9Gw8cPUaX1
           _I8sLGuSiVdt3C_Fn2PZ3Z8i744FPFGGcG1qs2Wz-Q",
      "e":"AQAB",
      "d":"GRtbIQmhOZtyszfgKdg4u_N-R_mZGU_9k7JQ_jn1DnfTuMdSNprTeaSTyWfS
           NkuaAwnOEbIQVy1IQbWVV25NY3ybc_IhUJtfri7bAXYEReWaCl3hdlPKXy9U
           vqPYGR0kIXTQRqns-dVJ7jahlI7LyckrpTmrM8dWBo4_PMaenNnPiQgO0xnu
           ToxutRZJfJvG4Ox4ka3GORQd9CsCZ2vsUDmsXOfUENOyMqADC6p1M3h33tsu
           rY15k9qMSpG9OX_IJAXmxzAh_tWiZOwk2K4yxH9tS3Lq1yX8C1EWmeRDkK2a
           hecG85-oLKQt5VEpWHKmjOi_gJSdSgqcN96X52esAQ",
      "p":"2rnSOV4hKSN8sS4CgcQHFbs08XboFDqKum3sc4h3GRxrTmQdl1ZK9uw-PIHf
           QP0FkxXVrx-WE-ZEbrqivH_2iCLUS7wAl6XvARt1KkIaUxPPSYB9yk31s0Q8
           UK96E3_OrADAYtAJs-M3JxCLfNgqh56HDnETTQhH3rCT5T3yJws",
      "q":"1u_RiFDP7LBYh3N4GXLT9OpSKYP0uQZyiaZwBtOCBNJgQxaj10RWjsZu0c6I
           edis4S7B_coSKB0Kj9PaPaBzg-IySRvvcQuPamQu66riMhjVtG6TlV8CLCYK
           rYl52ziqK0E_ym2QnkwsUX7eYTB7LbAHRK9GqocDE5B0f808I4s",
      "dp":"KkMTWqBUefVwZ2_Dbj1pPQqyHSHjj90L5x_MOzqYAJMcLMZtbUtwKqvVDq3
           tbEo3ZIcohbDtt6SbfmWzggabpQxNxuBpoOOf_a_HgMXK_lhqigI4y_kqS1w
           Y52IwjUn5rgRrJ-yYo1h41KR-vz2pYhEAeYrhttWtxVqLCRViD6c",
      "dq":"AvfS0-gRxvn0bwJoMSnFxYcK1WnuEjQFluMGfwGitQBWtfZ1Er7t1xDkbN9
           GQTB9yqpDoYaN06H7CFtrkxhJIBQaj6nkF5KKS3TQtQ5qCzkOkmxIe3KRbBy
           mXxkb5qwUpX5ELD5xFc6FeiafWYY63TmmEAu_lRFCOJ3xDea-ots",
      "qi":"lSQi-w9CpyUReMErP1RsBLk7wNtOvs5EQpPqmuMvqW57NBUczScEoPwmUqq
           abu9V0-Py4dQ57_bapoKRu1R90bvuFnU63SHWEFglZQvJDMeAvmj4sm-Fp0o
           Yu_neotgQ0hzbI5gry7ajdYy9-2lNx_76aBZoOUu9HCJ-UsfSOI8"}`),
	}

	for _, key := range keys {
		var jwk2 JsonWebKey
		err := jwk2.UnmarshalJSON([]byte(key))
		if err != nil {
			t.Error("unable to parse valid key:", key, err)
		}
	}
}
