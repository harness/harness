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
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	corestore "github.com/harness/gitness/app/store"
	gerrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var _ Artifact = (*LocalRegistry)(nil)
var _ Artifact = (*RemoteRegistry)(nil)

type ArtifactType int

const (
	LocalRegistryType ArtifactType = 1 << iota
	RemoteRegistryType
)

var TypeRegistry = map[ArtifactType]Artifact{}

type Controller struct {
	local                     *LocalRegistry
	remote                    *RemoteRegistry
	authorizer                authz.Authorizer
	DBStore                   *DBStore
	_                         dbtx.Transactor
	SpaceFinder               refcache.SpaceFinder
	quarantineFinder          quarantine.Finder
	dependencyFirewallChecker interfaces.DependencyFirewallChecker
}

type DBStore struct {
	RegistryDao      store.RegistryRepository
	ImageDao         store.ImageRepository
	ArtifactDao      store.ArtifactRepository
	SpaceStore       corestore.SpaceStore
	BandwidthStatDao store.BandwidthStatRepository
	DownloadStatDao  store.DownloadStatRepository
	NodeDao          store.NodesRepository
	UpstreamProxyDao store.UpstreamProxyConfigRepository
}

func NewController(
	local *LocalRegistry,
	remote *RemoteRegistry,
	authorizer authz.Authorizer,
	dBStore *DBStore,
	spaceFinder refcache.SpaceFinder,
	quarantineFinder quarantine.Finder,
	dependencyFirewallChecker interfaces.DependencyFirewallChecker,
) *Controller {
	c := &Controller{
		local:                     local,
		remote:                    remote,
		authorizer:                authorizer,
		DBStore:                   dBStore,
		SpaceFinder:               spaceFinder,
		quarantineFinder:          quarantineFinder,
		dependencyFirewallChecker: dependencyFirewallChecker,
	}

	TypeRegistry[LocalRegistryType] = local
	TypeRegistry[RemoteRegistryType] = remote
	return c
}

func NewDBStore(
	registryDao store.RegistryRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	spaceStore corestore.SpaceStore,
	bandwidthStatDao store.BandwidthStatRepository,
	downloadStatDao store.DownloadStatRepository,
	nodeDao store.NodesRepository,
	upstreamProxyDao store.UpstreamProxyConfigRepository,
) *DBStore {
	return &DBStore{
		RegistryDao:      registryDao,
		SpaceStore:       spaceStore,
		ImageDao:         imageDao,
		ArtifactDao:      artifactDao,
		BandwidthStatDao: bandwidthStatDao,
		DownloadStatDao:  downloadStatDao,
		NodeDao:          nodeDao,
		UpstreamProxyDao: upstreamProxyDao,
	}
}

func (c *Controller) factory(ctx context.Context, t ArtifactType) Artifact {
	switch t {
	case LocalRegistryType:
		return TypeRegistry[t]
	case RemoteRegistryType:
		return TypeRegistry[t]
	default:
		log.Ctx(ctx).Error().Stack().Msgf("Invalid artifact type %v", t)
		return nil
	}
}

func (c *Controller) GetArtifactRegistry(ctx context.Context, registry registrytypes.Registry) Artifact {
	if string(registry.Type) == string(artifact.RegistryTypeVIRTUAL) {
		return c.factory(ctx, LocalRegistryType)
	}
	return c.factory(ctx, RemoteRegistryType)
}

func (c *Controller) GetArtifact(ctx context.Context, info pkg.MavenArtifactInfo) *GetArtifactResponse {
	err := pkg.GetRegistryCheckAccess(ctx, c.authorizer, c.SpaceFinder, info.ParentID, *info.ArtifactInfo,
		enum.PermissionArtifactsDownload)
	if err != nil {
		return &GetArtifactResponse{
			Errors: []error{errcode.ErrCodeDenied},
		}
	}

	f := func(registry registrytypes.Registry, a Artifact) Response {
		info.UpdateRegistryInfo(registry)
		r, ok := a.(Registry)
		if !ok {
			log.Ctx(ctx).Error().Stack().Msgf("Proxy wrapper has invalid registry set")
			return nil
		}
		err = c.quarantineFinder.CheckArtifactQuarantineStatus(ctx, registry.ID, info.Image, info.Version, nil)
		if err != nil {
			if errors.Is(err, usererror.ErrQuarantinedArtifact) {
				return &GetArtifactResponse{
					Errors: []error{err},
				}
			}
			log.Ctx(ctx).Error().Stack().Err(err).Msgf("error "+
				"while checking the quarantine status of artifact: [%s], version: [%s], %v",
				info.Image, info.Version, err)
			return &GetArtifactResponse{
				Errors: []error{err},
			}
		}

		// Check dependency firewall violations if upstream proxy
		if registry.Type == artifact.RegistryTypeUPSTREAM {
			err = c.dependencyFirewallChecker.CheckPolicyViolation(ctx, registry.ID, info.Image, info.Version, nil)
			if err != nil {
				if errors.Is(err, usererror.ErrArtifactBlocked) {
					return &GetArtifactResponse{
						Errors: []error{err},
					}
				}
				log.Ctx(ctx).Error().Stack().Err(err).Msgf("error"+
					" while checking dependency firewall violations for artifact: [%s], version: [%s]",
					info.Image, info.Version)
				return &GetArtifactResponse{
					Errors: []error{err},
				}
			}
		}
		headers, body, fileReader, redirectURL, e := r.GetArtifact(ctx, info) //nolint:errcheck
		return &GetArtifactResponse{
			e, headers, redirectURL,
			body, fileReader,
		}
	}
	result, err := c.ProxyWrapper(ctx, f, info)
	if err != nil {
		return &GetArtifactResponse{
			Errors: []error{err},
		}
	}
	getArtifactResponse, ok := result.(*GetArtifactResponse)
	if !ok {
		return &GetArtifactResponse{
			[]error{fmt.Errorf("invalid response type: expected GetArtifactResponse")},
			nil, "", nil, nil}
	}

	return getArtifactResponse
}

func (c *Controller) HeadArtifact(ctx context.Context, info pkg.MavenArtifactInfo) *HeadArtifactResponse {
	err := pkg.GetRegistryCheckAccess(ctx, c.authorizer, c.SpaceFinder, info.ParentID, *info.ArtifactInfo,
		enum.PermissionArtifactsDownload)
	if err != nil {
		return &HeadArtifactResponse{
			Errors: []error{errcode.ErrCodeDenied},
		}
	}

	f := func(registry registrytypes.Registry, a Artifact) Response {
		info.UpdateRegistryInfo(registry)
		r, ok := a.(Registry)
		if !ok {
			log.Ctx(ctx).Error().Stack().Msgf("Proxy wrapper has invalid registry set")
			return nil
		}

		headers, e := r.HeadArtifact(ctx, info)
		return &HeadArtifactResponse{e, headers}
	}
	result, err := c.ProxyWrapper(ctx, f, info)
	if err != nil {
		return &HeadArtifactResponse{
			Errors: []error{err},
		}
	}
	headArtifactResponse, ok := result.(*HeadArtifactResponse)
	if !ok {
		return &HeadArtifactResponse{
			[]error{fmt.Errorf("invalid response type: expected HeadArtifactResponse")},
			nil}
	}

	return headArtifactResponse
}

func (c *Controller) PutArtifact(
	ctx context.Context,
	info pkg.MavenArtifactInfo,
	fileReader io.Reader,
) *PutArtifactResponse {
	err := pkg.GetRegistryCheckAccess(ctx, c.authorizer, c.SpaceFinder, info.ParentID, *info.ArtifactInfo,
		enum.PermissionArtifactsUpload)
	if err != nil {
		responseHeaders := &commons.ResponseHeaders{
			Code: http.StatusForbidden,
		}
		return &PutArtifactResponse{
			ResponseHeaders: responseHeaders,
			Errors:          []error{errcode.ErrCodeDenied},
		}
	}

	responseHeaders, errs := c.local.PutArtifact(ctx, info, fileReader)
	return &PutArtifactResponse{
		ResponseHeaders: responseHeaders,
		Errors:          errs,
	}
}

func (c *Controller) ProxyWrapper(
	ctx context.Context,
	f func(registry registrytypes.Registry, a Artifact) Response,
	info pkg.MavenArtifactInfo,
) (Response, error) {
	none := pkg.MavenArtifactInfo{}
	if info == none {
		err := fmt.Errorf("artifactinfo is not found")
		log.Ctx(ctx).Error().Stack().Msg(err.Error())
		return nil, err
	}

	var response Response
	var lastErr error
	requestRepoKey := info.RegIdentifier
	repos, err := c.GetOrderedRepos(ctx, requestRepoKey, info.Registry)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf(
			"GetOrderedRepos failed: registry=%s, parentID=%d",
			requestRepoKey, info.BaseInfo.ParentID)
		return response, err
	}
	for _, registry := range repos {
		log.Ctx(ctx).Info().Msgf("Using Repository: %s, Type: %s", registry.Name, registry.Type)
		art, ok := c.GetArtifactRegistry(ctx, registry).(Registry)
		if !ok {
			log.Ctx(ctx).Warn().Msgf("Invalid registry type for registry %s", registry.Name)
			continue
		}
		if art != nil {
			response = f(registry, art)
			if pkg.IsEmpty(response.GetErrors()) {
				return response, nil
			}
			if len(response.GetErrors()) > 0 {
				lastErr = response.GetErrors()[0]
			}
			log.Ctx(ctx).Warn().Msgf("Repository: %s, Type: %s, errors: %v", registry.Name, registry.Type,
				response.GetErrors())
		}
	}

	if lastErr != nil {
		return response, lastErr
	}

	return response, gerrors.NotFoundf("no matching artifacts found in registry %s", requestRepoKey)
}

func (c *Controller) GetOrderedRepos(
	ctx context.Context,
	repoKey string,
	registry registrytypes.Registry,
) ([]registrytypes.Registry, error) {
	result := []registrytypes.Registry{registry}

	proxies := registry.UpstreamProxies
	if len(proxies) > 0 {
		log.Ctx(ctx).Debug().Msgf("Fetching %d upstream proxies for registry %s: %v",
			len(proxies), repoKey, proxies)
		upstreamRepos, err := c.DBStore.RegistryDao.GetByIDIn(ctx, proxies)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("GetByIDIn failed for upstream proxies: %v", proxies)
			return result, err
		}
		result = append(result, *upstreamRepos...)
	}

	return result, nil
}
