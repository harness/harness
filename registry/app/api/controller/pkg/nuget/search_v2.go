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
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/nuget"
	"github.com/harness/gitness/registry/app/pkg/response"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	registrytypes "github.com/harness/gitness/registry/types"
)

func (c *controller) SearchPackageV2(ctx context.Context, info nugettype.ArtifactInfo,
	searchTerm string, limit, offset int) *SearchPackageV2Response {
	f := func(registry registrytypes.Registry, a pkg.Artifact, l, o int) response.Response {
		info.RegIdentifier = registry.Name
		info.RegistryID = registry.ID
		nugetRegistry, ok := a.(nuget.Registry)
		if !ok {
			return &SearchPackageV2Response{
				BaseResponse{
					fmt.Errorf("invalid registry type: expected nuget.Registry"),
					nil,
				}, nil,
			}
		}

		feedResponse, err := nugetRegistry.SearchPackageV2(ctx, info, searchTerm, l, o)
		return &SearchPackageV2Response{
			BaseResponse{
				err,
				nil,
			}, feedResponse,
		}
	}

	aggregatedResults, totalCount, err := base.SearchPackagesProxyWrapper(ctx,
		c.registryDao, f, extractResponseDataV2, info, limit, offset)

	if err != nil {
		return &SearchPackageV2Response{
			BaseResponse{
				err,
				nil,
			}, nil,
		}
	}

	// Create response using the aggregated results
	feedEntries := make([]*nugettype.FeedEntryResponse, 0, len(aggregatedResults))
	for _, result := range aggregatedResults {
		if entry, ok := result.(*nugettype.FeedEntryResponse); ok {
			feedEntries = append(feedEntries, entry)
		}
	}

	// Get the base URL for proper XML namespace fields
	baseURL := c.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")

	links := []nugettype.FeedEntryLink{
		{Rel: "self", Href: xml.CharData(baseURL)},
	}
	nextURL := ""
	if len(feedEntries) == limit {
		u, _ := url.Parse(baseURL)
		u = u.JoinPath("Search()")
		q := u.Query()
		q.Add("$skip", strconv.Itoa(limit+offset))
		q.Add("$top", strconv.Itoa(limit))
		if searchTerm != "" {
			q.Add("searchTerm", searchTerm)
		}
		u.RawQuery = q.Encode()
		nextURL = u.String()
	}

	if nextURL != "" {
		links = append(links, nugettype.FeedEntryLink{
			Rel:  "next",
			Href: xml.CharData(nextURL),
		})
	}
	// Create FeedResponse with proper XML namespace fields
	feedResponse := &nugettype.FeedResponse{
		Xmlns:   nuget.XMLNamespaceAtom,
		Base:    baseURL,
		XmlnsD:  nuget.XMLNamespaceDataServices,
		XmlnsM:  nuget.XMLNamespaceDataServicesMetadata,
		ID:      nuget.XMLNamespaceDataContract,
		Updated: time.Now(),
		Links:   links,
		Count:   totalCount,
		Entries: feedEntries,
	}

	return &SearchPackageV2Response{
		BaseResponse: BaseResponse{
			Error:           nil,
			ResponseHeaders: nil,
		},
		FeedResponse: feedResponse,
	}
}

func extractResponseDataV2(searchResponse response.Response) ([]interface{}, int64) {
	var nativeResults []interface{}
	var totalHits int64

	// Handle v2 response
	if searchV2Resp, ok := searchResponse.(*SearchPackageV2Response); ok && searchV2Resp.FeedResponse != nil {
		totalHits = searchV2Resp.FeedResponse.Count
		for _, entry := range searchV2Resp.FeedResponse.Entries {
			nativeResults = append(nativeResults, entry)
		}
	}

	return nativeResults, totalHits
}
