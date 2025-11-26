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

package npm

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	npm2 "github.com/harness/gitness/registry/app/metadata/npm"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/types/npm"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var _ pkg.Artifact = (*localRegistry)(nil)
var _ Registry = (*localRegistry)(nil)

type localRegistry struct {
	localBase   base.LocalBase
	fileManager filemanager.FileManager
	proxyStore  store.UpstreamProxyConfigRepository
	tx          dbtx.Transactor
	registryDao store.RegistryRepository
	imageDao    store.ImageRepository
	tagsDao     store.PackageTagRepository
	nodesDao    store.NodesRepository
	artifactDao store.ArtifactRepository
	urlProvider urlprovider.Provider
}

func (c *localRegistry) HeadPackageMetadata(ctx context.Context, info npm.ArtifactInfo) (bool, error) {
	return c.localBase.CheckIfVersionExists(ctx, info)
}

func (c *localRegistry) DownloadPackageFile(ctx context.Context,
	info npm.ArtifactInfo) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	headers, fileReader, redirectURL, err :=
		c.localBase.Download(ctx, info.ArtifactInfo, info.Version,
			info.Filename)
	if err != nil {
		return nil, nil, nil, "", err
	}
	return headers, fileReader, nil, redirectURL, nil
}

type LocalRegistry interface {
	Registry
}

func NewLocalRegistry(
	localBase base.LocalBase,
	fileManager filemanager.FileManager,
	proxyStore store.UpstreamProxyConfigRepository,
	tx dbtx.Transactor,
	registryDao store.RegistryRepository,
	tagDao store.PackageTagRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	nodesDao store.NodesRepository,
	urlProvider urlprovider.Provider,
) LocalRegistry {
	return &localRegistry{
		localBase:   localBase,
		fileManager: fileManager,
		proxyStore:  proxyStore,
		tx:          tx,
		tagsDao:     tagDao,
		registryDao: registryDao,
		imageDao:    imageDao,
		artifactDao: artifactDao,
		nodesDao:    nodesDao,
		urlProvider: urlProvider,
	}
}

func (c *localRegistry) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeVIRTUAL
}

func (c *localRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeNPM}
}

func (c *localRegistry) UploadPackageFile(
	ctx context.Context,
	info npm.ArtifactInfo,
	file io.ReadCloser,
) (headers *commons.ResponseHeaders, sha256 string, err error) {
	var packageMetadata npm2.PackageMetadata
	fileInfo, tempFileName, err := c.parseAndUploadNPMPackage(ctx, info, file, &packageMetadata)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to parse npm package: %v", err)
		return nil, "", err
	}
	log.Info().Str("packageName", info.Image).Msg("Successfully parsed and uploaded NPM package to tmp location")
	info.Metadata = packageMetadata
	info.Image = packageMetadata.Name
	for tag := range packageMetadata.DistTags {
		info.DistTags = append(info.DistTags, tag)
	}
	for _, meta := range packageMetadata.Versions {
		info.Version = meta.Version
	}
	info.Filename = info.Image + "-" + info.Version + ".tgz"
	fileInfo.Filename = info.Filename
	filePath := path.Join(info.Image, info.Version, fileInfo.Filename)

	_, sha256, _, _, err = c.localBase.MoveTempFileAndCreateArtifact(ctx, info.ArtifactInfo,
		tempFileName, info.Version, filePath,
		&npm2.NpmMetadata{
			PackageMetadata: info.Metadata,
		}, fileInfo, false)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to move npm package: %v", err)
		return nil, "", err
	}
	_, err = c.AddTag(ctx, info)

	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to add tag for npm package:%s, %v", info.Image, err)
		return nil, "", err
	}

	return nil, sha256, nil
}

func (c *localRegistry) GetPackageMetadata(ctx context.Context, info npm.ArtifactInfo) (npm2.PackageMetadata, error) {
	packageMetadata := npm2.PackageMetadata{}
	versions := make(map[string]*npm2.PackageMetadataVersion)
	artifacts, err := c.artifactDao.GetByRegistryIDAndImage(ctx, info.RegistryID, info.Image)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("Failed to fetch artifact for image:[%s], Reg:[%s]",
			info.BaseArtifactInfo().Image, info.BaseArtifactInfo().RegIdentifier)
		return packageMetadata, usererror.ErrInternal
	}

	if len(*artifacts) == 0 {
		return packageMetadata,
			usererror.NotFound(fmt.Sprintf("no artifacts found for registry %s and image %s", info.Registry.Name, info.Image))
	}
	regURL := c.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "npm")

	for _, artifact := range *artifacts {
		metadata := &npm2.NpmMetadata{}
		err = json.Unmarshal(artifact.Metadata, metadata)
		if err != nil {
			return packageMetadata, err
		}
		if packageMetadata.Name == "" {
			packageMetadata = metadata.PackageMetadata
		}
		for _, versionMetadata := range metadata.Versions {
			versions[artifact.Version] = CreatePackageMetadataVersion(regURL, versionMetadata)
		}
	}
	distTags, err := c.ListTags(ctx, info)
	if !commons.IsEmpty(err) {
		return npm2.PackageMetadata{}, err
	}
	packageMetadata.Versions = versions
	packageMetadata.DistTags = distTags
	return packageMetadata, nil
}

func (c *localRegistry) SearchPackage(ctx context.Context, info npm.ArtifactInfo,
	limit int, offset int) (*npm2.PackageSearch, error) {
	metadataList, err := c.artifactDao.SearchLatestByName(ctx, info.RegistryID, info.Image, limit, offset)

	if err != nil {
		log.Err(err).Msgf("Failed to search package for search term: [%s]", info.Image)
		return &npm2.PackageSearch{}, err
	}
	count, err := c.artifactDao.CountLatestByName(ctx, info.RegistryID, info.Image)

	if err != nil {
		log.Err(err).Msgf("Failed to search package for search term: [%s]", info.Image)
		return &npm2.PackageSearch{}, err
	}
	psList := make([]*npm2.PackageSearchObject, 0)
	registryURL := c.urlProvider.PackageURL(ctx,
		info.BaseArtifactInfo().RootIdentifier+"/"+info.BaseArtifactInfo().RegIdentifier, "npm")

	for _, metadata := range *metadataList {
		pso, err := mapToPackageSearch(metadata, registryURL)
		if err != nil {
			log.Err(err).Msgf("Failed to map search package results: [%s]", info.Image)
			return &npm2.PackageSearch{}, err
		}
		psList = append(psList, pso)
	}
	return &npm2.PackageSearch{
		Objects: psList,
		Total:   count,
	}, nil
}

func mapToPackageSearch(metadata types.Artifact, registryURL string) (*npm2.PackageSearchObject, error) {
	var art *npm2.NpmMetadata
	if err := json.Unmarshal(metadata.Metadata, &art); err != nil {
		return &npm2.PackageSearchObject{}, err
	}

	for _, version := range art.Versions {
		var author npm2.User
		if version.Author != nil {
			data, err := json.Marshal(version.Author)
			if err != nil {
				log.Err(err).Msgf("Failed to marshal search package results: [%s]", art.Name)
				return &npm2.PackageSearchObject{}, err
			}
			err = json.Unmarshal(data, &author)
			if err != nil {
				log.Err(err).Msgf("Failed to unmarshal search package results: [%s]", art.Name)
				return &npm2.PackageSearchObject{}, err
			}
		}

		return &npm2.PackageSearchObject{
			Package: &npm2.PackageSearchPackage{
				Name:        version.Name,
				Version:     version.Version,
				Description: version.Description,
				Date:        metadata.CreatedAt,

				Scope:       getScope(art.Name),
				Author:      npm2.User{Username: author.Name},
				Publisher:   npm2.User{Username: author.Name},
				Maintainers: getValueOrDefault(version.Maintainers, []npm2.User{}), // npm cli needs this field
				Keywords:    getValueOrDefault(version.Keywords, []string{}),
				Links: &npm2.PackageSearchPackageLinks{
					Registry:   registryURL,
					Homepage:   registryURL,
					Repository: registryURL,
				},
			},
		}, nil
	}
	return &npm2.PackageSearchObject{}, fmt.Errorf("no version found in the metadata for image:[%s]", art.Name)
}

func getValueOrDefault(value any, defaultValue any) any {
	if value != nil {
		return value
	}
	return defaultValue
}

func getScope(name string) string {
	if strings.HasPrefix(name, "@") {
		if i := strings.Index(name, "/"); i != -1 {
			return name[1:i] // Strip @ and return only the scope
		}
	}
	return "unscoped"
}

func CreatePackageMetadataVersion(registryURL string,
	metadata *npm2.PackageMetadataVersion) *npm2.PackageMetadataVersion {
	return &npm2.PackageMetadataVersion{
		ID:                   fmt.Sprintf("%s@%s", metadata.Name, metadata.Version),
		Name:                 metadata.Name,
		Version:              metadata.Version,
		Description:          metadata.Description,
		Author:               metadata.Author,
		Homepage:             registryURL,
		License:              metadata.License,
		Dependencies:         metadata.Dependencies,
		BundleDependencies:   metadata.BundleDependencies,
		DevDependencies:      metadata.DevDependencies,
		PeerDependencies:     metadata.PeerDependencies,
		OptionalDependencies: metadata.OptionalDependencies,
		Readme:               metadata.Readme,
		Bin:                  metadata.Bin,
		Dist: npm2.PackageDistribution{
			Shasum:    metadata.Dist.Shasum,
			Integrity: metadata.Dist.Integrity,
			Tarball: fmt.Sprintf("%s/%s/-/%s/%s", registryURL, metadata.Name, metadata.Version,
				metadata.Name+"-"+metadata.Version+".tgz"),
		},
	}
}

func (c *localRegistry) ListTags(ctx context.Context, info npm.ArtifactInfo) (map[string]string, error) {
	tags, err := c.tagsDao.FindByImageNameAndRegID(ctx, info.Image, info.RegistryID)
	if err != nil {
		return nil, err
	}

	pkgTags := make(map[string]string)

	for _, tag := range tags {
		pkgTags[tag.Name] = tag.Version
	}
	return pkgTags, nil
}

func (c *localRegistry) AddTag(ctx context.Context, info npm.ArtifactInfo) (map[string]string, error) {
	image, err := c.imageDao.GetByRepoAndName(ctx, info.ParentID, info.RegIdentifier, info.Image)
	if err != nil {
		return nil, err
	}
	version, err := c.artifactDao.GetByName(ctx, image.ID, info.Version)

	if err != nil {
		return nil, err
	}

	if len(info.DistTags) == 0 {
		return nil, usererror.BadRequest("Add tag error: distTags are empty")
	}
	packageTag := &types.PackageTag{
		ID:         uuid.NewString(),
		Name:       info.DistTags[0],
		ArtifactID: version.ID,
	}
	_, err = c.tagsDao.Create(ctx, packageTag)
	if err != nil {
		return nil, err
	}
	return c.ListTags(ctx, info)
}

func (c *localRegistry) DeleteTag(ctx context.Context, info npm.ArtifactInfo) (map[string]string, error) {
	if len(info.DistTags) == 0 {
		return nil, usererror.BadRequest("Delete tag error: distTags are empty")
	}
	err := c.tagsDao.DeleteByTagAndImageName(ctx, info.DistTags[0], info.Image, info.RegistryID)
	if err != nil {
		return nil, err
	}
	return c.ListTags(ctx, info)
}

func (c *localRegistry) DeletePackage(ctx context.Context, info npm.ArtifactInfo) error {
	return c.localBase.DeletePackage(ctx, info)
}

func (c *localRegistry) DeleteVersion(ctx context.Context, info npm.ArtifactInfo) error {
	return c.localBase.DeleteVersion(ctx, info)
}

func (c *localRegistry) parseAndUploadNPMPackage(ctx context.Context, info npm.ArtifactInfo,
	reader io.Reader, packageMetadata *npm2.PackageMetadata) (types.FileInfo, string, error) {
	// Use a buffered reader with controlled buffer size instead of unlimited buffering
	// This prevents the JSON decoder from buffering the entire file
	bufferedReader := bufio.NewReaderSize(reader, 32*1024) // 32KB buffer instead of unlimited

	// Create decoder with controlled buffering
	decoder := json.NewDecoder(bufferedReader)

	var fileInfo types.FileInfo
	var tmpFileName string

	// Parse top-level fields
	for {
		token, err := decoder.Token()

		if err != nil {
			// Check for both io.EOF and any error containing "EOF" in the message
			if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "EOF") {
				break
			}
			return types.FileInfo{}, "", fmt.Errorf("failed to parse JSON: %w", err)
		}

		//nolint:nestif
		if token, ok := token.(string); ok {
			switch token {
			case "_id":
				if err := decoder.Decode(&packageMetadata.ID); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse _id: %w", err)
				}
			case "name":
				if err := decoder.Decode(&packageMetadata.Name); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse name: %w", err)
				}
			case "description":
				if err := decoder.Decode(&packageMetadata.Description); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse description: %w", err)
				}
			case "dist-tags":
				packageMetadata.DistTags = make(map[string]string)
				if err := decoder.Decode(&packageMetadata.DistTags); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse dist-tags: %w", err)
				}
			case "versions":
				packageMetadata.Versions = make(map[string]*npm2.PackageMetadataVersion)
				if err := decoder.Decode(&packageMetadata.Versions); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse versions: %w", err)
				}
			case "readme":
				if err := decoder.Decode(&packageMetadata.Readme); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse readme: %w", err)
				}
			case "maintainers":
				if err := decoder.Decode(&packageMetadata.Maintainers); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse maintainers: %w", err)
				}
			case "time":
				packageMetadata.Time = make(map[string]time.Time)
				if err := decoder.Decode(&packageMetadata.Time); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse time: %w", err)
				}
			case "homepage":
				if err := decoder.Decode(&packageMetadata.Homepage); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse homepage: %w", err)
				}
			case "keywords":
				if err := decoder.Decode(&packageMetadata.Keywords); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse keywords: %w", err)
				}
			case "repository":
				if err := decoder.Decode(&packageMetadata.Repository); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse repository: %w", err)
				}
			case "author":
				if err := decoder.Decode(&packageMetadata.Author); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse author: %w", err)
				}
			case "readmeFilename":
				if err := decoder.Decode(&packageMetadata.ReadmeFilename); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse readmeFilename: %w", err)
				}
			case "users":
				if err := decoder.Decode(&packageMetadata.Users); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse users: %w", err)
				}
			case "license":
				if err := decoder.Decode(&packageMetadata.License); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse license: %w", err)
				}
			case "_attachments":

				// Process attachments with optimized streaming to minimize memory usage
				fileInfo, tmpFileName, err = c.processAttachmentsOptimized(ctx, info, decoder, bufferedReader)
				if err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to process attachments: %w", err)
				}
				log.Info().Str("packageName", info.Image).Msg("Successfully uploaded NPM package using optimized processing")

				// We're done processing attachments, break out of the main parsing loop
				return fileInfo, tmpFileName, nil
			default:
				var dummy any
				if err := decoder.Decode(&dummy); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to parse field %s: %w", token, err)
				}
			}
		}
	}

	return fileInfo, tmpFileName, nil
}

// processAttachmentsOptimized handles attachment processing with minimal memory buffering.
func (c *localRegistry) processAttachmentsOptimized(ctx context.Context, info npm.ArtifactInfo,
	decoder *json.Decoder, bufferedReader *bufio.Reader) (types.FileInfo, string, error) {
	// Parse the attachments map with minimal buffering
	t, err := decoder.Token()
	if err != nil {
		return types.FileInfo{}, "", fmt.Errorf("failed to parse _attachments: %w", err)
	}
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return types.FileInfo{}, "", fmt.Errorf("expected '{' at start of _attachments")
	}

	// Process each attachment (e.g., "test-large-package-2.0.0.tgz")
	for {
		//nolint:govet
		t, err := decoder.Token()
		if err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "EOF") {
				break
			}
			return types.FileInfo{}, "", fmt.Errorf("failed to parse JSON: %w", err)
		}
		if delim, ok := t.(json.Delim); ok && delim == '}' {
			break // End of _attachments object
		}
		attachmentKey, ok := t.(string)
		if !ok {
			return types.FileInfo{}, "", fmt.Errorf("expected string key in _attachments")
		}

		// Expect the start of the attachment object
		t, err = decoder.Token()
		if err != nil {
			return types.FileInfo{}, "", fmt.Errorf("failed to parse attachment %s: %w", attachmentKey, err)
		}
		if delim, ok := t.(json.Delim); !ok || delim != '{' {
			return types.FileInfo{}, "", fmt.Errorf("expected '{' for attachment %s", attachmentKey)
		}

		// Process fields within the attachment object with optimized streaming
		for {
			//nolint:govet
			t, err := decoder.Token()
			if err != nil {
				if err == io.EOF || strings.Contains(err.Error(), "EOF") {
					break
				}
				return types.FileInfo{}, "", fmt.Errorf("failed to parse attachment %s fields: %w", attachmentKey, err)
			}
			if delim, ok := t.(json.Delim); ok && delim == '}' {
				break // End of attachment object
			}
			field, ok := t.(string)
			if !ok {
				break
			}

			switch field {
			case "data":
				// Use optimized base64 streaming with minimal buffering
				return c.processBase64DataOptimized(ctx, info, decoder, bufferedReader, attachmentKey)
			default:
				// Skip other fields efficiently
				var dummy any
				if err := decoder.Decode(&dummy); err != nil {
					return types.FileInfo{}, "", fmt.Errorf("failed to skip field %s: %w", field, err)
				}
			}
		}
	}

	return types.FileInfo{}, "", fmt.Errorf("no attachment data found")
}

// processBase64DataOptimized handles base64 data processing with minimal memory usage.
func (c *localRegistry) processBase64DataOptimized(ctx context.Context, info npm.ArtifactInfo,
	decoder *json.Decoder, bufferedReader *bufio.Reader, attachmentKey string) (types.FileInfo, string, error) {
	// Get the remaining data from decoder's buffer + original reader
	// This avoids the memory-heavy io.MultiReader approach
	combinedReader := io.MultiReader(decoder.Buffered(), bufferedReader)

	// Use a smaller buffer for reading the base64 stream
	streamReader := bufio.NewReaderSize(combinedReader, 8*1024) // 8KB buffer instead of default

	// Expecting `:` character first
	startByte, err := streamReader.ReadByte()
	if err != nil {
		return types.FileInfo{}, "",
			fmt.Errorf("failed to upload attachment %s: Error while reading : character: %w", attachmentKey, err)
	}
	if startByte != ':' {
		return types.FileInfo{}, "",
			fmt.Errorf("failed to upload attachment %s: Expected : character, got %c", attachmentKey, startByte)
	}

	// Now expecting `"`, marking start of JSON string
	startByte, err = streamReader.ReadByte()
	if err != nil {
		return types.FileInfo{}, "", fmt.Errorf("failed to upload"+
			" attachment %s: Error while reading \" character: %w", attachmentKey, err)
	}
	if startByte != '"' {
		return types.FileInfo{}, "", fmt.Errorf("failed to upload"+
			" attachment %s: Expected \" character, got %c", attachmentKey, startByte)
	}

	// Use optimized JSON string reader with smaller buffer
	b64StreamReader := NewOptimizedJSONStringStreamReader(streamReader)

	// Wrap base64 decoder with error handling
	base64Reader := io.NopCloser(base64.NewDecoder(base64.StdEncoding, b64StreamReader))

	log.Info().Str("packageName", info.Image).Msg("Uploading NPM package with optimized streaming")
	fileInfo, tmpFileName, err := c.fileManager.UploadTempFile(ctx, info.RootIdentifier, nil, "tmp", base64Reader)
	if err != nil {
		if strings.Contains(err.Error(), "unexpected EOF") {
			return types.FileInfo{}, "",
				fmt.Errorf("failed to upload attachment %s: "+
					"base64 data may be corrupted or missing closing quote: %w", attachmentKey, err)
		}
		return types.FileInfo{}, "", fmt.Errorf("failed to upload attachment %s: %w", attachmentKey, err)
	}

	log.Info().Str("packageName", info.Image).Msg("Successfully uploaded NPM package with optimized streaming")
	return fileInfo, tmpFileName, nil
}

// NewOptimizedJSONStringStreamReader returns an io.Reader that stops at the closing quote of a JSON string
// with optimized chunk-based processing instead of byte-by-byte reading to reduce memory overhead.
func NewOptimizedJSONStringStreamReader(r *bufio.Reader) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		var escaped bool
		buf := make([]byte, 4096) // Process in 4KB chunks instead of byte-by-byte

		for {
			n, err := r.Read(buf)
			if err != nil && err != io.EOF {
				pw.CloseWithError(fmt.Errorf("error while reading base64 string: %w", err))
				return
			}

			if n == 0 {
				if err == io.EOF {
					pw.CloseWithError(fmt.Errorf("unexpected EOF while reading base64 string: missing closing quote"))
				}
				return
			}

			// Process the chunk for quotes and escapes
			writeStart := 0
			for i := range n {
				b := buf[i]

				if escaped {
					escaped = false
					continue
				}

				if b == '\\' {
					escaped = true
					continue
				}

				if b == '"' {
					// Found end quote - write remaining data up to this point and exit
					if i > writeStart {
						if _, writeErr := pw.Write(buf[writeStart:i]); writeErr != nil {
							pw.CloseWithError(writeErr)
							return
						}
					}
					return
				}
			}

			// Write the entire chunk if no end quote found
			if _, writeErr := pw.Write(buf[writeStart:n]); writeErr != nil {
				pw.CloseWithError(writeErr)
				return
			}

			if err == io.EOF {
				pw.CloseWithError(fmt.Errorf("unexpected EOF while reading base64 string: missing closing quote"))
				return
			}
		}
	}()

	return pr
}

func (c *localRegistry) UploadPackageFileWithoutParsing(
	ctx context.Context,
	info npm.ArtifactInfo,
	file io.ReadCloser,
) (headers *commons.ResponseHeaders, sha256 string, err error) {
	defer file.Close()
	path := pkg.JoinWithSeparator("/", info.Image, info.Version, info.Filename)
	response, sha, err := c.localBase.Upload(ctx, info.ArtifactInfo, info.Filename, info.Version, path, file,
		&npm2.NpmMetadata{
			PackageMetadata: info.Metadata,
		})
	if !commons.IsEmpty(err) {
		return nil, "", err
	}
	_, err = c.AddTag(ctx, info)
	if err != nil {
		return nil, "", err
	}
	return response, sha, nil
}
