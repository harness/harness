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

package repo

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
)

func TestCreateGitRepository_InvalidGitIgnore(t *testing.T) {
	c := &Controller{}
	session := &auth.Session{}

	tests := []struct {
		name      string
		gitIgnore string
	}{
		{name: "nonexistent template", gitIgnore: "nonexistent-xyz"},
		{name: "README.md is not a gitignore", gitIgnore: "README.md"},
		{name: "CONTRIBUTING.md is not a gitignore", gitIgnore: "CONTRIBUTING.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := c.createGitRepository(
				context.Background(),
				session,
				"test-repo",
				"",
				"main",
				CreateFileOptions{GitIgnore: tt.gitIgnore},
			)
			if err == nil {
				t.Fatalf("expected error for gitignore %q, got nil", tt.gitIgnore)
			}

			var userErr *usererror.Error
			if !errors.As(err, &userErr) {
				t.Fatalf("expected usererror.Error, got %T: %v", err, err)
			}
			if userErr.Status != http.StatusBadRequest {
				t.Errorf("expected HTTP 400, got %d", userErr.Status)
			}
			if !strings.Contains(userErr.Message, "Unsupported gitignore template") {
				t.Errorf("expected message to contain %q, got %q",
					"Unsupported gitignore template", userErr.Message)
			}
		})
	}
}

func TestCreateGitRepository_InvalidLicense(t *testing.T) {
	c := &Controller{}
	session := &auth.Session{}

	tests := []struct {
		name    string
		license string
	}{
		{name: "nonexistent license", license: "nonexistent-xyz"},
		{name: "random string", license: "not-a-real-license"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := c.createGitRepository(
				context.Background(),
				session,
				"test-repo",
				"",
				"main",
				CreateFileOptions{License: tt.license},
			)
			if err == nil {
				t.Fatalf("expected error for license %q, got nil", tt.license)
			}

			var userErr *usererror.Error
			if !errors.As(err, &userErr) {
				t.Fatalf("expected usererror.Error, got %T: %v", err, err)
			}
			if userErr.Status != http.StatusBadRequest {
				t.Errorf("expected HTTP 400, got %d", userErr.Status)
			}
			if !strings.Contains(userErr.Message, "Unsupported license template") {
				t.Errorf("expected message to contain %q, got %q",
					"Unsupported license template", userErr.Message)
			}
		})
	}
}
