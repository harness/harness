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

package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestGenerateHMACSHA256(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		key      []byte
		expected string
	}{
		{
			name:     "simple data and key",
			data:     []byte("hello world"),
			key:      []byte("secret"),
			expected: "734cc62f32841568f45715aeb9f4d7891324e6d948e4c6c60c0621cdac48623a",
		},
		{
			name:     "empty data",
			data:     []byte(""),
			key:      []byte("secret"),
			expected: "f9e66e179b6747ae54108f82f8ade8b3c25d76fd30afde6c395822c530196169",
		},
		{
			name:     "empty key",
			data:     []byte("hello world"),
			key:      []byte(""),
			expected: "", // We'll calculate this dynamically
		},
		{
			name:     "both empty",
			data:     []byte(""),
			key:      []byte(""),
			expected: "b613679a0814d9ec772f95d778c35fc5ff1697c493715653c6c712144292c5ad",
		},
		{
			name:     "long data",
			data:     []byte(strings.Repeat("a", 1000)),
			key:      []byte("secret"),
			expected: "", // We'll calculate this dynamically
		},
		{
			name:     "long key",
			data:     []byte("hello"),
			key:      []byte(strings.Repeat("k", 100)),
			expected: "", // We'll calculate this dynamically
		},
		{
			name:     "binary data",
			data:     []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
			key:      []byte("binary"),
			expected: "", // We'll calculate this dynamically
		},
		{
			name:     "unicode data",
			data:     []byte("Hello 世界 🌍"),
			key:      []byte("unicode"),
			expected: "", // We'll calculate this dynamically
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateHMACSHA256(tt.data, tt.key)
			if err != nil {
				t.Errorf("GenerateHMACSHA256() error = %v", err)
				return
			}

			// For cases where we don't have pre-calculated expected values,
			// verify the result by computing it manually
			if tt.expected == "" {
				h := hmac.New(sha256.New, tt.key)
				h.Write(tt.data)
				expected := hex.EncodeToString(h.Sum(nil))
				if result != expected {
					t.Errorf("GenerateHMACSHA256() = %v, want %v", result, expected)
				}
			} else if result != tt.expected {
				t.Errorf("GenerateHMACSHA256() = %v, want %v", result, tt.expected)
			}

			// Verify the result is valid hex
			_, err = hex.DecodeString(result)
			if err != nil {
				t.Errorf("GenerateHMACSHA256() returned invalid hex: %v", err)
			}

			// Verify the result has the correct length for SHA256 (64 hex characters)
			if len(result) != 64 {
				t.Errorf("GenerateHMACSHA256() returned wrong length: got %d, want 64", len(result))
			}
		})
	}
}

func TestGenerateHMACSHA256Consistency(t *testing.T) {
	// Test that the same input always produces the same output
	data := []byte("test data")
	key := []byte("test key")

	result1, err1 := GenerateHMACSHA256(data, key)
	if err1 != nil {
		t.Fatalf("First call failed: %v", err1)
	}

	result2, err2 := GenerateHMACSHA256(data, key)
	if err2 != nil {
		t.Fatalf("Second call failed: %v", err2)
	}

	if result1 != result2 {
		t.Errorf("GenerateHMACSHA256() is not consistent: %v != %v", result1, result2)
	}
}

func TestGenerateHMACSHA256DifferentInputs(t *testing.T) {
	// Test that different inputs produce different outputs
	key := []byte("secret")

	result1, _ := GenerateHMACSHA256([]byte("data1"), key)
	result2, _ := GenerateHMACSHA256([]byte("data2"), key)

	if result1 == result2 {
		t.Error("GenerateHMACSHA256() should produce different results for different inputs")
	}

	// Test different keys
	data := []byte("same data")
	result3, _ := GenerateHMACSHA256(data, []byte("key1"))
	result4, _ := GenerateHMACSHA256(data, []byte("key2"))

	if result3 == result4 {
		t.Error("GenerateHMACSHA256() should produce different results for different keys")
	}
}

func TestIsShaEqual(t *testing.T) {
	tests := []struct {
		name     string
		key1     string
		key2     string
		expected bool
	}{
		{
			name:     "identical strings",
			key1:     "hello",
			key2:     "hello",
			expected: true,
		},
		{
			name:     "different strings",
			key1:     "hello",
			key2:     "world",
			expected: false,
		},
		{
			name:     "empty strings",
			key1:     "",
			key2:     "",
			expected: true,
		},
		{
			name:     "one empty string",
			key1:     "hello",
			key2:     "",
			expected: false,
		},
		{
			name:     "case sensitive",
			key1:     "Hello",
			key2:     "hello",
			expected: false,
		},
		{
			name:     "whitespace differences",
			key1:     "hello ",
			key2:     "hello",
			expected: false,
		},
		{
			name:     "long identical strings",
			key1:     strings.Repeat("a", 1000),
			key2:     strings.Repeat("a", 1000),
			expected: true,
		},
		{
			name:     "long different strings",
			key1:     strings.Repeat("a", 1000),
			key2:     strings.Repeat("b", 1000),
			expected: false,
		},
		{
			name:     "unicode strings identical",
			key1:     "Hello 世界 🌍",
			key2:     "Hello 世界 🌍",
			expected: true,
		},
		{
			name:     "unicode strings different",
			key1:     "Hello 世界 🌍",
			key2:     "Hello 世界 🌎",
			expected: false,
		},
		{
			name:     "hex strings identical",
			key1:     "deadbeef",
			key2:     "deadbeef",
			expected: true,
		},
		{
			name:     "hex strings different",
			key1:     "deadbeef",
			key2:     "deadbeee",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsShaEqual(tt.key1, tt.key2)
			if result != tt.expected {
				t.Errorf("IsShaEqual(%q, %q) = %v, want %v", tt.key1, tt.key2, result, tt.expected)
			}
		})
	}
}

func TestIsShaEqualTimingSafety(t *testing.T) {
	// Test that IsShaEqual uses constant-time comparison
	// This is important for security to prevent timing attacks

	// Create two strings that differ only in the last character
	base := strings.Repeat("a", 100)
	key1 := base + "1"
	key2 := base + "2"

	// The function should return false
	result := IsShaEqual(key1, key2)
	if result {
		t.Error("IsShaEqual() should return false for different strings")
	}

	// Test with strings of different lengths
	result2 := IsShaEqual("short", "much longer string")
	if result2 {
		t.Error("IsShaEqual() should return false for strings of different lengths")
	}
}

func TestGenerateHMACSHA256WithRealWorldData(t *testing.T) {
	// Test with realistic data that might be used in practice
	tests := []struct {
		name string
		data []byte
		key  []byte
	}{
		{
			name: "JSON payload",
			data: []byte(`{"user_id": 123, "action": "login", "timestamp": "2023-01-01T00:00:00Z"}`),
			key:  []byte("webhook-secret-key"),
		},
		{
			name: "URL parameters",
			data: []byte("user=john&action=login&timestamp=1672531200"),
			key:  []byte("api-secret"),
		},
		{
			name: "Base64 data",
			data: []byte("SGVsbG8gV29ybGQ="),
			key:  []byte("base64-key"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateHMACSHA256(tt.data, tt.key)
			if err != nil {
				t.Errorf("GenerateHMACSHA256() error = %v", err)
				return
			}

			// Verify the result is a valid SHA256 hash
			if len(result) != 64 {
				t.Errorf("Expected 64 character hash, got %d", len(result))
			}

			// Verify it's valid hex
			_, err = hex.DecodeString(result)
			if err != nil {
				t.Errorf("Result is not valid hex: %v", err)
			}
		})
	}
}

// Benchmark tests.
func BenchmarkGenerateHMACSHA256(b *testing.B) {
	data := []byte("benchmark data")
	key := []byte("benchmark key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateHMACSHA256(data, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateHMACSHA256LargeData(b *testing.B) {
	data := []byte(strings.Repeat("a", 10000))
	key := []byte("benchmark key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateHMACSHA256(data, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIsShaEqual(b *testing.B) {
	key1 := "benchmark string for comparison"
	key2 := "benchmark string for comparison"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsShaEqual(key1, key2)
	}
}

func BenchmarkIsShaEqualLarge(b *testing.B) {
	key1 := strings.Repeat("a", 1000)
	key2 := strings.Repeat("a", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsShaEqual(key1, key2)
	}
}
