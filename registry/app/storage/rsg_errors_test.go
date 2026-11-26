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
	"errors"
	"fmt"
	"testing"

	"github.com/harness/gitness/registry/app/manifest"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

func TestRsg_StorageErrorsMessages(t *testing.T) {
	assert.Equal(t, "access denied", ErrAccessDenied.Error())
	assert.Equal(t, "operation unsupported", ErrUnsupported.Error())

	tagErr := TagUnknownError{Tag: "latest"}
	assert.Contains(t, tagErr.Error(), "unknown tag=latest")

	regErr := RegistryUnknownError{Name: "repo"}
	assert.Contains(t, regErr.Error(), "unknown registry name=repo")

	nameErr := RegistryNameInvalidError{Name: "bad", Reason: fmt.Errorf("reason")}
	assert.Contains(t, nameErr.Error(), `registry name "bad" invalid`)

	manErr := ManifestUnknownError{Name: "repo", Tag: "t1"}
	assert.Contains(t, manErr.Error(), "unknown manifest")

	rev := digest.Digest("sha256:" + rsgGenericSHA)
	revErr := ManifestUnknownRevisionError{Name: "repo", Revision: rev}
	assert.Contains(t, revErr.Error(), "unknown manifest")

	assert.Equal(t, "unverified manifest", ManifestUnverifiedError{}.Error())

	refErr := ManifestReferencesExceedLimitError{References: 10, Limit: 5}
	assert.Contains(t, refErr.Error(), "exceed reference limit")

	sizeErr := ManifestPayloadSizeExceedsLimitError{PayloadSize: 100, Limit: 50}
	assert.Contains(t, sizeErr.Error(), "exceeds limit")

	multi := ManifestVerificationErrors{errors.New("a"), errors.New("b")}
	assert.Contains(t, multi.Error(), "errors verifying manifest")

	blobErr := ManifestBlobUnknownError{Digest: rev}
	assert.Contains(t, blobErr.Error(), "unknown blob")

	nameInv := ManifestNameInvalidError{Name: "x", Reason: fmt.Errorf("bad")}
	assert.Contains(t, nameInv.Error(), `manifest name "x" invalid`)
}

func TestRsg_BlobErrorsMessages(t *testing.T) {
	assert.Equal(t, "blob exists", ErrBlobExists.Error())
	assert.Equal(t, "unknown blob", ErrBlobUnknown.Error())

	dgst := digest.FromBytes([]byte(rsgPayload))
	digErr := BlobInvalidDigestError{Digest: dgst, Reason: fmt.Errorf("mismatch")}
	assert.Contains(t, digErr.Error(), "invalid digest")

	mountErr := BlobMountedError{
		From:       nil,
		Descriptor: manifest.Descriptor{Digest: dgst},
	}
	assert.Contains(t, mountErr.Error(), "blob mounted from")
}
