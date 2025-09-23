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
	"encoding/json"
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
	ID               int64
	Name             string
	RepoName         string
	DownloadCount    int64
	PackageType      artifact.PackageType
	Labels           []string
	LatestVersion    string
	CreatedAt        time.Time
	ModifiedAt       time.Time
	Version          string
	Metadata         json.RawMessage
	IsQuarantined    bool
	QuarantineReason *string
	ArtifactType     *artifact.ArtifactType
	Tags             []string
}

type ImageMetadata struct {
	Name          string
	RepoName      string
	DownloadCount int64
	PackageType   artifact.PackageType
	ArtifactType  *artifact.ArtifactType
	LatestVersion string
	CreatedAt     time.Time
	ModifiedAt    time.Time
}

type OciVersionMetadata struct {
	Name             string
	Size             string
	PackageType      artifact.PackageType
	DigestCount      int
	ModifiedAt       time.Time
	SchemaVersion    int
	NonConformant    bool
	Payload          Payload
	MediaType        string
	Digest           string
	DownloadCount    int64
	Tags             []string
	IsQuarantined    bool
	QuarantineReason string
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

type TagInfo struct {
	Name   string
	Digest string
}

type QuarantineInfo struct {
	Reason    string
	CreatedAt int64
}

// ArtifactIdentifier represents an artifact by name, version, and registry ID.
type ArtifactIdentifier struct {
	Name         string
	Version      string
	RegistryName string
}
