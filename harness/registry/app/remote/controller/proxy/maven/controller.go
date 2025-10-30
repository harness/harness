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
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/maven/utils"
	"github.com/harness/gitness/registry/app/storage"
	cfg "github.com/harness/gitness/registry/config"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
)

type controller struct {
	localRegistry registryInterface
	secretService secret.Service
	spaceFinder   refcache.SpaceFinder
}

type Controller interface {
	UseLocalFile(ctx context.Context, info pkg.MavenArtifactInfo) (
		responseHeaders *commons.ResponseHeaders, fileReader *storage.FileReader, redirectURL string, useLocal bool,
	)

	ProxyFile(
		ctx context.Context, info pkg.MavenArtifactInfo, proxy types.UpstreamProxy, serveFile bool,
	) (*commons.ResponseHeaders, io.ReadCloser, error)
}

// NewProxyController -- get the proxy controller instance.
func NewProxyController(
	l registryInterface, secretService secret.Service,
	spaceFinder refcache.SpaceFinder,
) Controller {
	return &controller{
		localRegistry: l,
		secretService: secretService,
		spaceFinder:   spaceFinder,
	}
}

func (c *controller) UseLocalFile(ctx context.Context, info pkg.MavenArtifactInfo) (
	responseHeaders *commons.ResponseHeaders, fileReader *storage.FileReader, redirectURL string, useLocal bool,
) {
	responseHeaders, body, _, redirectURL, e := c.localRegistry.GetArtifact(ctx, info)
	return responseHeaders, body, redirectURL, len(e) == 0
}

func (c *controller) ProxyFile(
	ctx context.Context, info pkg.MavenArtifactInfo, proxy types.UpstreamProxy, serveFile bool,
) (responseHeaders *commons.ResponseHeaders, body io.ReadCloser, errs error) {
	responseHeaders = &commons.ResponseHeaders{
		Headers: make(map[string]string),
	}
	rHelper, err := NewRemoteHelper(ctx, c.spaceFinder, c.secretService, proxy)
	if err != nil {
		return responseHeaders, nil, err
	}

	filePath := utils.GetFilePath(info)

	filePath = strings.Trim(filePath, "/")

	if serveFile {
		responseHeaders, body, err = rHelper.GetFile(ctx, filePath)
	} else {
		responseHeaders, err = rHelper.HeadFile(ctx, filePath)
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
			log.Ctx(ctx).Error().Stack().Err(err).Msg("failed to get auth session from context")
			return
		}
		ctx2 := request.WithAuthSession(ctx, session)
		ctx2 = context.WithoutCancel(ctx2)
		ctx2 = context.WithValue(ctx2, cfg.GoRoutineKey, "goRoutine")
		err = c.putFileToLocal(ctx2, info, rHelper)
		if err != nil {
			log.Ctx(ctx2).Error().Stack().Err(err).Msgf("error while putting file to localRegistry, %v", err)
			return
		}
		log.Ctx(ctx2).Info().Msgf("Successfully updated file "+
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

	_, fileReader, err := r.GetFile(ctx, filePath)
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
