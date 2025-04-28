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

	"github.com/harness/gitness/registry/app/api/handler/generic"
	"github.com/harness/gitness/registry/app/api/handler/maven"
	"github.com/harness/gitness/registry/app/api/handler/oci"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/api/router/utils"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/docker"
	generic2 "github.com/harness/gitness/registry/app/pkg/generic"
	maven2 "github.com/harness/gitness/registry/app/pkg/maven"
	mavenutils "github.com/harness/gitness/registry/app/pkg/maven/utils"
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

func TrackDownloadStatForGenericArtifact(h *generic.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				methodType := r.Method
				ctx := r.Context()
				sw := &StatusWriter{ResponseWriter: w}

				if http.MethodGet == methodType {
					next.ServeHTTP(sw, r)
				} else {
					next.ServeHTTP(w, r)
					return
				}

				if sw.StatusCode != http.StatusOK && sw.StatusCode != http.StatusTemporaryRedirect {
					return
				}

				info, err := h.GetGenericArtifactInfo(r)
				if !commons.IsEmptyError(err) {
					log.Ctx(ctx).Error().Stack().Str("middleware",
						"TrackDownloadStat").Err(err).Msgf("error while putting download stat of artifact, %v",
						err)
					return
				}

				err = dbDownloadStatForGenericArtifact(ctx, h.Controller, info)
				if !commons.IsEmptyError(err) {
					log.Ctx(ctx).Error().Stack().Str("middleware",
						"TrackDownloadStat").Err(err).Msgf("error while putting download stat of artifact, %v",
						err)
					return
				}
			},
		)
	}
}

func TrackDownloadStatForMavenArtifact(h *maven.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				methodType := r.Method
				ctx := r.Context()
				sw := &StatusWriter{ResponseWriter: w}

				if http.MethodGet == methodType {
					next.ServeHTTP(sw, r)
				} else {
					next.ServeHTTP(w, r)
					return
				}

				if sw.StatusCode != http.StatusOK && sw.StatusCode != http.StatusTemporaryRedirect {
					return
				}

				info, err := h.GetArtifactInfo(r, true)
				if !commons.IsEmpty(err) {
					log.Ctx(ctx).Error().Stack().Str("middleware",
						"TrackDownloadStat").Err(err).Msgf("error while putting download stat of artifact, %v",
						err)
					return
				}

				if !mavenutils.IsMainArtifactFile(info) {
					return
				}

				err2 := dbDownloadStatForMavenArtifact(ctx, h.Controller, info)
				if !commons.IsEmptyError(err2) {
					log.Ctx(ctx).Error().Stack().Str("middleware",
						"TrackDownloadStat").Err(err).Msgf("error while putting download stat of artifact, %v",
						err)
					return
				}
			},
		)
	}
}

func dbDownloadStatForGenericArtifact(
	ctx context.Context,
	c *generic2.Controller,
	info pkg.GenericArtifactInfo,
) errcode.Error {
	registry, err := c.DBStore.RegistryDao.GetByParentIDAndName(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return errcode.ErrCodeInvalidRequest.WithDetail(err)
	}

	image, err := c.DBStore.ImageDao.GetByName(ctx, registry.ID, info.Image)
	if err != nil {
		return errcode.ErrCodeInvalidRequest.WithDetail(err)
	}

	artifact, err := c.DBStore.ArtifactDao.GetByName(ctx, image.ID, info.Version)
	if err != nil {
		return errcode.ErrCodeInvalidRequest.WithDetail(err)
	}

	downloadStat := &types.DownloadStat{
		ArtifactID: artifact.ID,
	}

	if err := c.DBStore.DownloadStatDao.Create(ctx, downloadStat); err != nil {
		return errcode.ErrCodeNameUnknown.WithDetail(err)
	}
	return errcode.Error{}
}

func dbDownloadStatForMavenArtifact(
	ctx context.Context,
	c *maven2.Controller,
	info pkg.MavenArtifactInfo,
) errcode.Error {
	imageName := info.GroupID + ":" + info.ArtifactID
	registry, err := c.DBStore.RegistryDao.GetByParentIDAndName(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return errcode.ErrCodeInvalidRequest.WithDetail(err)
	}

	image, err := c.DBStore.ImageDao.GetByName(ctx, registry.ID, imageName)
	if errors.Is(err, store.ErrResourceNotFound) {
		image, err = getMavenArtifactFromUpstreamProxy(ctx, c, info)
	}
	if err != nil {
		return errcode.ErrCodeInvalidRequest.WithDetail(err)
	}

	artifact, err := c.DBStore.ArtifactDao.GetByName(ctx, image.ID, info.Version)
	if err != nil {
		return errcode.ErrCodeInvalidRequest.WithDetail(err)
	}

	downloadStat := &types.DownloadStat{
		ArtifactID: artifact.ID,
	}

	if err := c.DBStore.DownloadStatDao.Create(ctx, downloadStat); err != nil {
		return errcode.ErrCodeNameUnknown.WithDetail(err)
	}
	return errcode.Error{}
}

func TrackDownloadStats(
	packageHandler packages.Handler,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				sw := &StatusWriter{ResponseWriter: w}
				next.ServeHTTP(sw, r)
				if sw.StatusCode != http.StatusOK && sw.StatusCode != http.StatusTemporaryRedirect {
					return
				}
				err := packageHandler.TrackDownloadStats(ctx, r)
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
