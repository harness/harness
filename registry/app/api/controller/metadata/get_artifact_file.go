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
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/api/utils"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types/enum"
)

func (c *APIController) GetArtifactFile(
	ctx context.Context,
	r artifact.GetArtifactFileRequestObject,
) (artifact.GetArtifactFileResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return artifact.GetArtifactFile400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.GetArtifactFile400JSONResponse{
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
		if errors.Is(err, apiauth.ErrUnauthorized) {
			return artifact.GetArtifactFile401JSONResponse{
				UnauthenticatedJSONResponse: artifact.UnauthenticatedJSONResponse(
					*GetErrorResponse(http.StatusUnauthorized, err.Error()),
				),
			}, nil
		}
		return artifact.GetArtifactFile403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	image := string(r.Artifact)
	version := string(r.Version)
	file := string(r.FileName)

	registry, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.ParentID, regInfo.RegistryIdentifier)

	if err != nil {
		return artifact.GetArtifactFile500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	var artifactType *artifact.ArtifactType
	if r.Params.ArtifactType != nil {
		artifactType, err = ValidateAndGetArtifactType(registry.PackageType, string(*r.Params.ArtifactType))
		if err != nil {
			return artifact.GetArtifactFile400JSONResponse{
				BadRequestJSONResponse: artifact.BadRequestJSONResponse(
					*GetErrorResponse(http.StatusBadRequest, err.Error()),
				),
			}, nil
		}
	}
	img, err := c.ImageStore.GetByNameAndType(ctx, regInfo.RegistryID, image, artifactType)

	if err != nil {
		if errors.Is(err, store.ErrResourceNotFound) {
			return artifact.GetArtifactFile404JSONResponse{
				NotFoundJSONResponse: artifact.NotFoundJSONResponse(
					*GetErrorResponse(http.StatusNotFound, "Artifact not found"),
				),
			}, nil
		}
		return artifact.GetArtifactFile500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	art, err := c.ArtifactStore.GetByName(ctx, img.ID, version)

	if err != nil {
		if errors.Is(err, store.ErrResourceNotFound) {
			return artifact.GetArtifactFile404JSONResponse{
				NotFoundJSONResponse: artifact.NotFoundJSONResponse(
					*GetErrorResponse(http.StatusNotFound, "Artifact version not found"),
				),
			}, nil
		}
		return artifact.GetArtifactFile500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	registryURL := c.URLProvider.PackageURL(ctx, regInfo.RootIdentifier+"/"+regInfo.RegistryIdentifier,
		strings.ToLower(string(registry.PackageType)))
	filePathPrefix, err := utils.GetFilePath(registry.PackageType, img.Name, art.Version)
	if err != nil {
		return artifact.GetArtifactFile500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	filePath := filePathPrefix + "/" + file
	fileInfo, err := c.fileManager.GetFileMetadata(ctx, img.RegistryID, filePath)

	if err != nil {
		if errors.Is(err, store.ErrResourceNotFound) {
			return artifact.GetArtifactFile404JSONResponse{
				NotFoundJSONResponse: artifact.NotFoundJSONResponse(
					*GetErrorResponse(http.StatusNotFound, "File not found"),
				),
			}, nil
		}
		return artifact.GetArtifactFile500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	return artifact.GetArtifactFile200JSONResponse{
		ArtifactFileResponseJSONResponse: *GetArtifactFileResponseJSONResponse(
			registryURL, registry.PackageType, img.Name, art.Version, fileInfo.Filename),
	}, nil
}
