// Copyright 2026 Harness, Inc.
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

package codeowners

import (
	"context"
	"reflect"
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func TestContentValidator_PathsComeFromConfig(t *testing.T) {
	paths := []string{"CODEOWNERS", ".harness/CODEOWNERS"}
	s := &Service{config: Config{FilePaths: paths}}

	if got := NewContentValidator(s).Paths(); !reflect.DeepEqual(got, paths) {
		t.Fatalf("Paths() = %v, want %v", got, paths)
	}
}

func TestContentValidator_MapsViolations(t *testing.T) {
	v := NewContentValidator(newValidationService(&stubPrincipalStore{}))

	got, err := v.Validate(context.Background(), &types.RepositoryCore{}, []byte("* ghost@harness.io\n"))
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Validate() = %+v, want one violation", got)
	}
	if got[0].Code != string(enum.CodeOwnerViolationCodeUserNotFound) || got[0].Line != 1 || got[0].Message == "" {
		t.Fatalf("Validate() = %+v, want user_not_found at line 1 with a message", got[0])
	}
}
