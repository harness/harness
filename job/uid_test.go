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

package job

import (
	"testing"
)

func TestUID(t *testing.T) {
	uid, err := UID()
	if err != nil {
		t.Fatalf("UID() returned error: %v", err)
	}

	// Verify UID is not empty
	if uid == "" {
		t.Errorf("UID() returned empty string")
	}

	// Verify UID has expected length (10 bytes / 5 * 8 = 16 characters)
	expectedLength := 16
	if len(uid) != expectedLength {
		t.Errorf("UID() length = %d, expected %d", len(uid), expectedLength)
	}

	// Verify UID contains only valid base32 characters
	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567="
	for _, c := range uid {
		found := false
		for _, valid := range validChars {
			if c == valid {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("UID() contains invalid character: %c", c)
		}
	}
}

func TestUIDUniqueness(t *testing.T) {
	// Generate multiple UIDs and verify they are unique
	const numUIDs = 1000
	uids := make(map[string]bool)

	for i := range numUIDs {
		uid, err := UID()
		if err != nil {
			t.Fatalf("UID() returned error on iteration %d: %v", i, err)
		}

		if uids[uid] {
			t.Errorf("UID() generated duplicate: %s", uid)
		}
		uids[uid] = true
	}

	// Verify we generated the expected number of unique UIDs
	if len(uids) != numUIDs {
		t.Errorf("Generated %d unique UIDs, expected %d", len(uids), numUIDs)
	}
}

func TestUIDConsistentLength(t *testing.T) {
	// Generate multiple UIDs and verify they all have the same length
	const numUIDs = 100
	expectedLength := 16

	for i := range numUIDs {
		uid, err := UID()
		if err != nil {
			t.Fatalf("UID() returned error on iteration %d: %v", i, err)
		}

		if len(uid) != expectedLength {
			t.Errorf("UID() iteration %d: length = %d, expected %d", i, len(uid), expectedLength)
		}
	}
}

func TestUIDBase32Encoding(t *testing.T) {
	// Generate a UID and verify it's valid base32
	uid, err := UID()
	if err != nil {
		t.Fatalf("UID() returned error: %v", err)
	}

	// Try to decode it as base32 - should not error
	// Note: We don't need to import encoding/base32 again as it's already in uid.go
	// Just verify the format is correct by checking characters
	for i, c := range uid {
		if (c < 'A' || c > 'Z') && (c < '2' || c > '7') && c != '=' {
			t.Errorf("UID() character at position %d (%c) is not valid base32", i, c)
		}
	}
}

func TestUIDNoPadding(t *testing.T) {
	// Generate multiple UIDs and verify they don't have padding
	// (10 bytes encodes to exactly 16 characters without padding)
	const numUIDs = 100

	for i := range numUIDs {
		uid, err := UID()
		if err != nil {
			t.Fatalf("UID() returned error on iteration %d: %v", i, err)
		}

		// Check if UID contains padding character '='
		for j, c := range uid {
			if c == '=' {
				t.Errorf("UID() iteration %d: contains padding at position %d", i, j)
			}
		}
	}
}

func BenchmarkUID(b *testing.B) {
	for b.Loop() {
		_, err := UID()
		if err != nil {
			b.Fatalf("UID() returned error: %v", err)
		}
	}
}

func BenchmarkUIDParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := UID()
			if err != nil {
				b.Fatalf("UID() returned error: %v", err)
			}
		}
	})
}
