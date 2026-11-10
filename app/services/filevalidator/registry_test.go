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

package filevalidator

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type stubValidator struct {
	paths []string
	out   []types.ContentViolation
	err   error
	calls int
}

func (s *stubValidator) Paths() []string { return s.paths }

func (s *stubValidator) Validate(
	_ context.Context,
	_ *types.RepositoryCore,
	_ []byte,
) ([]types.ContentViolation, error) {
	s.calls++
	return s.out, s.err
}

func codeOwnersStub() *stubValidator {
	return &stubValidator{
		paths: []string{"CODEOWNERS", ".harness/CODEOWNERS"},
		out:   []types.ContentViolation{{Code: "user_not_found", Message: "x", Line: 1}},
	}
}

func TestNormalizePath(t *testing.T) {
	require.Equal(t, ".harness/CODEOWNERS", NormalizePath(".harness/CODEOWNERS"))
	require.Equal(t, ".harness/CODEOWNERS", NormalizePath("./.harness/CODEOWNERS"))
	require.Equal(t, ".harness/CODEOWNERS", NormalizePath("/.harness/CODEOWNERS"))
	require.Equal(t, "codeowners", NormalizePath("codeowners"))
}

func TestRegistry_ValidateRoutesByPath(t *testing.T) {
	stub := codeOwnersStub()
	reg, err := NewRegistry(stub)
	require.NoError(t, err)

	for _, path := range []string{"CODEOWNERS", ".harness/CODEOWNERS", "./CODEOWNERS"} {
		got, err := reg.Validate(context.Background(), &types.RepositoryCore{}, path, []byte("x"))
		require.NoError(t, err, path)
		require.Equal(t, stub.out, got, path)
	}
}

func TestRegistry_ValidateIgnoresUnclaimedPaths(t *testing.T) {
	stub := codeOwnersStub()
	reg, err := NewRegistry(stub)
	require.NoError(t, err)

	for _, path := range []string{"CODEOWNERS.md", "docs/CODEOWNERS", "codeowners", "src/main.go", ""} {
		got, err := reg.Validate(context.Background(), &types.RepositoryCore{}, path, []byte("x"))
		require.NoError(t, err, path)
		require.Nil(t, got, path)
	}
	require.Zero(t, stub.calls)
}

func TestRegistry_HandlesMatchesValidate(t *testing.T) {
	reg, err := NewRegistry(codeOwnersStub())
	require.NoError(t, err)

	require.True(t, reg.Handles("CODEOWNERS"))
	require.True(t, reg.Handles("./.harness/CODEOWNERS"))
	require.False(t, reg.Handles("docs/CODEOWNERS"))
	require.False(t, reg.Handles(""))
}

func TestRegistry_ValidateWrapsValidatorError(t *testing.T) {
	boom := errors.New("boom")
	reg, err := NewRegistry(&stubValidator{paths: []string{"CODEOWNERS"}, err: boom})
	require.NoError(t, err)

	_, err = reg.Validate(context.Background(), &types.RepositoryCore{}, "CODEOWNERS", nil)
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "CODEOWNERS")
}

func TestNewRegistry_RejectsDuplicatePaths(t *testing.T) {
	_, err := NewRegistry(
		&stubValidator{paths: []string{"CODEOWNERS"}},
		&stubValidator{paths: []string{"CODEOWNERS"}},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "CODEOWNERS")

	_, err = NewRegistry(
		&stubValidator{paths: []string{"./CODEOWNERS"}},
		&stubValidator{paths: []string{"CODEOWNERS"}},
	)
	require.Error(t, err)
}

// A redundant entry in the operator-settable path list (GITNESS_CODEOWNERS_FILEPATH) must not stop the
// server from booting: one validator claiming the same path twice is a no-op, not a conflict.
func TestNewRegistry_AcceptsRepeatedPathsFromSameValidator(t *testing.T) {
	validator := &stubValidator{paths: []string{"CODEOWNERS", "./CODEOWNERS", "/CODEOWNERS", ".harness/CODEOWNERS"}}

	reg, err := NewRegistry(validator)

	require.NoError(t, err)
	require.Equal(t, []string{".harness/CODEOWNERS", "CODEOWNERS"}, reg.Paths())
}

func TestRegistry_ValidateNilRegistryIsNoop(t *testing.T) {
	var reg *Registry
	got, err := reg.Validate(context.Background(), &types.RepositoryCore{}, "CODEOWNERS", []byte("x"))
	require.NoError(t, err)
	require.Nil(t, got)
	require.False(t, reg.Handles("CODEOWNERS"))
}

func TestRegistry_PathsListsEveryClaimedPathSorted(t *testing.T) {
	registry, err := NewRegistry(
		&stubValidator{paths: []string{"b/OWNERS", "./A"}},
		&stubValidator{paths: []string{"c"}},
	)
	require.NoError(t, err)
	require.Equal(t, []string{"A", "b/OWNERS", "c"}, registry.Paths())

	var nilRegistry *Registry
	require.Empty(t, nilRegistry.Paths())
}
