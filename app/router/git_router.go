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

package router

import (
	"net/http"
	"strings"

	"github.com/harness/gitness/app/api/render"

	"github.com/rs/zerolog/log"
)

const GitMount = "/git"

type GitRouter struct {
	handler http.Handler

	// gitHost describes the optional host via which git traffic is identified.
	// Note: always stored as lowercase.
	gitHost string
}

func NewGitRouter(handler http.Handler, gitHost string) *GitRouter {
	return &GitRouter{handler: handler, gitHost: gitHost}
}

func (r *GitRouter) Handle(w http.ResponseWriter, req *http.Request) {
	// remove matched prefix to simplify API handlers (only if it's there)
	if err := StripPrefix(GitMount, req); err != nil {
		log.Ctx(req.Context()).Err(err).Msgf("Failed striping of prefix for git request.")
		render.InternalError(req.Context(), w)
		return
	}

	r.handler.ServeHTTP(w, req)
}

func (r *GitRouter) IsEligibleTraffic(req *http.Request) bool {
	// All Git originating traffic starts with "/space1/space2/repo.git".

	// git traffic is always reachable via the git mounting path.
	p := req.URL.Path
	if strings.HasPrefix(p, GitMount) {
		return true
	}

	// otherwise check if the request came in via the configured git host (if enabled)
	if len(r.gitHost) > 0 {
		// cut (optional) port off the host
		h, _, _ := strings.Cut(req.Host, ":")

		if strings.EqualFold(r.gitHost, h) {
			return true
		}
	}

	// otherwise we don't treat it as git traffic
	return false
}

func (r *GitRouter) Name() string {
	return "git"
}
