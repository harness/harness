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

package pkg

import (
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	v2 "github.com/distribution/distribution/v3/registry/api/v2"
)

type BaseInfo struct {
	PathPackageType artifact.PackageType
	PathRoot        string
	ParentID        int64
	RootIdentifier  string
	RootParentID    int64
}

type ArtifactInfo struct {
	*BaseInfo
	RegIdentifier string
	RegistryID    int64
	// Currently used only for Python packages
	// TODO: extend to all package types
	Registry     types.Registry
	Image        string
	ArtifactType *artifact.ArtifactType
}

func (a *ArtifactInfo) UpdateRegistryInfo(r types.Registry) {
	a.RegistryID = r.ID
	a.RegIdentifier = r.Name
	a.Registry = r
	a.ParentID = r.ParentID
}

type RegistryInfo struct {
	*ArtifactInfo
	Reference   string
	Digest      string
	Tag         string
	URLBuilder  *v2.URLBuilder
	Path        string
	PackageType artifact.PackageType
}

func (r *RegistryInfo) SetReference(ref string) {
	r.Reference = ref
}

func (a *ArtifactInfo) SetRepoKey(key string) {
	a.RegIdentifier = key
}

type MavenArtifactInfo struct {
	*ArtifactInfo
	GroupID    string
	ArtifactID string
	Version    string
	FileName   string
	Path       string
}

type GenericArtifactInfo struct {
	*ArtifactInfo
	FileName    string
	Version     string
	RegistryID  int64
	Description string
}

func (a *MavenArtifactInfo) SetMavenRepoKey(key string) {
	a.RegIdentifier = key
}

// BaseArtifactInfo implements pkg.PackageArtifactInfo interface.
func (a GenericArtifactInfo) BaseArtifactInfo() ArtifactInfo {
	return *a.ArtifactInfo
}

func (a GenericArtifactInfo) GetImageVersion() (exists bool, imageVersion string) {
	if a.Image != "" && a.Version != "" {
		return true, JoinWithSeparator(":", a.Image, a.Version)
	}
	return false, ""
}

func (a GenericArtifactInfo) GetVersion() string {
	return a.Version
}

func (a GenericArtifactInfo) GetFileName() string {
	return a.FileName
}

// BaseArtifactInfo implements pkg.PackageArtifactInfo interface.
func (a MavenArtifactInfo) BaseArtifactInfo() ArtifactInfo {
	return *a.ArtifactInfo
}

func (a MavenArtifactInfo) GetImageVersion() (exists bool, imageVersion string) {
	if a.Image != "" && a.Version != "" {
		return true, JoinWithSeparator(":", a.GroupID, a.ArtifactID, a.Version)
	}
	return false, ""
}

func (a MavenArtifactInfo) GetVersion() string {
	return a.Version
}

func (a MavenArtifactInfo) GetFileName() string {
	return a.FileName
}
