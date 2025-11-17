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
	"github.com/harness/gitness/registry/types"
)

type PackageHelper interface {
	GetPackageType() string
	GetPathPackageType() string
	GetPackageURL(ctx context.Context,
		rootIdentifier string,
		registryIdentifier string,
	) string
	GetFilePath(artifactName string, versionName string) string
	DeleteArtifact(
		ctx context.Context,
		regInfo *types.RegistryRequestBaseInfo,
		artifactName string,
	) error
	DeleteVersion(ctx context.Context,
		regInfo *types.RegistryRequestBaseInfo,
		imageInfo *types.Image,
		artifactName string,
		versionName string,
	) error
	ReportDeleteVersionEvent(ctx context.Context,
		principalID int64,
		registryID int64,
		artifact string,
		version string,
	)
	ReportBuildPackageIndexEvent(ctx context.Context, registryID int64, artifactName string)
	ReportBuildRegistryIndexEvent(ctx context.Context, registryID int64, sources []types.SourceRef)
	IsValidRepoType(repoType string) bool
	IsValidUpstreamSource(upstreamSource string) bool
	IsURLRequiredForUpstreamSource(upstreamSource string) bool
	GetArtifactMetadata(
		artifact types.ArtifactMetadata,
	) *artifact.ArtifactMetadata
	GetArtifactVersionMetadata(
		image string,
		tag types.NonOCIArtifactMetadata,
	) *artifact.ArtifactVersionMetadata
	GetFileMetadata(
		ctx context.Context,
		rootIdentifier string,
		registryIdentifier string,
		artifactName string,
		version string,
		file types.FileNodeMetadata,
	) *artifact.FileDetail
	GetArtifactDetail(
		img *types.Image,
		art *types.Artifact,
		downloadCount int64,
	) (*artifact.ArtifactDetail, error)
	GetClientSetupDetails(
		ctx context.Context,
		regRef string,
		image *artifact.ArtifactParam,
		tag *artifact.VersionParam,
		registryType artifact.RegistryType,
	) (*artifact.ClientSetupDetails, error)
	BuildRegistryIndexAsync(
		ctx context.Context,
		registry *types.Registry,
		payload types.BuildRegistryIndexTaskPayload,
	) error
	BuildPackageIndexAsync(
		ctx context.Context,
		registry *types.Registry,
		payload types.BuildPackageIndexTaskPayload,
	) error
	BuildPackageMetadataAsync(
		ctx context.Context,
		registry *types.Registry,
		payload types.BuildPackageMetadataTaskPayload,
	) error

	GetNodePathsForImage(
		artifactType *string,
		packageName string,
	) ([]string, error)
	GetNodePathsForArtifact(
		artifactType *string,
		packageName string,
		version string,
	) ([]string, error)
}
