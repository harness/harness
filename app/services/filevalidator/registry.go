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

// Package filevalidator validates the content of files based on their path.
package filevalidator

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/harness/gitness/types"
)

// Validator validates the content of the files at the paths it claims.
type Validator interface {
	// Paths returns the repository paths this validator applies to.
	Paths() []string
	// Validate validates the content of a claimed file.
	Validate(ctx context.Context, repo *types.RepositoryCore, content []byte) ([]types.ContentViolation, error)
}

// Registry routes file content to the validator registered for its path.
type Registry struct {
	byPath map[string]Validator
}

// NewRegistry creates a registry. It fails if two different validators claim the same (normalized) path.
// One validator repeating a path is accepted and deduplicated: its path list comes from operator
// configuration, where a redundant entry must not stop the server from starting.
func NewRegistry(validators ...Validator) (*Registry, error) {
	byPath := make(map[string]Validator)
	for _, v := range validators {
		for _, path := range v.Paths() {
			normalized := NormalizePath(path)
			if taken, ok := byPath[normalized]; ok && taken != v {
				return nil, fmt.Errorf("path %q is claimed by more than one content validator", normalized)
			}
			byPath[normalized] = v
		}
	}
	return &Registry{byPath: byPath}, nil
}

// NormalizePath trims a single leading "./" and then any leading "/". Matching is otherwise case-sensitive.
func NormalizePath(path string) string {
	path = strings.TrimPrefix(path, "./")
	return strings.TrimLeft(path, "/")
}

// Handles returns true if a validator is registered for the path. A nil registry handles nothing.
func (r *Registry) Handles(path string) bool {
	if r == nil {
		return false
	}
	_, ok := r.byPath[NormalizePath(path)]
	return ok
}

// Validate validates content with the validator registered for path. It returns nil, nil if no validator
// claims the path. A nil registry validates nothing.
func (r *Registry) Validate(
	ctx context.Context,
	repo *types.RepositoryCore,
	path string,
	content []byte,
) ([]types.ContentViolation, error) {
	if r == nil {
		return nil, nil //nolint:nilnil // no validator means nothing to report.
	}
	v, ok := r.byPath[NormalizePath(path)]
	if !ok {
		return nil, nil //nolint:nilnil // no validator means nothing to report.
	}
	violations, err := v.Validate(ctx, repo, content)
	if err != nil {
		return nil, fmt.Errorf("failed to validate %q: %w", NormalizePath(path), err)
	}
	return violations, nil
}

// Paths returns every normalized path that has a validator, sorted. A nil registry has none.
func (r *Registry) Paths() []string {
	if r == nil {
		return nil
	}
	paths := make([]string, 0, len(r.byPath))
	for path := range r.byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
