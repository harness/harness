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
	"reflect"
	"testing"
)

func TestSHA_MarshalJSON(t *testing.T) {
	type fields struct {
		bytes []byte
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				bytes: []byte(EmptyTree),
			},
			want: []byte("\"" + EmptyTree + "\""),
		},
		{
			name: "happy path - quotes",
			fields: fields{
				bytes: []byte("\"\""),
			},
			want: []byte("\"\"\"\""),
		},
		{
			name: "happy path - empty slice",
			fields: fields{
				bytes: []byte{},
			},
			want: []byte("\"\""),
		},
		{
			name: "happy path - nil slice",
			fields: fields{
				bytes: nil,
			},
			want: []byte("null"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SHA{
				bytes: tt.fields.bytes,
			}
			got, err := s.MarshalJSON()
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
	type fields struct {
		bytes []byte
	}
	type args struct {
		content []byte
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expected []byte
		wantErr  bool
	}{
		{
			name:     "happy path",
			args:     args{content: []byte("\"" + EmptyTree + "\"")},
			expected: []byte(EmptyTree),
			wantErr:  false,
		},
		{
			name:     "empty content return error",
			args:     args{content: []byte("\"\"")},
			expected: []byte{},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SHA{
				bytes: tt.fields.bytes,
			}
			if err := s.UnmarshalJSON(tt.args.content); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !bytes.Equal(s.bytes, tt.expected) {
				t.Errorf("bytes.Equal expected %s, got %s", tt.expected, s.bytes)
			}
		})
	}
}
