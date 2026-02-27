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

type PackageWrapper interface {
	IsFileOperationSupported(packageType string) bool
	IsValidPackageType(packageType string) bool
	IsValidPackageTypes(packageTypes []string) bool
	IsValidRepoType(repoType string) bool
	IsValidRepoTypes(repoTypes []string) bool
	ValidateRepoType(packageType string, repoType string) bool
	IsValidUpstreamSource(upstreamSource string) bool
	IsValidUpstreamSources(upstreamSources []string) bool
	ValidateUpstreamSource(packageType string, upstreamSource string) bool
	IsURLRequiredForUpstreamSource(packageType string, upstreamSource string) bool
	GetPackageTypeFromPathPackageType(pathPackageType string) (string, error)
	DeleteArtifactVersion(
		ctx context.Context,
		regInfo *types.RegistryRequestBaseInfo,
		imageInfo *types.Image,
		artifactName string,
		versionName string,
	) error
	DeleteArtifact(
		ctx context.Context,
		regInfo *types.RegistryRequestBaseInfo,
		artifactName string,
	) error
	GetFilePath(
		packageType string,
		artifactName string,
		versionName string,
	) (string, error)
	GetPackageURL(
		ctx context.Context,
		rootIdentifier string,
		registryIdentifier string,
		packageType string,
	) (string, error)
	GetArtifactMetadata(
		artifact types.ArtifactMetadata,
	) *artifact.ArtifactMetadata
	GetArtifactVersionMetadata(
		packageType string,
		image string,
		tag types.NonOCIArtifactMetadata,
	) *artifact.ArtifactVersionMetadata
	GetFileMetadata(
		ctx context.Context,
		rootIdentifier string,
		registryIdentifier string,
		packageType string,
		artifactName string,
		version string,
		file types.FileNodeMetadata,
	) *artifact.FileDetail
	GetArtifactDetail(
		packageType string,
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
		packageType string,
	) (*artifact.ClientSetupDetails, error)
	BuildRegistryIndexAsync(
		ctx context.Context,
		payload types.BuildRegistryIndexTaskPayload,
	) error
	BuildPackageIndexAsync(
		ctx context.Context,
		payload types.BuildPackageIndexTaskPayload,
	) error
	BuildPackageMetadataAsync(
		ctx context.Context,
		payload types.BuildPackageMetadataTaskPayload,
	) error
	ReportDeleteVersionEvent(
		ctx context.Context,
		registryID int64,
		artifactName string,
		versionName string,
	) error
	ReportBuildPackageIndexEvent(
		ctx context.Context,
		registryID int64,
		artifactName string,
	) error
	ReportBuildRegistryIndexEvent(
		ctx context.Context,
		registryID int64,
		sourceRefs []types.SourceRef,
	) error
	GetNodePathsForImage(
		packageType string,
		artifactType *string,
		packageName string,
	) ([]string, error)
	GetNodePathsForArtifact(
		packageType string,
		artifactType *string,
		packageName string,
		version string,
	) ([]string, error)
	GetPkgDownloadURL(
		ctx context.Context,
		packageType string,
		rootIdentifier string,
		registryIdentifier string,
		packageName string,
		artifactType string,
		version string,
		filename string,
		filepath string,
	) (string, error)
	GetPurlForArtifact(
		packageType string,
		packageName string,
		version string,
	) (string, error)
}
