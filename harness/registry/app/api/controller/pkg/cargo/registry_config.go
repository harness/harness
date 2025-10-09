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
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/harness/gitness/registry/app/api/controller/metadata"
	"github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg/commons"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	gitnessenum "github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *controller) GetRegistryConfig(
	ctx context.Context, info *cargotype.ArtifactInfo,
) (*GetRegistryConfigResponse, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}

	registryRef := metadata.GetRegistryRef(info.RootIdentifier, info.RegIdentifier)
	apiURL := c.urlProvider.PackageURL(ctx, registryRef, "cargo")
	downloadURL := c.getDownloadURL(apiURL)
	space, err := c.spaceFinder.FindByID(ctx, info.ParentID)
	if err != nil {
		log.Ctx(ctx).Error().Stack().Str("middleware",
			"OciCheckAuth").Err(err).Msgf("error while fetching the space with ID: %d err: %v", info.ParentID,
			err)
		responseHeaders.Code = http.StatusInternalServerError
		return &GetRegistryConfigResponse{
			BaseResponse: BaseResponse{
				ResponseHeaders: responseHeaders,
			},
		}, fmt.Errorf("failed to get space: %w", err)
	}

	publicAccessSupported, err := c.publicAccessService.
		Get(ctx, gitnessenum.PublicResourceTypeRegistry, space.Path+"/"+info.RegIdentifier)
	if err != nil {
		responseHeaders.Code = http.StatusInternalServerError
		return &GetRegistryConfigResponse{
			BaseResponse: BaseResponse{
				ResponseHeaders: responseHeaders,
			},
		}, fmt.Errorf("failed to check public access: %w", err)
	}

	responseHeaders.Code = http.StatusOK
	return &GetRegistryConfigResponse{
		BaseResponse: BaseResponse{
			ResponseHeaders: responseHeaders,
		},
		Config: &cargo.RegistryConfig{
			DownloadURL:  downloadURL,
			APIURL:       apiURL,
			AuthRequired: !publicAccessSupported,
		},
	}, nil
}

func (c *controller) getDownloadURL(apiURL string) string {
	base, _ := url.Parse(apiURL)
	segments := []string{base.Path}
	segments = append(segments, "api")
	segments = append(segments, "v1")
	segments = append(segments, "crates")
	base.Path = path.Join(segments...)
	return strings.TrimRight(base.String(), "/")
}
