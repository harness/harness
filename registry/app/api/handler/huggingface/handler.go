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
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/harness/gitness/registry/app/api/controller/pkg/huggingface"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/pkg"
	hftype "github.com/harness/gitness/registry/app/pkg/types/huggingface"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Repo validation regex.
var (
	repoIDMatcher = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9\-/]{0,126}[a-zA-Z0-9])?$`)
	allowedTypes  = map[string]bool{"model": true, "dataset": true}
)

// Handler defines the interface for Huggingface package operations.
type Handler interface {
	pkg.ArtifactInfoProvider
	ValidateYAML(writer http.ResponseWriter, request *http.Request)
	LfsInfo(writer http.ResponseWriter, request *http.Request)
	LfsUpload(writer http.ResponseWriter, request *http.Request)
	LfsVerify(writer http.ResponseWriter, request *http.Request)
	PreUpload(writer http.ResponseWriter, request *http.Request)
	RevisionInfo(w http.ResponseWriter, r *http.Request)
	CommitRevision(writer http.ResponseWriter, request *http.Request)
	HeadFile(w http.ResponseWriter, r *http.Request)
	DownloadFile(w http.ResponseWriter, r *http.Request)
}

// handler implements the Handler interface.
type handler struct {
	packages.Handler
	controller huggingface.Controller
}

// NewHandler creates a new Huggingface handler.
func NewHandler(
	controller huggingface.Controller,
	packageHandler packages.Handler,
) Handler {
	return &handler{
		Handler:    packageHandler,
		controller: controller,
	}
}

var _ Handler = (*handler)(nil)

// GetPackageArtifactInfo retrieves artifact information from the request.
func (h *handler) GetPackageArtifactInfo(r *http.Request) (pkg.PackageArtifactInfo, error) {
	info, err := h.Handler.GetArtifactInfo(r)
	if err != nil {
		return nil, err
	}

	repoType := chi.URLParam(r, "repoType")
	repo := chi.URLParam(r, "repo")
	rev := chi.URLParam(r, "rev")
	sha256 := chi.URLParam(r, "sha256")
	if repoType != "" {
		repoType = strings.TrimSuffix(repoType, "s")
	} else {
		repoType = "model"
	}

	// Validate repoType
	if !allowedTypes[repoType] {
		log.Ctx(r.Context()).Error().Msgf("unsupported repoType: %s", repoType)
		return nil, fmt.Errorf("unsupported repoType")
	}

	// Validate repoID
	if repo != "" && !isValidRepoID(repo) {
		log.Ctx(r.Context()).Error().Msgf("Invalid repo ID: %s", repo)
		return nil, fmt.Errorf("invalid repo ID format")
	}

	// Set default revision if not provided
	if rev == "" {
		rev = "main"
	}
	info.Image = repo

	return &hftype.ArtifactInfo{
		ArtifactInfo: info,
		Repo:         repo,
		Revision:     rev,
		RepoType:     repoType,
		SHA256:       sha256,
	}, nil
}

// isValidRepoID validates the format of a repo ID.
func isValidRepoID(repoID string) bool {
	return repoIDMatcher.MatchString(repoID)
}
