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
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types/enum"

	"github.com/opencontainers/go-digest"
)

func (c *APIController) GetDockerArtifactManifest(
	ctx context.Context,
	r artifact.GetDockerArtifactManifestRequestObject,
) (artifact.GetDockerArtifactManifestResponseObject, error) {
	regInfo, err := c.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return artifact.GetDockerArtifactManifest400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	space, err := c.spaceStore.FindByRef(ctx, regInfo.parentRef)
	if err != nil {
		return artifact.GetDockerArtifactManifest400JSONResponse{
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
		return artifact.GetDockerArtifactManifest403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	imageName := string(r.Artifact)
	dgst := string(r.Params.Digest)
	manifestDigest, err := types.NewDigest(digest.Digest(dgst))
	if err != nil {
		return getArtifactManifestErrorResponse(err)
	}

	manifestPayload, err := c.ManifestStore.GetManifestPayload(
		ctx,
		regInfo.parentID,
		regInfo.RegistryIdentifier,
		imageName,
		manifestDigest,
	)

	if err != nil {
		return getArtifactManifestErrorResponse(err)
	}

	payload := *manifestPayload
	return artifact.GetDockerArtifactManifest200JSONResponse{
		DockerArtifactManifestResponseJSONResponse: artifact.DockerArtifactManifestResponseJSONResponse{
			Data: artifact.DockerArtifactManifest{
				Manifest: string(payload),
			},
			Status: artifact.StatusSUCCESS,
		},
	}, nil
}

func getArtifactManifestErrorResponse(err error) (artifact.GetDockerArtifactManifestResponseObject, error) {
	return artifact.GetDockerArtifactManifest500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}, nil
}
