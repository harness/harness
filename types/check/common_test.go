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

package check

import (
	"errors"
	"strings"
	"testing"
)

func TestDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		wantErr     error
	}{
		{name: "empty", description: ""},
		{name: "colon and hyphens", description: "names of the following: - a - b"},
		{name: "unicode", description: "repo für Büro 日本語"},
		{name: "newline", description: "line one\nline two", wantErr: ErrDescriptionInvalidCharacters},
		{name: "carriage return", description: "line one\r\nline two", wantErr: ErrDescriptionInvalidCharacters},
		{name: "tab", description: "a\tb", wantErr: ErrDescriptionInvalidCharacters},
		{name: "delete char", description: "a\x7fb", wantErr: ErrDescriptionInvalidCharacters},
		{name: "too long", description: strings.Repeat("a", maxDescriptionLength+1), wantErr: ErrDescriptionTooLong},
		{name: "max length", description: strings.Repeat("a", maxDescriptionLength)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Description(tt.description); !errors.Is(got, tt.wantErr) {
				t.Errorf("Description() = %v, want %v", got, tt.wantErr)
			}
		})
	}
}

func TestDisplayName(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		wantErr     error
	}{
		{name: "valid", displayName: "Jane Doe"},
		{name: "empty", displayName: "", wantErr: ErrDisplayNameLength},
		{name: "newline", displayName: "Jane\nDoe", wantErr: ErrDisplayNameInvalidCharacters},
		{name: "tab", displayName: "Jane\tDoe", wantErr: ErrDisplayNameInvalidCharacters},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DisplayName(tt.displayName); !errors.Is(got, tt.wantErr) {
				t.Errorf("DisplayName() = %v, want %v", got, tt.wantErr)
			}
		})
	}
}
