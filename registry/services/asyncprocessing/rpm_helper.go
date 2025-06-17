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

package asyncprocessing

//nolint:gosec
import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	rpmmetadata "github.com/harness/gitness/registry/app/metadata/rpm"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/rpm"
	"github.com/harness/gitness/registry/app/store"
	rpmtypes "github.com/harness/gitness/registry/app/utils/rpm/types"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"
)

const (
	RepoMdFile           = "repomd.xml"
	RepoDataPrefix       = "repodata/"
	PrimaryFile          = "primary.xml.gz"
	OtherFile            = "other.xml.gz"
	FileListsFile        = "filelists.xml.gz"
	artifactBatchLimit   = 50
	packageStartElements = "package"
)

type RpmHelper interface {
	BuildRegistryFiles(
		ctx context.Context,
		registry types.Registry,
		principalID int64,
	) error
}

type rpmHelper struct {
	fileManager        filemanager.FileManager
	artifactDao        store.ArtifactRepository
	upstreamProxyStore store.UpstreamProxyConfigRepository
	spaceFinder        refcache.SpaceFinder
	secretService      secret.Service
	registryDao        store.RegistryRepository
}

type registryData interface {
	getReader(ctx context.Context) (io.ReadCloser, error)
}

func (l localRepoData) getReader(ctx context.Context) (io.ReadCloser, error) {
	primaryReader, _, _, err := l.fileManager.DownloadFile(
		ctx, "/"+l.fileRef, l.registryID, l.registryIdentifier, l.rootIdentifier, false,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary.xml.gz: %w", err)
	}
	return primaryReader, nil
}

type localRepoData struct {
	fileManager        filemanager.FileManager
	fileRef            string
	registryID         int64
	registryIdentifier string
	rootIdentifier     string
}

type remoteRepoData struct {
	helper  rpm.RemoteRegistryHelper
	fileRef string
}

func (r remoteRepoData) getReader(ctx context.Context) (io.ReadCloser, error) {
	readCloser, err := r.helper.GetMetadataFile(ctx, "/"+r.fileRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary.xml.gz: %w", err)
	}
	return readCloser, nil
}

func NewRpmHelper(
	fileManager filemanager.FileManager,
	artifactDao store.ArtifactRepository,
	upstreamProxyStore store.UpstreamProxyConfigRepository,
	spaceFinder refcache.SpaceFinder,
	secretService secret.Service,
	registryDao store.RegistryRepository,
) RpmHelper {
	return &rpmHelper{
		fileManager:        fileManager,
		artifactDao:        artifactDao,
		upstreamProxyStore: upstreamProxyStore,
		spaceFinder:        spaceFinder,
		secretService:      secretService,
		registryDao:        registryDao,
	}
}

func (l *rpmHelper) BuildRegistryFiles(
	ctx context.Context,
	registry types.Registry,
	principalID int64,
) error {
	rootSpace, err := l.spaceFinder.FindByID(ctx, registry.RootParentID)
	if err != nil {
		return fmt.Errorf("failed to find root space by ID: %w", err)
	}
	existingPackageInfos, err := l.getExistingArtifactInfos(ctx, registry.ID)
	if err != nil {
		return err
	}
	if registry.Type == artifact.RegistryTypeUPSTREAM {
		return l.buildForUpstream(ctx, registry.ID, registry.RootParentID, existingPackageInfos,
			rootSpace.Identifier, principalID)
	}
	return l.buildForVirtual(ctx, registry.ID, registry.Name, registry.RootParentID, registry.ParentID,
		rootSpace.Identifier, existingPackageInfos, principalID)
}

func (l *rpmHelper) buildForVirtual(
	ctx context.Context,
	registryID int64,
	registryIdentifier string,
	rootParentID int64,
	parentID int64,
	rootIdentifier string,
	existingPackageInfos []*rpmtypes.PackageInfo,
	principalID int64,
) error {
	registries, err := base.GetOrderedRepos(ctx, l.registryDao, registryIdentifier, parentID, true)
	if err != nil {
		return err
	}

	var primary, fileLists, other *rpmtypes.RepoData

	primaryRegistryData, err := l.getRegistryData(ctx, registries, rootIdentifier, "primary")
	if err != nil {
		return err
	}
	primary, err = l.buildPrimary(ctx, existingPackageInfos, registryID, rootParentID, rootIdentifier,
		"", false, primaryRegistryData, principalID)
	if err != nil {
		return err
	}

	fileListsRegistryData, err := l.getRegistryData(ctx, registries, rootIdentifier, "filelists")
	if err != nil {
		return err
	}
	if fileListsRegistryData != nil {
		fileLists, err = l.buildFileLists(ctx, existingPackageInfos, registryID, rootParentID,
			rootIdentifier, fileListsRegistryData, principalID)
		if err != nil {
			return err
		}
	}

	otherRegistryData, err := l.getRegistryData(ctx, registries, rootIdentifier, "other")
	if err != nil {
		return err
	}
	if otherRegistryData != nil {
		other, err = l.buildOther(ctx, existingPackageInfos, registryID, rootParentID,
			rootIdentifier, otherRegistryData, principalID)
		if err != nil {
			return err
		}
	}

	err = l.buildRepoMDFile(ctx, registryID, rootParentID,
		primary, fileLists, other, rootIdentifier, principalID)
	return err
}

func (l *rpmHelper) buildForUpstream(
	ctx context.Context,
	registryID int64,
	rootParentID int64,
	existingPackageInfos []*rpmtypes.PackageInfo,
	rootIdentifier string,
	principalID int64,
) error {
	var primary, fileLists, other *rpmtypes.RepoData
	upstream, err := l.upstreamProxyStore.Get(ctx, registryID)
	if err != nil {
		return err
	}

	helper, err := rpm.NewRemoteRegistryHelper(ctx, l.spaceFinder, *upstream, l.secretService)
	if err != nil {
		return err
	}

	primaryRef, fileListsRef, otherRef, err := l.getRefs(ctx, helper)
	if err != nil {
		return err
	}

	prd := &remoteRepoData{
		helper:  helper,
		fileRef: primaryRef,
	}
	primary, err = l.buildPrimary(ctx, existingPackageInfos, registryID, rootParentID, rootIdentifier,
		upstream.RepoKey, true, []registryData{prd}, principalID)
	if err != nil {
		return err
	}

	if fileListsRef != "" {
		frd := &remoteRepoData{
			helper:  helper,
			fileRef: fileListsRef,
		}
		fileLists, err = l.buildFileLists(ctx, existingPackageInfos, registryID, rootParentID,
			rootIdentifier, []registryData{frd}, principalID)
		if err != nil {
			return err
		}
	}

	if otherRef != "" {
		ord := &remoteRepoData{
			helper:  helper,
			fileRef: otherRef,
		}
		other, err = l.buildOther(ctx, existingPackageInfos, registryID, rootParentID,
			rootIdentifier, []registryData{ord}, principalID)
		if err != nil {
			return err
		}
	}

	// Build the repodata file
	err = l.buildRepoMDFile(ctx, registryID, rootParentID, primary, fileLists,
		other, rootIdentifier, principalID)
	if err != nil {
		return err
	}
	return nil
}

func (l *rpmHelper) getRegistryData(
	ctx context.Context,
	registries []types.Registry,
	rootIdentifier string,
	refType string,
) ([]registryData, error) {
	var rd []registryData
	for i := 1; i < len(registries); i++ {
		r := registries[i]
		fileRef, err := l.getRefsForHarnessRepos(ctx, r.ID, r.Name, rootIdentifier, refType)
		if err != nil {
			return nil, err
		}
		if fileRef != "" {
			rd = append(rd, &localRepoData{
				l.fileManager,
				fileRef,
				r.ID,
				r.Name,
				rootIdentifier,
			})
		}
	}
	return rd, nil
}

func (l *rpmHelper) getExistingArtifactInfos(
	ctx context.Context,
	registryID int64,
) ([]*rpmtypes.PackageInfo, error) {
	lastArtifactID := int64(0)
	var packageInfos []*rpmtypes.PackageInfo
	for {
		artifacts, err := l.artifactDao.GetAllArtifactsByRepo(ctx, registryID, artifactBatchLimit, lastArtifactID)
		if err != nil {
			return nil, err
		}

		for _, a := range *artifacts {
			metadata := rpmmetadata.RpmMetadata{}
			err := json.Unmarshal(a.Metadata, &metadata)
			if err != nil {
				return nil, err
			}

			packageInfos = append(packageInfos, &rpmtypes.PackageInfo{
				Name:            a.Name,
				Sha256:          metadata.GetFiles()[0].Sha256,
				Size:            metadata.GetFiles()[0].Size,
				VersionMetadata: &metadata.VersionMetadata,
				FileMetadata:    &metadata.FileMetadata,
			})
			if a.ID > lastArtifactID {
				lastArtifactID = a.ID
			}
		}
		if len(*artifacts) < artifactBatchLimit {
			break
		}
	}
	return packageInfos, nil
}

func (l *rpmHelper) buildRepoMDFile(
	ctx context.Context,
	registryID int64,
	rootParentID int64,
	primary *rpmtypes.RepoData,
	fileLists *rpmtypes.RepoData,
	other *rpmtypes.RepoData,
	rootIdentifier string,
	principalID int64,
) error {
	err := l.buildRepomd(ctx, []*rpmtypes.RepoData{
		primary,
		fileLists,
		other,
	}, registryID, rootParentID, rootIdentifier, principalID)
	return err
}

func (l *rpmHelper) getRefsForHarnessRepos(
	ctx context.Context,
	registryID int64,
	registryIdentifier string,
	rootIdentifier string,
	refType string,
) (string, error) {
	fileReader, _, _, err := l.fileManager.DownloadFile(
		ctx, "/repodata/repomd.xml", registryID, registryIdentifier, rootIdentifier, false,
	)
	if err != nil {
		return "", err
	}
	defer fileReader.Close()
	var repoMD rpmtypes.Repomd
	if err := xml.NewDecoder(fileReader).Decode(&repoMD); err != nil {
		return "", fmt.Errorf("failed to parse repomd.xml: %w", err)
	}
	for _, data := range repoMD.Data {
		if refType == data.Type {
			return data.Location.Href, nil
		}
	}
	return "", nil
}

func (l *rpmHelper) getRefs(ctx context.Context, helper rpm.RemoteRegistryHelper) (string, string, string, error) {
	closer, err := helper.GetMetadataFile(ctx, "/repodata/repomd.xml")
	if err != nil {
		return "", "", "", err
	}
	defer closer.Close()
	var repoMD rpmtypes.Repomd
	var primaryRef string
	var filelistsRef string
	var otherRef string
	if err := xml.NewDecoder(closer).Decode(&repoMD); err != nil {
		return "", "", "", fmt.Errorf("failed to parse repomd.xml: %w", err)
	}
	for _, data := range repoMD.Data {
		switch data.Type {
		case "primary":
			primaryRef = data.Location.Href
		case "filelists":
			filelistsRef = data.Location.Href
		case "other":
			otherRef = data.Location.Href
		}
	}
	return primaryRef, filelistsRef, otherRef, nil
}

func (l *rpmHelper) validateRootElement(fileListsDecoder *xml.Decoder, rootElement string) error {
	var startElem xml.StartElement
	for {
		token, err := fileListsDecoder.Token()
		if err != nil {
			return fmt.Errorf("failed to read start element: %w", err)
		}

		if se, ok := token.(xml.StartElement); ok {
			startElem = se
			break
		}
	}

	if startElem.Name.Local != rootElement {
		return fmt.Errorf("unexpected root element: %v", startElem.Name.Local)
	}
	return nil
}

func (l *rpmHelper) buildPrimary(
	ctx context.Context,
	existingPis []*rpmtypes.PackageInfo,
	registryID int64,
	rootParentID int64,
	rootIdentifier string,
	repoKey string,
	overridePath bool,
	repoDataList []registryData,
	principalID int64,
) (*rpmtypes.RepoData, error) {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()

		set := make(map[string]struct{})
		gzw := gzip.NewWriter(pw)
		defer gzw.Close()

		encoder := xml.NewEncoder(gzw)
		if _, err := gzw.Write([]byte(xml.Header)); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to write XML header: %w", err))
			return
		}

		if err := encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "metadata"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "xmlns"}, Value: "http://linux.duke.edu/metadata/common"},
				{Name: xml.Name{Local: "xmlns:rpm"}, Value: "http://linux.duke.edu/metadata/rpm"},
				{Name: xml.Name{Local: "packages"}, Value: fmt.Sprintf("%d", 0)}, // TODO fix size
			}}); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to encode metadata start: %w", err))
			return
		}

		for _, pi := range existingPis {
			_, pp := getPrimaryPackage(pi)
			if err := encoder.Encode(pp); err != nil {
				pw.CloseWithError(fmt.Errorf("failed to encode package: %w", err))
				return
			}
			set[fmt.Sprintf("%s:%s-%s:%s", pp.Name, pp.Version.Version, pp.Version.Release,
				pp.Architecture)] = struct{}{}
		}

		for _, rd := range repoDataList {
			readCloser, err := rd.getReader(ctx)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to get primary.xml.gz: %w", err))
				return
			}
			defer readCloser.Close()

			gzipReader, err := gzip.NewReader(readCloser)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to create gzip reader: %w", err))
				return
			}
			defer gzipReader.Close()

			decoder := xml.NewDecoder(gzipReader)
			err = l.validateRootElement(decoder, "metadata")
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to validate primary.xml root element: %w", err))
				return
			}

			for {
				token, err := decoder.Token()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					pw.CloseWithError(fmt.Errorf("error reading primary.xml: %w", err))
					return
				}

				packageStartElement, ok := token.(xml.StartElement)
				if !ok || packageStartElement.Name.Local != packageStartElements {
					continue
				}

				var pkg rpmtypes.PrimaryPackage
				if err := decoder.DecodeElement(&pkg, &packageStartElement); err != nil {
					pw.CloseWithError(fmt.Errorf("failed to decode package: %w", err))
					return
				}

				if overridePath {
					packageVersion := fmt.Sprintf("%s-%s", pkg.Version.Version, pkg.Version.Release)
					pkg.Location.Href = fmt.Sprintf("../../%s/rpm/package/%s/%s/%s/%s/%s", repoKey,
						url.PathEscape(pkg.Name),
						url.PathEscape(packageVersion),
						url.PathEscape(pkg.Architecture),
						url.PathEscape(fmt.Sprintf("%s-%s.%s.rpm", pkg.Name, packageVersion, pkg.Architecture)),
						pkg.Location.Href)
				}

				pkgref := fmt.Sprintf("%s:%s-%s:%s", pkg.Name, pkg.Version.Version,
					pkg.Version.Release, pkg.Architecture)
				if _, exists := set[pkgref]; !exists {
					if err := encoder.Encode(pkg); err != nil {
						pw.CloseWithError(fmt.Errorf("failed to encode package: %w", err))
						return
					}
					set[fmt.Sprintf("%s:%s-%s:%s", pkg.Name, pkg.Version.Version, pkg.Version.Release,
						pkg.Architecture)] = struct{}{}
				}
			}
		}

		if err := encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "metadata"}}); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to encode metadata end: %w", err))
			return
		}

		if err := encoder.Flush(); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to flush encoder: %w", err))
			return
		}
	}()

	info, tempFileName, err := l.fileManager.UploadTempFile(ctx, rootIdentifier, nil, PrimaryFile, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	filePath := RepoDataPrefix + info.Sha256 + "-" + PrimaryFile
	err = l.fileManager.MoveTempFile(ctx, filePath, registryID, rootParentID, rootIdentifier,
		info, tempFileName, principalID)
	if err != nil {
		return nil, fmt.Errorf("failed to move temp file [%s] to [%s]: %w", tempFileName, filePath, err)
	}

	return getRepoData(info, filePath, "primary"), nil
}

func (l *rpmHelper) buildOther(
	ctx context.Context,
	pds []*rpmtypes.PackageInfo,
	registryID int64,
	rootParentID int64,
	rootIdentifier string,
	repoDataList []registryData,
	principalID int64,
) (*rpmtypes.RepoData, error) {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		set := make(map[string]struct{})

		gzw := gzip.NewWriter(pw)
		defer gzw.Close()

		encoder := xml.NewEncoder(gzw)
		if _, err := gzw.Write([]byte(xml.Header)); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to write XML header: %w", err))
			return
		}

		if err := encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "otherdata"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "xmlns"}, Value: "http://linux.duke.edu/metadata/other"},
				{Name: xml.Name{Local: "packages"}, Value: fmt.Sprintf("%d", 0)}, // TODO fix size
			}}); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to encode metadata start: %w", err))
			return
		}

		for _, pd := range pds {
			op := getOtherPackage(pd)
			if err := encoder.Encode(op); err != nil {
				pw.CloseWithError(fmt.Errorf("failed to encode package: %w", err))
				return
			}
			set[op.Pkgid] = struct{}{}
		}
		for _, rd := range repoDataList {
			readCloser, err := rd.getReader(ctx)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to get other.xml.gz: %w", err))
				return
			}
			defer readCloser.Close()

			otherGzipReader, err := gzip.NewReader(readCloser)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to create gzip reader for other: %w", err))
				return
			}
			defer otherGzipReader.Close()
			otherDecoder := xml.NewDecoder(otherGzipReader)

			err = l.validateRootElement(otherDecoder, "otherdata")
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to validate other.xml root element: %w", err))
				return
			}
			for {
				token, err := otherDecoder.Token()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					pw.CloseWithError(fmt.Errorf("error reading other.xml: %w", err))
				}

				packageStartElement, ok := token.(xml.StartElement)
				if !ok || packageStartElement.Name.Local != packageStartElements {
					continue
				}

				var pkg rpmtypes.OtherPackage
				if err := otherDecoder.DecodeElement(&pkg, &packageStartElement); err != nil {
					pw.CloseWithError(fmt.Errorf("failed to decode other package: %w", err))
				}
				if _, exists := set[pkg.Pkgid]; !exists {
					if err := encoder.Encode(pkg); err != nil {
						pw.CloseWithError(fmt.Errorf("failed to encode package: %w", err))
						return
					}
					set[pkg.Pkgid] = struct{}{}
				}
			}
		}

		if err := encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "otherdata"}}); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to encode metadata end: %w", err))
			return
		}

		if err := encoder.Flush(); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to flush encoder: %w", err))
			return
		}
	}()

	info, tempFileName, err := l.fileManager.UploadTempFile(ctx, rootIdentifier, nil, OtherFile, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	filePath := RepoDataPrefix + info.Sha256 + "-" + OtherFile
	err = l.fileManager.MoveTempFile(ctx, filePath, registryID,
		rootParentID, rootIdentifier, info, tempFileName, principalID)
	if err != nil {
		return nil, fmt.Errorf("failed to move temp file [%s] to [%s]: %w", tempFileName, filePath, err)
	}

	return getRepoData(info, filePath, "other"), nil
}

func (l *rpmHelper) buildFileLists(
	ctx context.Context,
	pis []*rpmtypes.PackageInfo,
	registryID int64,
	rootParentID int64,
	rootIdentifier string,
	repoDataList []registryData,
	principalID int64,
) (*rpmtypes.RepoData, error) { //nolint:dupl
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		set := make(map[string]struct{})
		gzw := gzip.NewWriter(pw)
		defer gzw.Close()

		encoder := xml.NewEncoder(gzw)
		if _, err := gzw.Write([]byte(xml.Header)); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to write XML header: %w", err))
			return
		}

		if err := encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "filelists"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "xmlns"}, Value: "http://linux.duke.edu/metadata/filelists"},
				{Name: xml.Name{Local: "packages"}, Value: fmt.Sprintf("%d", 0)}, // TODO fix size
			}}); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to encode metadata start: %w", err))
			return
		}

		for _, pi := range pis {
			fp := getFileListsPackage(pi)
			if err := encoder.Encode(fp); err != nil {
				pw.CloseWithError(fmt.Errorf("failed to encode package: %w", err))
				return
			}
			set[fp.Pkgid] = struct{}{}
		}
		for _, rd := range repoDataList {
			readCloser, err := rd.getReader(ctx)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to get filelists.xml.gz: %w", err))
				return
			}
			defer readCloser.Close()

			filelistsGzipReader, err := gzip.NewReader(readCloser)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to create gzip reader for filelists: %w", err))
				return
			}
			defer filelistsGzipReader.Close()
			fileListsDecoder := xml.NewDecoder(filelistsGzipReader)

			err = l.validateRootElement(fileListsDecoder, "filelists")
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to validate filelists.xml root element: %w", err))
				return
			}
			for {
				token, err := fileListsDecoder.Token()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					pw.CloseWithError(fmt.Errorf("error reading filelists.xml: %w", err))
					return
				}

				packageStartElement, ok := token.(xml.StartElement)
				if !ok || packageStartElement.Name.Local != packageStartElements {
					continue
				}

				var pkg rpmtypes.FileListPackage
				if err := fileListsDecoder.DecodeElement(&pkg, &packageStartElement); err != nil {
					pw.CloseWithError(fmt.Errorf("failed to decode filelists package: %w", err))
					return
				}
				if _, exists := set[pkg.Pkgid]; !exists {
					if err := encoder.Encode(pkg); err != nil {
						pw.CloseWithError(fmt.Errorf("failed to encode package: %w", err))
						return
					}
					set[pkg.Pkgid] = struct{}{}
				}
			}
		}

		if err := encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "filelists"}}); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to encode metadata end: %w", err))
			return
		}

		if err := encoder.Flush(); err != nil {
			pw.CloseWithError(fmt.Errorf("failed to flush encoder: %w", err))
			return
		}
	}()

	info, tempFileName, err := l.fileManager.UploadTempFile(ctx, rootIdentifier, nil, FileListsFile, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	filePath := RepoDataPrefix + info.Sha256 + "-" + FileListsFile
	err = l.fileManager.MoveTempFile(ctx, filePath, registryID,
		rootParentID, rootIdentifier, info, tempFileName, principalID)
	if err != nil {
		return nil, fmt.Errorf("failed to move temp file [%s] to [%s]: %w", tempFileName, filePath, err)
	}

	return getRepoData(info, filePath, "filelists"), nil
}

func getRepoData(
	info types.FileInfo,
	filePath string,
	dataType string,
) *rpmtypes.RepoData {
	return &rpmtypes.RepoData{
		Type: dataType,
		Checksum: rpmtypes.RepoChecksum{
			Type:  "sha256",
			Value: info.Sha256,
		},
		OpenChecksum: rpmtypes.RepoChecksum{
			Type:  "sha256",
			Value: info.Sha256,
		},
		Location: rpmtypes.RepoLocation{
			Href: filePath,
		},
		Timestamp: time.Now().Unix(),
		Size:      info.Size,
		OpenSize:  info.Size,
	}
}

func getFileListsPackage(pi *rpmtypes.PackageInfo) *rpmtypes.FileListPackage {
	fp := &rpmtypes.FileListPackage{
		Pkgid:        pi.Sha256,
		Name:         pi.Name,
		Architecture: pi.FileMetadata.Architecture,
		Version: rpmtypes.FileListVersion{
			Epoch:   pi.FileMetadata.Epoch,
			Version: pi.FileMetadata.Version,
			Release: pi.FileMetadata.Release,
		},
		Files: pi.FileMetadata.Files,
	}
	return fp
}

func getOtherPackage(pd *rpmtypes.PackageInfo) *rpmtypes.OtherPackage {
	op := &rpmtypes.OtherPackage{
		Pkgid:        pd.Sha256,
		Name:         pd.Name,
		Architecture: pd.FileMetadata.Architecture,
		Version: rpmtypes.OtherVersion{
			Epoch:   pd.FileMetadata.Epoch,
			Version: pd.FileMetadata.Version,
			Release: pd.FileMetadata.Release,
		},
		Changelogs: pd.FileMetadata.Changelogs,
	}
	return op
}

func (l *rpmHelper) buildRepomd(
	ctx context.Context,
	data []*rpmtypes.RepoData,
	registryID int64,
	rootParentID int64,
	rootIdentifier string,
	principalID int64,
) error {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	if err := xml.NewEncoder(&buf).Encode(&rpmtypes.Repomd{
		Xmlns:    "http://linux.duke.edu/metadata/repo",
		XmlnsRpm: "http://linux.duke.edu/metadata/rpm",
		Data:     data,
	}); err != nil {
		return err
	}
	repomdContent, _ := rpmtypes.CreateHashedBufferFromReader(&buf)
	defer repomdContent.Close()

	_, err := l.fileManager.UploadFile(ctx, RepoDataPrefix+RepoMdFile, registryID,
		rootParentID, rootIdentifier, repomdContent, repomdContent, RepoMdFile, principalID)
	if err != nil {
		return err
	}
	return nil
}

func getPrimaryPackage(pi *rpmtypes.PackageInfo) (string, *rpmtypes.PrimaryPackage) {
	files := make([]*rpmmetadata.File, 0, 3)
	for _, f := range pi.FileMetadata.Files {
		if f.IsExecutable {
			files = append(files, f)
		}
	}
	packageVersion := fmt.Sprintf("%s-%s", pi.FileMetadata.Version, pi.FileMetadata.Release)
	key := fmt.Sprintf("%s:%s:%s", pi.Name, packageVersion, pi.FileMetadata.Architecture)
	pp := &rpmtypes.PrimaryPackage{
		Type:         "rpm",
		Name:         pi.Name,
		Architecture: pi.FileMetadata.Architecture,
		Version: rpmtypes.PrimaryVersion{
			Epoch:   pi.FileMetadata.Epoch,
			Version: pi.FileMetadata.Version,
			Release: pi.FileMetadata.Release,
		},
		Checksum: rpmtypes.PrimaryChecksum{
			Type:     "sha256",
			Checksum: pi.Sha256,
			Pkgid:    "YES",
		},
		Summary:     pi.VersionMetadata.Summary,
		Description: pi.VersionMetadata.Description,
		Packager:    pi.FileMetadata.Packager,
		URL:         pi.VersionMetadata.ProjectURL,
		Time: rpmtypes.PrimaryTimes{
			File:  pi.FileMetadata.FileTime,
			Build: pi.FileMetadata.BuildTime,
		},
		Size: rpmtypes.PrimarySizes{
			Package:   pi.Size,
			Installed: pi.FileMetadata.InstalledSize,
			Archive:   pi.FileMetadata.ArchiveSize,
		},
		Location: rpmtypes.PrimaryLocation{
			Href: fmt.Sprintf("package/%s/%s/%s/%s",
				url.PathEscape(pi.Name),
				url.PathEscape(packageVersion),
				url.PathEscape(pi.FileMetadata.Architecture),
				url.PathEscape(fmt.Sprintf("%s-%s.%s.rpm", pi.Name, packageVersion, pi.FileMetadata.Architecture))),
		},
		Format: rpmtypes.PrimaryFormat{
			License:   pi.VersionMetadata.License,
			Vendor:    pi.FileMetadata.Vendor,
			Group:     pi.FileMetadata.Group,
			Buildhost: pi.FileMetadata.BuildHost,
			Sourcerpm: pi.FileMetadata.SourceRpm,
			Provides: rpmtypes.PrimaryEntryList{
				Entries: pi.FileMetadata.Provides,
			},
			Requires: rpmtypes.PrimaryEntryList{
				Entries: pi.FileMetadata.Requires,
			},
			Conflicts: rpmtypes.PrimaryEntryList{
				Entries: pi.FileMetadata.Conflicts,
			},
			Obsoletes: rpmtypes.PrimaryEntryList{
				Entries: pi.FileMetadata.Obsoletes,
			},
			Files: files,
		},
	}
	return key, pp
}
