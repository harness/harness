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

package generic

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	usercontroller "github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	corestore "github.com/harness/gitness/app/store"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/controller/metadata"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/api/handler/utils"
	artifact2 "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/generic"
	"github.com/harness/gitness/registry/app/pkg/types/npm"
	"github.com/harness/gitness/registry/request"

	"github.com/rs/zerolog/log"
)

const (
	packageNameRegex = `^[a-zA-Z0-9][a-zA-Z0-9._-]*[a-zA-Z0-9]$`
	versionRegex     = `^[a-z0-9][a-z0-9.-]*[a-z0-9]$`
	// Add other route types here.
)

func NewGenericArtifactHandler(
	spaceStore corestore.SpaceStore, controller *generic.Controller, tokenStore corestore.TokenStore,
	userCtrl *usercontroller.Controller, authenticator authn.Authenticator, urlProvider urlprovider.Provider,
	authorizer authz.Authorizer, packageHandler packages.Handler, spaceFinder refcache.SpaceFinder,
) *Handler {
	return &Handler{
		Handler:       packageHandler,
		Controller:    controller,
		SpaceStore:    spaceStore,
		TokenStore:    tokenStore,
		UserCtrl:      userCtrl,
		Authenticator: authenticator,
		URLProvider:   urlProvider,
		Authorizer:    authorizer,
		SpaceFinder:   spaceFinder,
	}
}

type Handler struct {
	packages.Handler
	Controller    *generic.Controller
	SpaceStore    corestore.SpaceStore
	TokenStore    corestore.TokenStore
	UserCtrl      *usercontroller.Controller
	Authenticator authn.Authenticator
	URLProvider   urlprovider.Provider
	Authorizer    authz.Authorizer
	SpaceFinder   refcache.SpaceFinder
}

func (h *Handler) GetGenericArtifactInfo(r *http.Request) (pkg.GenericArtifactInfo, errcode.Error) {
	ctx := r.Context()
	info, ok := request.ArtifactInfoFrom(ctx).(pkg.GenericArtifactInfo)
	if ok {
		return info, errcode.Error{}
	}
	path := r.URL.Path
	rootIdentifier, registryIdentifier, artifact, tag, fileName, description, err := ExtractPathVars(r)

	if err != nil {
		return pkg.GenericArtifactInfo{}, errcode.ErrCodeInvalidRequest.WithDetail(err)
	}

	if err := metadata.ValidateIdentifier(registryIdentifier); err != nil {
		return pkg.GenericArtifactInfo{}, errcode.ErrCodeInvalidRequest.WithDetail(err)
	}

	if err := validatePackageVersion(artifact, tag); err != nil {
		return pkg.GenericArtifactInfo{}, errcode.ErrCodeInvalidRequest.WithDetail(err)
	}

	rootSpaceID, err := h.SpaceStore.FindByRefCaseInsensitive(ctx, rootIdentifier)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Root spaceID not found: %s", rootIdentifier)
		return pkg.GenericArtifactInfo{}, errcode.ErrCodeRootNotFound.WithDetail(err)
	}
	rootSpace, err := h.SpaceFinder.FindByID(ctx, rootSpaceID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Root space not found: %d", rootSpaceID)
		return pkg.GenericArtifactInfo{}, errcode.ErrCodeRootNotFound.WithDetail(err)
	}

	registry, err := h.Controller.DBStore.RegistryDao.GetByRootParentIDAndName(ctx, rootSpace.ID, registryIdentifier)
	if err != nil {
		log.Ctx(ctx).Error().Msgf(
			"registry %s not found for root: %s. Reason: %s", registryIdentifier, rootSpace.Identifier, err,
		)
		return pkg.GenericArtifactInfo{}, errcode.ErrCodeRegNotFound.WithDetail(err)
	}

	if registry.PackageType != artifact2.PackageTypeGENERIC {
		log.Ctx(ctx).Error().Msgf(
			"registry %s is not a generic artifact registry for root: %s", registryIdentifier, rootSpace.Identifier,
		)
		return pkg.GenericArtifactInfo{}, errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("registry %s is"+
			" not a generic artifact registry", registryIdentifier))
	}

	_, err = h.SpaceFinder.FindByID(r.Context(), registry.ParentID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Parent space not found: %d", registry.ParentID)
		return pkg.GenericArtifactInfo{}, errcode.ErrCodeParentNotFound.WithDetail(err)
	}

	info = pkg.GenericArtifactInfo{
		ArtifactInfo: &pkg.ArtifactInfo{
			BaseInfo: &pkg.BaseInfo{
				RootIdentifier: rootIdentifier,
				RootParentID:   rootSpace.ID,
				ParentID:       registry.ParentID,
			},
			RegIdentifier: registryIdentifier,
			Image:         artifact,
		},
		RegistryID:  registry.ID,
		Version:     tag,
		FileName:    fileName,
		Description: description,
	}

	log.Ctx(ctx).Info().Msgf("Dispatch: URI: %s", path)
	if commons.IsEmpty(rootSpace.Identifier) {
		log.Ctx(ctx).Error().Msgf("ParentRef not found in context")
		return pkg.GenericArtifactInfo{}, errcode.ErrCodeParentNotFound.WithDetail(err)
	}

	if commons.IsEmpty(registryIdentifier) {
		log.Ctx(ctx).Warn().Msgf("registry not found in context")
		return pkg.GenericArtifactInfo{}, errcode.ErrCodeRegNotFound.WithDetail(err)
	}

	if !commons.IsEmpty(info.Image) && !commons.IsEmpty(info.Version) && !commons.IsEmpty(info.FileName) {
		flag, err2 := utils.IsPatternAllowed(registry.AllowedPattern, registry.BlockedPattern,
			info.Image+":"+info.Version+":"+info.FileName)
		if !flag || err2 != nil {
			return pkg.GenericArtifactInfo{}, errcode.ErrCodeInvalidRequest.WithDetail(err2)
		}
	}

	return info, errcode.Error{}
}

// ExtractPathVars extracts registry,image, reference, digest and tag from the path
// Path format: /generic/:rootSpace/:registry/:image/:tag (for ex:
// /generic/myRootSpace/reg1/alpine/v1).
func ExtractPathVars(r *http.Request) (
	rootIdentifier, registry, artifact,
	tag, fileName string, description string, err error,
) {
	path := r.URL.Path

	// Ensure the path starts with "/generic/"
	if !strings.HasPrefix(path, "/generic/") {
		return "", "", "", "", "", "", fmt.Errorf("invalid path: must start with /generic/")
	}

	trimmedPath := strings.TrimPrefix(path, "/generic/")
	firstSlashIndex := strings.Index(trimmedPath, "/")
	if firstSlashIndex == -1 {
		return "", "", "", "", "", "", fmt.Errorf("invalid path format: missing rootIdentifier or registry")
	}
	rootIdentifier = trimmedPath[:firstSlashIndex]

	remainingPath := trimmedPath[firstSlashIndex+1:]
	secondSlashIndex := strings.Index(remainingPath, "/")
	if secondSlashIndex == -1 {
		return "", "", "", "", "", "", fmt.Errorf("invalid path format: missing registry")
	}
	registry = remainingPath[:secondSlashIndex]

	// Extract the artifact and tag from the remaining path
	artifactPath := remainingPath[secondSlashIndex+1:]

	// Check if the artifactPath contains a ":" for tag and filename
	if strings.Contains(artifactPath, ":") {
		segments := strings.SplitN(artifactPath, ":", 3)
		if len(segments) < 3 {
			return "", "", "", "", "", "", fmt.Errorf("invalid artifact format: %s", artifactPath)
		}
		artifact = segments[0]
		tag = segments[1]
		fileName = segments[2]
	} else {
		segments := strings.SplitN(artifactPath, "/", 2)
		if len(segments) < 2 {
			return "", "", "", "", "", "", fmt.Errorf("invalid artifact format: %s", artifactPath)
		}
		artifact = segments[0]
		tag = segments[1]
	}

	return rootIdentifier, registry, artifact, tag, fileName, description, nil
}

func handleErrors(ctx context.Context, err errcode.Error, w http.ResponseWriter) {
	if !commons.IsEmptyError(err) {
		w.WriteHeader(err.Code.Descriptor().HTTPStatusCode)
		_ = errcode.ServeJSON(w, err)
		log.Ctx(ctx).Error().Msgf("Error occurred while performing generic artifact action: %s", err.Message)
	}
}

func validatePackageVersion(packageName, version string) error {
	// Compile the regular expressions
	packageNameRe := regexp.MustCompile(packageNameRegex)
	versionRe := regexp.MustCompile(versionRegex)

	// Validate package name
	if !packageNameRe.MatchString(packageName) {
		return fmt.Errorf("invalid package name: %s", packageName)
	}

	// Validate version
	if !versionRe.MatchString(version) {
		return fmt.Errorf("invalid version: %s", version)
	}

	return nil
}

func (h *Handler) GetPackageArtifactInfo(r *http.Request) (pkg.PackageArtifactInfo, error) {
	info, e := h.GetArtifactInfo(r)

	if !commons.IsEmpty(e) {
		return npm.ArtifactInfo{}, e
	}

	info.Image = r.PathValue("package")
	version := r.PathValue("version")

	if err := validatePackageVersion(info.Image, version); err != nil {
		log.Error().Msgf("Invalid image name/version/fileName: %s/%s", info.Image, version)
		return nil, err
	}

	return pkg.GenericArtifactInfo{
		ArtifactInfo: &info,
		Version:      version,
		RegistryID:   info.RegistryID,
	}, nil
}
