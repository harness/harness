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

	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/utils"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	registrypostprocessingevents "github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"
)

type registryHelper struct {
	ArtifactStore          store.ArtifactRepository
	FileManager            filemanager.FileManager
	ImageStore             store.ImageRepository
	ArtifactEventReporter  *registryevents.Reporter
	PostProcessingReporter *registrypostprocessingevents.Reporter
	tx                     dbtx.Transactor
}

func NewRegistryHelper(
	artifactStore store.ArtifactRepository,
	fileManager filemanager.FileManager,
	imageStore store.ImageRepository,
	artifactEventReporter *registryevents.Reporter,
	postProcessingReporter *registrypostprocessingevents.Reporter,
	tx dbtx.Transactor,
) interfaces.RegistryHelper {
	return &registryHelper{
		ArtifactStore:          artifactStore,
		FileManager:            fileManager,
		ImageStore:             imageStore,
		ArtifactEventReporter:  artifactEventReporter,
		PostProcessingReporter: postProcessingReporter,
		tx:                     tx,
	}
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
