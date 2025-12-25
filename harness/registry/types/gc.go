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

type GCBlobTask struct {
	BlobID      int64
	ReviewAfter int64
	ReviewCount int
	CreatedAt   int64
	Event       string
}

// GCManifestTask represents a row in the gc_manifest_review_queue table.
type GCManifestTask struct {
	RegistryID  int64
	ManifestID  int64
	ReviewAfter int64
	ReviewCount int
	CreatedAt   int64
	Event       string
}
