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
	"fmt"

	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
)

type BucketKey string

func (k BucketKey) String() string { return string(k) }

type BlobLocator struct {
	Digest        digest.Digest
	BlobID        int64
	GenericBlobID uuid.UUID
	RootParentID  int64
	RegistryID    int64
}

func (b BlobLocator) String() string {
	return fmt.Sprintf("%s:%d:%s:%d:%d", b.Digest, b.BlobID, b.GenericBlobID, b.RootParentID, b.RegistryID)
}

type StorageLookup struct {
	BlobLocator
	ClientIP string
}
