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

package maven

import (
	"context"
	"io"
	"strings"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/maven/utils"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
)

type controller struct {
	localRegistry  registryInterface
	secretService  secret.Service
	spacePathStore store.SpacePathStore
}

type Controller interface {
	UseLocalFile(ctx context.Context, info pkg.MavenArtifactInfo) (
		responseHeaders *commons.ResponseHeaders, fileReader *storage.FileReader, redirectURL string, useLocal bool)

	ProxyFile(
		ctx context.Context, info pkg.MavenArtifactInfo, proxy types.UpstreamProxy, serveFile bool,
	) (*commons.ResponseHeaders, io.ReadCloser, error)
}

// NewProxyController -- get the proxy controller instance.
func NewProxyController(
	l registryInterface, secretService secret.Service,
	spacePathStore store.SpacePathStore,
) Controller {
	return &controller{
		localRegistry:  l,
		secretService:  secretService,
		spacePathStore: spacePathStore,
	}
}

func (c *controller) UseLocalFile(ctx context.Context, info pkg.MavenArtifactInfo) (
	responseHeaders *commons.ResponseHeaders, fileReader *storage.FileReader, redirectURL string, useLocal bool) {
	responseHeaders, body, _, redirectURL, e := c.localRegistry.GetArtifact(ctx, info)
	return responseHeaders, body, redirectURL, len(e) == 0
}

func (c *controller) ProxyFile(
	ctx context.Context, info pkg.MavenArtifactInfo, proxy types.UpstreamProxy, serveFile bool,
) (responseHeaders *commons.ResponseHeaders, body io.ReadCloser, errs error) {
	responseHeaders = &commons.ResponseHeaders{
		Headers: make(map[string]string),
	}
	rHelper, err := NewRemoteHelper(ctx, c.spacePathStore, c.secretService, proxy)
	if err != nil {
		return responseHeaders, nil, err
	}

	filePath := utils.GetFilePath(info)

	filePath = strings.Trim(filePath, "/")

	if serveFile {
		responseHeaders, body, err = rHelper.GetFile(filePath)
	} else {
		responseHeaders, _, err = rHelper.HeadFile(filePath)
	}

	if err != nil {
		return responseHeaders, nil, err
	}

	if !serveFile {
		return responseHeaders, nil, nil
	}

	go func(info pkg.MavenArtifactInfo) {
		// Cloning Context.
		session, ok := request.AuthSessionFrom(ctx)
		if !ok {
			log.Error().Stack().Err(err).Msg("failed to get auth session from context")
			return
		}
		ctx2 := request.WithAuthSession(context.Background(), session)

		err = c.putFileToLocal(ctx2, info, rHelper)
		if err != nil {
			log.Ctx(ctx2).Error().Str("goRoutine",
				"AddMavenFile").Stack().Err(err).Msgf("error while putting file to localRegistry, %v", err)
			return
		}
		log.Ctx(ctx2).Info().Str("goRoutine", "AddMavenFile").Msgf("Successfully updated file "+
			"to registry:  %s with file path: %s",
			info.RegIdentifier, filePath)
	}(info)
	return responseHeaders, body, nil
}

func (c *controller) putFileToLocal(
	ctx context.Context,
	info pkg.MavenArtifactInfo,
	r RemoteInterface,
) error {
	filePath := utils.GetFilePath(info)

	filePath = strings.Trim(filePath, "/")

	_, fileReader, err := r.GetFile(filePath)
	if err != nil {
		return err
	}
	defer fileReader.Close()
	_, errs := c.localRegistry.PutArtifact(ctx, info, fileReader)
	if len(errs) > 0 {
		return errs[0]
	}
	return err
}
