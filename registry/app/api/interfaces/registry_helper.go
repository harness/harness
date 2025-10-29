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

package interfaces

import (
	"context"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/types"
)

type RegistryHelper interface {
	GetAuthHeaderPrefix() string
	// DeleteVersion deletes the version
	DeleteVersion(ctx context.Context,
		regInfo *types.RegistryRequestBaseInfo,
		imageInfo *types.Image,
		artifactName string,
		versionName string) error

	// DeleteArtifact deletes the artifact
	DeleteGenericImage(ctx context.Context,
		regInfo *types.RegistryRequestBaseInfo,
		artifactName string, filePath string,
	) error

	// ReportDeleteVersionEvent reports the delete version event
	ReportDeleteVersionEvent(
		ctx context.Context,
		payload *registryevents.ArtifactDeletedPayload,
	)

	// ReportBuildPackageIndexEvent reports the build package index event
	ReportBuildPackageIndexEvent(ctx context.Context, registryID int64, artifactName string)

	// ReportBuildRegistryIndexEvent reports the build registry index event
	ReportBuildRegistryIndexEvent(ctx context.Context, registryID int64, sources []types.SourceRef)

	// GetPackageURL returns the package URL
	GetPackageURL(
		ctx context.Context,
		rootIdentifier string,
		registryIdentifier string,
		packageTypePathParam string,
	) string

	GetHostName(
		ctx context.Context,
		rootSpace string,
	) string

	GetArtifactMetadata(
		artifact types.ArtifactMetadata,
		pullCommand string,
	) *artifact.ArtifactMetadata

	GetArtifactVersionMetadata(
		tag types.NonOCIArtifactMetadata,
		pullCommand string,
		packageType string,
	) *artifact.ArtifactVersionMetadata

	GetFileMetadata(
		file types.FileNodeMetadata,
		filename string,
		downloadCommand string,
	) *artifact.FileDetail

	GetArtifactDetail(
		img *types.Image,
		art *types.Artifact,
		metadata map[string]any,
		downloadCount int64,
	) *artifact.ArtifactDetail

	ReplacePlaceholders(
		ctx context.Context,
		clientSetupSections *[]artifact.ClientSetupSection,
		username string,
		regRef string,
		image *artifact.ArtifactParam,
		version *artifact.VersionParam,
		registryURL string,
		groupID string,
		uploadURL string,
		hostname string,
	)
}
