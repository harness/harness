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
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/interfaces"
	artifactapi "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/api/utils"
	"github.com/harness/gitness/registry/app/metadata"
	npm2 "github.com/harness/gitness/registry/app/metadata/npm"
	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

func GetArtifactMetadata(
	ctx context.Context,
	artifacts []types.ArtifactMetadata,
	rootIdentifier string,
	urlProvider url.Provider,
	setupDetailsAuthHeaderPrefix string,
	untaggedImagesEnabled bool,
	packageWrapper interfaces.PackageWrapper,
) []artifactapi.ArtifactMetadata {
	artifactMetadataList := make([]artifactapi.ArtifactMetadata, 0, len(artifacts))
	for _, artifact := range artifacts {
		metadata := packageWrapper.GetArtifactMetadata(artifact)
		if metadata != nil {
			artifactMetadataList = append(artifactMetadataList, *metadata)
			continue
		}
		registryURL := urlProvider.RegistryURL(ctx, rootIdentifier, artifact.RepoName)
		if artifact.PackageType == artifactapi.PackageTypeGENERIC {
			registryURL = urlProvider.RegistryURL(ctx, rootIdentifier, "generic", artifact.RepoName)
		}
		artifactMetadata := mapToArtifactMetadata(ctx, artifact, registryURL,
			setupDetailsAuthHeaderPrefix, untaggedImagesEnabled)
		if len(artifact.Tags) > 0 {
			artifactMetadata.Metadata = &artifactapi.ArtifactEntityMetadata{
				"tags": artifact.Tags,
			}
		}
		artifactMetadataList = append(artifactMetadataList, *artifactMetadata)
	}
	return artifactMetadataList
}

func GetRegistryArtifactMetadata(artifacts []types.ArtifactMetadata) []artifactapi.RegistryArtifactMetadata {
	artifactMetadataList := make([]artifactapi.RegistryArtifactMetadata, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifactMetadata := mapToRegistryArtifactMetadata(artifact)
		artifactMetadataList = append(artifactMetadataList, *artifactMetadata)
	}
	return artifactMetadataList
}

func GetMavenArtifactDetail(
	image *types.Image, artifact *types.Artifact,
	mavenMetadata metadata.MavenMetadata,
) artifactapi.ArtifactDetail {
	createdAt := GetTimeInMs(artifact.CreatedAt)
	modifiedAt := GetTimeInMs(artifact.UpdatedAt)
	var size int64
	for _, file := range mavenMetadata.Files {
		size += file.Size
	}
	sizeVal := GetSize(size)
	artifactDetail := &artifactapi.ArtifactDetail{
		CreatedAt:  &createdAt,
		ModifiedAt: &modifiedAt,
		Name:       &image.Name,
		Version:    artifact.Version,
		Size:       &sizeVal,
	}
	return *artifactDetail
}

func mapToArtifactMetadata(
	ctx context.Context,
	artifact types.ArtifactMetadata,
	registryURL string,
	setupDetailsAuthHeaderPrefix string,
	untaggedImagesEnabled bool,
) *artifactapi.ArtifactMetadata {
	lastModified := GetTimeInMs(artifact.ModifiedAt)
	packageType := artifact.PackageType
	pullCommand := GetPullCommand(ctx, artifact.Name, artifact.Version,
		string(packageType), registryURL, setupDetailsAuthHeaderPrefix, artifact.ArtifactType, !untaggedImagesEnabled)
	return &artifactapi.ArtifactMetadata{
		RegistryIdentifier: artifact.RepoName,
		Name:               artifact.Name,
		Version:            &artifact.Version,
		Labels:             &artifact.Labels,
		LastModified:       &lastModified,
		PackageType:        packageType,
		DownloadsCount:     &artifact.DownloadCount,
		PullCommand:        &pullCommand,
		IsQuarantined:      &artifact.IsQuarantined,
		QuarantineReason:   artifact.QuarantineReason,
		ArtifactType:       artifact.ArtifactType,
	}
}

func mapToRegistryArtifactMetadata(artifact types.ArtifactMetadata) *artifactapi.RegistryArtifactMetadata {
	lastModified := GetTimeInMs(artifact.ModifiedAt)
	packageType := artifact.PackageType
	return &artifactapi.RegistryArtifactMetadata{
		RegistryIdentifier: artifact.RepoName,
		Name:               artifact.Name,
		LatestVersion:      artifact.LatestVersion,
		Labels:             &artifact.Labels,
		LastModified:       &lastModified,
		PackageType:        packageType,
		DownloadsCount:     &artifact.DownloadCount,
		IsQuarantined:      &artifact.IsQuarantined,
		ArtifactType:       artifact.ArtifactType,
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
	case string(artifactapi.PackageTypePYTHON):
		return artifactapi.PackageTypePYTHON, nil
	case string(artifactapi.PackageTypeNPM):
		return artifactapi.PackageTypeNPM, nil
	case string(artifactapi.PackageTypeRPM):
		return artifactapi.PackageTypeRPM, nil
	case string(artifactapi.PackageTypeNUGET):
		return artifactapi.PackageTypeNUGET, nil
	case string(artifactapi.PackageTypeGO):
		return artifactapi.PackageTypeGO, nil
	case string(artifactapi.PackageTypeHUGGINGFACE):
		return artifactapi.PackageTypeHUGGINGFACE, nil
	default:
		return "", errors.New("invalid package type")
	}
}

func GetTagMetadata(
	ctx context.Context,
	ociVersions *[]types.OciVersionMetadata,
	image string,
	registryURL string,
	setupDetailsAuthHeaderPrefix string,
	untaggedImagesEnabled bool,
) []artifactapi.ArtifactVersionMetadata {
	artifactVersionMetadataList := []artifactapi.ArtifactVersionMetadata{}
	for _, tag := range *ociVersions {
		modifiedAt := GetTimeInMs(tag.ModifiedAt)
		size := GetImageSize(tag.Size)
		digestCount := tag.DigestCount
		command := GetPullCommand(ctx, image, tag.Name, string(tag.PackageType), registryURL,
			setupDetailsAuthHeaderPrefix, nil, !untaggedImagesEnabled)
		packageType, err := toPackageType(string(tag.PackageType))
		downloadCount := tag.DownloadCount
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("Error converting package type %s", tag.PackageType)
			continue
		}
		artifactVersionMetadata := &artifactapi.ArtifactVersionMetadata{
			PackageType:    packageType,
			Name:           tag.Name,
			Size:           &size,
			LastModified:   &modifiedAt,
			DigestCount:    &digestCount,
			PullCommand:    &command,
			DownloadsCount: &downloadCount,
			Metadata: &artifactapi.ArtifactEntityMetadata{
				"tags": tag.Tags,
			},
			IsQuarantined:    &tag.IsQuarantined,
			QuarantineReason: &tag.QuarantineReason,
		}

		artifactVersionMetadataList = append(artifactVersionMetadataList, *artifactVersionMetadata)
	}
	return artifactVersionMetadataList
}

func GetAllArtifactResponse(
	ctx context.Context,
	artifacts *[]types.ArtifactMetadata,
	count int64,
	pageNumber int64,
	pageSize int,
	rootIdentifier string,
	urlProvider url.Provider,
	setupDetailsAuthHeaderPrefix string,
	untaggedImagesEnabled bool,
	packageWrapper interfaces.PackageWrapper,
) *artifactapi.ListArtifactResponseJSONResponse {
	var artifactMetadataList []artifactapi.ArtifactMetadata
	if artifacts == nil {
		artifactMetadataList = make([]artifactapi.ArtifactMetadata, 0)
	} else {
		artifactMetadataList = GetArtifactMetadata(ctx, *artifacts, rootIdentifier, urlProvider,
			setupDetailsAuthHeaderPrefix, untaggedImagesEnabled, packageWrapper)
	}
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

func GetAllArtifactFilesResponse(
	ctx context.Context,
	files *[]types.FileNodeMetadata, count int64, pageNumber int64, pageSize int,
	registryURL string, artifactName string, version string,
	packageType artifactapi.PackageType, setupDetailsAuthHeaderPrefix string,
	artifactType *artifactapi.ArtifactType, isAnonymous bool, rootIdentifier string,
	registryIdentifier string, packageWrapper interfaces.PackageWrapper,
) *artifactapi.FileDetailResponseJSONResponse {
	var fileMetadataList []artifactapi.FileDetail
	if files == nil || len(*files) == 0 {
		fileMetadataList = make([]artifactapi.FileDetail, 0)
	} else {
		fileMetadataList = GetArtifactFilesMetadata(ctx, files, registryURL, artifactName, version, packageType,
			setupDetailsAuthHeaderPrefix, artifactType, isAnonymous, rootIdentifier, registryIdentifier, packageWrapper)
	}
	pageCount := GetPageCount(count, pageSize)
	listFileDetail := &artifactapi.ListFileDetail{
		ItemCount: &count,
		PageCount: &pageCount,
		PageIndex: &pageNumber,
		PageSize:  &pageSize,
		Files:     fileMetadataList,
	}
	response := &artifactapi.FileDetailResponseJSONResponse{
		Data:   *listFileDetail,
		Status: artifactapi.StatusSUCCESS,
	}
	return response
}

func GetArtifactFileResponseJSONResponse(
	registryURL string,
	packageType artifactapi.PackageType,
	artifactName string,
	version string,
	fileName string,
) *artifactapi.ArtifactFileResponseJSONResponse {
	return &artifactapi.ArtifactFileResponseJSONResponse{
		Status:      artifactapi.StatusSUCCESS,
		DownloadUrl: getDownloadURL(registryURL, packageType, artifactName, version, fileName),
	}
}
func GetArtifactFilesMetadata(
	ctx context.Context,
	metadata *[]types.FileNodeMetadata, registryURL string, artifactName string,
	version string, packageType artifactapi.PackageType, setupDetailsAuthHeaderPrefix string,
	artifactType *artifactapi.ArtifactType, isAnonymous bool, rootIdentifier string,
	registryIdentifier string, packageWrapper interfaces.PackageWrapper,
) []artifactapi.FileDetail {
	var files []artifactapi.FileDetail
	for _, file := range *metadata {
		filePathPrefix := "/" + artifactName + "/" + version + "/"
		filename := strings.Replace(file.Path, filePathPrefix, "", 1)
		var downloadCommand string
		//nolint:exhaustive
		switch packageType {
		case artifactapi.PackageTypeGENERIC:
			downloadCommand = GetGenericArtifactFileDownloadCommand(registryURL, artifactName,
				version, filename, setupDetailsAuthHeaderPrefix, isAnonymous)
		case artifactapi.PackageTypePYTHON:
			downloadCommand = GetGenericFileDownloadCommand(registryURL, artifactName,
				version, filename, setupDetailsAuthHeaderPrefix, isAnonymous)
		case artifactapi.PackageTypeNPM:
			downloadCommand = GetNPMArtifactFileDownloadCommand(registryURL, artifactName,
				version, filename, isAnonymous)
		case artifactapi.PackageTypeMAVEN:
			artifactName = strings.ReplaceAll(artifactName, ".", "/")
			artifactName = strings.ReplaceAll(artifactName, ":", "/")
			filePathPrefix = "/" + artifactName + "/" + version + "/"
			filename = strings.Replace(file.Path, filePathPrefix, "", 1)
			downloadCommand = GetMavenArtifactFileDownloadCommand(registryURL, artifactName,
				version, filename, setupDetailsAuthHeaderPrefix, isAnonymous)
		case artifactapi.PackageTypeRPM:
			downloadCommand = GetRPMArtifactFileDownloadCommand(registryURL, filename,
				setupDetailsAuthHeaderPrefix, isAnonymous)
			_, filename, _ = paths.DisectLeaf(filename)
		case artifactapi.PackageTypeNUGET:
			downloadCommand = GetNugetArtifactFileDownloadCommand(registryURL, artifactName,
				version, filename, setupDetailsAuthHeaderPrefix, isAnonymous)
		case artifactapi.PackageTypeGO:
			goFilePath := utils.GetGoFilePath(artifactName, version)
			filename = strings.Replace(file.Path, goFilePath+"/", "", 1)
			downloadCommand = GetGoArtifactFileDownloadCommand(registryURL, artifactName,
				filename, setupDetailsAuthHeaderPrefix, isAnonymous)
		case artifactapi.PackageTypeHUGGINGFACE:
			filename = strings.Replace(filename, "/"+string(*artifactType), "", 1)
			downloadCommand = GetHuggingFaceArtifactFileDownloadCommand(registryURL, artifactName,
				version, filename, setupDetailsAuthHeaderPrefix, artifactType, isAnonymous)
		default:
			metadata := packageWrapper.GetFileMetadata(
				ctx, rootIdentifier, registryIdentifier, string(packageType), artifactName, version, file)
			if metadata != nil {
				files = append(files, *metadata)
				continue
			}
		}
		files = append(files, artifactapi.FileDetail{
			Checksums:       getCheckSums(file),
			Size:            GetSize(file.Size),
			CreatedAt:       fmt.Sprint(file.CreatedAt),
			Name:            filename,
			DownloadCommand: downloadCommand,
		})
	}
	return files
}

func GetQuarantinePathJSONResponse(
	id string,
	registryID int64,
	artifactID int64,
	versionID int64,
	reason string,
	filePath string,
) *artifactapi.QuarantinePathResponseJSONResponse {
	return &artifactapi.QuarantinePathResponseJSONResponse{
		Status: artifactapi.StatusSUCCESS,
		Data: artifactapi.QuarantinePath{
			Id:         id,
			RegistryId: registryID,
			ArtifactId: artifactID,
			VersionId:  &versionID,
			FilePath:   &filePath,
			Reason:     reason,
		},
	}
}

func getCheckSums(file types.FileNodeMetadata) []string {
	return []string{
		fmt.Sprintf("SHA-512: %s", file.Sha512),
		fmt.Sprintf("SHA-256: %s", file.Sha256),
		fmt.Sprintf("SHA-1: %s", file.Sha1),
		fmt.Sprintf("MD5: %s", file.MD5),
	}
}

func GetAllArtifactByRegistryResponse(
	artifacts *[]types.ArtifactMetadata,
	count int64,
	pageNumber int64,
	pageSize int,
) *artifactapi.ListRegistryArtifactResponseJSONResponse {
	var artifactMetadataList []artifactapi.RegistryArtifactMetadata
	if artifacts == nil {
		artifactMetadataList = make([]artifactapi.RegistryArtifactMetadata, 0)
	} else {
		artifactMetadataList = GetRegistryArtifactMetadata(*artifacts)
	}
	pageCount := GetPageCount(count, pageSize)
	listArtifact := &artifactapi.ListRegistryArtifact{
		ItemCount: &count,
		PageCount: &pageCount,
		PageIndex: &pageNumber,
		PageSize:  &pageSize,
		Artifacts: artifactMetadataList,
	}
	response := &artifactapi.ListRegistryArtifactResponseJSONResponse{
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
	ociVersions *[]types.OciVersionMetadata,
	image string,
	count int64,
	pageNumber int64,
	pageSize int,
	registryURL string,
	setupDetailsAuthHeaderPrefix string,
	untaggedImagesEnabled bool,
) *artifactapi.ListArtifactVersionResponseJSONResponse {
	artifactVersionMetadataList := GetTagMetadata(
		ctx, ociVersions, image, registryURL, setupDetailsAuthHeaderPrefix, untaggedImagesEnabled,
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

func GetNonOCIAllArtifactVersionResponse(
	ctx context.Context,
	artifacts *[]types.NonOCIArtifactMetadata,
	image string,
	count int64,
	pageNumber int64,
	pageSize int,
	registryURL string,
	setupDetailsAuthHeaderPrefix string,
	pkgType string,
	packageWrapper interfaces.PackageWrapper,
) *artifactapi.ListArtifactVersionResponseJSONResponse {
	artifactVersionMetadataList := GetNonOCIArtifactMetadata(
		ctx, artifacts, image, registryURL, setupDetailsAuthHeaderPrefix, pkgType, packageWrapper)
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

func GetNonOCIArtifactMetadata(
	ctx context.Context,
	tags *[]types.NonOCIArtifactMetadata,
	image string,
	registryURL string,
	setupDetailsAuthHeaderPrefix string,
	pkgType string,
	packageWrapper interfaces.PackageWrapper,
) []artifactapi.ArtifactVersionMetadata {
	artifactVersionMetadataList := []artifactapi.ArtifactVersionMetadata{}
	for _, tag := range *tags {
		metadata := packageWrapper.GetArtifactVersionMetadata(pkgType, image, tag)
		if metadata != nil {
			artifactVersionMetadataList = append(artifactVersionMetadataList, *metadata)
			continue
		}
		modifiedAt := GetTimeInMs(tag.ModifiedAt)
		size := GetImageSize(tag.Size)
		command := GetPullCommand(ctx, image, tag.Name, pkgType, registryURL, setupDetailsAuthHeaderPrefix,
			tag.ArtifactType, true)
		packageType, err := toPackageType(pkgType)
		downloadCount := tag.DownloadCount
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("Error converting package type %s", tag.PackageType)
			continue
		}
		fileCount := tag.FileCount
		artifactVersionMetadata := &artifactapi.ArtifactVersionMetadata{
			PackageType:      packageType,
			FileCount:        &fileCount,
			Name:             tag.Name,
			Size:             &size,
			LastModified:     &modifiedAt,
			PullCommand:      &command,
			DownloadsCount:   &downloadCount,
			IsQuarantined:    &tag.IsQuarantined,
			QuarantineReason: tag.QuarantineReason,
			ArtifactType:     tag.ArtifactType,
		}
		artifactVersionMetadataList = append(artifactVersionMetadataList, *artifactVersionMetadata)
	}
	return artifactVersionMetadataList
}

func GetDockerArtifactDetails(
	registry *types.Registry,
	imageName string,
	version string,
	created time.Time,
	modified time.Time,
	digest string,
	size int64,
	registryURL string,
	downloadCount int64,
	isQuarantined bool,
	quarantineReason *string,
) *artifactapi.DockerArtifactDetailResponseJSONResponse {
	repoPath := getRepoPath(registry.Name, imageName, digest)
	createdAt := GetTimeInMs(created)
	modifiedAt := GetTimeInMs(modified)
	s := GetSize(size)
	artifactDetail := &artifactapi.DockerArtifactDetail{
		ImageName:        imageName,
		Version:          version,
		PackageType:      registry.PackageType,
		CreatedAt:        &createdAt,
		ModifiedAt:       &modifiedAt,
		RegistryPath:     repoPath,
		Url:              GetTagURL(imageName, version, registryURL),
		Size:             &s,
		DownloadsCount:   &downloadCount,
		IsQuarantined:    &isQuarantined,
		QuarantineReason: quarantineReason,
	}

	response := &artifactapi.DockerArtifactDetailResponseJSONResponse{
		Data:   *artifactDetail,
		Status: artifactapi.StatusSUCCESS,
	}
	return response
}

func GetHelmArtifactDetails(
	registry *types.Registry,
	imageName string,
	version string,
	created time.Time,
	updated time.Time,
	manifest *types.Manifest,
	registryURL string,
) *artifactapi.HelmArtifactDetailResponseJSONResponse {
	repoPath := getRepoPath(registry.Name, imageName, manifest.Digest.String())
	createdAt := GetTimeInMs(created)
	modifiedAt := GetTimeInMs(updated)
	size := GetSize(manifest.TotalSize)
	artifactDetail := &artifactapi.HelmArtifactDetail{
		Artifact:     &imageName,
		Version:      version,
		PackageType:  registry.PackageType,
		CreatedAt:    &createdAt,
		ModifiedAt:   &modifiedAt,
		RegistryPath: repoPath,
		Url:          GetTagURL(imageName, version, registryURL),
		Size:         &size,
	}

	response := &artifactapi.HelmArtifactDetailResponseJSONResponse{
		Data:   *artifactDetail,
		Status: artifactapi.StatusSUCCESS,
	}
	return response
}

func GetGenericArtifactDetail(
	image *types.Image, artifact *types.Artifact,
	metadata metadata.GenericMetadata,
) artifactapi.ArtifactDetail {
	createdAt := GetTimeInMs(artifact.CreatedAt)
	modifiedAt := GetTimeInMs(artifact.UpdatedAt)
	artifactDetail := &artifactapi.ArtifactDetail{
		CreatedAt:  &createdAt,
		ModifiedAt: &modifiedAt,
		Name:       &image.Name,
		Version:    artifact.Version,
	}
	err := artifactDetail.FromGenericArtifactDetailConfig(artifactapi.GenericArtifactDetailConfig{
		Description: &metadata.Description,
	})
	if err != nil {
		return artifactapi.ArtifactDetail{}
	}
	return *artifactDetail
}

func GetPythonArtifactDetail(
	image *types.Image, artifact *types.Artifact,
	metadata map[string]any,
) artifactapi.ArtifactDetail {
	createdAt := GetTimeInMs(artifact.CreatedAt)
	modifiedAt := GetTimeInMs(artifact.UpdatedAt)
	artifactDetail := &artifactapi.ArtifactDetail{
		CreatedAt:  &createdAt,
		ModifiedAt: &modifiedAt,
		Name:       &image.Name,
		Version:    artifact.Version,
	}
	err := artifactDetail.FromPythonArtifactDetailConfig(artifactapi.PythonArtifactDetailConfig{
		Metadata: &metadata,
	})
	if err != nil {
		return artifactapi.ArtifactDetail{}
	}
	return *artifactDetail
}

func GetNugetArtifactDetail(
	image *types.Image, artifact *types.Artifact,
	metadata map[string]any,
	downloadCount int64,
) artifactapi.ArtifactDetail {
	createdAt := GetTimeInMs(artifact.CreatedAt)
	modifiedAt := GetTimeInMs(artifact.UpdatedAt)
	size, ok := metadata["size"].(float64)
	if !ok {
		log.Error().Msg("failed to get size from Nuget metadata")
	}
	totalSize := GetSize(int64(size))
	artifactDetail := &artifactapi.ArtifactDetail{
		CreatedAt:     &createdAt,
		ModifiedAt:    &modifiedAt,
		Name:          &image.Name,
		Version:       artifact.Version,
		DownloadCount: &downloadCount,
		Size:          &totalSize,
	}
	err := artifactDetail.FromNugetArtifactDetailConfig(artifactapi.NugetArtifactDetailConfig{
		Metadata: &metadata,
	})
	if err != nil {
		log.Error().Err(err).Msgf("Error setting the artifact details for nuget package: [%s]", image.Name)
		return artifactapi.ArtifactDetail{}
	}
	return *artifactDetail
}

func GetHFArtifactDetail(
	image *types.Image, artifact *types.Artifact,
	metadata map[string]any,
	downloadCount int64,
) artifactapi.ArtifactDetail {
	createdAt := GetTimeInMs(artifact.CreatedAt)
	modifiedAt := GetTimeInMs(artifact.UpdatedAt)
	size, ok := metadata["size"].(float64)
	if !ok {
		log.Error().Msg("failed to get size from Hugging face metadata")
	}
	totalSize := GetSize(int64(size))
	artifactDetail := &artifactapi.ArtifactDetail{
		CreatedAt:     &createdAt,
		ModifiedAt:    &modifiedAt,
		Name:          &image.Name,
		Version:       artifact.Version,
		DownloadCount: &downloadCount,
		Size:          &totalSize,
		ArtifactType:  image.ArtifactType,
	}
	err := artifactDetail.FromHuggingFaceArtifactDetailConfig(artifactapi.HuggingFaceArtifactDetailConfig{
		Metadata: &metadata,
	})
	if err != nil {
		log.Error().Err(err).Msgf("Error setting the artifact details for hugging face package: [%s]", image.Name)
		return artifactapi.ArtifactDetail{}
	}
	return *artifactDetail
}

func GetGoArtifactDetail(
	image *types.Image, artifact *types.Artifact,
	metadata map[string]any,
	downloadCount int64,
) artifactapi.ArtifactDetail {
	createdAt := GetTimeInMs(artifact.CreatedAt)
	modifiedAt := GetTimeInMs(artifact.UpdatedAt)
	size, ok := metadata["size"].(float64)
	if !ok {
		log.Error().Msg("failed to get size from Go metadata")
	}
	totalSize := GetSize(int64(size))
	artifactDetail := &artifactapi.ArtifactDetail{
		CreatedAt:     &createdAt,
		ModifiedAt:    &modifiedAt,
		Name:          &image.Name,
		Version:       artifact.Version,
		DownloadCount: &downloadCount,
		Size:          &totalSize,
	}
	err := artifactDetail.FromGoArtifactDetailConfig(artifactapi.GoArtifactDetailConfig{
		Metadata: &metadata,
	})
	if err != nil {
		log.Error().Err(err).Msgf("Error setting the artifact details for go package: [%s]", image.Name)
		return artifactapi.ArtifactDetail{}
	}
	return *artifactDetail
}

func GetNPMArtifactDetail(
	image *types.Image, artifact *types.Artifact,
	metadata map[string]any,
	downloadCount int64,
) artifactapi.ArtifactDetail {
	createdAt := GetTimeInMs(artifact.CreatedAt)
	modifiedAt := GetTimeInMs(artifact.UpdatedAt)
	var npmMetadata npm2.NpmMetadata
	err := json.Unmarshal(artifact.Metadata, &npmMetadata)
	if err != nil {
		log.Error().Err(err).Msgf("Error unmarshalling the artifact metadata "+
			"for image: [%s], version: [%s]", image.Name, artifact.Version)
		return artifactapi.ArtifactDetail{}
	}
	totalSize := strconv.FormatInt(npmMetadata.Size, 10)
	artifactDetail := &artifactapi.ArtifactDetail{
		CreatedAt:     &createdAt,
		ModifiedAt:    &modifiedAt,
		Name:          &image.Name,
		Version:       artifact.Version,
		DownloadCount: &downloadCount,
		Size:          &totalSize,
	}
	err = artifactDetail.FromNpmArtifactDetailConfig(artifactapi.NpmArtifactDetailConfig{
		Metadata: &metadata,
	})
	if err != nil {
		log.Error().Err(err).Msgf("Error setting the artifact details for image: [%s]", image.Name)
		return artifactapi.ArtifactDetail{}
	}
	return *artifactDetail
}

func GetRPMArtifactDetail(
	image *types.Image, artifact *types.Artifact,
	metadata map[string]any,
	downloadCount int64,
) artifactapi.ArtifactDetail {
	createdAt := GetTimeInMs(artifact.CreatedAt)
	modifiedAt := GetTimeInMs(artifact.UpdatedAt)
	size, ok := metadata["size"].(float64)
	if !ok {
		log.Error().Msg("failed to get size from RPM metadata")
	}
	fileMetadata, ok := metadata["file_metadata"].(map[string]any)
	if ok {
		delete(fileMetadata, "files")
		delete(fileMetadata, "changelogs")
	}

	totalSize := strconv.FormatInt(int64(size), 10)
	artifactDetail := &artifactapi.ArtifactDetail{
		CreatedAt:     &createdAt,
		ModifiedAt:    &modifiedAt,
		Name:          &image.Name,
		Version:       artifact.Version,
		DownloadCount: &downloadCount,
		Size:          &totalSize,
	}
	err := artifactDetail.FromRpmArtifactDetailConfig(artifactapi.RpmArtifactDetailConfig{
		Metadata: &metadata,
	})
	if err != nil {
		log.Error().Err(err).Msgf("Error setting the artifact details for artifact: [%s/%s].", image.Name,
			artifact.Version)
		return artifactapi.ArtifactDetail{}
	}
	return *artifactDetail
}

func GetArtifactSummary(artifact types.ImageMetadata) *artifactapi.ArtifactSummaryResponseJSONResponse {
	createdAt := GetTimeInMs(artifact.CreatedAt)
	modifiedAt := GetTimeInMs(artifact.ModifiedAt)
	artifactVersionSummary := &artifactapi.ArtifactSummary{
		CreatedAt:      &createdAt,
		ModifiedAt:     &modifiedAt,
		DownloadsCount: &artifact.DownloadCount,
		ImageName:      artifact.Name,
		PackageType:    artifact.PackageType,
		ArtifactType:   artifact.ArtifactType,
	}
	response := &artifactapi.ArtifactSummaryResponseJSONResponse{
		Data:   *artifactVersionSummary,
		Status: artifactapi.StatusSUCCESS,
	}
	return response
}

func GetArtifactVersionSummary(
	artifactName string,
	packageType artifactapi.PackageType,
	version string,
	isQuarantined bool,
	quarantineReason string,
	artifactType *artifactapi.ArtifactType,
) *artifactapi.ArtifactVersionSummaryResponseJSONResponse {
	artifactVersionSummary := &artifactapi.ArtifactVersionSummary{
		ImageName:        artifactName,
		PackageType:      packageType,
		Version:          version,
		IsQuarantined:    &isQuarantined,
		QuarantineReason: &quarantineReason,
		ArtifactType:     artifactType,
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
