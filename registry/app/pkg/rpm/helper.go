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
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding"
	"errors"
	"fmt"
	"hash"
	"io"
	"math"
	"os"
	"strings"

	rpmmetadata "github.com/harness/gitness/registry/app/metadata/rpm"
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

	DefaultMemorySize = 32 * 1024 * 1024
)

var (
	ErrInvalidMemorySize = errors.New("memory size must be greater 0 and lower math.MaxInt32")
	ErrWriteAfterRead    = errors.New("write is unsupported after a read operation")
)

func parsePackage(r io.Reader) (*rpmPackage, error) {
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

	p := &rpmPackage{
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
