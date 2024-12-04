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

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
)

// Tag DTO object.
type Tag struct {
	ID         int64
	Name       string
	ImageName  string
	RegistryID int64
	ManifestID int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
	CreatedBy  int64
	UpdatedBy  int64
}

type ArtifactMetadata struct {
	Name          string
	RepoName      string
	DownloadCount int64
	PackageType   artifact.PackageType
	Labels        []string
	LatestVersion string
	CreatedAt     time.Time
	ModifiedAt    time.Time
	Version       string
}

type TagMetadata struct {
	Name            string
	Size            string
	PackageType     artifact.PackageType
	DigestCount     int
	IsLatestVersion bool
	ModifiedAt      time.Time
	SchemaVersion   int
	NonConformant   bool
	Payload         Payload
	MediaType       string
	DownloadCount   int64
}

type TagDetail struct {
	ID            int64
	Name          string
	ImageName     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Size          string
	DownloadCount int64
}
