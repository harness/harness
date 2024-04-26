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
	"reflect"
	"testing"
)

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SHA{}
			if err := s.UnmarshalJSON(tt.input); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(s, tt.expected) {
				t.Errorf("bytes.Equal expected %s, got %s", tt.expected, s)
			}
		})
	}
}
