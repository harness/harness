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
	"strings"
	"testing"
)

const testKey32Bytes = "12345678901234567890123456789012"

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		compat    bool
		expectErr bool
	}{
		{
			name:      "valid 32-byte key",
			key:       testKey32Bytes,
			compat:    false,
			expectErr: false,
		},
		{
			name:      "valid 32-byte key with compat mode",
			key:       testKey32Bytes,
			compat:    true,
			expectErr: false,
		},
		{
			name:      "invalid key - too short",
			key:       "short",
			compat:    false,
			expectErr: true,
		},
		{
			name:      "invalid key - too long",
			key:       "123456789012345678901234567890123",
			compat:    false,
			expectErr: true,
		},
		{
			name:      "empty key",
			key:       "",
			compat:    false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypter, err := New(tt.key, tt.compat)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if encrypter != nil {
					t.Errorf("expected nil encrypter but got %v", encrypter)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if encrypter == nil {
				t.Errorf("expected encrypter but got nil")
			}
		})
	}
}

func TestAesgcmEncryptDecrypt(t *testing.T) {
	key := testKey32Bytes
	encrypter, err := New(key, false)
	if err != nil {
		t.Fatalf("failed to create encrypter: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple text",
			plaintext: "hello world",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "long text",
			plaintext: strings.Repeat("a", 1000),
		},
		{
			name:      "special characters",
			plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
		{
			name:      "unicode text",
			plaintext: "Hello 世界 🌍",
		},
		{
			name:      "newlines and tabs",
			plaintext: "line1\nline2\tline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := encrypter.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("encryption failed: %v", err)
			}

			// Verify ciphertext is not empty
			if len(ciphertext) == 0 {
				t.Errorf("ciphertext is empty")
			}

			// Decrypt
			decrypted, err := encrypter.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("decryption failed: %v", err)
			}

			// Verify decrypted matches original
			if decrypted != tt.plaintext {
				t.Errorf("decrypted text does not match original\nexpected: %q\ngot: %q", tt.plaintext, decrypted)
			}
		})
	}
}

func TestAesgcmEncryptUniqueness(t *testing.T) {
	key := testKey32Bytes
	encrypter, err := New(key, false)
	if err != nil {
		t.Fatalf("failed to create encrypter: %v", err)
	}

	plaintext := "test message"

	// Encrypt the same plaintext multiple times
	ciphertext1, err := encrypter.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encryption 1 failed: %v", err)
	}

	ciphertext2, err := encrypter.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encryption 2 failed: %v", err)
	}

	// Verify ciphertexts are different (due to random nonce)
	if string(ciphertext1) == string(ciphertext2) {
		t.Errorf("ciphertexts should be different due to random nonce")
	}

	// But both should decrypt to the same plaintext
	decrypted1, err := encrypter.Decrypt(ciphertext1)
	if err != nil {
		t.Fatalf("decryption 1 failed: %v", err)
	}

	decrypted2, err := encrypter.Decrypt(ciphertext2)
	if err != nil {
		t.Fatalf("decryption 2 failed: %v", err)
	}

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Errorf("decrypted texts should match original plaintext")
	}
}

func TestAesgcmDecryptInvalidCiphertext(t *testing.T) {
	key := testKey32Bytes
	encrypter, err := New(key, false)
	if err != nil {
		t.Fatalf("failed to create encrypter: %v", err)
	}

	tests := []struct {
		name       string
		ciphertext []byte
		expectErr  bool
	}{
		{
			name:       "empty ciphertext",
			ciphertext: []byte{},
			expectErr:  true,
		},
		{
			name:       "too short ciphertext",
			ciphertext: []byte{1, 2, 3},
			expectErr:  true,
		},
		{
			name:       "corrupted ciphertext",
			ciphertext: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encrypter.Decrypt(tt.ciphertext)
			if tt.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
		})
	}
}

func TestAesgcmCompatMode(t *testing.T) {
	key := testKey32Bytes
	encrypter, err := New(key, true)
	if err != nil {
		t.Fatalf("failed to create encrypter: %v", err)
	}

	aesgcm, _ := encrypter.(*Aesgcm)
	if !aesgcm.Compat {
		t.Errorf("compat mode should be enabled")
	}

	// Test that invalid ciphertext returns the ciphertext as plaintext in compat mode
	invalidCiphertext := []byte("not encrypted data")
	decrypted, err := encrypter.Decrypt(invalidCiphertext)
	if err != nil {
		t.Errorf("compat mode should not return error for invalid ciphertext: %v", err)
	}
	if decrypted != string(invalidCiphertext) {
		t.Errorf(
			"compat mode should return ciphertext as plaintext\nexpected: %q\ngot: %q",
			string(invalidCiphertext),
			decrypted,
		)
	}

	// Test that valid encrypted data still works in compat mode
	plaintext := "test message"
	ciphertext, err := encrypter.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err = encrypter.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted text does not match original\nexpected: %q\ngot: %q", plaintext, decrypted)
	}
}

func TestAesgcmCompatModeShortCiphertext(t *testing.T) {
	key := testKey32Bytes
	encrypter, err := New(key, true)
	if err != nil {
		t.Fatalf("failed to create encrypter: %v", err)
	}

	// Test with very short ciphertext (less than nonce size)
	shortCiphertext := []byte("short")
	decrypted, err := encrypter.Decrypt(shortCiphertext)
	if err != nil {
		t.Errorf("compat mode should not return error for short ciphertext: %v", err)
	}
	if decrypted != string(shortCiphertext) {
		t.Errorf(
			"compat mode should return ciphertext as plaintext\nexpected: %q\ngot: %q",
			string(shortCiphertext),
			decrypted,
		)
	}
}

func TestAesgcmNonCompatModeInvalidCiphertext(t *testing.T) {
	key := testKey32Bytes
	encrypter, err := New(key, false)
	if err != nil {
		t.Fatalf("failed to create encrypter: %v", err)
	}

	// Test that invalid ciphertext returns error in non-compat mode
	invalidCiphertext := []byte("not encrypted data")
	_, err = encrypter.Decrypt(invalidCiphertext)
	if err == nil {
		t.Errorf("non-compat mode should return error for invalid ciphertext")
	}
}
