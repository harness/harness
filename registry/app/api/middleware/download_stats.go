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

package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/harness/gitness/registry/app/api/handler/oci"
	"github.com/harness/gitness/registry/app/api/router/utils"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/docker"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store"

	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

func TrackDownloadStat(h *oci.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path
				methodType := r.Method

				requestType := utils.GetRouteTypeV2(path)

				sw := &StatusWriter{ResponseWriter: w}

				if utils.Manifests == requestType && http.MethodGet == methodType {
					next.ServeHTTP(sw, r)
				} else {
					next.ServeHTTP(w, r)
					return
				}

				if sw.StatusCode != http.StatusOK {
					return
				}
				ctx := r.Context()

				info, err := h.GetRegistryInfo(r, true)
				if err != nil {
					log.Ctx(ctx).Error().Stack().Str("middleware",
						"TrackDownloadStat").Err(err).Msgf("error while putting download stat of artifact, %v",
						err)
					return
				}

				err = dbDownloadStat(ctx, h.Controller, info)
				if err != nil {
					log.Ctx(ctx).Error().Stack().Str("middleware",
						"TrackDownloadStat").Err(err).Msgf("error while putting download stat of artifact, %v",
						err)
					return
				}
			},
		)
	}
}

func dbDownloadStat(
	ctx context.Context,
	c *docker.Controller,
	info pkg.RegistryInfo,
) error {
	registry, err := c.RegistryDao.GetByParentIDAndName(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return err
	}

	image, err := c.DBStore.ImageDao.GetByName(ctx, registry.ID, info.Image)
	if errors.Is(err, store.ErrResourceNotFound) {
		image, err = getImageFromUpstreamProxy(ctx, c, info)
	}
	if err != nil {
		return err
	}

	dgst, err := types.NewDigest(digest.Digest(info.Digest))
	if err != nil {
		return err
	}

	artifact, err := c.DBStore.ArtifactDao.GetByName(ctx, image.ID, dgst.String())
	if err != nil {
		return err
	}

	downloadStat := &types.DownloadStat{
		ArtifactID: artifact.ID,
	}

	if err := c.DBStore.DownloadStatDao.Create(ctx, downloadStat); err != nil {
		return err
	}
	return nil
}
