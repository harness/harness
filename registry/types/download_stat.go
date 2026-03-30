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

// DownloadStat DTO object.
type DownloadStat struct {
	ID         int64
	ArtifactID int64
	Timestamp  time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	CreatedBy  int64
	UpdatedBy  int64
}

// DownloadCount pairs an entity ID with its download count for ExtendedCache compatibility.
type DownloadCount struct {
	EntityID int64
	Count    int64
}

func (d *DownloadCount) Identifier() int64 {
	return d.EntityID
}

// ManifestDownloadCount pairs a composite key with its download count for ExtendedCache compatibility.
type ManifestDownloadCount struct {
	Key   string
	Count int64
}

func (d *ManifestDownloadCount) Identifier() string {
	return d.Key
}
