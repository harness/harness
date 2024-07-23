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
	"github.com/harness/gitness/logging"

	"github.com/rs/zerolog/log"
)

const APIMount = "/api"

type APIRouter struct {
	handler http.Handler
}

func NewAPIRouter(handler http.Handler) *APIRouter {
	return &APIRouter{handler: handler}
}

func (r *APIRouter) Handle(w http.ResponseWriter, req *http.Request) {
	req = req.WithContext(logging.NewContext(req.Context(), WithLoggingRouter("api")))

	// remove matched prefix to simplify API handlers
	if err := StripPrefix(APIMount, req); err != nil {
		log.Ctx(req.Context()).Err(err).Msgf("Failed striping of prefix for api request.")
		render.InternalError(req.Context(), w)
		return
	}

	r.handler.ServeHTTP(w, req)
}

func (r *APIRouter) IsEligibleTraffic(req *http.Request) bool {
	// All Rest API calls start with "/api/", and thus can be uniquely identified.
	p := req.URL.Path
	return strings.HasPrefix(p, APIMount)
}

func (r *APIRouter) Name() string {
	return "api"
}
