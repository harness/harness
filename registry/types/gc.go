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
)

type GCBlobTask struct {
	BlobID      int64
	ReviewAfter time.Time
	ReviewCount int
	CreatedAt   time.Time
	Event       string
}

// GCManifestTask represents a row in the gc_manifest_review_queue table.
type GCManifestTask struct {
	RegistryID  int64
	ManifestID  int64
	ReviewAfter time.Time
	ReviewCount int
	CreatedAt   time.Time
	Event       string
}

// GCReviewAfterDefault represents a row in the gc_review_after_defaults table.
type GCReviewAfterDefault struct {
	Event string
	Value time.Duration
}
