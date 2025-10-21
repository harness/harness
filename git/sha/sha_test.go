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

package sha

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"
)

const emptyTreeSHA = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid sha",
			input:   emptyTreeSHA,
			wantErr: false,
		},
		{
			name:    "valid short sha",
			input:   "4b825dc6",
			wantErr: false,
		},
		{
			name:    "valid sha with spaces",
			input:   "  " + emptyTreeSHA + "  ",
			wantErr: false,
		},
		{
			name:    "valid sha uppercase",
			input:   "4B825DC642CB6EB9A060E54BF8D69288FBEE4904",
			wantErr: false,
		},
		{
			name:    "invalid sha - too short",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "invalid sha - invalid characters",
			input:   "gggggggggggggggggggggggggggggggggggggggg",
			wantErr: true,
		},
		{
			name:    "invalid sha - empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.String() == "" {
				t.Error("expected non-empty SHA")
			}
		})
	}
}

func TestNewOrEmpty(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		isEmpty bool
	}{
		{
			name:    "valid sha",
			input:   emptyTreeSHA,
			wantErr: false,
			isEmpty: false,
		},
		{
			name:    "empty string returns None",
			input:   "",
			wantErr: false,
			isEmpty: true,
		},
		{
			name:    "invalid sha",
			input:   "invalid",
			wantErr: true,
			isEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewOrEmpty(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOrEmpty() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.IsEmpty() != tt.isEmpty {
				t.Errorf("NewOrEmpty() isEmpty = %v, want %v", got.IsEmpty(), tt.isEmpty)
			}
		})
	}
}

func TestSHA_IsNil(t *testing.T) {
	tests := []struct {
		name  string
		sha   SHA
		isNil bool
	}{
		{
			name:  "nil sha",
			sha:   Nil,
			isNil: true,
		},
		{
			name:  "non-nil sha",
			sha:   EmptyTree,
			isNil: false,
		},
		{
			name:  "empty sha",
			sha:   None,
			isNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sha.IsNil(); got != tt.isNil {
				t.Errorf("IsNil() = %v, want %v", got, tt.isNil)
			}
		})
	}
}

func TestSHA_IsEmpty(t *testing.T) {
	tests := []struct {
		name    string
		sha     SHA
		isEmpty bool
	}{
		{
			name:    "empty sha",
			sha:     None,
			isEmpty: true,
		},
		{
			name:    "non-empty sha",
			sha:     EmptyTree,
			isEmpty: false,
		},
		{
			name:    "nil sha",
			sha:     Nil,
			isEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sha.IsEmpty(); got != tt.isEmpty {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.isEmpty)
			}
		})
	}
}

func TestSHA_Equal(t *testing.T) {
	sha1 := Must(emptyTreeSHA)
	sha2 := Must(emptyTreeSHA)
	sha3 := Must("1234567890abcdef1234567890abcdef12345678")

	tests := []struct {
		name  string
		sha1  SHA
		sha2  SHA
		equal bool
	}{
		{
			name:  "equal shas",
			sha1:  sha1,
			sha2:  sha2,
			equal: true,
		},
		{
			name:  "different shas",
			sha1:  sha1,
			sha2:  sha3,
			equal: false,
		},
		{
			name:  "empty shas",
			sha1:  None,
			sha2:  None,
			equal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sha1.Equal(tt.sha2); got != tt.equal {
				t.Errorf("Equal() = %v, want %v", got, tt.equal)
			}
		})
	}
}

func TestSHA_String(t *testing.T) {
	sha := Must(emptyTreeSHA)
	if got := sha.String(); got != emptyTreeSHA {
		t.Errorf("String() = %v, want %v", got, emptyTreeSHA)
	}
}

func TestSHA_Value(t *testing.T) {
	sha := Must(emptyTreeSHA)
	val, err := sha.Value()
	if err != nil {
		t.Errorf("Value() error = %v", err)
	}
	if val != emptyTreeSHA {
		t.Errorf("Value() = %v, want %v", val, emptyTreeSHA)
	}
}

func TestSHA_GobEncodeDecode(t *testing.T) {
	original := Must(emptyTreeSHA)

	// Encode
	encoded, err := original.GobEncode()
	if err != nil {
		t.Fatalf("GobEncode() error = %v", err)
	}

	// Decode
	var decoded SHA
	err = decoded.GobDecode(encoded)
	if err != nil {
		t.Fatalf("GobDecode() error = %v", err)
	}

	if !original.Equal(decoded) {
		t.Errorf("GobEncode/Decode round trip failed: got %v, want %v", decoded, original)
	}
}

func TestSHA_GobEncodeDecodeWithGob(t *testing.T) {
	original := Must(emptyTreeSHA)

	// Encode using gob
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(original)
	if err != nil {
		t.Fatalf("gob.Encode() error = %v", err)
	}

	// Decode using gob
	var decoded SHA
	dec := gob.NewDecoder(&buf)
	err = dec.Decode(&decoded)
	if err != nil {
		t.Fatalf("gob.Decode() error = %v", err)
	}

	if !original.Equal(decoded) {
		t.Errorf("gob Encode/Decode round trip failed: got %v, want %v", decoded, original)
	}
}

func TestMust(t *testing.T) {
	t.Run("valid sha", func(t *testing.T) {
		sha := Must(emptyTreeSHA)
		if sha.String() != emptyTreeSHA {
			t.Errorf("Must() = %v, want %v", sha.String(), emptyTreeSHA)
		}
	})

	t.Run("invalid sha panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Must() did not panic on invalid SHA")
			}
		}()
		Must("invalid")
	})
}

func TestSHA_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   SHA
		want    []byte
		wantErr bool
	}{
		{
			name:  "happy path",
			input: EmptyTree,
			want:  []byte("\"" + EmptyTree.String() + "\""),
		},
		{
			name: "happy path - quotes",
			input: SHA{
				str: "\"\"",
			},
			want: []byte("\"\"\"\""),
		},
		{
			name:  "happy path - empty string",
			input: SHA{},
			want:  []byte("\"\""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.input.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSHA_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected SHA
		wantErr  bool
	}{
		{
			name:     "happy path",
			input:    []byte("\"" + EmptyTree.String() + "\""),
			expected: EmptyTree,
			wantErr:  false,
		},
		{
			name:     "empty content returns empty",
			input:    []byte("\"\""),
			expected: SHA{},
			wantErr:  false,
		},
		{
			name:     "invalid json",
			input:    []byte("invalid"),
			expected: SHA{},
			wantErr:  true,
		},
		{
			name:     "invalid sha",
			input:    []byte("\"invalid\""),
			expected: SHA{},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SHA{}
			if err := s.UnmarshalJSON(tt.input); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(s, tt.expected) {
				t.Errorf("bytes.Equal expected %s, got %s", tt.expected, s)
			}
		})
	}
}

func TestSHA_JSONSchema(t *testing.T) {
	sha := Must(emptyTreeSHA)
	schema, err := sha.JSONSchema()
	if err != nil {
		t.Errorf("JSONSchema() error = %v", err)
	}
	if schema.Description == nil || *schema.Description != "Git object hash" {
		t.Errorf("JSONSchema() description = %v, want 'Git object hash'", schema.Description)
	}
}
