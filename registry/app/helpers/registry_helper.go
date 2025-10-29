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

package helpers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/paths"
	localurlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/interfaces"
	artifactapi "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/api/utils"
	"github.com/harness/gitness/registry/app/common"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	registrypostprocessingevents "github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	registryutils "github.com/harness/gitness/registry/utils"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/inhies/go-bytesize"
	"github.com/rs/zerolog/log"
)

type registryHelper struct {
	ArtifactStore                store.ArtifactRepository
	FileManager                  filemanager.FileManager
	ImageStore                   store.ImageRepository
	ArtifactEventReporter        *registryevents.Reporter
	PostProcessingReporter       *registrypostprocessingevents.Reporter
	tx                           dbtx.Transactor
	URLProvider                  localurlprovider.Provider
	SetupDetailsAuthHeaderPrefix string
}

func NewRegistryHelper(
	artifactStore store.ArtifactRepository,
	fileManager filemanager.FileManager,
	imageStore store.ImageRepository,
	artifactEventReporter *registryevents.Reporter,
	postProcessingReporter *registrypostprocessingevents.Reporter,
	tx dbtx.Transactor,
	urlProvider localurlprovider.Provider,
	setupDetailsAuthHeaderPrefix string,
) interfaces.RegistryHelper {
	return &registryHelper{
		ArtifactStore:                artifactStore,
		FileManager:                  fileManager,
		ImageStore:                   imageStore,
		ArtifactEventReporter:        artifactEventReporter,
		PostProcessingReporter:       postProcessingReporter,
		tx:                           tx,
		URLProvider:                  urlProvider,
		SetupDetailsAuthHeaderPrefix: setupDetailsAuthHeaderPrefix,
	}
}

func (r *registryHelper) GetAuthHeaderPrefix() string {
	return r.SetupDetailsAuthHeaderPrefix
}

func (r *registryHelper) DeleteVersion(ctx context.Context,
	regInfo *types.RegistryRequestBaseInfo,
	imageInfo *types.Image,
	artifactName string,
	versionName string) error {
	_, err := r.ArtifactStore.GetByName(ctx, imageInfo.ID, versionName)
	if err != nil {
		return fmt.Errorf("version doesn't exist with for image %v: %w", imageInfo.Name, err)
	}

	// get the file path based on package type
	filePath, err := utils.GetFilePath(regInfo.PackageType, artifactName, versionName)
	if err != nil {
		return fmt.Errorf("failed to get file path: %w", err)
	}

	err = r.tx.WithTx(
		ctx,
		func(ctx context.Context) error {
			// delete nodes from nodes store
			err = r.FileManager.DeleteNode(ctx, regInfo.RegistryID, filePath)
			if err != nil {
				return err
			}

			// delete artifacts from artifacts store
			err = r.ArtifactStore.DeleteByVersionAndImageName(ctx, artifactName, versionName, regInfo.RegistryID)
			if err != nil {
				return fmt.Errorf("failed to delete version: %w", err)
			}

			// delete image if no other artifacts linked
			err = r.ImageStore.DeleteByImageNameIfNoLinkedArtifacts(ctx, regInfo.RegistryID, artifactName)
			if err != nil {
				return fmt.Errorf("failed to delete image: %w", err)
			}

			return nil
		},
	)

	if err != nil {
		return err
	}

	return nil
}

func (r *registryHelper) ReportDeleteVersionEvent(
	ctx context.Context,
	payload *registryevents.ArtifactDeletedPayload,
) {
	r.ArtifactEventReporter.ArtifactDeleted(ctx, payload)
}

func (r *registryHelper) ReportBuildPackageIndexEvent(
	ctx context.Context, registryID int64, artifactName string,
) {
	if r.PostProcessingReporter != nil {
		r.PostProcessingReporter.BuildPackageIndex(ctx, registryID, artifactName)
	}
}

func (r *registryHelper) ReportBuildRegistryIndexEvent(
	ctx context.Context, registryID int64, sources []types.SourceRef,
) {
	if r.PostProcessingReporter != nil {
		r.PostProcessingReporter.BuildRegistryIndex(ctx, registryID, sources)
	}
}

func (r *registryHelper) DeleteGenericImage(ctx context.Context,
	regInfo *types.RegistryRequestBaseInfo,
	artifactName string, filePath string,
) error {
	err := r.tx.WithTx(
		ctx, func(ctx context.Context) error {
			// Delete Artifact Files
			err := r.FileManager.DeleteNode(ctx, regInfo.RegistryID, filePath)
			if err != nil {
				return fmt.Errorf("failed to delete artifact files: %w", err)
			}
			// Delete Artifacts
			err = r.ArtifactStore.DeleteByImageNameAndRegistryID(ctx, regInfo.RegistryID, artifactName)
			if err != nil {
				return fmt.Errorf("failed to delete versions: %w", err)
			}
			// Delete image
			err = r.ImageStore.DeleteByImageNameAndRegID(
				ctx, regInfo.RegistryID, artifactName,
			)
			if err != nil {
				return fmt.Errorf("failed to delete artifact: %w", err)
			}
			return nil
		},
	)
	return err
}

func (r *registryHelper) GetPackageURL(
	ctx context.Context,
	rootIdentifier string,
	registryIdentifier string,
	packageTypePathParam string,
) string {
	return r.URLProvider.PackageURL(ctx, rootIdentifier+"/"+registryIdentifier, packageTypePathParam)
}

func (r *registryHelper) GetHostName(
	ctx context.Context,
	rootSpace string,
) string {
	return common.TrimURLScheme(r.URLProvider.RegistryURL(ctx, rootSpace))
}

func (r *registryHelper) GetArtifactMetadata(
	artifact types.ArtifactMetadata,
	pullCommand string,
) *artifactapi.ArtifactMetadata {
	lastModified := GetTimeInMs(artifact.ModifiedAt)
	return &artifactapi.ArtifactMetadata{
		RegistryIdentifier: artifact.RepoName,
		Name:               artifact.Name,
		Version:            &artifact.Version,
		Labels:             &artifact.Labels,
		LastModified:       &lastModified,
		PackageType:        artifact.PackageType,
		DownloadsCount:     &artifact.DownloadCount,
		PullCommand:        &pullCommand,
		IsQuarantined:      &artifact.IsQuarantined,
		QuarantineReason:   artifact.QuarantineReason,
		ArtifactType:       artifact.ArtifactType,
	}
}

func (r *registryHelper) GetArtifactVersionMetadata(
	tag types.NonOCIArtifactMetadata,
	pullCommand string,
	packageType string,
) *artifactapi.ArtifactVersionMetadata {
	modifiedAt := GetTimeInMs(tag.ModifiedAt)
	size := GetImageSize(tag.Size)
	downloadCount := tag.DownloadCount
	fileCount := tag.FileCount
	return &artifactapi.ArtifactVersionMetadata{
		PackageType:      artifactapi.PackageType(packageType),
		FileCount:        &fileCount,
		Name:             tag.Name,
		Size:             &size,
		LastModified:     &modifiedAt,
		PullCommand:      &pullCommand,
		DownloadsCount:   &downloadCount,
		IsQuarantined:    &tag.IsQuarantined,
		QuarantineReason: tag.QuarantineReason,
		ArtifactType:     tag.ArtifactType,
	}
}

func (r *registryHelper) GetFileMetadata(
	file types.FileNodeMetadata,
	filename string,
	downloadCommand string,
) *artifactapi.FileDetail {
	return &artifactapi.FileDetail{
		Checksums:       GetCheckSums(file),
		Size:            GetSize(file.Size),
		CreatedAt:       fmt.Sprint(file.CreatedAt),
		Name:            filename,
		DownloadCommand: downloadCommand,
	}
}

func (r *registryHelper) GetArtifactDetail(
	img *types.Image,
	art *types.Artifact,
	metadata map[string]any,
	downloadCount int64,
) *artifactapi.ArtifactDetail {
	createdAt := GetTimeInMs(art.CreatedAt)
	modifiedAt := GetTimeInMs(art.UpdatedAt)
	size, ok := metadata["size"].(float64)
	if !ok {
		log.Error().Msg(fmt.Sprintf("failed to get size from metadata: %s, %s", img.Name, art.Version))
	}
	totalSize := GetSize(int64(size))
	artifactDetail := &artifactapi.ArtifactDetail{
		CreatedAt:     &createdAt,
		ModifiedAt:    &modifiedAt,
		Name:          &img.Name,
		Version:       art.Version,
		DownloadCount: &downloadCount,
		Size:          &totalSize,
	}
	return artifactDetail
}

func GetTimeInMs(t time.Time) string {
	return fmt.Sprint(t.UnixMilli())
}

func GetImageSize(size string) string {
	sizeVal, _ := strconv.ParseInt(size, 10, 64)
	return GetSize(sizeVal)
}

func GetSize(sizeVal int64) string {
	size := bytesize.New(float64(sizeVal))
	return size.String()
}

func GetCheckSums(file types.FileNodeMetadata) []string {
	return []string{
		fmt.Sprintf("SHA-512: %s", file.Sha512),
		fmt.Sprintf("SHA-256: %s", file.Sha256),
		fmt.Sprintf("SHA-1: %s", file.Sha1),
		fmt.Sprintf("MD5: %s", file.MD5),
	}
}

func (r *registryHelper) ReplacePlaceholders(
	ctx context.Context,
	clientSetupSections *[]artifactapi.ClientSetupSection,
	username string,
	regRef string,
	image *artifactapi.ArtifactParam,
	version *artifactapi.VersionParam,
	registryURL string,
	groupID string,
	uploadURL string,
	hostname string,
) {
	for i := range *clientSetupSections {
		tab, err := (*clientSetupSections)[i].AsTabSetupStepConfig()
		if err != nil || tab.Tabs == nil {
			//nolint:lll
			r.ReplacePlaceholdersInSection(ctx, &(*clientSetupSections)[i], username, regRef, image, version,
				registryURL, groupID, uploadURL, hostname)
		} else {
			for j := range *tab.Tabs {
				r.ReplacePlaceholders(ctx, (*tab.Tabs)[j].Sections, username, regRef, image, version, registryURL, groupID,
					uploadURL, hostname)
			}
			_ = (*clientSetupSections)[i].FromTabSetupStepConfig(tab)
		}
	}
}

func (r *registryHelper) ReplacePlaceholdersInSection(
	ctx context.Context,
	clientSetupSection *artifactapi.ClientSetupSection,
	username string,
	regRef string,
	image *artifactapi.ArtifactParam,
	version *artifactapi.VersionParam,
	registryURL string,
	groupID string,
	uploadURL string,
	hostname string,
) {
	_, registryName, _ := paths.DisectLeaf(regRef)

	sec, err := clientSetupSection.AsClientSetupStepConfig()
	if err != nil || sec.Steps == nil {
		return
	}
	for _, st := range *sec.Steps {
		if st.Commands == nil {
			continue
		}
		for j := range *st.Commands {
			r.ReplaceText(ctx, username, st, j, hostname, registryName, image, version, registryURL, groupID, uploadURL)
		}
	}
	_ = clientSetupSection.FromClientSetupStepConfig(sec)
}

func (r *registryHelper) ReplaceText(
	ctx context.Context,
	username string,
	st artifactapi.ClientSetupStep,
	i int,
	hostname string,
	repoName string,
	image *artifactapi.ArtifactParam,
	version *artifactapi.VersionParam,
	registryURL string,
	groupID string,
	uploadURL string,
) {
	if r.SetupDetailsAuthHeaderPrefix != "" {
		(*st.Commands)[i].Value = registryutils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value,
			"<AUTH_HEADER_PREFIX>", r.SetupDetailsAuthHeaderPrefix))
	}
	if username != "" {
		(*st.Commands)[i].Value = registryutils.StringPtr(
			strings.ReplaceAll(*(*st.Commands)[i].Value, "<USERNAME>", username),
		)
		if (*st.Commands)[i].Label != nil {
			(*st.Commands)[i].Label = registryutils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Label, "<USERNAME>",
				username))
		}
	}
	if groupID != "" {
		(*st.Commands)[i].Value = registryutils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<GROUP_ID>", groupID))
	}
	if registryURL != "" {
		(*st.Commands)[i].Value = registryutils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<REGISTRY_URL>",
			registryURL))
		if (*st.Commands)[i].Label != nil {
			(*st.Commands)[i].Label = registryutils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Label,
				"<REGISTRY_URL>", registryURL))
		}
	}
	if uploadURL != "" {
		(*st.Commands)[i].Value = registryutils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<UPLOAD_URL>",
			uploadURL))
	}
	if hostname != "" {
		(*st.Commands)[i].Value = registryutils.StringPtr(
			strings.ReplaceAll(*(*st.Commands)[i].Value, "<HOSTNAME>", hostname),
		)
	}
	if hostname != "" {
		(*st.Commands)[i].Value = registryutils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value,
			"<LOGIN_HOSTNAME>", common.GetHost(ctx, hostname)))
	}
	if repoName != "" {
		(*st.Commands)[i].Value = registryutils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<REGISTRY_NAME>",
			repoName))
	}
	if image != nil {
		(*st.Commands)[i].Value = registryutils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<IMAGE_NAME>",
			string(*image)))
		(*st.Commands)[i].Value = registryutils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<ARTIFACT_ID>",
			string(*image)))
		(*st.Commands)[i].Value = registryutils.StringPtr(strings.ReplaceAll(*(*st.Commands)[i].Value, "<ARTIFACT_NAME>",
			string(*image)))
	}
	if version != nil {
		(*st.Commands)[i].Value = registryutils.StringPtr(
			strings.ReplaceAll(*(*st.Commands)[i].Value, "<TAG>", string(*version)),
		)
		(*st.Commands)[i].Value = registryutils.StringPtr(
			strings.ReplaceAll(*(*st.Commands)[i].Value, "<VERSION>", string(*version)),
		)
		(*st.Commands)[i].Value = registryutils.StringPtr(
			strings.ReplaceAll(*(*st.Commands)[i].Value, "<DIGEST>", string(*version)),
		)
	}
}
