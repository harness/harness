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

type StatusWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (w *StatusWriter) WriteHeader(code int) {
	w.StatusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *StatusWriter) Write(p []byte) (n int, err error) {
	n, err = w.ResponseWriter.Write(p)
	if w.StatusCode == 0 {
		w.StatusCode = http.StatusOK
	}
	return
}

func TrackBandwidthStat(h *oci.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path
				methodType := r.Method

				requestType := utils.GetRouteTypeV2(path)

				sw := &StatusWriter{ResponseWriter: w}

				bandwidthType := types.BandwidthTypeUPLOAD
				//nolint:gocritic
				if utils.Blobs == requestType && http.MethodGet == methodType {
					next.ServeHTTP(sw, r)
					bandwidthType = types.BandwidthTypeDOWNLOAD
				} else if utils.BlobsUploadsSession == requestType && http.MethodPut == methodType {
					next.ServeHTTP(sw, r)
				} else {
					next.ServeHTTP(w, r)
					return
				}

				if types.BandwidthTypeUPLOAD == bandwidthType && sw.StatusCode != http.StatusCreated {
					return
				} else if types.BandwidthTypeDOWNLOAD == bandwidthType && sw.StatusCode != http.StatusOK {
					return
				}
				ctx := r.Context()

				info, err := h.GetRegistryInfo(r, true)
				if err != nil {
					log.Ctx(ctx).Error().Stack().Str("middleware",
						"TrackBandwidthStat").Err(err).Msgf("error while putting bandwidth stat for artifact, %v",
						err)
					return
				}

				err = dbBandwidthStat(ctx, h.Controller, info, bandwidthType)
				if err != nil {
					log.Ctx(ctx).Error().Stack().Str("middleware",
						"TrackBandwidthStat").Err(err).Msgf("error while putting bandwidth stat for artifact [%s:%s], %v",
						info.RegIdentifier, info.Image, err)
					return
				}
			},
		)
	}
}

func dbBandwidthStat(
	ctx context.Context,
	c *docker.Controller,
	info pkg.RegistryInfo,
	bandwidthType types.BandwidthType,
) error {
	dgst := digest.Digest(info.Digest)
	registry, err := c.RegistryDao.GetByParentIDAndName(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return err
	}

	blob, err := c.DBStore.BlobRepo.FindByDigestAndRootParentID(ctx, dgst, info.RootParentID)
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

	bandwidthStat := &types.BandwidthStat{
		ImageID: image.ID,
		Type:    bandwidthType,
		Bytes:   blob.Size,
	}

	if err := c.DBStore.BandwidthStatDao.Create(ctx, bandwidthStat); err != nil {
		return err
	}
	return nil
}

func getImageFromUpstreamProxy(ctx context.Context, c *docker.Controller, info pkg.RegistryInfo) (*types.Image, error) {
	repos, err := c.GetOrderedRepos(ctx, info.RegIdentifier, *info.BaseInfo)
	if err != nil {
		return nil, err
	}
	for _, registry := range repos {
		log.Ctx(ctx).Info().Msgf("Using Repository: %s, Type: %s", registry.Name, registry.Type)
		image, err := c.DBStore.ImageDao.GetByName(ctx, registry.ID, info.Image)
		if err == nil && image != nil {
			return image, nil
		}
	}
	return nil, errors.New("image not found in upstream proxy")
}
