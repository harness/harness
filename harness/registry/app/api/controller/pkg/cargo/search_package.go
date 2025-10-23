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

package cargo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg/commons"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

var (
	DefaultPageSize    = 10
	MaxPageSize        = 100
	DefaultSortByField = "image_created_at"
	DefaultSortByOrder = "ASC"
)

func (c *controller) SearchPackage(
	ctx context.Context,
	info *cargotype.ArtifactInfo,
	requestParams *cargotype.SearchPackageRequestParams,
) (*SearchPackageResponse, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	requestInfo := c.getSearchPackageRequestInfo(requestParams)
	registry, err := c.registryFinder.FindByRootParentID(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry %s: %w", info.RegIdentifier, err)
	}

	registryList := make([]string, 0)

	registryList = append(registryList, registry.Name)

	if len(registry.UpstreamProxies) > 0 {
		repoKeys, err := c.registryDao.FetchUpstreamProxyKeys(ctx, registry.UpstreamProxies)
		if err != nil {
			return nil, fmt.Errorf("failed to get upstream proxies repokeys %s: %w", info.RegIdentifier, err)
		}
		registryList = append(registryList, repoKeys...)
	}
	artifacts, err := c.artifactDao.GetAllArtifactsByParentID(
		ctx, info.ParentID, &registryList, requestInfo.SortField, requestInfo.SortOrder,
		requestInfo.Limit, requestInfo.Offset, requestInfo.SearchTerm, true, []string{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts %s: %w", info.RegIdentifier, err)
	}

	count, err := c.artifactDao.CountAllArtifactsByParentID(
		ctx, info.ParentID, &registryList, requestInfo.SearchTerm, true, []string{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts count %s: %w", info.RegIdentifier, err)
	}

	crates := make([]SearchPackageResponseCrate, 0)
	if artifacts != nil {
		for _, artifact := range *artifacts {
			crates = append(crates, *c.mapArtifactToSearchPackageCrate(ctx, artifact))
		}
	}

	responseHeaders.Code = http.StatusOK
	return &SearchPackageResponse{
		BaseResponse: BaseResponse{
			ResponseHeaders: responseHeaders,
		},
		Crates: crates,
		Metadata: SearchPackageResponseMetadata{
			Total: count,
		},
	}, nil
}

func (c *controller) getSearchPackageRequestInfo(
	params *cargotype.SearchPackageRequestParams,
) *cargotype.SearchPackageRequestInfo {
	var requestInfo cargotype.SearchPackageRequestInfo
	if params == nil {
		return &requestInfo
	}

	if params.SearchTerm != nil {
		requestInfo.SearchTerm = *params.SearchTerm
	}
	if params.Size != nil {
		requestInfo.Limit = int(*params.Size)
	} else {
		requestInfo.Limit = DefaultPageSize
	}
	requestInfo.Limit = min(requestInfo.Limit, MaxPageSize)
	requestInfo.Offset = int(0)
	requestInfo.SortField = DefaultSortByField
	requestInfo.SortOrder = DefaultSortByOrder
	return &requestInfo
}

func (c *controller) mapArtifactToSearchPackageCrate(
	ctx context.Context,
	artifact types.ArtifactMetadata,
) *SearchPackageResponseCrate {
	metadata := cargometadata.VersionMetadataDB{}
	err := json.Unmarshal(artifact.Metadata, &metadata)
	if err != nil {
		log.Error().Ctx(ctx).Msg(
			fmt.Sprintf(
				"Failed to unmarshal metadata for package %s version %s: %s",
				artifact.Name, artifact.Version, err.Error(),
			),
		)
	}
	return &SearchPackageResponseCrate{
		Name:        artifact.Name,
		MaxVersion:  artifact.Version,
		Description: metadata.Description,
	}
}
