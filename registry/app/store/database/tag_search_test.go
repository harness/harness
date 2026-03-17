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

package database

import (
	"strings"
	"testing"

	"github.com/harness/gitness/registry/types"
	databaseg "github.com/harness/gitness/store/database"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildOciVersionsSearchQuery replicates the search filtering logic from
// GetAllOciVersionsByRepoAndImage to allow unit testing the generated SQL.
func buildOciVersionsSearchQuery(search string) (string, []interface{}, error) {
	q := databaseg.Builder.Select("m.manifest_digest").
		From("manifests m").
		LeftJoin("tags t ON m.manifest_id = t.tag_manifest_id").
		Join("registries r ON m.manifest_registry_id = r.registry_id").
		Join("images i ON i.image_registry_id = r.registry_id AND i.image_name = m.manifest_image_name").
		Where("r.registry_parent_id = ? AND r.registry_name = ? AND m.manifest_image_name = ?",
			int64(1), "myrepo", "myimage").
		Where("i.image_deleted_at IS NULL")

	if search != "" {
		digestBytes, err := types.GetDigestBytes(digest.Digest(search))
		if err == nil {
			q = q.Where("m.manifest_digest = ?", digestBytes)
		} else {
			q = q.Where(
				"EXISTS (SELECT 1 FROM tags st WHERE st.tag_manifest_id = m.manifest_id AND st.tag_name LIKE ?)",
				sqlPartialMatch(search),
			)
		}
	}

	q = q.GroupBy("m.manifest_digest")

	return q.ToSql()
}

// buildCountOciVersionsSearchQuery replicates the search filtering logic from
// CountOciVersionByRepoAndImage to allow unit testing the generated SQL.
func buildCountOciVersionsSearchQuery(search string) (string, []interface{}, error) {
	stmt := databaseg.Builder.Select("COUNT(*)").
		From("manifests m").
		Join("registries r ON m.manifest_registry_id = r.registry_id").
		Join("images i ON i.image_registry_id = r.registry_id AND i.image_name = m.manifest_image_name").
		Where("r.registry_parent_id = ? AND r.registry_name = ? AND m.manifest_image_name = ?",
			int64(1), "myrepo", "myimage").
		Where("i.image_deleted_at IS NULL")

	if search != "" {
		digestBytes, err := types.GetDigestBytes(digest.Digest(search))
		if err == nil {
			stmt = stmt.Where("m.manifest_digest = ?", digestBytes)
		} else {
			stmt = stmt.Where(
				"EXISTS (SELECT 1 FROM tags st WHERE st.tag_manifest_id = m.manifest_id AND st.tag_name LIKE ?)",
				sqlPartialMatch(search),
			)
		}
	}

	return stmt.ToSql()
}

func TestOciVersionsSearch_TagName(t *testing.T) {
	sql, args, err := buildOciVersionsSearchQuery("latest")
	require.NoError(t, err)

	// Should contain tag search EXISTS subquery (with positional param $4).
	assert.Contains(t, sql,
		"EXISTS (SELECT 1 FROM tags st WHERE st.tag_manifest_id = m.manifest_id AND st.tag_name LIKE $4)")

	// Should NOT contain manifest_digest filter.
	assert.NotContains(t, sql, "m.manifest_digest =")

	// Last arg should be the search pattern.
	require.NotEmpty(t, args)
	lastArg := args[len(args)-1]
	assert.Equal(t, sqlPartialMatch("latest"), lastArg)
}

func TestOciVersionsSearch_Digest(t *testing.T) {
	validDigest := "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sql, args, err := buildOciVersionsSearchQuery(validDigest)
	require.NoError(t, err)

	// Should contain digest filter.
	assert.Contains(t, sql, "m.manifest_digest = $4")

	// Should NOT contain tag search.
	assert.NotContains(t, sql, "EXISTS (SELECT 1 FROM tags st")

	// Last arg should be digest bytes.
	require.NotEmpty(t, args)
	lastArg := args[len(args)-1]
	_, ok := lastArg.([]byte)
	assert.True(t, ok, "last arg should be []byte for digest search")
}

func TestOciVersionsSearch_Empty(t *testing.T) {
	sql, args, err := buildOciVersionsSearchQuery("")
	require.NoError(t, err)

	// Should not have any search filter.
	assert.NotContains(t, sql, "LIKE")
	assert.NotContains(t, sql, "manifest_digest =")
	assert.NotContains(t, sql, "EXISTS")

	// Should only have the base args (parentID, repoKey, image).
	assert.Len(t, args, 3)
}

func TestOciVersionsSearch_PartialTagName(t *testing.T) {
	sql, _, err := buildOciVersionsSearchQuery("v1")
	require.NoError(t, err)

	// Partial tag names should trigger tag search, not digest search.
	assert.Contains(t, sql, "st.tag_name LIKE")
	assert.NotContains(t, sql, "m.manifest_digest =")
}

func TestOciVersionsSearch_SpecialCharsEscaped(t *testing.T) {
	_, args, err := buildOciVersionsSearchQuery("my_tag%test")
	require.NoError(t, err)

	lastArg := args[len(args)-1]
	searchVal, ok := lastArg.(string)
	require.True(t, ok)

	// Special chars should be escaped.
	assert.Contains(t, searchVal, `\_`)
	assert.Contains(t, searchVal, `\%`)
	assert.True(t, strings.HasPrefix(searchVal, "%"))
	assert.True(t, strings.HasSuffix(searchVal, "%"))
}

func TestCountOciVersionsSearch_TagName(t *testing.T) {
	sql, args, err := buildCountOciVersionsSearchQuery("v1.0")
	require.NoError(t, err)

	assert.Contains(t, sql, "SELECT COUNT(*)")
	assert.Contains(t, sql,
		"EXISTS (SELECT 1 FROM tags st WHERE st.tag_manifest_id = m.manifest_id AND st.tag_name LIKE $4)")
	assert.NotContains(t, sql, "m.manifest_digest =")

	require.NotEmpty(t, args)
	lastArg := args[len(args)-1]
	assert.Equal(t, sqlPartialMatch("v1.0"), lastArg)
}

func TestCountOciVersionsSearch_Digest(t *testing.T) {
	validDigest := "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sql, _, err := buildCountOciVersionsSearchQuery(validDigest)
	require.NoError(t, err)

	assert.Contains(t, sql, "SELECT COUNT(*)")
	assert.Contains(t, sql, "m.manifest_digest = $4")
	assert.NotContains(t, sql, "EXISTS (SELECT 1 FROM tags st")
}

func TestCountOciVersionsSearch_Empty(t *testing.T) {
	sql, args, err := buildCountOciVersionsSearchQuery("")
	require.NoError(t, err)

	assert.Contains(t, sql, "SELECT COUNT(*)")
	assert.NotContains(t, sql, "LIKE")
	assert.NotContains(t, sql, "manifest_digest = ?")
	assert.Len(t, args, 3)
}
