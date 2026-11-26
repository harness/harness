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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestValidationError(t *testing.T) {
	err := NewValidationError("boom")
	require.Equal(t, "boom", err.Error())

	errf := NewValidationErrorf("bad %s %d", "value", 7)
	require.Equal(t, "bad value 7", errf.Error())

	// any validation error matches the ErrAny sentinel
	require.True(t, errors.Is(err, ErrAny))
	require.True(t, errors.Is(err, NewValidationError("boom")))
	// non validation errors never match
	require.False(t, errors.Is(err, errors.New("boom")))

	// wrapped validation errors are still recognized
	require.True(t, errors.Is(fmt.Errorf("wrapped: %w", err), ErrAny))
}

func TestIdentifier(t *testing.T) {
	require.NoError(t, Identifier("valid-name_1.0"))
	require.ErrorIs(t, Identifier(""), ErrIdentifierLength)
	require.ErrorIs(t, Identifier(strings.Repeat("a", MaxIdentifierLength+1)), ErrIdentifierLength)
	require.NoError(t, Identifier(strings.Repeat("a", MaxIdentifierLength)))
	require.ErrorIs(t, Identifier("has space"), ErrIdentifierRegex)
	require.ErrorIs(t, Identifier("slash/inside"), ErrIdentifierRegex)
}

func TestRepoIdentifierDefault(t *testing.T) {
	require.NoError(t, RepoIdentifierDefault("my-repo", nil))
	require.ErrorIs(t, RepoIdentifierDefault("", nil), ErrIdentifierLength)
	require.ErrorIs(t, RepoIdentifierDefault("repo.git", nil), ErrIllegalRepoSpaceIdentifierSuffix)
	require.ErrorIs(t, RepoIdentifierDefault("REPO.GIT", nil), ErrIllegalRepoSpaceIdentifierSuffix)
	require.NoError(t, RepoIdentifierDefault("repo.github", nil))
}

func TestPrincipalUIDDefault(t *testing.T) {
	require.NoError(t, PrincipalUIDDefault("jane"))
	require.ErrorIs(t, PrincipalUIDDefault(""), ErrIdentifierLength)
	require.ErrorIs(t, PrincipalUIDDefault("bad uid"), ErrIdentifierRegex)
	require.ErrorIs(t, PrincipalUIDDefault(types.AnonymousPrincipalUID), ErrIllegalPrincipalUID)
	require.ErrorIs(t, PrincipalUIDDefault(strings.ToUpper(types.AnonymousPrincipalUID)), ErrIllegalPrincipalUID)
}

func TestSpaceIdentifierDefault(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		isRoot     bool
		want       error
	}{
		{"valid root", "acme", true, nil},
		{"valid child", "child", false, nil},
		{"invalid chars", "no space", false, ErrIdentifierRegex},
		{"git suffix", "space.git", false, ErrIllegalRepoSpaceIdentifierSuffix},
		{"numeric root", "12345", true, ErrIllegalRootSpaceIdentifierNumber},
		{"numeric child is allowed", "12345", false, nil},
		{"reserved api", "api", true, ErrIllegalRootSpaceIdentifier},
		{"reserved git case insensitive", "GIT", true, ErrIllegalRootSpaceIdentifier},
		{"reserved name as child is allowed", "api", false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SpaceIdentifierDefault(tt.identifier, tt.isRoot)
			if tt.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestEmail(t *testing.T) {
	require.NoError(t, Email("a@b.c"))
	require.ErrorIs(t, Email(""), ErrEmailLen)
	require.ErrorIs(t, Email(strings.Repeat("a", maxEmailLength+1)), ErrEmailLen)
	require.NoError(t, Email(strings.Repeat("a", maxEmailLength)))
}

func TestPassword(t *testing.T) {
	require.NoError(t, Password("secret"))
	require.ErrorIs(t, Password(""), ErrPasswordLength)
	require.ErrorIs(t, Password(strings.Repeat("a", maxPasswordLength+1)), ErrPasswordLength)
	require.NoError(t, Password(strings.Repeat("a", maxPasswordLength)))
}

func TestPath(t *testing.T) {
	deep := func(segments int) string {
		parts := make([]string, segments)
		for i := range parts {
			parts[i] = fmt.Sprintf("s%d", i+1)
		}
		return strings.Join(parts, "/")
	}

	tests := []struct {
		name    string
		path    string
		isSpace bool
		wantErr error
	}{
		{"valid space", "root/child", true, nil},
		{"valid repo", "root/child/repo", false, nil},
		{"empty", "", true, ErrPathEmpty},
		{"leading separator", "/root", true, ErrPathCantBeginOrEndWithSeparator},
		{"trailing separator", "root/", true, ErrPathCantBeginOrEndWithSeparator},
		{"empty segment", "root//child", true, ErrEmptyPathSegment},
		{"max space depth", deep(MaxSpacePathDepth), true, nil},
		{"space too deep", deep(MaxSpacePathDepth + 1), true, ErrPathInvalidDepth},
		{"repo one deeper than space", deep(MaxRepoPathDepth), false, nil},
		{"repo too deep", deep(MaxRepoPathDepth + 1), false, ErrPathInvalidDepth},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Path(tt.path, tt.isSpace, SpaceIdentifierDefault)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}

	t.Run("invalid segment is reported", func(t *testing.T) {
		err := Path("root/bad segment", true, SpaceIdentifierDefault)
		require.ErrorIs(t, err, ErrIdentifierRegex)
		require.Contains(t, err.Error(), "bad segment")
	})

	t.Run("only first segment is root", func(t *testing.T) {
		err := Path("api/child", true, SpaceIdentifierDefault)
		require.ErrorIs(t, err, ErrIllegalRootSpaceIdentifier)
		require.NoError(t, Path("root/api", true, SpaceIdentifierDefault))
	})
}

func TestPathDepth(t *testing.T) {
	require.NoError(t, PathDepth("a/b", true))
	require.ErrorIs(t, PathDepth(strings.Repeat("a/", MaxSpacePathDepth)+"a", true), ErrPathInvalidDepth)

	require.False(t, IsPathTooDeep(strings.Repeat("a/", MaxSpacePathDepth-1)+"a", true))
	require.True(t, IsPathTooDeep(strings.Repeat("a/", MaxSpacePathDepth)+"a", true))
	require.False(t, IsPathTooDeep(strings.Repeat("a/", MaxRepoPathDepth-1)+"a", false))
	require.True(t, IsPathTooDeep(strings.Repeat("a/", MaxRepoPathDepth)+"a", false))
}

func TestServiceAccountParent(t *testing.T) {
	require.NoError(t, ServiceAccountParent(enum.ParentResourceTypeRepo, 1))
	require.NoError(t, ServiceAccountParent(enum.ParentResourceTypeSpace, 1))
	require.ErrorIs(t, ServiceAccountParent("other", 1), ErrServiceAccountParentTypeIsInvalid)
	require.ErrorIs(t, ServiceAccountParent(enum.ParentResourceTypeSpace, 0), ErrServiceAccountParentIDInvalid)
	require.ErrorIs(t, ServiceAccountParent(enum.ParentResourceTypeRepo, -3), ErrServiceAccountParentIDInvalid)
}

func TestTokenLifetime(t *testing.T) {
	ptr := func(d time.Duration) *time.Duration { return &d }

	require.ErrorIs(t, TokenLifetime(nil, false), ErrTokenLifeTimeRequired)
	require.NoError(t, TokenLifetime(nil, true))
	require.NoError(t, TokenLifetime(ptr(minTokenLifeTime), false))
	require.NoError(t, TokenLifetime(ptr(maxTokenLifeTime), false))
	require.ErrorIs(t, TokenLifetime(ptr(minTokenLifeTime-time.Second), false), ErrTokenLifeTimeOutOfBounds)
	require.ErrorIs(t, TokenLifetime(ptr(maxTokenLifeTime+time.Second), true), ErrTokenLifeTimeOutOfBounds)
}

func TestWireProviders(t *testing.T) {
	require.NoError(t, ProvideSpaceIdentifierCheck()("space", true))
	require.Error(t, ProvideSpaceIdentifierCheck()("api", true))
	require.NoError(t, ProvidePrincipalUIDCheck()("jane"))
	require.Error(t, ProvidePrincipalUIDCheck()(""))
	require.NoError(t, ProvideRepoIdentifierCheck()("repo", nil))
	require.Error(t, ProvideRepoIdentifierCheck()("repo.git", nil))
}
