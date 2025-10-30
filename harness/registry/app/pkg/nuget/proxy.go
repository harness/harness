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

package nuget

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	cfg "github.com/harness/gitness/registry/config"
	"github.com/harness/gitness/secret"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/rs/zerolog/log"

	_ "github.com/harness/gitness/registry/app/remote/adapter/nuget" // This is required to init nuget adapter
)

const (
	// XML namespace constants for NuGet feed responses.
	xmlnsDataServices         = "http://schemas.microsoft.com/ado/2007/08/dataservices"
	xmlnsDataServicesMetadata = "http://schemas.microsoft.com/ado/2007/08/dataservices/metadata"
)

var _ pkg.Artifact = (*proxy)(nil)
var _ Registry = (*proxy)(nil)

type proxy struct {
	fileManager         filemanager.FileManager
	proxyStore          store.UpstreamProxyConfigRepository
	tx                  dbtx.Transactor
	registryDao         store.RegistryRepository
	imageDao            store.ImageRepository
	artifactDao         store.ArtifactRepository
	urlProvider         urlprovider.Provider
	spaceFinder         refcache.SpaceFinder
	service             secret.Service
	localRegistryHelper LocalRegistryHelper
}

func (r *proxy) UploadPackage(
	ctx context.Context, _ nugettype.ArtifactInfo,
	_ io.ReadCloser, _ FileBundleType,
) (*commons.ResponseHeaders, string, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, "", errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

func (r *proxy) DownloadPackage(ctx context.Context, info nugettype.ArtifactInfo) (*commons.ResponseHeaders,
	*storage.FileReader, string, io.ReadCloser, error) {
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return nil, nil, "", nil, err
	}

	exists := r.localRegistryHelper.FileExists(ctx, info)
	if exists {
		headers, fileReader, redirectURL, err := r.localRegistryHelper.DownloadFile(ctx, info)
		if err == nil {
			return headers, fileReader, redirectURL, nil, nil
		}
		log.Warn().Ctx(ctx).Msgf("failed to pull from local, attempting streaming from remote, %v", err)
	}

	remote, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	if err != nil {
		return nil, nil, "", nil, err
	}

	file, err := remote.GetFile(ctx, info.Image, info.Version, info.ProxyEndpoint, info.Filename)
	if err != nil {
		return nil, nil, "", nil, err
	}
	go func(info nugettype.ArtifactInfo) {
		ctx2 := context.WithoutCancel(ctx)
		ctx2 = context.WithValue(ctx2, cfg.GoRoutineKey, "goRoutine")
		err = r.putFileToLocal(ctx2, &info, remote)
		if err != nil {
			log.Ctx(ctx2).Error().Stack().Err(err).Msgf("error while putting file to localRegistry, %v", err)
			return
		}
		log.Ctx(ctx2).Info().Msgf("Successfully updated file: %s, registry: %s", info.Filename, info.RegIdentifier)
	}(info)

	return nil, nil, "", file, nil
}

func (r *proxy) DeletePackage(
	ctx context.Context,
	_ nugettype.ArtifactInfo,
) (*commons.ResponseHeaders, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

func (r *proxy) CountPackageVersionV2(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) (count int64, err error) {
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return 0, err
	}

	helper, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	if err != nil {
		return 0, err
	}

	// Use the adapter's CountPackageVersionV2 method directly
	count, err = helper.CountPackageVersionV2(ctx, info.Image)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *proxy) CountPackageV2(
	ctx context.Context, info nugettype.ArtifactInfo,
	searchTerm string,
) (count int64, err error) {
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return 0, err
	}

	helper, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	if err != nil {
		return 0, err
	}

	// Use the adapter's CountPackageV2 method directly
	count, err = helper.CountPackageV2(ctx, searchTerm)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *proxy) SearchPackageV2(
	ctx context.Context, info nugettype.ArtifactInfo,
	searchTerm string, limit int, offset int,
) (*nugettype.FeedResponse, error) {
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return &nugettype.FeedResponse{}, err
	}

	helper, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	if err != nil {
		return &nugettype.FeedResponse{}, err
	}

	fileReader, err := helper.SearchPackageV2(ctx, searchTerm, limit, offset)
	if err != nil {
		return &nugettype.FeedResponse{}, err
	}
	defer fileReader.Close()

	var result nugettype.FeedResponse
	if err = xml.NewDecoder(fileReader).Decode(&result); err != nil {
		return &nugettype.FeedResponse{}, err
	}

	// Update URLs to point to our proxy, similar to ListPackageVersionV2
	packageURL := r.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")
	result.Xmlns = "http://www.w3.org/2005/Atom"
	result.XmlnsD = xmlnsDataServices
	result.XmlnsM = xmlnsDataServicesMetadata
	result.Base = packageURL
	result.ID = "http://schemas.datacontract.org/2004/07/"
	result.Updated = time.Now()

	links := []nugettype.FeedEntryLink{
		{Rel: "self", Href: xml.CharData(packageURL)},
	}
	result.Links = links

	// Update each entry's content URLs to point to our proxy
	for _, entry := range result.Entries {
		re := regexp.MustCompile(`Version='([^']+)'`)
		matches := re.FindStringSubmatch(entry.ID)
		if len(matches) > 1 {
			version := matches[1]
			err = modifyContent(entry, packageURL, info.Image, version)
			if err != nil {
				return &nugettype.FeedResponse{}, fmt.Errorf("failed to modify content: %w", err)
			}
		}
	}

	return &result, nil
}

func (r *proxy) SearchPackage(
	ctx context.Context, info nugettype.ArtifactInfo,
	searchTerm string, limit int, offset int,
) (*nugettype.SearchResultResponse, error) {
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return nil, err
	}

	helper, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	if err != nil {
		return nil, err
	}

	// Use the v3 search API directly
	fileReader, err := helper.SearchPackage(ctx, searchTerm, limit, offset)
	if err != nil {
		return nil, err
	}
	defer fileReader.Close()

	// Parse the v3 search response directly
	var result nugettype.SearchResultResponse
	if err = json.NewDecoder(fileReader).Decode(&result); err != nil {
		return nil, err
	}

	// Update URLs in search results to point to our proxy
	packageURL := r.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")

	for _, searchResult := range result.Data {
		if searchResult != nil {
			// Update RegistrationIndexURL to point to our proxy
			if searchResult.RegistrationIndexURL != "" {
				registrationURL := getRegistrationIndexURL(packageURL, searchResult.ID)
				searchResult.RegistrationIndexURL = registrationURL
			}

			// Update RegistrationLeafURL in versions to point to our proxy
			for _, version := range searchResult.Versions {
				if version != nil && version.RegistrationLeafURL != "" {
					registrationURL := getRegistrationIndexURL(packageURL, searchResult.ID)
					version.RegistrationLeafURL = getProxyURL(registrationURL, version.RegistrationLeafURL)
				}
			}
		}
	}

	return &result, nil
}

func (r *proxy) ListPackageVersion(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) (*nugettype.PackageVersion, error) {
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return &nugettype.PackageVersion{}, err
	}

	helper, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	if err != nil {
		return &nugettype.PackageVersion{}, err
	}
	fileReader, err := helper.ListPackageVersion(ctx, info.Image)
	if err != nil {
		return &nugettype.PackageVersion{}, err
	}
	var result nugettype.PackageVersion
	if err = json.NewDecoder(fileReader).Decode(&result); err != nil {
		return &nugettype.PackageVersion{}, err
	}
	return &result, nil
}

func (r *proxy) GetPackageMetadata(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) (nugettype.RegistrationResponse, error) {
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return &nugettype.RegistrationIndexResponse{}, err
	}

	helper, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	if err != nil {
		return &nugettype.RegistrationIndexResponse{}, err
	}
	fileReader, err := helper.GetPackageMetadata(ctx, info.Image, info.ProxyEndpoint)
	if err != nil {
		return &nugettype.RegistrationIndexResponse{}, err
	}

	packageURL := r.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")

	if info.ProxyEndpoint != "" {
		metadata, err2 := parseRegistrationIndexPageResponse(fileReader)
		if err2 != nil {
			//todo: add handling for registration leaf
			return &nugettype.RegistrationIndexPageResponse{}, err
		}
		updateRegistrationIndexPageResponse(metadata, packageURL, info.Image)
		return metadata, nil
	}
	metadata, err2 := parseRegistrationIndexResponse(fileReader)
	if err2 != nil {
		return &nugettype.RegistrationIndexResponse{}, err
	}
	updateRegistrationIndexResponse(metadata, packageURL, info.Image)
	return metadata, nil
}

func (r *proxy) ListPackageVersionV2(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) (*nugettype.FeedResponse, error) {
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return &nugettype.FeedResponse{}, err
	}
	helper, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	if err != nil {
		return &nugettype.FeedResponse{}, err
	}
	fileReader, err := helper.ListPackageVersionV2(ctx, info.Image)
	if err != nil {
		return &nugettype.FeedResponse{}, err
	}
	var result nugettype.FeedResponse
	if err = xml.NewDecoder(fileReader).Decode(&result); err != nil {
		return &nugettype.FeedResponse{}, err
	}
	packageURL := r.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")
	result.Xmlns = "http://www.w3.org/2005/Atom"
	result.XmlnsD = xmlnsDataServices
	result.XmlnsM = xmlnsDataServicesMetadata
	result.Base = packageURL
	result.ID = "http://schemas.datacontract.org/2004/07/"
	result.Updated = time.Now()
	links := []nugettype.FeedEntryLink{
		{Rel: "self", Href: xml.CharData(packageURL)},
	}
	result.Links = links

	for _, entry := range result.Entries {
		re := regexp.MustCompile(`Version='([^']+)'`)
		matches := re.FindStringSubmatch(entry.ID)
		if len(matches) > 1 {
			version := matches[1]
			err = modifyContent(entry, packageURL, info.Image, version)
			if err != nil {
				return &nugettype.FeedResponse{}, fmt.Errorf("failed to modify content: %w", err)
			}
		}
	}

	return &result, nil
}

func (r *proxy) GetPackageVersionMetadataV2(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) (*nugettype.FeedEntryResponse, error) {
	packageURL := r.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return &nugettype.FeedEntryResponse{}, err
	}
	helper, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	if err != nil {
		return &nugettype.FeedEntryResponse{}, err
	}
	fileReader, err := helper.GetPackageVersionMetadataV2(ctx, info.Image, info.Version)
	if err != nil {
		return &nugettype.FeedEntryResponse{}, err
	}
	var result nugettype.FeedEntryResponse
	if err = xml.NewDecoder(fileReader).Decode(&result); err != nil {
		return &nugettype.FeedEntryResponse{}, err
	}
	result.XmlnsD = xmlnsDataServices
	result.XmlnsM = xmlnsDataServicesMetadata
	err = modifyContent(&result, packageURL, info.Image, info.Version)
	if err != nil {
		return &nugettype.FeedEntryResponse{}, fmt.Errorf("failed to modify content: %w", err)
	}
	return &result, nil
}

func parseRegistrationIndexResponse(r io.ReadCloser) (*nugettype.RegistrationIndexResponse, error) {
	var result nugettype.RegistrationIndexResponse
	if err := json.NewDecoder(r).Decode(&result); err != nil {
		return &nugettype.RegistrationIndexResponse{}, err
	}
	return &result, nil
}

func parseRegistrationIndexPageResponse(r io.ReadCloser) (*nugettype.RegistrationIndexPageResponse, error) {
	var result nugettype.RegistrationIndexPageResponse
	if err := json.NewDecoder(r).Decode(&result); err != nil {
		return &nugettype.RegistrationIndexPageResponse{}, err
	}
	return &result, nil
}

func updateRegistrationIndexPageResponse(r *nugettype.RegistrationIndexPageResponse, packageURL, pkg string) {
	registrationURL := getRegistrationIndexURL(packageURL, pkg)
	if r.RegistrationPageURL != "" {
		r.RegistrationPageURL = getProxyURL(registrationURL, r.RegistrationPageURL)
	}
	for _, item := range r.Items {
		if item.RegistrationLeafURL != "" {
			item.RegistrationLeafURL = getProxyURL(registrationURL, item.RegistrationLeafURL)
		}
		if item.CatalogEntry != nil {
			packageContentURL := getPackageDownloadURL(packageURL, pkg, item.CatalogEntry.Version)
			item.PackageContentURL = packageContentURL
			item.CatalogEntry.PackageContentURL = item.PackageContentURL
		}
	}
}

func updateRegistrationIndexResponse(r *nugettype.RegistrationIndexResponse, packageURL, pkg string) {
	registrationURL := getRegistrationIndexURL(packageURL, pkg)
	if r.RegistrationIndexURL != "" {
		r.RegistrationIndexURL = registrationURL
	}
	for _, page := range r.Pages {
		updateRegistrationIndexPageResponse(page, packageURL, pkg)
	}
}

func (r *proxy) GetPackageVersionMetadata(
	ctx context.Context,
	_ nugettype.ArtifactInfo,
) (*nugettype.RegistrationLeafResponse, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

func (r *proxy) GetServiceEndpoint(ctx context.Context, info nugettype.ArtifactInfo) *nugettype.ServiceEndpoint {
	packageURL := r.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")
	serviceEndpoints := buildServiceEndpoint(packageURL)
	return serviceEndpoints
}

func (r *proxy) GetServiceEndpointV2(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) *nugettype.ServiceEndpointV2 {
	packageURL := r.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")
	serviceEndpoints := buildServiceV2Endpoint(packageURL)
	return serviceEndpoints
}

func (r *proxy) GetServiceMetadataV2(_ context.Context, _ nugettype.ArtifactInfo) *nugettype.ServiceMetadataV2 {
	return getServiceMetadataV2()
}

type Proxy interface {
	Registry
}

func NewProxy(
	fileManager filemanager.FileManager,
	proxyStore store.UpstreamProxyConfigRepository,
	tx dbtx.Transactor,
	registryDao store.RegistryRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	urlProvider urlprovider.Provider,
	spaceFinder refcache.SpaceFinder,
	service secret.Service,
	localRegistryHelper LocalRegistryHelper,
) Proxy {
	return &proxy{
		fileManager:         fileManager,
		proxyStore:          proxyStore,
		tx:                  tx,
		registryDao:         registryDao,
		imageDao:            imageDao,
		artifactDao:         artifactDao,
		urlProvider:         urlProvider,
		spaceFinder:         spaceFinder,
		service:             service,
		localRegistryHelper: localRegistryHelper,
	}
}

func (r *proxy) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeUPSTREAM
}

func (r *proxy) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeNUGET}
}

func (r *proxy) putFileToLocal(
	ctx context.Context, info *nugettype.ArtifactInfo,
	remote RemoteRegistryHelper,
) error {
	file, err := remote.GetFile(ctx, info.Image, info.Version, info.ProxyEndpoint, info.Filename)
	if err != nil {
		log.Ctx(ctx).Error().Stack().Err(err).Msgf("fetching file for pkg: %s failed, %v", info.Image, err)
		return err
	}
	defer file.Close()

	_, sha256, err2 := r.localRegistryHelper.UploadPackageFile(ctx, *info, file)
	if err2 != nil {
		log.Ctx(ctx).Error().Stack().Err(err2).Msgf("uploading file for pkg: %s failed, %v", info.Image, err)
		return err2
	}
	log.Ctx(ctx).Info().Msgf("Successfully uploaded file for pkg: %s , version: %s with SHA256: %s",
		info.Image, info.Version, sha256)
	return nil
}
