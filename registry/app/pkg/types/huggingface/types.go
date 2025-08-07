// Copyright 2023 Harness, Inc.
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

package huggingface

import (
	huggingfacemetadata "github.com/harness/gitness/registry/app/metadata/huggingface"
	"github.com/harness/gitness/registry/app/pkg"
)

// ArtifactInfo represents information about a Huggingface artifact.
type ArtifactInfo struct {
	pkg.ArtifactInfo
	Repo     string
	Revision string
	RepoType string
	SHA256   string
}

// BaseArtifactInfo implements pkg.PackageArtifactInfo interface.
func (a ArtifactInfo) BaseArtifactInfo() pkg.ArtifactInfo {
	return a.ArtifactInfo
}

// GetImageVersion returns the image and version information.
func (a ArtifactInfo) GetImageVersion() (exists bool, imageVersion string) {
	if a.Repo != "" && a.Revision != "" {
		return true, pkg.JoinWithSeparator(":", a.Repo, a.Revision)
	}
	return false, ""
}

// GetVersion returns the version (revision) of the Huggingface artifact.
func (a ArtifactInfo) GetVersion() string {
	return a.Revision
}

func (a ArtifactInfo) GetFileName() string {
	return a.SHA256
}

// File represents a file in a Huggingface package.
type File struct {
	FileURL  string
	Name     string
	RepoType string
}

// PackageMetadata represents metadata for a Huggingface package.
type PackageMetadata struct {
	RepoID   string
	RepoType string
	Files    []File
}

type Debug struct {
	Message *string `json:"message"`
	Type    *string `json:"type"`
	Help    *string `json:"help"`
}

type ValidateYamlRequest struct {
	Content  *string `json:"content"`
	RepoType *string `json:"repoType"`
}

type ValidateYamlResponse struct {
	Errors   *[]Debug `json:"errors"`
	Warnings *[]Debug `json:"warnings"`
}

type PreUploadResponse struct {
	Files *[]PreUploadResponseFile `json:"files"`
}

type PreUploadRequest struct {
	Files *[]PreUploadRequestFile `json:"files"`
}

type PreUploadResponseFile struct {
	Path         string     `json:"path"`
	UploadMode   UploadMode `json:"uploadMode"`
	ShouldIgnore bool       `json:"shouldIgnore"` // NEW
}

type PreUploadRequestFile struct {
	Path   string  `json:"path"`
	Sample *string `json:"sample"`
	Size   int64   `json:"size"` // NEW
}

type RevisionInfoResponse struct {
	huggingfacemetadata.Metadata
	XetEnabled bool `json:"xetEnabled"`
}

type LfsInfoRequest struct {
	Operation string              `json:"operation"` // "upload", "download"
	Transfers []string            `json:"transfers"` // "basic", "multipart"
	Objects   *[]LfsObjectRequest `json:"objects"`
	HashAlgo  *string             `json:"hash_algo,omitempty"`
	Ref       *LfsRefRequest      `json:"ref,omitempty"`
}

type LfsObjectRequest struct {
	Oid  string `json:"oid"`
	Size int64  `json:"size"`
}

type LfsRefRequest struct {
	Name string `json:"name"`
}

type LfsInfoResponse struct {
	Objects *[]LfsObjectResponse `json:"objects"`
}

type LfsVerifyResponse struct {
	Success bool      `json:"success"`
	Error   *LfsError `json:"error,omitempty"`
}

type LfsUploadResponse struct {
	Success bool      `json:"success"`
	Error   *LfsError `json:"error,omitempty"`
}

type CommitRevisionResponse struct {
	CommitURL         string    `json:"commitUrl"`
	CommitMessage     string    `json:"commitMessage"`
	CommitDescription string    `json:"commitDescription"`
	OID               string    `json:"commitOid"`
	Success           bool      `json:"success"`
	Error             *LfsError `json:"error,omitempty"`
}

type CommitEntry struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"` // Value can be HeaderInfo or LFSFileInfo
}

type HeaderInfo struct {
	Summary     string `json:"summary"`
	Description string `json:"description"`
}

type LfsFileInfo struct {
	Path string `json:"path"`
	Algo string `json:"algo"`
	Oid  string `json:"oid"`
	Size int64  `json:"size"`
}

type LfsAction struct {
	Href      string            `json:"href"`
	Header    map[string]string `json:"header,omitempty"`
	ExpiresAt string            `json:"expires_at,omitempty"`
}

type LfsError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type LfsObjectResponse struct {
	Oid     string                `json:"oid"`
	Size    int64                 `json:"size"`
	Actions *map[string]LfsAction `json:"actions,omitempty"`
	Error   *LfsError             `json:"error,omitempty"`
}

// UploadMode represents the mode for uploading files
type UploadMode string

const (
	// Regular represents regular file upload mode
	RegularUpload UploadMode = "regular"
	// LFS represents Large File Storage upload mode
	LFSUpload UploadMode = "lfs"
)

// NewPreUploadResponseFile creates a new PreUploadResponseFile with default values.
func NewPreUploadResponseFile(path string) PreUploadResponseFile {
	return PreUploadResponseFile{
		Path:         path,
		UploadMode:   LFSUpload, // Default to LFS upload mode
		ShouldIgnore: false,
	}
}
