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

package python

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/harness/gitness/registry/app/api/controller/pkg/python"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/api/handler/utils"
	python2 "github.com/harness/gitness/registry/app/metadata/python"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"
	"github.com/harness/gitness/registry/validation"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// https://peps.python.org/pep-0426/#name
var (
	normalizer  = strings.NewReplacer(".", "-", "_", "-")
	nameMatcher = regexp.MustCompile(`\A(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\.\-_]*[a-zA-Z0-9])\z`)
)

// https://peps.python.org/pep-0440/#appendix-b-parsing-version-strings-with-regular-expressions
var versionMatcher = regexp.MustCompile(`\Av?` +
	`(?:[0-9]+!)?` + // epoch
	`[0-9]+(?:\.[0-9]+)*` + // release segment
	`(?:[-_\.]?(?:a|b|c|rc|alpha|beta|pre|preview)[-_\.]?[0-9]*)?` + // pre-release
	`(?:-[0-9]+|[-_\.]?(?:post|rev|r)[-_\.]?[0-9]*)?` + // post release
	`(?:[-_\.]?dev[-_\.]?[0-9]*)?` + // dev release
	`(?:\+[a-z0-9]+(?:[-_\.][a-z0-9]+)*)?` + // local version
	`\z`)

type Handler interface {
	pkg.ArtifactInfoProvider
	UploadPackageFile(writer http.ResponseWriter, request *http.Request)
	DownloadPackageFile(http.ResponseWriter, *http.Request)
	PackageMetadata(writer http.ResponseWriter, request *http.Request)
}

type handler struct {
	packages.Handler
	controller python.Controller
}

func NewHandler(
	controller python.Controller,
	packageHandler packages.Handler,
) Handler {
	return &handler{
		Handler:    packageHandler,
		controller: controller,
	}
}

var _ Handler = (*handler)(nil)

func (h *handler) GetPackageArtifactInfo(r *http.Request) (pkg.PackageArtifactInfo, error) {
	info, err := h.Handler.GetArtifactInfo(r)
	if !commons.IsEmptyError(err) {
		return nil, err
	}

	image := chi.URLParam(r, "image")
	filename := chi.URLParam(r, "filename")
	version := chi.URLParam(r, "version")

	var md python2.Metadata
	err2 := utils.FillFromForm(r, &md)

	if err2 == nil {
		if image == "" {
			image = md.Name
		}
		if version == "" {
			version = md.Version
		}
	}

	image = normalizer.Replace(image)
	if image != "" && version != "" && !isValidNameAndVersion(image, version) {
		log.Info().Msgf("Invalid image name/version: %s/%s", info.Image, version)
		return nil, fmt.Errorf("invalid name or version")
	}

	md.HomePage = getHomePage(md)
	info.Image = image

	return &pythontype.ArtifactInfo{
		ArtifactInfo: info,
		Metadata:     md,
		Filename:     filename,
		Version:      version,
	}, nil
}

func getHomePage(md python2.Metadata) string {
	var homepageURL string
	if len(md.ProjectURLs) > 0 {
		for k, v := range md.ProjectURLs {
			if normalizeLabel(k) != "homepage" {
				continue
			}
			homepageURL = strings.TrimSpace(v)
			break
		}
	}

	if len(homepageURL) == 0 {
		homepageURL = md.HomePage
	}

	if !validation.IsValidURL(homepageURL) {
		homepageURL = ""
	}
	return homepageURL
}

func isValidNameAndVersion(image, version string) bool {
	return nameMatcher.MatchString(image) && versionMatcher.MatchString(version)
}

// Normalizes a Project-URL label.
// See https://packaging.python.org/en/latest/specifications/well-known-project-urls/#label-normalization.
func normalizeLabel(label string) string {
	var builder strings.Builder

	// "A label is normalized by deleting all ASCII punctuation and whitespace, and then converting the result
	// to lowercase."
	for _, r := range label {
		if unicode.IsPunct(r) || unicode.IsSpace(r) {
			continue
		}
		builder.WriteRune(unicode.ToLower(r))
	}

	return builder.String()
}
