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

	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/registry/app/api/handler/oci"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/api/router/utils"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/docker"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store"

	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

func CheckQuarantineStatus(
	packageHandler packages.Handler,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				sw := &StatusWriter{ResponseWriter: w}
				err := packageHandler.CheckQuarantineStatus(ctx)
				if err != nil {
					log.Ctx(ctx).Error().Stack().Str("middleware",
						"CheckQuarantineStatus").Err(err).Msgf("error while putting download stat of artifact, %v",
						err)
					render.TranslatedUserError(r.Context(), w, err)
					return
				}
				next.ServeHTTP(sw, r)
			},
		)
	}
}

func CheckQuarantineStatusOCI(h *oci.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path
				methodType := r.Method

				requestType := utils.GetRouteTypeV2(path)

				sw := &StatusWriter{ResponseWriter: w}

				if utils.Manifests != requestType || (http.MethodGet != methodType && http.MethodHead != methodType) {
					next.ServeHTTP(sw, r)
					return
				}
				ctx := r.Context()

				info, err := h.GetRegistryInfo(r, true)
				if err != nil {
					log.Ctx(ctx).Error().Stack().Str("middleware",
						"CheckQuarantineStatus").Err(err).Msgf("error while fetching the artifact info: %v",
						err)
					next.ServeHTTP(sw, r)
					return
				}

				err = dbQuarantineStatusOCI(ctx, h.Controller, info)
				if err != nil {
					if errors.Is(err, errcode.ErrCodeManifestQuarantined) {
						render.Forbiddenf(ctx, w, "%s", err.Error())
						return
					}
					log.Ctx(ctx).Error().Stack().Str("middleware",
						"CheckQuarantineStatus").Err(err).Msgf("error while checking the quarantine status of artifact, %v",
						err)
				}
				next.ServeHTTP(sw, r)
			},
		)
	}
}

func dbQuarantineStatusOCI(
	ctx context.Context,
	c *docker.Controller,
	info pkg.RegistryInfo,
) error {
	registry := info.Registry

	var parsedDigest digest.Digest
	var err error
	if info.Digest == "" { //nolint:nestif
		dbManifest, err := c.DBStore.ManifestDao.FindManifestDigestByTagName(ctx, registry.ID, info.Image, info.Tag)
		if err != nil {
			if errors.Is(err, store.ErrResourceNotFound) {
				return nil
			}
			return err
		}
		parsedDigest, err = dbManifest.Parse()
		if err != nil {
			return err
		}
	} else {
		// Convert raw digest string to proper digest format with prefix
		parsedDigest, err = digest.Parse(info.Digest)
		if err != nil {
			return err
		}
	}
	typesDigest, err := types.NewDigest(parsedDigest)
	if err != nil {
		return err
	}
	digestVal := typesDigest.String()
	quarantineArtifacts, err := c.DBStore.QuarantineDao.GetByFilePath(ctx, "", registry.ID, info.Image, digestVal, nil)
	if err != nil {
		return err
	}
	if len(quarantineArtifacts) > 0 {
		return errcode.ErrCodeManifestQuarantined
	}
	return nil
}
