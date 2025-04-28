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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	npm3 "github.com/harness/gitness/registry/app/api/controller/pkg/npm"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	npm2 "github.com/harness/gitness/registry/app/metadata/npm"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/types/npm"

	"github.com/rs/zerolog/log"
)

var (
	ErrInvalidPackageVersion = errors.New("package version is invalid")
	ErrInvalidAttachment     = errors.New("package attachment is invalid")
	packageNameRegex         = regexp.MustCompile(`^(?:@[\w.-]+\/)?[\w.-]+$`)
	versionRegex             = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([\w.-]+))?(?:\+([\w.-]+))?$`)
)

type Handler interface {
	pkg.ArtifactInfoProvider
	UploadPackage(writer http.ResponseWriter, request *http.Request)
	DownloadPackageFile(http.ResponseWriter, *http.Request)
	GetPackageMetadata(http.ResponseWriter, *http.Request)
	DownloadPackageFileByName(http.ResponseWriter, *http.Request)
	HeadPackageFileByName(http.ResponseWriter, *http.Request)

	ListPackageTag(http.ResponseWriter, *http.Request)
	AddPackageTag(http.ResponseWriter, *http.Request)
	DeletePackageTag(http.ResponseWriter, *http.Request)
	DeletePackage(w http.ResponseWriter, r *http.Request)
	DeleteVersion(w http.ResponseWriter, r *http.Request)
	DeletePreview(w http.ResponseWriter, r *http.Request)
	SearchPackage(w http.ResponseWriter, r *http.Request)
}

type handler struct {
	packages.Handler
	controller npm3.Controller
}

func NewHandler(
	controller npm3.Controller,
	packageHandler packages.Handler,
) Handler {
	return &handler{
		Handler:    packageHandler,
		controller: controller,
	}
}

var _ Handler = (*handler)(nil)

func (h *handler) GetPackageArtifactInfo(r *http.Request) (pkg.PackageArtifactInfo, error) {
	info, e := h.GetArtifactInfo(r)

	if !commons.IsEmpty(e) {
		return npm.ArtifactInfo{}, e
	}

	info.Image = PackageNameFromParams(r)
	version := GetVersionFromParams(r)
	fileName := GetFileName(r)

	if info.Image != "" && version != "" && !isValidNameAndVersion(info.Image, version) {
		log.Info().Msgf("Invalid image name/version: %s/%s", info.Image, version)
		return nil, fmt.Errorf("invalid name or version")
	}
	distTags := r.PathValue("tag")

	npmInfo := npm.ArtifactInfo{
		ArtifactInfo:        info,
		Filename:            fileName,
		Version:             version,
		ParentRegIdentifier: info.RegIdentifier,
		DistTags:            []string{distTags},
	}

	if r.Body == nil || r.ContentLength == 0 {
		return npmInfo, nil
	}

	if strings.Contains(r.URL.Path, "/-rev/") {
		return npmInfo, nil
	}

	if strings.Contains(r.URL.Path, "/-/package/") && strings.Contains(r.URL.Path, "/dist-tags/") {
		// Process the payload only for add tag requests
		if r.Body == nil || r.ContentLength == 0 {
			return npmInfo, nil
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			return npm.ArtifactInfo{}, err
		}
		npmInfo.Version = strings.Trim(string(body), "\"")
		npmInfo.DistTags = []string{r.PathValue("tag")}
		return npmInfo, err
	}

	return GetNPMMetadata(r, info)
}

func GetNPMMetadata(r *http.Request, info pkg.ArtifactInfo) (pkg.PackageArtifactInfo, error) {
	var md npm2.PackageUpload

	// Read body into a buffer
	var buf bytes.Buffer
	tee := io.TeeReader(r.Body, &buf)
	if err := json.NewDecoder(tee).Decode(&md); err != nil {
		return npm.ArtifactInfo{}, err
	}

	r.Body = io.NopCloser(&buf)

	for _, meta := range md.Versions {
		a := npm.ArtifactInfo{
			ArtifactInfo:        info,
			Metadata:            md.PackageMetadata,
			Version:             meta.Version,
			DistTags:            make([]string, 0),
			ParentRegIdentifier: info.RegIdentifier,
		}
		for tag := range md.DistTags {
			a.DistTags = append(a.DistTags, tag)
		}
		a.Filename = strings.ToLower(fmt.Sprintf("%s-%s.tgz", md.Name, a.Version))
		return a, nil
	}
	return npm.ArtifactInfo{}, ErrInvalidPackageVersion
}

func GetNPMFile(r *http.Request) (io.ReadCloser, error) {
	var md npm2.PackageUpload
	if err := json.NewDecoder(r.Body).Decode(&md); err != nil {
		return nil, err
	}
	attachment := func() *npm2.PackageAttachment {
		for _, a := range md.Attachments {
			return a
		}
		return nil
	}()
	if attachment == nil || len(attachment.Data) == 0 {
		return nil, ErrInvalidAttachment
	}
	return io.NopCloser(base64.NewDecoder(base64.StdEncoding,
		strings.NewReader(attachment.Data))), nil
}

func isValidNameAndVersion(image, version string) bool {
	return packageNameRegex.MatchString(image) && versionRegex.MatchString(version)
}
