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
	"github.com/harness/gitness/registry/types"
	store2 "github.com/harness/gitness/store"
	"github.com/harness/gitness/types/enum"

	"github.com/opencontainers/go-digest"
)

func (c *APIController) GetDockerArtifactLayers(
	ctx context.Context,
	r artifact.GetDockerArtifactLayersRequestObject,
) (artifact.GetDockerArtifactLayersResponseObject, error) {
	regInfo, _ := c.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))

	space, err := c.spaceStore.FindByRef(ctx, regInfo.parentRef)
	if err != nil {
		return artifact.GetDockerArtifactLayers400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := getPermissionChecks(space, regInfo.RegistryIdentifier, enum.PermissionRegistryView)
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
		return getLayersErrorResponse(err)
	}
	registry, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.parentID, regInfo.RegistryIdentifier)
	if err != nil {
		return getLayersErrorResponse(err)
	}
	if registry == nil {
		return getLayersErrorResponse(fmt.Errorf("repository not found"))
	}

	m, err := c.ManifestStore.FindManifestByDigest(ctx, registry.ID, image, dgst)
	if err != nil {
		if errors.Is(err, store2.ErrResourceNotFound) {
			return getLayersErrorResponse(fmt.Errorf("manifest not found"))
		}
		return getLayersErrorResponse(err)
	}

	mConfig, err := getManifestConfig(ctx, m.Configuration.Digest, regInfo.rootIdentifier, c.StorageDriver)
	if err != nil {
		return getLayersErrorResponse(err)
	}

	layersSummary := &artifact.DockerLayersSummary{
		Digest: m.Digest.String(),
	}

	if mConfig != nil {
		osArch := fmt.Sprintf("%s/%s", mConfig.Os, mConfig.Arch)
		layersSummary.OsArch = &osArch
		var historyLayers []artifact.DockerLayerEntry
		for _, history := range mConfig.History {
			historyLayers = append(
				historyLayers, artifact.DockerLayerEntry{
					Command: history.CreatedBy,
				},
			)
		}
		layersSummary.Layers = &historyLayers
	} else {
		return getLayersErrorResponse(fmt.Errorf("manifest config not found"))
	}

	return artifact.GetDockerArtifactLayers200JSONResponse{
		DockerLayersResponseJSONResponse: artifact.DockerLayersResponseJSONResponse{
			Data:   *layersSummary,
			Status: artifact.StatusSUCCESS,
		},
	}, nil
}

func getLayersErrorResponse(err error) (artifact.GetDockerArtifactLayersResponseObject, error) {
	return artifact.GetDockerArtifactLayers500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}, nil
}
