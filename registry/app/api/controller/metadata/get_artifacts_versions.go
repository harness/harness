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
	"fmt"
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	ml "github.com/harness/gitness/registry/app/manifest/manifestlist"
	os "github.com/harness/gitness/registry/app/manifest/ocischema"
	s2 "github.com/harness/gitness/registry/app/manifest/schema2"
	"github.com/harness/gitness/registry/app/pkg/docker"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) GetAllArtifactVersions(
	ctx context.Context,
	r artifact.GetAllArtifactVersionsRequestObject,
) (artifact.GetAllArtifactVersionsResponseObject, error) {
	regInfo, _ := c.GetRegistryRequestInfo(
		ctx, nil, r.Params.Page, r.Params.Size,
		r.Params.SearchTerm, ArtifactVersionResource, "", string(r.RegistryRef),
		nil, r.Params.SortOrder, r.Params.SortField,
	)

	space, err := c.spaceStore.FindByRef(ctx, regInfo.parentRef)
	if err != nil {
		return artifact.GetAllArtifactVersions400JSONResponse{
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
		return artifact.GetAllArtifactVersions403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	image := string(r.Artifact)

	tags, err := c.TagStore.GetAllTagsByRepoAndImage(
		ctx, regInfo.parentID, regInfo.RegistryIdentifier,
		image, regInfo.sortByField, regInfo.sortByOrder, regInfo.limit, regInfo.offset, regInfo.searchTerm,
	)

	latestTag, _ := c.TagStore.GetLatestTagName(ctx, regInfo.parentID, regInfo.RegistryIdentifier, image)

	count, _ := c.TagStore.CountAllTagsByRepoAndImage(
		ctx, regInfo.parentID, regInfo.RegistryIdentifier,
		image, regInfo.searchTerm,
	)

	if err != nil {
		return throw500Error(err)
	}
	err = setDigestCount(ctx, *tags)
	if err != nil {
		return throw500Error(err)
	}

	return artifact.GetAllArtifactVersions200JSONResponse{
		ListArtifactVersionResponseJSONResponse: *GetAllArtifactVersionResponse(
			ctx, tags, latestTag, image, count,
			regInfo, regInfo.pageNumber, regInfo.limit, regInfo.rootIdentifier, c.URLProvider.RegistryURL(),
		),
	}, nil
}

func setDigestCount(ctx context.Context, tags []types.TagMetadata) error {
	for i := range tags {
		err := setDigestCountInTagMetadata(ctx, &tags[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func setDigestCountInTagMetadata(ctx context.Context, t *types.TagMetadata) error {
	m := types.Manifest{
		SchemaVersion: t.SchemaVersion,
		MediaType:     t.MediaType,
		NonConformant: t.NonConformant,
		Payload:       t.Payload,
	}
	manifest, err := docker.DBManifestToManifest(&m)
	if err != nil {
		log.Ctx(ctx).Error().Stack().Err(err).Msg("Failed to convert DBManifest to Manifest")
		return err
	}
	switch reqManifest := manifest.(type) {
	case *s2.DeserializedManifest, *os.DeserializedManifest:
		t.DigestCount = 1
	case *ml.DeserializedManifestList:
		t.DigestCount = len(reqManifest.Manifests)
	default:
		err = fmt.Errorf("unknown manifest type: %T", manifest)
		log.Ctx(ctx).Error().Stack().Err(err).Msg("Failed to set digest count")
	}
	return nil
}

func throw500Error(err error) (artifact.GetAllArtifactVersionsResponseObject, error) {
	wrappedErr := fmt.Errorf("internal server error: %w", err)
	return artifact.GetAllArtifactVersions500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, wrappedErr.Error()),
		),
	}, nil
}
