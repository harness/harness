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

package encrypt

import (
	"bytes"
	"testing"
)

func TestNone_Encrypt(t *testing.T) {
	encrypter := &none{}

	t.Run("encrypt simple string", func(t *testing.T) {
		plaintext := "hello world"
		ciphertext, err := encrypter.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !bytes.Equal(ciphertext, []byte(plaintext)) {
			t.Errorf("expected ciphertext to be %v, got %v", []byte(plaintext), ciphertext)
		}
	})

	t.Run("encrypt empty string", func(t *testing.T) {
		plaintext := ""
		ciphertext, err := encrypter.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !bytes.Equal(ciphertext, []byte(plaintext)) {
			t.Errorf("expected ciphertext to be empty, got %v", ciphertext)
		}
	})

	t.Run("encrypt special characters", func(t *testing.T) {
		plaintext := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
		ciphertext, err := encrypter.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !bytes.Equal(ciphertext, []byte(plaintext)) {
			t.Errorf("expected ciphertext to be %v, got %v", []byte(plaintext), ciphertext)
		}
	})

	t.Run("encrypt unicode characters", func(t *testing.T) {
		plaintext := "こんにちは世界 🌍"
		ciphertext, err := encrypter.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !bytes.Equal(ciphertext, []byte(plaintext)) {
			t.Errorf("expected ciphertext to be %v, got %v", []byte(plaintext), ciphertext)
		}
	})
}

func TestNone_Decrypt(t *testing.T) {
	encrypter := &none{}

	t.Run("decrypt simple bytes", func(t *testing.T) {
		ciphertext := []byte("hello world")
		plaintext, err := encrypter.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if plaintext != string(ciphertext) {
			t.Errorf("expected plaintext to be %s, got %s", string(ciphertext), plaintext)
		}
	})

	t.Run("decrypt empty bytes", func(t *testing.T) {
		ciphertext := []byte("")
		plaintext, err := encrypter.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if plaintext != "" {
			t.Errorf("expected plaintext to be empty, got %s", plaintext)
		}
	})

	t.Run("decrypt nil bytes", func(t *testing.T) {
		var ciphertext []byte
		plaintext, err := encrypter.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if plaintext != "" {
			t.Errorf("expected plaintext to be empty, got %s", plaintext)
		}
	})

	t.Run("decrypt special characters", func(t *testing.T) {
		ciphertext := []byte("!@#$%^&*()_+-=[]{}|;':\",./<>?")
		plaintext, err := encrypter.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if plaintext != string(ciphertext) {
			t.Errorf("expected plaintext to be %s, got %s", string(ciphertext), plaintext)
		}
	})
}

func TestNone_EncryptDecrypt_RoundTrip(t *testing.T) {
	encrypter := &none{}

	testCases := []string{
		"hello world",
		"",
		"!@#$%^&*()",
		"こんにちは世界",
		"multi\nline\nstring",
		"tab\tseparated\tvalues",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			ciphertext, err := encrypter.Encrypt(tc)
			if err != nil {
				t.Fatalf("encrypt failed: %v", err)
			}

			plaintext, err := encrypter.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("decrypt failed: %v", err)
			}

			if plaintext != tc {
				t.Errorf("round trip failed: expected %s, got %s", tc, plaintext)
			}
		})
	}
}
