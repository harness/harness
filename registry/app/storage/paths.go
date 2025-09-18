// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"fmt"
	"path"
	"strings"

	a "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"

	"github.com/opencontainers/go-digest"
)

const (
	storagePathRoot = "/"
	docker          = "docker"
	blobs           = "blobs"
)

// PackageType constants using iota.
const (
	PackageTypeDOCKER = iota
)

func pathFor(spec pathSpec) (string, error) {
	rootPrefix := []string{storagePathRoot}
	switch v := spec.(type) {
	case blobsPathSpec:
		blobsPathPrefix := rootPrefix
		blobsPathPrefix = append(blobsPathPrefix, blobs)
		return path.Join(blobsPathPrefix...), nil
	case blobPathSpec:
		components, err := digestPathComponents(v.digest, true)
		if err != nil {
			return "", err
		}
		blobPathPrefix := rootPrefix
		blobPathPrefix = append(blobPathPrefix, v.path, docker, blobs)
		return path.Join(append(blobPathPrefix, components...)...), nil
	case blobDataPathSpec:
		components, err := digestPathComponents(v.digest, true)
		if err != nil {
			return "", err
		}

		components = append(components, "data")
		blobPathPrefix := rootPrefix
		blobPathPrefix = append(blobPathPrefix, v.path, docker, "blobs")
		return path.Join(append(blobPathPrefix, components...)...), nil

	case uploadDataPathSpec:
		return path.Join(append(rootPrefix, v.path, docker, "_uploads", v.repoName, v.id, "data")...), nil
	case uploadHashStatePathSpec:
		offset := fmt.Sprintf("%d", v.offset)
		if v.list {
			offset = "" // Limit to the prefix for listing offsets.
		}
		return path.Join(
			append(
				rootPrefix, v.path, docker, "_uploads", v.repoName, v.id, "hashstates",
				string(v.alg), offset,
			)...,
		), nil
	case repositoriesRootPathSpec:
		return path.Join(rootPrefix...), nil
	case uploadFilePathSpec:
		return path.Join(append(rootPrefix, v.path)...), nil
	default:
		return "", fmt.Errorf("unknown path spec: %#v", v)
	}
}

// pathSpec is a type to mark structs as path specs. There is no
// implementation because we'd like to keep the specs and the mappers
// decoupled.
type pathSpec interface {
	pathSpec()
}

// blobAlgorithmReplacer does some very simple path sanitization for user
// input. Paths should be "safe" before getting this far due to strict digest
// requirements but we can add further path conversion here, if needed.
var blobAlgorithmReplacer = strings.NewReplacer(
	"+", "/",
	".", "/",
	";", "/",
)

// blobsPathSpec contains the path for the blobs directory.
type blobsPathSpec struct{}

func (blobsPathSpec) pathSpec() {}

// blobPathSpec contains the path for the registry global blob store.
type blobPathSpec struct {
	digest digest.Digest
	path   string
}

func (blobPathSpec) pathSpec() {}

// blobDataPathSpec contains the path for the StorageService global blob store. For
// now, this contains layer data, exclusively.
type blobDataPathSpec struct {
	digest digest.Digest
	path   string
}

func (blobDataPathSpec) pathSpec() {}

// uploadDataPathSpec defines the path parameters of the data file for
// uploads.
type uploadDataPathSpec struct {
	path     string
	repoName string
	id       string
}

func (uploadDataPathSpec) pathSpec() {}

// uploadDataPathSpec defines the path parameters of the data file for
// uploads.
type uploadFilePathSpec struct {
	path string
}

func (uploadFilePathSpec) pathSpec() {}

// uploadHashStatePathSpec defines the path parameters for the file that stores
// the hash function state of an upload at a specific byte offset. If `list` is
// set, then the path mapper will generate a list prefix for all hash state
// offsets for the upload identified by the name, id, and alg.
type uploadHashStatePathSpec struct {
	path     string
	repoName string
	id       string
	alg      digest.Algorithm
	offset   int64
	list     bool
}

func (uploadHashStatePathSpec) pathSpec() {}

// repositoriesRootPathSpec returns the root of repositories.
type repositoriesRootPathSpec struct{}

func (repositoriesRootPathSpec) pathSpec() {}

// digestPathComponents provides a consistent path breakdown for a given
// digest. For a generic digest, it will be as follows:
//
//	<algorithm>/<hex digest>
//
// If multilevel is true, the first two bytes of the digest will separate
// groups of digest folder. It will be as follows:
//
//	<algorithm>/<first two bytes of digest>/<full digest>
func digestPathComponents(dgst digest.Digest, multilevel bool) ([]string, error) {
	if err := dgst.Validate(); err != nil {
		return nil, err
	}

	algorithm := blobAlgorithmReplacer.Replace(string(dgst.Algorithm()))
	hex := dgst.Encoded()
	prefix := []string{algorithm}

	var suffix []string

	if multilevel {
		suffix = append(suffix, hex[:2])
	}

	suffix = append(suffix, hex)

	return append(prefix, suffix...), nil
}

// BlobPath returns the path for a blob based on the package type.
func BlobPath(acctID string, packageType string, sha256 string) (string, error) {
	// sample =  sha256:50f564aff30aeb53eb88b0eb2c2ba59878e9854681989faa5ff7396bdfaf509b
	sha256 = strings.TrimPrefix(sha256, "sha256:")
	sha256Prefix := sha256[:2]

	switch packageType {
	case string(a.PackageTypeDOCKER):
		acctID = strings.ToLower(acctID) // lowercase for OCI compliance
		// format: /accountId(lowercase)/docker/blobs/sha256/(2 character prefix of sha)/sha/data
		return fmt.Sprintf("/%s/docker/blobs/sha256/%s/%s/data", acctID, sha256Prefix, sha256), nil
	default:
		return fmt.Sprintf("/%s/files/%s", acctID, sha256), nil
	}
}
