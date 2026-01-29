//  Copyright 2023 Harness, Inc.
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

package types

import (
	"time"

	"github.com/opencontainers/go-digest"
)

// Blob DTO object.
type Blob struct {
	ID           int64
	RootParentID int64
	// This media type is for S3. The caller should look this up
	// and override the value for the specific repository.
	MediaType   string
	MediaTypeID int64
	Digest      digest.Digest
	Size        int64
	CreatedAt   time.Time
	CreatedBy   int64
}

// Blobs is a slice of Blob pointers.
type Blobs []*Blob

type BlobDigests struct {
	SHA1   digest.Digest
	SHA256 digest.Digest
	SHA512 digest.Digest
	MD5    digest.Digest
}
