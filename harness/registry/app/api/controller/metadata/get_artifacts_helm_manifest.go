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
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	store2 "github.com/harness/gitness/store"
	"github.com/harness/gitness/types/enum"

	"github.com/opencontainers/go-digest"
)

func (c *APIController) GetHelmArtifactManifest(
	ctx context.Context,
	r artifact.GetHelmArtifactManifestRequestObject,
) (artifact.GetHelmArtifactManifestResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return c.get400Error(err)
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.GetHelmArtifactManifest400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space, regInfo.RegistryIdentifier,
		enum.PermissionRegistryView)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		return artifact.GetHelmArtifactManifest403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}
	imageName := string(r.Artifact)
	version := string(r.Version)

	var manifestPayload *types.Payload
	if c.UntaggedImagesEnabled(ctx) {
		manifestDigest, err2 := types.NewDigest(digest.Digest(version))
		if err2 != nil {
			return artifact.GetHelmArtifactManifest400JSONResponse{
				BadRequestJSONResponse: artifact.BadRequestJSONResponse(
					*GetErrorResponse(http.StatusBadRequest, err2.Error()),
				),
			}, nil
		}
		manifestPayload, err = c.ManifestStore.GetManifestPayload(
			ctx,
			regInfo.ParentID,
			regInfo.RegistryIdentifier,
			imageName,
			manifestDigest,
		)
	} else {
		manifestPayload, err = c.ManifestStore.FindManifestPayloadByTagName(
			ctx,
			regInfo.ParentID,
			regInfo.RegistryIdentifier,
			imageName,
			version,
		)
	}

	if err != nil {
		if errors.Is(err, store2.ErrResourceNotFound) {
			return artifact.GetHelmArtifactManifest400JSONResponse{
				BadRequestJSONResponse: artifact.BadRequestJSONResponse(
					*GetErrorResponse(http.StatusBadRequest, err.Error()),
				),
			}, nil
		}
		return artifact.GetHelmArtifactManifest500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	payload := *manifestPayload
	return artifact.GetHelmArtifactManifest200JSONResponse{
		HelmArtifactManifestResponseJSONResponse: artifact.HelmArtifactManifestResponseJSONResponse{
			Data: artifact.HelmArtifactManifest{
				Manifest: string(payload),
			},
			Status: artifact.StatusSUCCESS,
		},
	}, nil
}

func (c *APIController) get400Error(err error) (artifact.GetHelmArtifactManifestResponseObject, error) {
	return artifact.GetHelmArtifactManifest400JSONResponse{
		BadRequestJSONResponse: artifact.BadRequestJSONResponse(
			*GetErrorResponse(http.StatusBadRequest, err.Error()),
		),
	}, nil
}
