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

package packages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	usercontroller "github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/app/auth/authz"
	corestore "github.com/harness/gitness/app/store"
	urlprovider "github.com/harness/gitness/app/url"
	artifact2 "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/request"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func NewHandler(
	registryDao store.RegistryRepository,
	downloadStatDao store.DownloadStatRepository,
	spaceStore corestore.SpaceStore, tokenStore corestore.TokenStore,
	userCtrl *usercontroller.Controller, authenticator authn.Authenticator,
	urlProvider urlprovider.Provider, authorizer authz.Authorizer,
) Handler {
	return &handler{
		RegistryDao:     registryDao,
		DownloadStatDao: downloadStatDao,
		SpaceStore:      spaceStore,
		TokenStore:      tokenStore,
		UserCtrl:        userCtrl,
		Authenticator:   authenticator,
		URLProvider:     urlProvider,
		Authorizer:      authorizer,
	}
}

type handler struct {
	RegistryDao     store.RegistryRepository
	DownloadStatDao store.DownloadStatRepository
	SpaceStore      corestore.SpaceStore
	TokenStore      corestore.TokenStore
	UserCtrl        *usercontroller.Controller
	Authenticator   authn.Authenticator
	URLProvider     urlprovider.Provider
	Authorizer      authz.Authorizer
}

type Handler interface {
	GetRegistryCheckAccess(
		ctx context.Context,
		r *http.Request,
		reqPermissions ...enum.Permission,
	) error
	GetArtifactInfo(r *http.Request) (pkg.ArtifactInfo, error)

	TrackDownloadStats(
		ctx context.Context,
		r *http.Request,
	) error
	GetAuthenticator() authn.Authenticator
	HandleErrors2(ctx context.Context, errors errcode.Error, w http.ResponseWriter)
	HandleErrors(ctx context.Context, errors errcode.Errors, w http.ResponseWriter)
	HandleError(ctx context.Context, w http.ResponseWriter, err error)
	ServeContent(
		w http.ResponseWriter, r *http.Request, fileReader *storage.FileReader, filename string,
	)
}

type PathPackageType string

const (
	PathPackageTypeGeneric PathPackageType = "generic"
	PathPackageTypeMaven   PathPackageType = "maven"
	PathPackageTypePython  PathPackageType = "python"
	PathPackageTypeNuget   PathPackageType = "nuget"
	PathPackageTypeNpm     PathPackageType = "npm"
	PathPackageTypeRPM     PathPackageType = "rpm"
)

var packageTypeMap = map[PathPackageType]artifact2.PackageType{
	PathPackageTypeGeneric: artifact2.PackageTypeGENERIC,
	PathPackageTypeMaven:   artifact2.PackageTypeMAVEN,
	PathPackageTypePython:  artifact2.PackageTypePYTHON,
	PathPackageTypeNuget:   artifact2.PackageTypeNUGET,
	PathPackageTypeNpm:     artifact2.PackageTypeNPM,
	PathPackageTypeRPM:     artifact2.PackageTypeRPM,
}

func (h *handler) GetAuthenticator() authn.Authenticator {
	return h.Authenticator
}

func (h *handler) GetRegistryCheckAccess(
	ctx context.Context,
	r *http.Request,
	reqPermissions ...enum.Permission,
) error {
	info, err := h.GetArtifactInfo(r)
	if err != nil {
		return err
	}
	return pkg.GetRegistryCheckAccess(ctx, h.RegistryDao, h.Authorizer,
		h.SpaceStore,
		info.RegIdentifier, info.ParentID, reqPermissions...)
}

func (h *handler) TrackDownloadStats(
	ctx context.Context,
	r *http.Request,
) error {
	info := request.ArtifactInfoFrom(r.Context()) //nolint:contextcheck
	if err := h.DownloadStatDao.CreateByRegistryIDImageAndArtifactName(ctx,
		info.BaseArtifactInfo().RegistryID, info.BaseArtifactInfo().Image, info.GetVersion()); err != nil {
		log.Error().Msgf("failed to create download stat: %v", err.Error())
		return usererror.ErrInternal
	}
	return nil
}

func (h *handler) GetArtifactInfo(r *http.Request) (pkg.ArtifactInfo, error) {
	ctx := r.Context()
	rootIdentifier, registryIdentifier, pathPackageType, err := extractPathVars(r)

	if err != nil {
		return pkg.ArtifactInfo{}, errcode.ErrCodeInvalidRequest.WithDetail(err)
	}

	rootSpace, err := h.SpaceStore.FindByRefCaseInsensitive(ctx, rootIdentifier)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Root space not found: %s", rootIdentifier)
		return pkg.ArtifactInfo{}, usererror.NotFoundf("Root not found: %s", rootIdentifier)
	}

	registry, err := h.RegistryDao.GetByRootParentIDAndName(ctx, rootSpace.ID, registryIdentifier)

	if err != nil {
		log.Ctx(ctx).Error().Msgf(
			"registry %s not found for root: %s. Reason: %s", registryIdentifier, rootSpace.Identifier, err,
		)
		return pkg.ArtifactInfo{}, usererror.NotFoundf("Registry not found: %s", registryIdentifier)
	}

	_, err = h.SpaceStore.Find(r.Context(), registry.ParentID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Parent space not found: %d", registry.ParentID)
		return pkg.ArtifactInfo{}, usererror.NotFoundf("Parent not found for registry: %s", registryIdentifier)
	}

	return pkg.ArtifactInfo{
		BaseInfo: &pkg.BaseInfo{
			RootIdentifier:  rootIdentifier,
			RootParentID:    rootSpace.ID,
			ParentID:        registry.ParentID,
			PathPackageType: pathPackageType,
		},
		RegIdentifier: registryIdentifier,
		RegistryID:    registry.ID,
		Registry:      *registry,
		Image:         "",
	}, nil
}

func (h *handler) HandleErrors2(ctx context.Context, err errcode.Error, w http.ResponseWriter) {
	if !commons.IsEmptyError(err) {
		w.WriteHeader(err.Code.Descriptor().HTTPStatusCode)
		_ = errcode.ServeJSON(w, err)
		log.Ctx(ctx).Error().Msgf("Error occurred while performing artifact action: %s", err.Message)
	}
}

// HandleErrors TODO: Improve Error Handling
// HandleErrors handles errors and writes the appropriate response to the client.
func (h *handler) HandleErrors(ctx context.Context, errs errcode.Errors, w http.ResponseWriter) {
	if !commons.IsEmpty(errs) {
		LogError(errs)
		log.Ctx(ctx).Error().Errs("errs occurred during artifact operation: ", errs).Msgf("Error occurred")
		err := errs[0]
		var e *commons.Error
		if errors.As(err, &e) {
			code := e.Status
			w.WriteHeader(code)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(errs)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("Error occurred during artifact error encoding")
		}
	}
}

func (h *handler) HandleError(ctx context.Context, w http.ResponseWriter, err error) {
	if nil != err {
		log.Error().Err(err).Ctx(ctx).Msgf("error: %v", err)
		render.TranslatedUserError(ctx, w, err)
		return
	}
}

func LogError(errList errcode.Errors) {
	for _, e1 := range errList {
		log.Error().Err(e1).Msgf("error: %v", e1)
	}
}

// extractPathVars extracts rootSpace, registryId, pathPackageType from the path
// Path format: /pkg/:rootSpace/:registry/:pathPackageType/...
func extractPathVars(r *http.Request) (
	rootIdentifier string,
	registry string,
	packageType artifact2.PackageType,
	err error,
) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		return "", "", "", fmt.Errorf("invalid path: %s", path)
	}
	rootIdentifier = parts[2]
	registry = parts[3]
	pathPackageType := PathPackageType(parts[4])
	if _, ok := packageTypeMap[pathPackageType]; !ok {
		return "", "", "", fmt.Errorf("invalid package type: %s", packageType)
	}
	return rootIdentifier, registry, packageTypeMap[pathPackageType], nil
}

func (h *handler) ServeContent(
	w http.ResponseWriter, r *http.Request, fileReader *storage.FileReader, filename string,
) {
	if fileReader != nil {
		http.ServeContent(w, r, filename, time.Time{}, fileReader)
	}
}
