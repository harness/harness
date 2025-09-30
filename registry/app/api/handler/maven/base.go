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

package maven

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	usercontroller "github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	corestore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/registry/app/api/controller/metadata"
	"github.com/harness/gitness/registry/app/api/handler/utils"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/maven"
	mavenutils "github.com/harness/gitness/registry/app/pkg/maven/utils"
	"github.com/harness/gitness/registry/request"

	"github.com/rs/zerolog/log"
)

type Handler struct {
	Controller          *maven.Controller
	SpaceStore          corestore.SpaceStore
	TokenStore          corestore.TokenStore
	UserCtrl            *usercontroller.Controller
	Authenticator       authn.Authenticator
	Authorizer          authz.Authorizer
	SpaceFinder         refcache.SpaceFinder
	PublicAccessService publicaccess.Service
}

func NewHandler(
	controller *maven.Controller, spaceStore corestore.SpaceStore, tokenStore corestore.TokenStore,
	userCtrl *usercontroller.Controller, authenticator authn.Authenticator, authorizer authz.Authorizer,
	spaceFinder refcache.SpaceFinder, publicAccessService publicaccess.Service,
) *Handler {
	return &Handler{
		Controller:          controller,
		SpaceStore:          spaceStore,
		TokenStore:          tokenStore,
		UserCtrl:            userCtrl,
		Authenticator:       authenticator,
		Authorizer:          authorizer,
		SpaceFinder:         spaceFinder,
		PublicAccessService: publicAccessService,
	}
}

var (
	illegalCharacters = regexp.MustCompile(`[\\/:"<>|?\*]`)
	invalidPathFormat = "invalid path format: %s"
)

func (h *Handler) GetArtifactInfo(r *http.Request, remoteSupport bool) (pkg.MavenArtifactInfo, error) {
	ctx := r.Context()
	path := r.URL.Path
	rootIdentifier, registryIdentifier, groupID, artifactID, version, fileName, err := ExtractPathVars(path)
	if err != nil {
		return pkg.MavenArtifactInfo{}, err
	}
	if err = metadata.ValidateIdentifier(registryIdentifier); err != nil {
		return pkg.MavenArtifactInfo{}, err
	}

	rootSpaceID, err := h.SpaceStore.FindByRefCaseInsensitive(ctx, rootIdentifier)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Root spaceID not found: %s", rootIdentifier)
		return pkg.MavenArtifactInfo{}, errcode.ErrCodeRootNotFound.WithDetail(err)
	}
	rootSpace, err := h.SpaceFinder.FindByID(ctx, rootSpaceID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Root space not found: %d", rootSpaceID)
		return pkg.MavenArtifactInfo{}, errcode.ErrCodeRootNotFound.WithDetail(err)
	}

	registry, err := h.Controller.DBStore.RegistryDao.GetByRootParentIDAndName(ctx, rootSpace.ID, registryIdentifier)
	if err != nil {
		log.Ctx(ctx).Error().Msgf(
			"registry %s not found for root: %s. Reason: %s", registryIdentifier, rootSpace.Identifier, err,
		)
		return pkg.MavenArtifactInfo{}, errcode.ErrCodeRegNotFound
	}
	_, err = h.SpaceFinder.FindByID(r.Context(), registry.ParentID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Parent space not found: %d", registry.ParentID)
		return pkg.MavenArtifactInfo{}, errcode.ErrCodeParentNotFound
	}

	pathRoot := getPathRoot(r.Context())

	info := &pkg.MavenArtifactInfo{
		ArtifactInfo: &pkg.ArtifactInfo{
			BaseInfo: &pkg.BaseInfo{
				PathRoot:       pathRoot,
				RootIdentifier: rootIdentifier,
				RootParentID:   rootSpace.ID,
				ParentID:       registry.ParentID,
			},
			Registry:      *registry,
			RegIdentifier: registryIdentifier,
			RegistryID:    registry.ID,
			Image:         groupID + ":" + artifactID,
		},
		GroupID:    groupID,
		ArtifactID: artifactID,
		Version:    version,
		FileName:   fileName,
		Path:       r.URL.Path,
	}

	log.Ctx(ctx).Info().Msgf("Dispatch: URI: %s", path)
	if commons.IsEmpty(rootSpace.Identifier) {
		log.Ctx(ctx).Error().Msgf("ParentRef not found in context")
		return pkg.MavenArtifactInfo{}, errcode.ErrCodeParentNotFound
	}

	if commons.IsEmpty(registryIdentifier) {
		log.Ctx(ctx).Warn().Msgf("registry not found in context")
		return pkg.MavenArtifactInfo{}, errcode.ErrCodeRegNotFound
	}

	if !commons.IsEmpty(info.GroupID) && !commons.IsEmpty(info.ArtifactID) && !commons.IsEmpty(info.Version) {
		err2 := utils.PatternAllowed(registry.AllowedPattern, registry.BlockedPattern,
			info.GroupID+":"+info.ArtifactID+":"+info.Version)
		if err2 != nil {
			return pkg.MavenArtifactInfo{}, errcode.ErrCodeDenied
		}
	}

	if registry.Type == artifact.RegistryTypeUPSTREAM && !remoteSupport {
		log.Ctx(ctx).Warn().Msgf("Remote registryIdentifier %s not supported", registryIdentifier)
		return pkg.MavenArtifactInfo{}, errcode.ErrCodeDenied
	}

	return *info, nil
}

// ExtractPathVars extracts registry, groupId, artifactId, version and tag from the path
// Path format: /maven/:rootSpace/:registry/:groupId/artifactId/:version/:filename (for ex:
// /maven/myRootSpace/reg1/io/example/my-app/1.0/my-app-1.0.jar.
func ExtractPathVars(path string) (rootIdentifier, registry, groupID, artifactID, version, fileName string, err error) {
	path = strings.Trim(path, "/")
	segments := strings.Split(path, "/")
	if len(segments) < 6 {
		err = fmt.Errorf(invalidPathFormat, path)
		return "", "", "", "", "", "", err
	}
	rootIdentifier = segments[1]
	registry = segments[2]
	fileName = segments[len(segments)-1]

	if segments[0] == "pkg" {
		segments = segments[4 : len(segments)-1]
	} else {
		segments = segments[3 : len(segments)-1]
	}

	version = segments[len(segments)-1]
	if mavenutils.IsMetadataFile(fileName) && !strings.HasSuffix(version, "-SNAPSHOT") {
		version = ""
	} else {
		segments = segments[:len(segments)-1]
		if len(segments) < 2 {
			err = fmt.Errorf(invalidPathFormat, path)
			return rootIdentifier, registry, groupID, artifactID, version, fileName, err
		}
	}

	artifactID = segments[len(segments)-1]
	groupID = strings.Join(segments[:len(segments)-1], ".")

	if illegalCharacters.MatchString(groupID) || illegalCharacters.MatchString(artifactID) ||
		illegalCharacters.MatchString(version) {
		err = fmt.Errorf(invalidPathFormat, path)
		return rootIdentifier, registry, groupID, artifactID, version, fileName, err
	}
	return rootIdentifier, registry, groupID, artifactID, version, fileName, nil
}

func getPathRoot(ctx context.Context) string {
	originalURL := request.OriginalURLFrom(ctx)
	pathRoot := ""
	if originalURL != "" {
		originalURL = strings.Trim(originalURL, "/")
		segments := strings.Split(originalURL, "/")
		if len(segments) > 1 {
			pathRoot = segments[1]
		}
	}
	return pathRoot
}

func handleErrors(ctx context.Context, errs errcode.Errors, w http.ResponseWriter, headers *commons.ResponseHeaders) {
	if !commons.IsEmpty(errs) {
		LogError(errs)
		log.Ctx(ctx).Error().Errs("errs occurred during maven operation: ", errs).Msgf("Error occurred")
		err := errs[0]
		var e *commons.Error
		switch {
		case headers != nil:
			headers.WriteToResponse(w)
		case errors.As(err, &e):
			code := e.Status
			w.WriteHeader(code)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(errs)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("Error occurred during maven error encoding")
		}
	}
}

func LogError(errList errcode.Errors) {
	for _, e1 := range errList {
		log.Error().Err(e1).Msgf("error: %v", e1)
	}
}

func (h *Handler) GetPackageArtifactInfo(r *http.Request) (pkg.PackageArtifactInfo, error) {
	info, e := h.GetArtifactInfo(r, true)
	if e != nil {
		return pkg.MavenArtifactInfo{}, e
	}

	artifactInfo := info.ArtifactInfo
	artifactInfo.PathPackageType = artifact.PackageTypeMAVEN
	return pkg.MavenArtifactInfo{
		ArtifactInfo: artifactInfo,
		Version:      info.Version,
	}, nil
}
