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

package nuget

import (
	"context"
	"fmt"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/nuget"
	"github.com/harness/gitness/registry/app/pkg/response"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	registrytypes "github.com/harness/gitness/registry/types"
)

func (c *controller) SearchPackage(ctx context.Context, info nugettype.ArtifactInfo,
	searchTerm string, limit, offset int) *SearchPackageResponse {
	f := func(registry registrytypes.Registry, a pkg.Artifact, l, o int) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		nugetRegistry, ok := a.(nuget.Registry)
		if !ok {
			return &SearchPackageResponse{
				BaseResponse{
					fmt.Errorf("invalid registry type: expected nuget.Registry"),
					nil,
				}, nil,
			}
		}
		feedResponse, err := nugetRegistry.SearchPackage(ctx, info, searchTerm, l, o)
		return &SearchPackageResponse{
			BaseResponse{
				err,
				nil,
			}, feedResponse,
		}
	}

	aggregatedResults, totalCount, err := base.SearchPackagesProxyWrapper(ctx, c.registryDao,
		f, extractResponseData, info, limit, offset)

	if err != nil {
		return &SearchPackageResponse{
			BaseResponse{
				err,
				nil,
			}, nil,
		}
	}

	// Create response using the aggregated results
	searchResults := make([]*nugettype.SearchResult, 0, len(aggregatedResults))
	for _, result := range aggregatedResults {
		if searchResult, ok := result.(*nugettype.SearchResult); ok {
			searchResults = append(searchResults, searchResult)
		}
	}

	searchResultResponse := &nugettype.SearchResultResponse{
		TotalHits: totalCount,
		Data:      searchResults,
	}

	return &SearchPackageResponse{
		BaseResponse: BaseResponse{
			Error:           nil,
			ResponseHeaders: nil,
		},
		SearchResponse: searchResultResponse,
	}
}

func extractResponseData(searchResponse response.Response) ([]interface{}, int64) {
	var nativeResults []interface{}
	var totalHits int64

	if searchResp, ok := searchResponse.(*SearchPackageResponse); ok && searchResp.SearchResponse != nil {
		totalHits = searchResp.SearchResponse.TotalHits
		for _, result := range searchResp.SearchResponse.Data {
			nativeResults = append(nativeResults, result)
		}
	}

	return nativeResults, totalHits
}
