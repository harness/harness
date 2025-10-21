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

package types

import (
	"encoding/json"
	"testing"
)

func TestSpacePathSegment_MarshalJSON(t *testing.T) {
	t.Run("marshal with all fields", func(t *testing.T) {
		segment := SpacePathSegment{
			ID:         123,
			Identifier: "test-identifier",
			IsPrimary:  true,
			SpaceID:    456,
			ParentID:   789,
			CreatedBy:  111,
			Created:    1234567890,
			Updated:    1234567900,
		}

		data, err := json.Marshal(segment)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		// Check that uid field is present and matches identifier
		uid, ok := result["uid"].(string)
		if !ok {
			t.Error("uid field not found or not a string")
		}
		if uid != "test-identifier" {
			t.Errorf("expected uid to be 'test-identifier', got %s", uid)
		}

		// Check that identifier field is also present
		identifier, ok := result["identifier"].(string)
		if !ok {
			t.Error("identifier field not found or not a string")
		}
		if identifier != "test-identifier" {
			t.Errorf("expected identifier to be 'test-identifier', got %s", identifier)
		}

		// Check other fields
		if isPrimary, ok := result["is_primary"].(bool); !ok || !isPrimary {
			t.Errorf("expected is_primary to be true, got %v", result["is_primary"])
		}

		if spaceID, ok := result["space_id"].(float64); !ok || int64(spaceID) != 456 {
			t.Errorf("expected space_id to be 456, got %v", result["space_id"])
		}
	})

	t.Run("marshal with empty identifier", func(t *testing.T) {
		segment := SpacePathSegment{
			ID:         1,
			Identifier: "",
			IsPrimary:  false,
			SpaceID:    2,
		}

		data, err := json.Marshal(segment)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		// Check that uid field is present and empty
		uid, ok := result["uid"].(string)
		if !ok {
			t.Error("uid field not found or not a string")
		}
		if uid != "" {
			t.Errorf("expected uid to be empty, got %s", uid)
		}
	})

	t.Run("marshal zero values", func(t *testing.T) {
		segment := SpacePathSegment{}

		data, err := json.Marshal(segment)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		// Check that uid field is present and empty
		uid, ok := result["uid"].(string)
		if !ok {
			t.Error("uid field not found or not a string")
		}
		if uid != "" {
			t.Errorf("expected uid to be empty, got %s", uid)
		}

		// Check that identifier field is also present and empty
		identifier, ok := result["identifier"].(string)
		if !ok {
			t.Error("identifier field not found or not a string")
		}
		if identifier != "" {
			t.Errorf("expected identifier to be empty, got %s", identifier)
		}
	})
}
