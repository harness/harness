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

package types

//nolint:gosec
import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding"
	"encoding/xml"
	"errors"
	"hash"
	"io"
	"math"
	"os"

	rpmmetadata "github.com/harness/gitness/registry/app/metadata/rpm"
)

const (
	sizeMD5    = 92
	sizeSHA1   = 96
	sizeSHA256 = 108
	sizeSHA512 = 204
	size       = sizeMD5 + sizeSHA1 + sizeSHA256 + sizeSHA512

	DefaultMemorySize = 32 * 1024 * 1024
)

var (
	ErrInvalidMemorySize = errors.New("memory size must be greater 0 and lower math.MaxInt32")
	ErrWriteAfterRead    = errors.New("write is unsupported after a read operation")
)

type PrimaryVersion struct {
	Epoch   string `xml:"epoch,attr"`
	Version string `xml:"ver,attr"`
	Release string `xml:"rel,attr"`
}

type PrimaryChecksum struct {
	Checksum string `xml:",chardata"` //nolint: tagliatelle
	Type     string `xml:"type,attr"`
	Pkgid    string `xml:"pkgid,attr"`
}

type PrimaryTimes struct {
	File  uint64 `xml:"file,attr"`
	Build uint64 `xml:"build,attr"`
}

type PrimarySizes struct {
	Package   int64  `xml:"package,attr"`
	Installed uint64 `xml:"installed,attr"`
	Archive   uint64 `xml:"archive,attr"`
}

type PrimaryLocation struct {
	Href string `xml:"href,attr"`
}

type PrimaryEntryList struct {
	Entries []*rpmmetadata.Entry `xml:"entry"`
}

// MarshalXML implements custom XML marshaling for primaryEntryList to add rpm prefix.
func (l PrimaryEntryList) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(struct {
		Entries []*rpmmetadata.Entry `xml:"rpm:entry"`
	}{
		Entries: l.Entries,
	}, start)
}

type PrimaryFormat struct {
	License   string              `xml:"license"`
	Vendor    string              `xml:"vendor"`
	Group     string              `xml:"group"`
	Buildhost string              `xml:"buildhost"`
	Sourcerpm string              `xml:"sourcerpm"`
	Provides  PrimaryEntryList    `xml:"provides"`
	Requires  PrimaryEntryList    `xml:"requires"`
	Conflicts PrimaryEntryList    `xml:"conflicts"`
	Obsoletes PrimaryEntryList    `xml:"obsoletes"`
	Files     []*rpmmetadata.File `xml:"file"`
}

// MarshalXML implements custom XML marshaling for primaryFormat to add rpm prefix.
func (f PrimaryFormat) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(struct {
		License   string              `xml:"rpm:license"`
		Vendor    string              `xml:"rpm:vendor"`
		Group     string              `xml:"rpm:group"`
		Buildhost string              `xml:"rpm:buildhost"`
		Sourcerpm string              `xml:"rpm:sourcerpm"`
		Provides  PrimaryEntryList    `xml:"rpm:provides"`
		Requires  PrimaryEntryList    `xml:"rpm:requires"`
		Conflicts PrimaryEntryList    `xml:"rpm:conflicts"`
		Obsoletes PrimaryEntryList    `xml:"rpm:obsoletes"`
		Files     []*rpmmetadata.File `xml:"file"`
	}{
		License:   f.License,
		Vendor:    f.Vendor,
		Group:     f.Group,
		Buildhost: f.Buildhost,
		Sourcerpm: f.Sourcerpm,
		Provides:  f.Provides,
		Requires:  f.Requires,
		Conflicts: f.Conflicts,
		Obsoletes: f.Obsoletes,
		Files:     f.Files,
	}, start)
}

type PrimaryPackage struct {
	XMLName      xml.Name        `xml:"package"`
	Type         string          `xml:"type,attr"`
	Name         string          `xml:"name"`
	Architecture string          `xml:"arch"`
	Version      PrimaryVersion  `xml:"version"`
	Checksum     PrimaryChecksum `xml:"checksum"`
	Summary      string          `xml:"summary"`
	Description  string          `xml:"description"`
	Packager     string          `xml:"packager"`
	URL          string          `xml:"url"`
	Time         PrimaryTimes    `xml:"time"`
	Size         PrimarySizes    `xml:"size"`
	Location     PrimaryLocation `xml:"location"`
	Format       PrimaryFormat   `xml:"format"`
}

type OtherVersion struct {
	Epoch   string `xml:"epoch,attr"`
	Version string `xml:"ver,attr"`
	Release string `xml:"rel,attr"`
}

type OtherPackage struct {
	XMLName      xml.Name                 `xml:"package"`
	Pkgid        string                   `xml:"pkgid,attr"`
	Name         string                   `xml:"name,attr"`
	Architecture string                   `xml:"arch,attr"`
	Version      OtherVersion             `xml:"version"`
	Changelogs   []*rpmmetadata.Changelog `xml:"changelog"`
}

type FileListVersion struct {
	Epoch   string `xml:"epoch,attr"`
	Version string `xml:"ver,attr"`
	Release string `xml:"rel,attr"`
}

type FileListPackage struct {
	XMLName      xml.Name            `xml:"package"`
	Pkgid        string              `xml:"pkgid,attr"`
	Name         string              `xml:"name,attr"`
	Architecture string              `xml:"arch,attr"`
	Version      FileListVersion     `xml:"version"`
	Files        []*rpmmetadata.File `xml:"file"`
}

type Repomd struct {
	XMLName  xml.Name    `xml:"repomd"`
	Xmlns    string      `xml:"xmlns,attr"`
	XmlnsRpm string      `xml:"xmlns:rpm,attr"`
	Data     []*RepoData `xml:"data"`
}

type RepoChecksum struct {
	Value string `xml:",chardata"` //nolint: tagliatelle
	Type  string `xml:"type,attr"`
}

type RepoLocation struct {
	Href string `xml:"href,attr"`
}

type RepoData struct {
	Type         string       `xml:"type,attr"`
	Checksum     RepoChecksum `xml:"checksum"`
	OpenChecksum RepoChecksum `xml:"open-checksum"` //nolint: tagliatelle
	Location     RepoLocation `xml:"location"`
	Timestamp    int64        `xml:"timestamp"`
	Size         int64        `xml:"size"`
	OpenSize     int64        `xml:"open-size"` //nolint: tagliatelle
}

type PackageInfo struct {
	Name            string
	Sha256          string
	Size            int64
	VersionMetadata *rpmmetadata.VersionMetadata
	FileMetadata    *rpmmetadata.FileMetadata
}

type Package struct {
	Name            string
	Version         string
	VersionMetadata *rpmmetadata.VersionMetadata
	FileMetadata    *rpmmetadata.FileMetadata
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
