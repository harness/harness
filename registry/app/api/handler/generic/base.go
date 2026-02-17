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
	"path"
	"path/filepath"
	"regexp"
	"strings"

	usercontroller "github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	corestore "github.com/harness/gitness/app/store"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/controller/metadata"
	"github.com/harness/gitness/registry/app/api/controller/pkg/generic"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/api/handler/utils"
	artifact2 "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	generic2 "github.com/harness/gitness/registry/app/pkg/types/generic"
	refcache2 "github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/request"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

const (
	packageNameRegex  = `^[a-zA-Z0-9][a-zA-Z0-9._-]*[a-zA-Z0-9]$`
	versionRegex      = `^[a-z0-9][a-z0-9.-]*[a-z0-9]$`
	versionRegexV2    = `^[A-Za-z0-9](?:[A-Za-z0-9.-]*[A-Za-z0-9])?$`
	filePathRegex     = `^(?:[A-Za-z0-9._~@-]+/)*[A-Za-z0-9._~@-]+$`
	maxFilePathLength = 4000
)

var (
	packageNameRe = regexp.MustCompile(packageNameRegex)
	versionRe     = regexp.MustCompile(versionRegex)
	versionV2Re   = regexp.MustCompile(versionRegexV2)
	filePathRe    = regexp.MustCompile(filePathRegex)
)

func NewGenericArtifactHandler(
	spaceStore corestore.SpaceStore, controller *generic.Controller, tokenStore corestore.TokenStore,
	userCtrl *usercontroller.Controller, authenticator authn.Authenticator, urlProvider urlprovider.Provider,
	authorizer authz.Authorizer, packageHandler packages.Handler, spaceFinder refcache.SpaceFinder,
	registryFinder refcache2.RegistryFinder, auditService audit.Service,
) *Handler {
	return &Handler{
		Handler:        packageHandler,
		Controller:     controller,
		SpaceStore:     spaceStore,
		TokenStore:     tokenStore,
		UserCtrl:       userCtrl,
		Authenticator:  authenticator,
		URLProvider:    urlProvider,
		Authorizer:     authorizer,
		SpaceFinder:    spaceFinder,
		RegistryFinder: registryFinder,
		AuditService:   auditService,
	}
}

type Handler struct {
	packages.Handler
	Controller     *generic.Controller
	SpaceStore     corestore.SpaceStore
	TokenStore     corestore.TokenStore
	UserCtrl       *usercontroller.Controller
	Authenticator  authn.Authenticator
	URLProvider    urlprovider.Provider
	Authorizer     authz.Authorizer
	SpaceFinder    refcache.SpaceFinder
	RegistryFinder refcache2.RegistryFinder
	AuditService   audit.Service
}

//nolint:staticcheck
func (h *Handler) GetGenericArtifactInfo(r *http.Request) (
	pkg.GenericArtifactInfo,
	errcode.Error,
) {
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

	rootSpace, err := h.SpaceFinder.FindByRef(ctx, rootIdentifier)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Root space not found: %s", rootIdentifier)
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

	if info.Image != "" && info.Version != "" && info.FileName != "" {
		err2 := utils.PatternAllowed(registry.AllowedPattern, registry.BlockedPattern,
			info.Image+":"+info.Version+":"+info.FileName)
		if err2 != nil {
			return pkg.GenericArtifactInfo{}, errcode.ErrCodeInvalidRequest.WithDetail(err2)
		}
	}

	return info, errcode.Error{}
}

func (h *Handler) GetGenericArtifactInfoV2(r *http.Request) (generic2.ArtifactInfo, error) {
	ctx := r.Context()
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/")
	splits := strings.Split(path, "/")
	filePath := strings.Join(splits[6:], "/")
	fileName := splits[len(splits)-1]

	rootIdentifier, registryIdentifier, packageName, version :=
		chi.URLParam(r, "rootIdentifier"),
		chi.URLParam(r, "registryIdentifier"),
		chi.URLParam(r, "package"),
		chi.URLParam(r, "version")

	if err := validatePackageVersionV2(packageName, version); err != nil {
		return generic2.ArtifactInfo{}, fmt.Errorf("invalid image name/version/fileName: %q/%q %w", packageName,
			version, err)
	}

	if err := validateFilePath(filePath); err != nil {
		return generic2.ArtifactInfo{}, usererror.BadRequestf("%v", err)
	}

	rootSpace, err := h.SpaceFinder.FindByRef(ctx, rootIdentifier)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Root space not found: %q", rootIdentifier)
		return generic2.ArtifactInfo{}, usererror.NotFoundf("Root %q not found", rootIdentifier)
	}

	registry, err := h.RegistryFinder.FindByRootRef(ctx, rootSpace.Identifier, registryIdentifier)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf(
			"registry %q not found for root: %q. Reason: %q", registryIdentifier, rootSpace.Identifier, err,
		)
		return generic2.ArtifactInfo{}, usererror.NotFoundf("Registry %q not found for root: %q", registryIdentifier,
			rootSpace.Identifier)
	}

	info := generic2.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo: &pkg.BaseInfo{
				PathPackageType: registry.PackageType,
				ParentID:        registry.ParentID,
				RootIdentifier:  rootIdentifier,
				RootParentID:    rootSpace.ID,
			},
			RegIdentifier: registryIdentifier,
			RegistryID:    registry.ID,
			Registry:      *registry,
			Image:         packageName,
			ArtifactType:  nil,
		},
		FileName:    fileName,
		FilePath:    filePath,
		Version:     version,
		Description: "",
	}

	log.Ctx(ctx).Info().Msgf("Dispatch: URI: %s", path)
	if info.Image == "" || info.Version == "" || info.FileName == "" || info.FilePath == "" {
		log.Ctx(ctx).Warn().Msgf("Invalid request")
		return generic2.ArtifactInfo{}, usererror.BadRequestf("Invalid request Image: %q, Version: %q, FileName: %q",
			info.Image, info.Version,
			info.FileName)
	}

	if info.Image != "" && info.Version != "" && info.FilePath != "" {
		artifact := info.Image + ":" + info.Version + ":" + info.FilePath
		err2 := utils.PatternAllowed(registry.AllowedPattern, registry.BlockedPattern,
			artifact)
		if err2 != nil {
			return generic2.ArtifactInfo{}, usererror.BadRequestf("Invalid request: File path %q not "+
				"allowed due to allowed / blocked patterns",
				artifact)
		}
	}

	return info, nil
}

func (h *Handler) GetPackageArtifactInfo(r *http.Request) (pkg.PackageArtifactInfo, error) {
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/")
	splits := strings.Split(path, "/")
	if len(splits) >= 7 && splits[0] == "pkg" && splits[3] == "files" {
		artifactInfo, err := h.GetGenericArtifactInfoV2(r)
		if err != nil {
			return nil, fmt.Errorf("failed to get generic artifact info: %w", err)
		}
		return artifactInfo, nil
	} else if len(splits) >= 4 && splits[0] == "pkg" && splits[3] == "files" {
		// Missing required path segments (version or filepath)
		return nil, usererror.BadRequestf(
			"Invalid request path: expected /pkg/{root}/{registry}/files/{package}/{version}/{filepath}, got: %s",
			r.URL.Path)
	}
	info, e := h.GetArtifactInfo(r)

	if e != nil {
		log.Ctx(r.Context()).Error().Msgf("failed to get artifact info: %q", e)
		return nil, fmt.Errorf("failed to get artifact info: %w", e)
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

func validatePackageVersionV2(packageName, version string) error {
	// Compile the regular expressions
	// Validate package name
	if !packageNameRe.MatchString(packageName) {
		return usererror.BadRequestf("Invalid package name: %q. Should follow pattern: %q", packageName,
			packageNameRegex)
	}

	// Validate version
	if !versionV2Re.MatchString(version) {
		return usererror.BadRequestf("Invalid version: %q. Should follow pattern: %q", version, versionRegexV2)
	}

	return nil
}

func validatePackageVersion(packageName, version string) error {
	// Compile the regular expressions
	// Validate package name
	if !packageNameRe.MatchString(packageName) {
		return usererror.BadRequestf("Invalid package name: %q. Should follow pattern: %q", packageName,
			packageNameRegex)
	}

	// Validate version
	if !versionRe.MatchString(version) {
		return usererror.BadRequestf("Invalid version: %q. Should follow pattern: %q", version, versionRegex)
	}

	return nil
}

func validateFilePath(filePath string) error {
	if !filePathRe.MatchString(filePath) {
		return fmt.Errorf("invalid file path: %q, should follow pattern: %q", filePath, filePathRegex)
	}
	if path.Clean(filePath) != filePath {
		return fmt.Errorf("relative segments not allowed in file path: %q", filePath)
	}

	parts := strings.SplitSeq(filePath, "/")
	for e := range parts {
		if e == "" || e == "." || e == ".." || filepath.IsAbs(e) {
			return fmt.Errorf("unsafe path element: %q", e)
		}
		if strings.HasPrefix(e, "...") {
			return fmt.Errorf("invalid path component detected: %q", e)
		}
	}
	if filePath == "" || strings.HasPrefix(filePath, "/") {
		return fmt.Errorf("invalid file path: [%s]", filePath)
	}

	if len(filePath) > maxFilePathLength {
		return fmt.Errorf("file path too long: %s", filePath)
	}

	return nil
}
