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

package rpm

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"time"

	rpmmetadata "github.com/harness/gitness/registry/app/metadata/rpm"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	rpmtype "github.com/harness/gitness/registry/app/pkg/types/rpm"
	"github.com/harness/gitness/registry/app/store"
)

const artifactBatchLimit = 50

type LocalRegistryHelper interface {
	BuildRegistryFiles(ctx context.Context, info rpmtype.ArtifactInfo) error
}

type localRegistryHelper struct {
	fileManager filemanager.FileManager
	artifactDao store.ArtifactRepository
}

func NewLocalRegistryHelper(
	fileManager filemanager.FileManager,
	artifactDao store.ArtifactRepository,
) LocalRegistryHelper {
	return &localRegistryHelper{
		fileManager: fileManager,
		artifactDao: artifactDao,
	}
}

func (l *localRegistryHelper) BuildRegistryFiles(ctx context.Context, info rpmtype.ArtifactInfo) error {
	lastArtifactID := int64(0)
	var packageInfos []*packageInfo

	for {
		artifacts, err := l.artifactDao.GetAllArtifactsByRepo(ctx, info.RegistryID, artifactBatchLimit, lastArtifactID)
		if err != nil {
			return err
		}

		for _, a := range *artifacts {
			metadata := rpmmetadata.RpmMetadata{}
			err := json.Unmarshal(a.Metadata, &metadata)
			if err != nil {
				return err
			}

			packageInfos = append(packageInfos, &packageInfo{
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

	primary, err := l.buildPrimary(ctx, packageInfos, info)
	if err != nil {
		return err
	}
	fileLists, err := l.buildFilelists(ctx, packageInfos, info)
	if err != nil {
		return err
	}
	other, err := l.buildOther(ctx, packageInfos, info)
	if err != nil {
		return err
	}
	return l.buildRepomd(ctx, []*repoData{
		primary,
		fileLists,
		other,
	}, info)
}

func (l *localRegistryHelper) buildPrimary(
	ctx context.Context,
	pds []*packageInfo,
	info rpmtype.ArtifactInfo,
) (*repoData, error) {
	packages := make([]*primaryPackage, 0, len(pds))
	for _, pd := range pds {
		files := make([]*rpmmetadata.File, 0, 3)
		for _, f := range pd.FileMetadata.Files {
			if f.IsExecutable {
				files = append(files, f)
			}
		}
		packageVersion := fmt.Sprintf("%s-%s", pd.FileMetadata.Version, pd.FileMetadata.Release)
		packages = append(packages, &primaryPackage{
			Type:         "rpm",
			Name:         pd.Name,
			Architecture: pd.FileMetadata.Architecture,
			Version: primaryVersion{
				Epoch:   pd.FileMetadata.Epoch,
				Version: pd.FileMetadata.Version,
				Release: pd.FileMetadata.Release,
			},
			Checksum: primaryChecksum{
				Type:     "sha256",
				Checksum: pd.Sha256,
				Pkgid:    "YES",
			},
			Summary:     pd.VersionMetadata.Summary,
			Description: pd.VersionMetadata.Description,
			Packager:    pd.FileMetadata.Packager,
			URL:         pd.VersionMetadata.ProjectURL,
			Time: primaryTimes{
				File:  pd.FileMetadata.FileTime,
				Build: pd.FileMetadata.BuildTime,
			},
			Size: primarySizes{
				Package:   pd.Size,
				Installed: pd.FileMetadata.InstalledSize,
				Archive:   pd.FileMetadata.ArchiveSize,
			},
			Location: PrimaryLocation{
				Href: fmt.Sprintf("package/%s/%s/%s/%s",
					url.PathEscape(pd.Name),
					url.PathEscape(packageVersion),
					url.PathEscape(pd.FileMetadata.Architecture),
					url.PathEscape(fmt.Sprintf("%s-%s.%s.rpm", pd.Name, packageVersion, pd.FileMetadata.Architecture))),
			},
			Format: primaryFormat{
				License:   pd.VersionMetadata.License,
				Vendor:    pd.FileMetadata.Vendor,
				Group:     pd.FileMetadata.Group,
				Buildhost: pd.FileMetadata.BuildHost,
				Sourcerpm: pd.FileMetadata.SourceRpm,
				Provides: primaryEntryList{
					Entries: pd.FileMetadata.Provides,
				},
				Requires: primaryEntryList{
					Entries: pd.FileMetadata.Requires,
				},
				Conflicts: primaryEntryList{
					Entries: pd.FileMetadata.Conflicts,
				},
				Obsoletes: primaryEntryList{
					Entries: pd.FileMetadata.Obsoletes,
				},
				Files: files,
			},
		})
	}

	return l.addDataAsFileToRepo(ctx, "primary", &primaryMetadata{
		Xmlns:        "http://linux.duke.edu/metadata/common",
		XmlnsRpm:     "http://linux.duke.edu/metadata/rpm",
		PackageCount: len(pds),
		Packages:     packages,
	}, info)
}

func (l *localRegistryHelper) buildOther(
	ctx context.Context,
	pds []*packageInfo,
	info rpmtype.ArtifactInfo,
) (*repoData, error) {
	packages := make([]*otherPackage, 0, len(pds))
	for _, pd := range pds {
		packages = append(packages, &otherPackage{
			Pkgid:        pd.Sha256,
			Name:         pd.Name,
			Architecture: pd.FileMetadata.Architecture,
			Version: otherVersion{
				Epoch:   pd.FileMetadata.Epoch,
				Version: pd.FileMetadata.Version,
				Release: pd.FileMetadata.Release,
			},
			Changelogs: pd.FileMetadata.Changelogs,
		})
	}

	return l.addDataAsFileToRepo(ctx, "other", &otherdata{
		Xmlns:        "http://linux.duke.edu/metadata/other",
		PackageCount: len(pds),
		Packages:     packages,
	}, info)
}

func (l *localRegistryHelper) buildFilelists(
	ctx context.Context,
	pds []*packageInfo,
	info rpmtype.ArtifactInfo,
) (*repoData, error) { //nolint:dupl
	packages := make([]*fileListPackage, 0, len(pds))
	for _, pd := range pds {
		packages = append(packages, &fileListPackage{
			Pkgid:        pd.Sha256,
			Name:         pd.Name,
			Architecture: pd.FileMetadata.Architecture,
			Version: fileListVersion{
				Epoch:   pd.FileMetadata.Epoch,
				Version: pd.FileMetadata.Version,
				Release: pd.FileMetadata.Release,
			},
			Files: pd.FileMetadata.Files,
		})
	}

	return l.addDataAsFileToRepo(ctx, "filelists", &filelists{
		Xmlns:        "http://linux.duke.edu/metadata/other",
		PackageCount: len(pds),
		Packages:     packages,
	}, info)
}

func (l *localRegistryHelper) buildRepomd(
	ctx context.Context,
	data []*repoData,
	info rpmtype.ArtifactInfo,
) error {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	if err := xml.NewEncoder(&buf).Encode(&repomd{
		Xmlns:    "http://linux.duke.edu/metadata/repo",
		XmlnsRpm: "http://linux.duke.edu/metadata/rpm",
		Data:     data,
	}); err != nil {
		return err
	}
	repomdContent, _ := CreateHashedBufferFromReader(&buf)
	defer repomdContent.Close()

	_, err := l.fileManager.UploadFile(ctx, RepoDataPrefix+RepoMdFile, info.RegIdentifier, info.RegistryID,
		info.RootParentID, info.RootIdentifier, repomdContent, repomdContent, RepoMdFile)
	if err != nil {
		return err
	}
	return nil
}

func (l *localRegistryHelper) addDataAsFileToRepo(
	ctx context.Context,
	filetype string,
	obj any,
	info rpmtype.ArtifactInfo,
) (*repoData, error) {
	content, _ := NewHashedBuffer()
	defer content.Close()

	gzw := gzip.NewWriter(content)
	wc := &writtenCounter{}
	h := sha256.New()

	w := io.MultiWriter(gzw, wc, h)
	_, _ = w.Write([]byte(xml.Header))

	if err := xml.NewEncoder(w).Encode(obj); err != nil {
		return nil, err
	}

	if err := gzw.Close(); err != nil {
		return nil, err
	}

	filename := filetype + ".xml.gz"
	_, err := l.fileManager.UploadFile(ctx, RepoDataPrefix+filename, info.RegIdentifier, info.RegistryID,
		info.RootParentID, info.RootIdentifier, content, content, filename)
	if err != nil {
		return nil, err
	}
	_, _, hashSHA256, _ := content.Sums()

	return &repoData{
		Type: filetype,
		Checksum: repoChecksum{
			Type:  "sha256",
			Value: hex.EncodeToString(hashSHA256),
		},
		OpenChecksum: repoChecksum{
			Type:  "sha256",
			Value: hex.EncodeToString(h.Sum(nil)),
		},
		Location: repoLocation{
			Href: "repodata/" + filename,
		},
		Timestamp: time.Now().Unix(),
		Size:      content.Size(),
		OpenSize:  wc.Written(),
	}, nil
}
