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

//nolint:gosec
import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"hash"
	"io"
	"math"
	"net/url"
	"os"
	"strings"
	"time"

	rpmmetadata "github.com/harness/gitness/registry/app/metadata/rpm"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/validation"

	"github.com/sassoftware/go-rpmutils"
)

const (
	sIFMT  = 0xf000
	sIFDIR = 0x4000
	sIXUSR = 0x40
	sIXGRP = 0x8
	sIXOTH = 0x1

	sizeMD5    = 92
	sizeSHA1   = 96
	sizeSHA256 = 108
	sizeSHA512 = 204
	size       = sizeMD5 + sizeSHA1 + sizeSHA256 + sizeSHA512

	RepoMdFile     = "repomd.xml"
	RepoDataPrefix = "repodata/"

	DefaultMemorySize  = 32 * 1024 * 1024
	artifactBatchLimit = 50
)

var (
	ErrInvalidMemorySize = errors.New("memory size must be greater 0 and lower math.MaxInt32")
	ErrWriteAfterRead    = errors.New("write is unsupported after a read operation")
)

type RegistryHelper interface {
	BuildRegistryFiles(ctx context.Context, registryID int64, rootParentID int64, rootIdentifier string) error
}

type registryHelper struct {
	fileManager filemanager.FileManager
	artifactDao store.ArtifactRepository
}

func NewRegistryHelper(
	fileManager filemanager.FileManager,
	artifactDao store.ArtifactRepository,
) RegistryHelper {
	return &registryHelper{
		fileManager: fileManager,
		artifactDao: artifactDao,
	}
}

func (l *registryHelper) BuildRegistryFiles(
	ctx context.Context,
	registryID int64,
	rootParentID int64,
	rootIdentifier string,
) error {
	lastArtifactID := int64(0)
	var packageInfos []*packageInfo

	for {
		artifacts, err := l.artifactDao.GetAllArtifactsByRepo(ctx, registryID, artifactBatchLimit, lastArtifactID)
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

	primary, err := l.buildPrimary(ctx, packageInfos, registryID, rootParentID, rootIdentifier)
	if err != nil {
		return err
	}
	fileLists, err := l.buildFileLists(ctx, packageInfos, registryID, rootParentID, rootIdentifier)
	if err != nil {
		return err
	}
	other, err := l.buildOther(ctx, packageInfos, registryID, rootParentID, rootIdentifier)
	if err != nil {
		return err
	}
	return l.buildRepomd(ctx, []*repoData{
		primary,
		fileLists,
		other,
	}, registryID, rootParentID, rootIdentifier)
}

func (l *registryHelper) buildPrimary(
	ctx context.Context,
	pds []*packageInfo,
	registryID int64,
	rootParentID int64,
	rootIdentifier string,
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

	primaryData := &primaryMetadata{
		Xmlns:        "http://linux.duke.edu/metadata/common",
		XmlnsRpm:     "http://linux.duke.edu/metadata/rpm",
		PackageCount: len(pds),
		Packages:     packages,
	}
	return l.addDataAsFileToRepo(ctx, "primary", primaryData, registryID, rootParentID, rootIdentifier)
}

func (l *registryHelper) buildOther(
	ctx context.Context,
	pds []*packageInfo,
	registryID int64,
	rootParentID int64,
	rootIdentifier string,
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
	otherData := &otherdata{
		Xmlns:        "http://linux.duke.edu/metadata/other",
		PackageCount: len(pds),
		Packages:     packages,
	}

	return l.addDataAsFileToRepo(ctx, "other", otherData, registryID, rootParentID, rootIdentifier)
}

func (l *registryHelper) buildFileLists(
	ctx context.Context,
	pds []*packageInfo,
	registryID int64,
	rootParentID int64,
	rootIdentifier string,
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

	fileLists := &filelists{
		Xmlns:        "http://linux.duke.edu/metadata/other",
		PackageCount: len(pds),
		Packages:     packages,
	}

	return l.addDataAsFileToRepo(ctx, "filelists", fileLists, registryID, rootParentID, rootIdentifier)
}

func (l *registryHelper) buildRepomd(
	ctx context.Context,
	data []*repoData,
	registryID int64,
	rootParentID int64,
	rootIdentifier string,
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

	_, err := l.fileManager.UploadFile(ctx, RepoDataPrefix+RepoMdFile, registryID,
		rootParentID, rootIdentifier, repomdContent, repomdContent, RepoMdFile)
	if err != nil {
		return err
	}
	return nil
}

func (l *registryHelper) addDataAsFileToRepo(
	ctx context.Context,
	filetype string,
	obj any,
	registryID int64,
	rootParentID int64,
	rootIdentifier string,
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
	_, err := l.fileManager.UploadFile(ctx, RepoDataPrefix+filename, registryID,
		rootParentID, rootIdentifier, content, content, filename)
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

func ParsePackage(r io.Reader) (*Package, error) {
	rpm, err := rpmutils.ReadRpm(r)
	if err != nil {
		return nil, err
	}

	nevra, err := rpm.Header.GetNEVRA()
	if err != nil {
		return nil, err
	}

	version := fmt.Sprintf("%s-%s", nevra.Version, nevra.Release)
	if nevra.Epoch != "" && nevra.Epoch != "0" {
		version = fmt.Sprintf("%s-%s", nevra.Epoch, version)
	}

	p := &Package{
		Name:    nevra.Name,
		Version: version,
		VersionMetadata: &rpmmetadata.VersionMetadata{
			Summary:     getString(rpm.Header, rpmutils.SUMMARY),
			Description: getString(rpm.Header, rpmutils.DESCRIPTION),
			License:     getString(rpm.Header, rpmutils.LICENSE),
			ProjectURL:  getString(rpm.Header, rpmutils.URL),
		},
		FileMetadata: &rpmmetadata.FileMetadata{
			Architecture:  nevra.Arch,
			Epoch:         nevra.Epoch,
			Version:       nevra.Version,
			Release:       nevra.Release,
			Vendor:        getString(rpm.Header, rpmutils.VENDOR),
			Group:         getString(rpm.Header, rpmutils.GROUP),
			Packager:      getString(rpm.Header, rpmutils.PACKAGER),
			SourceRpm:     getString(rpm.Header, rpmutils.SOURCERPM),
			BuildHost:     getString(rpm.Header, rpmutils.BUILDHOST),
			BuildTime:     getUInt64(rpm.Header, rpmutils.BUILDTIME),
			FileTime:      getUInt64(rpm.Header, rpmutils.FILEMTIMES),
			InstalledSize: getUInt64(rpm.Header, rpmutils.SIZE),
			ArchiveSize:   getUInt64(rpm.Header, rpmutils.SIG_PAYLOADSIZE),

			Provides:   getEntries(rpm.Header, rpmutils.PROVIDENAME, rpmutils.PROVIDEVERSION, rpmutils.PROVIDEFLAGS),
			Requires:   getEntries(rpm.Header, rpmutils.REQUIRENAME, rpmutils.REQUIREVERSION, rpmutils.REQUIREFLAGS),
			Conflicts:  getEntries(rpm.Header, rpmutils.CONFLICTNAME, rpmutils.CONFLICTVERSION, rpmutils.CONFLICTFLAGS),
			Obsoletes:  getEntries(rpm.Header, rpmutils.OBSOLETENAME, rpmutils.OBSOLETEVERSION, rpmutils.OBSOLETEFLAGS),
			Files:      getFiles(rpm.Header),
			Changelogs: getChangelogs(rpm.Header),
		},
	}

	if !validation.IsValidURL(p.VersionMetadata.ProjectURL) {
		p.VersionMetadata.ProjectURL = ""
	}

	return p, nil
}

func getString(h *rpmutils.RpmHeader, tag int) string {
	values, err := h.GetStrings(tag)
	if err != nil || len(values) < 1 {
		return ""
	}
	return values[0]
}

func getUInt64(h *rpmutils.RpmHeader, tag int) uint64 {
	values, err := h.GetUint64s(tag)
	if err != nil || len(values) < 1 {
		return 0
	}
	return values[0]
}

// nolint: gocritic
func getEntries(h *rpmutils.RpmHeader, namesTag, versionsTag, flagsTag int) []*rpmmetadata.Entry {
	names, err := h.GetStrings(namesTag)
	if err != nil || len(names) == 0 {
		return nil
	}
	flags, err := h.GetUint64s(flagsTag)
	if err != nil || len(flags) == 0 {
		return nil
	}
	versions, err := h.GetStrings(versionsTag)
	if err != nil || len(versions) == 0 {
		return nil
	}
	if len(names) != len(flags) || len(names) != len(versions) {
		return nil
	}

	entries := make([]*rpmmetadata.Entry, 0, len(names))
	for i := range names {
		e := &rpmmetadata.Entry{
			Name: names[i],
		}

		flags := flags[i]
		if (flags&rpmutils.RPMSENSE_GREATER) != 0 && (flags&rpmutils.RPMSENSE_EQUAL) != 0 {
			e.Flags = "GE"
		} else if (flags&rpmutils.RPMSENSE_LESS) != 0 && (flags&rpmutils.RPMSENSE_EQUAL) != 0 {
			e.Flags = "LE"
		} else if (flags & rpmutils.RPMSENSE_GREATER) != 0 {
			e.Flags = "GT"
		} else if (flags & rpmutils.RPMSENSE_LESS) != 0 {
			e.Flags = "LT"
		} else if (flags & rpmutils.RPMSENSE_EQUAL) != 0 {
			e.Flags = "EQ"
		}

		version := versions[i]
		if version != "" {
			parts := strings.Split(version, "-")

			versionParts := strings.Split(parts[0], ":")
			if len(versionParts) == 2 {
				e.Version = versionParts[1]
				e.Epoch = versionParts[0]
			} else {
				e.Version = versionParts[0]
				e.Epoch = "0"
			}

			if len(parts) > 1 {
				e.Release = parts[1]
			}
		}

		entries = append(entries, e)
	}
	return entries
}

func getFiles(h *rpmutils.RpmHeader) []*rpmmetadata.File {
	baseNames, _ := h.GetStrings(rpmutils.BASENAMES)
	dirNames, _ := h.GetStrings(rpmutils.DIRNAMES)
	dirIndexes, _ := h.GetUint32s(rpmutils.DIRINDEXES)
	fileFlags, _ := h.GetUint32s(rpmutils.FILEFLAGS)
	fileModes, _ := h.GetUint32s(rpmutils.FILEMODES)

	files := make([]*rpmmetadata.File, 0, len(baseNames))
	for i := range baseNames {
		if len(dirIndexes) <= i {
			continue
		}
		dirIndex := dirIndexes[i]
		if len(dirNames) <= int(dirIndex) {
			continue
		}

		var fileType string
		var isExecutable bool
		if i < len(fileFlags) && (fileFlags[i]&rpmutils.RPMFILE_GHOST) != 0 {
			fileType = "ghost"
		} else if i < len(fileModes) {
			if (fileModes[i] & sIFMT) == sIFDIR {
				fileType = "dir"
			} else {
				mode := fileModes[i] & ^uint32(sIFMT)
				isExecutable = (mode&sIXUSR) != 0 || (mode&sIXGRP) != 0 || (mode&sIXOTH) != 0
			}
		}

		files = append(files, &rpmmetadata.File{
			Path:         dirNames[dirIndex] + baseNames[i],
			Type:         fileType,
			IsExecutable: isExecutable,
		})
	}

	return files
}

func getChangelogs(h *rpmutils.RpmHeader) []*rpmmetadata.Changelog {
	texts, err := h.GetStrings(rpmutils.CHANGELOGTEXT)
	if err != nil || len(texts) == 0 {
		return nil
	}
	authors, err := h.GetStrings(rpmutils.CHANGELOGNAME)
	if err != nil || len(authors) == 0 {
		return nil
	}
	times, err := h.GetUint32s(rpmutils.CHANGELOGTIME)
	if err != nil || len(times) == 0 {
		return nil
	}
	if len(texts) != len(authors) || len(texts) != len(times) {
		return nil
	}

	changelogs := make([]*rpmmetadata.Changelog, 0, len(texts))
	for i := range texts {
		changelogs = append(changelogs, &rpmmetadata.Changelog{
			Author: authors[i],
			Date:   int64(times[i]),
			Text:   texts[i],
		})
	}
	return changelogs
}

type writtenCounter struct {
	written int64
}

func (wc *writtenCounter) Write(buf []byte) (int, error) {
	n := len(buf)

	wc.written += int64(n)

	return n, nil
}

func (wc *writtenCounter) Written() int64 {
	return wc.written
}

type readAtSeeker interface {
	io.ReadSeeker
	io.ReaderAt
}

type FileBackedBuffer struct {
	maxMemorySize int64
	size          int64
	buffer        bytes.Buffer
	file          *os.File
	reader        readAtSeeker
}

func NewFileBackedBuffer(maxMemorySize int) (*FileBackedBuffer, error) {
	if maxMemorySize < 0 || maxMemorySize > math.MaxInt32 {
		return nil, ErrInvalidMemorySize
	}

	return &FileBackedBuffer{
		maxMemorySize: int64(maxMemorySize),
	}, nil
}

//nolint:nestif
func (b *FileBackedBuffer) Write(p []byte) (int, error) {
	if b.reader != nil {
		return 0, ErrWriteAfterRead
	}

	var n int
	var err error

	if b.file != nil {
		n, err = b.file.Write(p)
	} else {
		if b.size+int64(len(p)) > b.maxMemorySize {
			b.file, err = os.CreateTemp("", "gitness-buffer-")
			if err != nil {
				return 0, err
			}

			_, err = io.Copy(b.file, &b.buffer)
			if err != nil {
				return 0, err
			}

			return b.Write(p)
		}

		n, err = b.buffer.Write(p)
	}

	if err != nil {
		return n, err
	}
	b.size += int64(n)
	return n, nil
}

func (b *FileBackedBuffer) Size() int64 {
	return b.size
}

func (b *FileBackedBuffer) switchToReader() error {
	if b.reader != nil {
		return nil
	}

	if b.file != nil {
		if _, err := b.file.Seek(0, io.SeekStart); err != nil {
			return err
		}
		b.reader = b.file
	} else {
		b.reader = bytes.NewReader(b.buffer.Bytes())
	}
	return nil
}

func (b *FileBackedBuffer) Read(p []byte) (int, error) {
	if err := b.switchToReader(); err != nil {
		return 0, err
	}

	return b.reader.Read(p)
}

func (b *FileBackedBuffer) ReadAt(p []byte, off int64) (int, error) {
	if err := b.switchToReader(); err != nil {
		return 0, err
	}

	return b.reader.ReadAt(p, off)
}

func (b *FileBackedBuffer) Seek(offset int64, whence int) (int64, error) {
	if err := b.switchToReader(); err != nil {
		return 0, err
	}

	return b.reader.Seek(offset, whence)
}

func (b *FileBackedBuffer) Close() error {
	if b.file != nil {
		err := b.file.Close()
		os.Remove(b.file.Name())
		b.file = nil
		return err
	}
	return nil
}

type HashedBuffer struct {
	*FileBackedBuffer
	hash           *MultiHasher
	combinedWriter io.Writer
}

func NewHashedBuffer() (*HashedBuffer, error) {
	return NewHashedBufferWithSize(DefaultMemorySize)
}

func NewHashedBufferWithSize(maxMemorySize int) (*HashedBuffer, error) {
	b, err := NewFileBackedBuffer(maxMemorySize)
	if err != nil {
		return nil, err
	}

	hash := NewMultiHasher()

	combinedWriter := io.MultiWriter(b, hash)

	return &HashedBuffer{
		b,
		hash,
		combinedWriter,
	}, nil
}

func CreateHashedBufferFromReader(r io.Reader) (*HashedBuffer, error) {
	return CreateHashedBufferFromReaderWithSize(r, DefaultMemorySize)
}

func CreateHashedBufferFromReaderWithSize(r io.Reader, maxMemorySize int) (*HashedBuffer, error) {
	b, err := NewHashedBufferWithSize(maxMemorySize)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(b, r)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (b *HashedBuffer) Write(p []byte) (int, error) {
	return b.combinedWriter.Write(p)
}

func (b *HashedBuffer) Sums() (hashMD5, hashSHA1, hashSHA256, hashSHA512 []byte) {
	return b.hash.Sums()
}

type MultiHasher struct {
	md5    hash.Hash
	sha1   hash.Hash
	sha256 hash.Hash
	sha512 hash.Hash

	combinedWriter io.Writer
}

//nolint:gosec
func NewMultiHasher() *MultiHasher {
	md5 := md5.New()
	sha1 := sha1.New()
	sha256 := sha256.New()
	sha512 := sha512.New()

	combinedWriter := io.MultiWriter(md5, sha1, sha256, sha512)

	return &MultiHasher{
		md5,
		sha1,
		sha256,
		sha512,
		combinedWriter,
	}
}

// nolint:errcheck
func (h *MultiHasher) MarshalBinary() ([]byte, error) {
	md5Bytes, err := h.md5.(encoding.BinaryMarshaler).MarshalBinary()
	if err != nil {
		return nil, err
	}
	sha1Bytes, err := h.sha1.(encoding.BinaryMarshaler).MarshalBinary()
	if err != nil {
		return nil, err
	}
	sha256Bytes, err := h.sha256.(encoding.BinaryMarshaler).MarshalBinary()
	if err != nil {
		return nil, err
	}
	sha512Bytes, err := h.sha512.(encoding.BinaryMarshaler).MarshalBinary()
	if err != nil {
		return nil, err
	}

	b := make([]byte, 0, size)
	b = append(b, md5Bytes...)
	b = append(b, sha1Bytes...)
	b = append(b, sha256Bytes...)
	b = append(b, sha512Bytes...)
	return b, nil
}

func (h *MultiHasher) Write(p []byte) (int, error) {
	return h.combinedWriter.Write(p)
}

func (h *MultiHasher) Sums() (hashMD5, hashSHA1, hashSHA256, hashSHA512 []byte) {
	hashMD5 = h.md5.Sum(nil)
	hashSHA1 = h.sha1.Sum(nil)
	hashSHA256 = h.sha256.Sum(nil)
	hashSHA512 = h.sha512.Sum(nil)
	return hashMD5, hashSHA1, hashSHA256, hashSHA512
}
