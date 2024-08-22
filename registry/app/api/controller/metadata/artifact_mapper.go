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

package metadata

import (
	"context"
	"errors"
	"path/filepath"

	artifactapi "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

func GetArtifactMetadata(artifacts *[]types.ArtifactMetadata) []artifactapi.ArtifactMetadata {
	artifactMetadataList := make([]artifactapi.ArtifactMetadata, 0, len(*artifacts))
	for _, artifact := range *artifacts {
		artifactMetadata := mapToArtifactMetadata(artifact)
		artifactMetadataList = append(artifactMetadataList, *artifactMetadata)
	}
	return artifactMetadataList
}

func mapToArtifactMetadata(artifact types.ArtifactMetadata) *artifactapi.ArtifactMetadata {
	lastModified := GetTimeInMs(artifact.ModifiedAt)
	packageType := artifact.PackageType
	return &artifactapi.ArtifactMetadata{
		RegistryIdentifier: artifact.RepoName,
		Name:               artifact.Name,
		LatestVersion:      artifact.LatestVersion,
		Labels:             &artifact.Labels,
		LastModified:       &lastModified,
		PackageType:        &packageType,
		DownloadsCount:     &artifact.DownloadCount,
	}
}

func toPackageType(packageTypeStr string) (artifactapi.PackageType, error) {
	switch packageTypeStr {
	case string(artifactapi.PackageTypeDOCKER):
		return artifactapi.PackageTypeDOCKER, nil
	case string(artifactapi.PackageTypeGENERIC):
		return artifactapi.PackageTypeGENERIC, nil
	case string(artifactapi.PackageTypeHELM):
		return artifactapi.PackageTypeHELM, nil
	case string(artifactapi.PackageTypeMAVEN):
		return artifactapi.PackageTypeMAVEN, nil
	default:
		return "", errors.New("invalid package type")
	}
}

func GetTagMetadata(
	ctx context.Context,
	tags *[]types.TagMetadata,
	latestTag string,
	image string,
	regIdentifier string,
	rootIdentifier string,
	registryURL string,
) []artifactapi.ArtifactVersionMetadata {
	artifactVersionMetadataList := []artifactapi.ArtifactVersionMetadata{}
	digestCount := int64(1)
	for _, tag := range *tags {
		modifiedAt := GetTimeInMs(tag.ModifiedAt)
		size := GetImageSize(tag.Size)
		isLatestVersion := latestTag == tag.Name
		command := GetPullCommand(rootIdentifier, regIdentifier, image, tag.Name, string(tag.PackageType), registryURL)
		packageType, err := toPackageType(string(tag.PackageType))
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("Error converting package type %s", tag.PackageType)
			continue
		}
		artifactVersionMetadata := &artifactapi.ArtifactVersionMetadata{
			PackageType:     &packageType,
			Name:            tag.Name,
			Size:            &size,
			LastModified:    &modifiedAt,
			DigestCount:     &digestCount,
			IslatestVersion: &isLatestVersion,
			PullCommand:     &command,
		}
		artifactVersionMetadataList = append(artifactVersionMetadataList, *artifactVersionMetadata)
	}
	return artifactVersionMetadataList
}

func GetAllArtifactResponse(
	artifacts *[]types.ArtifactMetadata,
	count int64,
	pageNumber int64,
	pageSize int,
) *artifactapi.ListArtifactResponseJSONResponse {
	artifactMetadataList := GetArtifactMetadata(artifacts)
	pageCount := GetPageCount(count, pageSize)
	listArtifact := &artifactapi.ListArtifact{
		ItemCount: &count,
		PageCount: &pageCount,
		PageIndex: &pageNumber,
		PageSize:  &pageSize,
		Artifacts: artifactMetadataList,
	}
	response := &artifactapi.ListArtifactResponseJSONResponse{
		Data:   *listArtifact,
		Status: artifactapi.StatusSUCCESS,
	}
	return response
}

func GetAllArtifactLabelsResponse(
	artifactLabels *[]string,
	count int64,
	pageNumber int64,
	pageSize int,
) *artifactapi.ListArtifactLabelResponseJSONResponse {
	pageCount := GetPageCount(count, pageSize)
	listArtifactLabels := &artifactapi.ListArtifactLabel{
		ItemCount: &count,
		PageCount: &pageCount,
		PageIndex: &pageNumber,
		PageSize:  &pageSize,
		Labels:    *artifactLabels,
	}
	response := &artifactapi.ListArtifactLabelResponseJSONResponse{
		Data:   *listArtifactLabels,
		Status: artifactapi.StatusSUCCESS,
	}
	return response
}

func GetAllArtifactVersionResponse(
	ctx context.Context,
	tags *[]types.TagMetadata,
	latestTag string,
	image string,
	count int64,
	regInfo *RegistryRequestInfo,
	pageNumber int64,
	pageSize int,
	rootIdentifier string,
	registryURL string,
) *artifactapi.ListArtifactVersionResponseJSONResponse {
	artifactVersionMetadataList := GetTagMetadata(
		ctx, tags, latestTag, image,
		regInfo.RegistryIdentifier, rootIdentifier, registryURL,
	)
	pageCount := GetPageCount(count, pageSize)
	listArtifactVersions := &artifactapi.ListArtifactVersion{
		ItemCount:        &count,
		PageCount:        &pageCount,
		PageIndex:        &pageNumber,
		PageSize:         &pageSize,
		ArtifactVersions: &artifactVersionMetadataList,
	}
	response := &artifactapi.ListArtifactVersionResponseJSONResponse{
		Data:   *listArtifactVersions,
		Status: artifactapi.StatusSUCCESS,
	}
	return response
}

func GetDockerArtifactDetails(
	registry *types.Registry,
	tag *types.TagDetail,
	manifest *types.Manifest,
	isLatestTag bool,
	regInfo *RegistryRequestBaseInfo,
	registryURL string,
) *artifactapi.DockerArtifactDetailResponseJSONResponse {
	repoPath := getRepoPath(registry.Name, tag.ImageName, manifest.Digest.String())
	pullCommand := GetDockerPullCommand(regInfo.rootIdentifier, registry.Name, tag.ImageName, tag.Name, registryURL)
	createdAt := GetTimeInMs(tag.CreatedAt)
	modifiedAt := GetTimeInMs(tag.UpdatedAt)
	size := GetSize(manifest.TotalSize)
	artifactDetail := &artifactapi.DockerArtifactDetail{
		ImageName:       tag.ImageName,
		Version:         tag.Name,
		PackageType:     registry.PackageType,
		IsLatestVersion: &isLatestTag,
		CreatedAt:       &createdAt,
		ModifiedAt:      &modifiedAt,
		RegistryPath:    repoPath,
		PullCommand:     &pullCommand,
		Url:             GetTagURL(regInfo.rootIdentifier, tag.ImageName, tag.Name, registry.Name, registryURL),
		Size:            &size,
	}

	response := &artifactapi.DockerArtifactDetailResponseJSONResponse{
		Data:   *artifactDetail,
		Status: artifactapi.StatusSUCCESS,
	}
	return response
}

func GetHelmArtifactDetails(
	registry *types.Registry,
	tag *types.TagDetail,
	manifest *types.Manifest,
	isLatestTag bool,
	rootIdentifier string,
	registryURL string,
) *artifactapi.HelmArtifactDetailResponseJSONResponse {
	repoPath := getRepoPath(registry.Name, tag.ImageName, manifest.Digest.String())
	pullCommand := GetHelmPullCommand(rootIdentifier, registry.Name, tag.ImageName, tag.Name, registryURL)
	createdAt := GetTimeInMs(tag.CreatedAt)
	modifiedAt := GetTimeInMs(tag.UpdatedAt)
	size := GetSize(manifest.TotalSize)
	artifactDetail := &artifactapi.HelmArtifactDetail{
		Artifact:        &tag.ImageName,
		Version:         tag.Name,
		PackageType:     registry.PackageType,
		IsLatestVersion: &isLatestTag,
		CreatedAt:       &createdAt,
		ModifiedAt:      &modifiedAt,
		RegistryPath:    repoPath,
		PullCommand:     &pullCommand,
		Url:             GetTagURL(rootIdentifier, tag.ImageName, tag.Name, registry.Name, registryURL),
		Size:            &size,
	}

	response := &artifactapi.HelmArtifactDetailResponseJSONResponse{
		Data:   *artifactDetail,
		Status: artifactapi.StatusSUCCESS,
	}
	return response
}

func GetArtifactSummary(artifact types.ArtifactMetadata) *artifactapi.ArtifactSummaryResponseJSONResponse {
	downloads := int64(0)
	createdAt := GetTimeInMs(artifact.CreatedAt)
	modifiedAt := GetTimeInMs(artifact.ModifiedAt)
	artifactVersionSummary := &artifactapi.ArtifactSummary{
		CreatedAt:      &createdAt,
		ModifiedAt:     &modifiedAt,
		DownloadsCount: &downloads,
		ImageName:      artifact.Name,
		Labels:         &artifact.Labels,
		PackageType:    artifact.PackageType,
	}
	response := &artifactapi.ArtifactSummaryResponseJSONResponse{
		Data:   *artifactVersionSummary,
		Status: artifactapi.StatusSUCCESS,
	}
	return response
}

func GetArtifactVersionSummary(
	tag *types.TagMetadata,
	artifactName string,
	isLatestTag bool,
) *artifactapi.ArtifactVersionSummaryResponseJSONResponse {
	artifactVersionSummary := &artifactapi.ArtifactVersionSummary{
		ImageName:       artifactName,
		IsLatestVersion: &isLatestTag,
		PackageType:     tag.PackageType,
		Version:         tag.Name,
	}
	response := &artifactapi.ArtifactVersionSummaryResponseJSONResponse{
		Data:   *artifactVersionSummary,
		Status: artifactapi.StatusSUCCESS,
	}
	return response
}

func getRepoPath(registry string, image string, tag string) string {
	return filepath.Join(registry, image, tag)
}
