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

package metadata

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	m "github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/manifest/ocischema"
	"github.com/harness/gitness/registry/app/manifest/schema2"
	"github.com/harness/gitness/registry/app/pkg/docker"
	"github.com/harness/gitness/registry/types"
	store2 "github.com/harness/gitness/store"
	"github.com/harness/gitness/types/enum"

	"github.com/distribution/distribution/v3/registry/api/errcode"
	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

const (
	KB          = 1024
	MB          = 1024 * KB
	GB          = 1024 * MB
	DefaultSize = "0 B"
)

func (c *APIController) GetDockerArtifactLayers(
	ctx context.Context,
	r artifact.GetDockerArtifactLayersRequestObject,
) (artifact.GetDockerArtifactLayersResponseObject, error) {
	regInfo, _ := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.GetDockerArtifactLayers400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space,
		regInfo.RegistryIdentifier, enum.PermissionRegistryView)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		return artifact.GetDockerArtifactLayers403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	manifestDigest := string(r.Params.Digest)
	image := string(r.Artifact)

	dgst, err := types.NewDigest(digest.Digest(manifestDigest))
	if err != nil {
		return getLayersErrorResponse(ctx, err)
	}
	registry, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.ParentID, regInfo.RegistryIdentifier)
	if err != nil {
		return getLayersErrorResponse(ctx, err)
	}
	if registry == nil {
		return getLayersErrorResponse(ctx, fmt.Errorf("repository not found"))
	}

	m, err := c.ManifestStore.FindManifestByDigest(ctx, registry.ID, image, dgst)
	if err != nil {
		if errors.Is(err, store2.ErrResourceNotFound) {
			return getLayersErrorResponse(ctx, fmt.Errorf("manifest not found"))
		}
		return getLayersErrorResponse(ctx, err)
	}

	mConfig, err := getManifestConfig(ctx, m.Configuration.Digest, regInfo.RootIdentifier, c.StorageDriver)
	if err != nil {
		return getLayersErrorResponse(ctx, err)
	}

	layersSummary := &artifact.DockerLayersSummary{
		Digest: m.Digest.String(),
	}

	if mConfig != nil {
		osArch := fmt.Sprintf("%s/%s", mConfig.Os, mConfig.Arch)
		layersSummary.OsArch = &osArch
		var historyLayers []artifact.DockerLayerEntry
		manifest, err := docker.DBManifestToManifest(m)
		if err != nil {
			return getLayersErrorResponse(ctx, fmt.Errorf("failed to convert DB manifest to manifest: %w", err))
		}
		layers, err := getManifestLayers(manifest)
		if err != nil {
			return getLayersErrorResponse(ctx, err)
		}
		layerIndex := 0

		for _, history := range mConfig.History {
			var layerEntry = &artifact.DockerLayerEntry{
				Command: history.CreatedBy,
			}
			sizeString := DefaultSize
			if !history.EmptyLayer && len(layers) > layerIndex {
				sizeString = GetSizeString(layers[layerIndex].Size)
				layerIndex++
			}
			layerEntry.Size = &sizeString
			historyLayers = append(
				historyLayers, *layerEntry,
			)
		}
		layersSummary.Layers = &historyLayers
	} else {
		return getLayersErrorResponse(ctx, fmt.Errorf("manifest config not found"))
	}

	return artifact.GetDockerArtifactLayers200JSONResponse{
		DockerLayersResponseJSONResponse: artifact.DockerLayersResponseJSONResponse{
			Data:   *layersSummary,
			Status: artifact.StatusSUCCESS,
		},
	}, nil
}

func getLayersErrorResponse(ctx context.Context, err error) (artifact.GetDockerArtifactLayersResponseObject, error) {
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to get layers: %v", err)
	}
	return artifact.GetDockerArtifactLayers500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}, nil
}

func getManifestLayers(
	manifest m.Manifest,
) ([]m.Descriptor, error) {
	switch manifest.(type) {
	case *schema2.DeserializedManifest:
		deserializedManifest := &schema2.DeserializedManifest{}
		mediaType, bytes, _ := manifest.Payload()
		err := deserializedManifest.UnmarshalJSON(bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s manifest: %w", mediaType, err)
		}
		return deserializedManifest.Layers(), nil
	case *ocischema.DeserializedManifest:
		deserializedManifest := &ocischema.DeserializedManifest{}
		mediaType, bytes, _ := manifest.Payload()
		err := deserializedManifest.UnmarshalJSON(bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s manifest: %w", mediaType, err)
		}
		return deserializedManifest.Layers(), nil
	default:
		return nil, errcode.ErrorCodeManifestInvalid.WithDetail("manifest type unsupported")
	}
}

func GetSizeString(size int64) string {
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
