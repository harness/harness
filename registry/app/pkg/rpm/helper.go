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

package rpm

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/registry/app/events/asyncprocessing"
	rpmmetadata "github.com/harness/gitness/registry/app/metadata/rpm"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/types/rpm"
	rpmutil "github.com/harness/gitness/registry/app/utils/rpm"
	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

type RegistryHelper interface {
	UploadPackage(ctx context.Context, info rpm.ArtifactInfo, file io.Reader) (*commons.ResponseHeaders, string, error)
}

type registryHelper struct {
	localBase              base.LocalBase
	fileManager            filemanager.FileManager
	postProcessingReporter *asyncprocessing.Reporter
}

func NewRegistryHelper(
	localBase base.LocalBase,
	fileManager filemanager.FileManager,
	postProcessingReporter *asyncprocessing.Reporter,
) RegistryHelper {
	return &registryHelper{
		localBase:              localBase,
		fileManager:            fileManager,
		postProcessingReporter: postProcessingReporter,
	}
}

func (c *registryHelper) UploadPackage(
	ctx context.Context,
	info rpm.ArtifactInfo,
	file io.Reader,
) (headers *commons.ResponseHeaders, sha256 string, err error) {
	fileInfo, err := c.fileManager.UploadFileNoDBUpdate(ctx, info.RootIdentifier, nil, file, info.RootParentID,
		info.RegistryID)
	if err != nil {
		return nil, "", err
	}
	r, err := c.fileManager.DownloadFileByDigest(ctx, info.RootIdentifier, fileInfo, info.RootParentID, info.RegistryID)
	if err != nil {
		return nil, "", err
	}
	defer r.Close()

	p, err := rpmutil.ParsePackage(r)
	if err != nil {
		log.Printf("failed to parse rpm package: %v", err)
		return nil, "", err
	}

	artifactVersion := p.Version + "." + p.FileMetadata.Architecture
	if p.FileMetadata.Epoch != "" && p.FileMetadata.Epoch != "0" {
		artifactVersion = fmt.Sprintf("%s:%s", p.FileMetadata.Epoch, artifactVersion)
	}
	info.Image = p.Name
	info.Version = artifactVersion
	info.Metadata = rpmmetadata.Metadata{
		VersionMetadata: *p.VersionMetadata,
		FileMetadata:    *p.FileMetadata,
	}

	pathVersion := fmt.Sprintf("%s/%s", p.Version, p.FileMetadata.Architecture)
	if p.FileMetadata.Epoch != "" && p.FileMetadata.Epoch != "0" {
		pathVersion = fmt.Sprintf("%s/%s", pathVersion, p.FileMetadata.Epoch)
	}

	rpmFileName := fmt.Sprintf("%s-%s.%s.rpm", p.Name, p.Version, p.FileMetadata.Architecture)
	path := fmt.Sprintf("%s/%s/%s", p.Name, pathVersion, rpmFileName)
	fileInfo.Filename = rpmFileName
	rs, sha256, artifactID, existent, err := c.localBase.UpdateFileManagerAndCreateArtifact(ctx, info.ArtifactInfo,
		info.Version, path, &rpmmetadata.RpmMetadata{
			Metadata: info.Metadata,
		}, fileInfo, true)

	if err != nil {
		return nil, "", err
	}

	if !existent {
		sources := make([]types.SourceRef, 0)
		sources = append(sources, types.SourceRef{Type: types.SourceTypeArtifact, ID: artifactID})
		c.postProcessingReporter.BuildRegistryIndex(ctx, info.RegistryID, sources)
	}
	return rs, sha256, err
}
