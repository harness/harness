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

package storage

import (
	"strings"
	"testing"

	a "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRsg_PathForBlobPaths(t *testing.T) {
	dgst, err := digest.Parse("sha256:" + rsgGenericSHA)
	require.NoError(t, err)

	p, err := pathFor(blobsPathSpec{})
	require.NoError(t, err)
	assert.Equal(t, "/blobs", p)

	p, err = pathFor(blobPathSpec{digest: dgst, path: rsgAcctPrefix})
	require.NoError(t, err)
	assert.Contains(t, p, "docker/blobs/sha256")
	assert.Contains(t, p, rsgGenericSHA)

	p, err = pathFor(blobDataPathSpec{digest: dgst, path: rsgAcctPrefix})
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(p, "/data"))

	p, err = pathFor(globalBlobPathSpec{digest: dgst})
	require.NoError(t, err)
	assert.Contains(t, p, "/blobs/sha256")
}

func TestRsg_PathForUploadPaths(t *testing.T) {
	p, err := pathFor(uploadDataPathSpec{path: rsgAcctPrefix, repoName: rsgRepoName, id: "uid1"})
	require.NoError(t, err)
	assert.Contains(t, p, "_uploads")
	assert.Contains(t, p, rsgRepoName)

	p, err = pathFor(genericUploadDataPathSpec{rootIdentifier: rsgGenericRoot, id: "tid"})
	require.NoError(t, err)
	assert.Contains(t, p, "tmp/tid")

	p, err = pathFor(genericDataPathSpec{rootIdentifier: rsgGenericRoot, sha256: rsgGenericSHA})
	require.NoError(t, err)
	assert.Contains(t, p, "files/"+rsgGenericSHA)

	p, err = pathFor(globalUploadDataPathSpec{id: "gid"})
	require.NoError(t, err)
	assert.Equal(t, "/_uploads/gid/data", p)
}

func TestRsg_PathForHashStatePaths(t *testing.T) {
	p, err := pathFor(uploadHashStatePathSpec{
		path: rsgAcctPrefix, repoName: rsgRepoName, id: "u1",
		alg: digest.SHA256, offset: 42,
	})
	require.NoError(t, err)
	assert.Contains(t, p, "hashstates/sha256/42")

	p, err = pathFor(uploadHashStatePathSpec{
		path: rsgAcctPrefix, repoName: rsgRepoName, id: "u1",
		alg: digest.SHA256, list: true,
	})
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(p, "hashstates/sha256"))

	p, err = pathFor(globalUploadHashStatePathSpec{id: "g1", alg: digest.SHA512, offset: 7})
	require.NoError(t, err)
	assert.Contains(t, p, "/_uploads/g1/hashstates/sha512/7")
}

func TestRsg_PathForOtherSpecs(t *testing.T) {
	p, err := pathFor(repositoriesRootPathSpec{})
	require.NoError(t, err)
	assert.Equal(t, "/", p)

	p, err = pathFor(uploadFilePathSpec{path: "/custom/path"})
	require.NoError(t, err)
	assert.Equal(t, "/custom/path", p)

	_, err = pathFor(rsgBadPathSpec{})
	require.Error(t, err)
}

func TestRsg_DigestPathComponentsInvalid(t *testing.T) {
	_, err := digestPathComponents("not-a-digest", 1)
	require.Error(t, err)
}

func TestRsg_DigestPathComponentsLevels(t *testing.T) {
	dgst, err := digest.Parse("sha256:" + rsgGenericSHA)
	require.NoError(t, err)

	parts, err := digestPathComponents(dgst, 0)
	require.NoError(t, err)
	assert.Len(t, parts, 2)
	assert.Equal(t, "sha256", parts[0])

	parts, err = digestPathComponents(dgst, 2)
	require.NoError(t, err)
	assert.Len(t, parts, 4)
	assert.Equal(t, rsgGenericSHA[:2], parts[1])
	assert.Equal(t, rsgGenericSHA[2:4], parts[2])
}

func TestRsg_BlobPathPackageTypes(t *testing.T) {
	p, err := BlobPath("Acct", string(a.PackageTypeDOCKER), "sha256:"+rsgGenericSHA)
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(p), "acct/docker/blobs")
	assert.Contains(t, p, rsgGenericSHA[:2])

	p, err = BlobPath(rsgGenericRoot, "maven", rsgGenericSHA)
	require.NoError(t, err)
	assert.Equal(t, "/"+rsgGenericRoot+"/files/"+rsgGenericSHA, p)
}

func TestRsg_PathFn(t *testing.T) {
	dgst := rsgHelloDigest(t)
	p, err := PathFn(rsgAcctPrefix, dgst)
	require.NoError(t, err)
	assert.Contains(t, p, rsgAcctPrefix)
	assert.Contains(t, p, dgst.Encoded())
}

type rsgBadPathSpec struct{}

func (rsgBadPathSpec) pathSpec() {}
