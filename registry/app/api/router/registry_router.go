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

	"github.com/harness/gitness/registry/utils"
)

const RegistryMount = "/api/v1/registry"
const APIMount = "/api"

type RegistryRouter struct {
	handler http.Handler
}

func NewRegistryRouter(handler http.Handler) *RegistryRouter {
	return &RegistryRouter{handler: handler}
}

func (r *RegistryRouter) Handle(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}

func (r *RegistryRouter) IsEligibleTraffic(req *http.Request) bool {
	urlPath := req.URL.Path
	if req.URL.RawPath != "" {
		urlPath = req.URL.RawPath
	}
	if utils.HasAnyPrefix(urlPath, []string{RegistryMount, "/v2/", "/registry/", "/maven/", "/generic/"}) ||
		(strings.HasPrefix(urlPath, APIMount+"/v1/spaces/") &&
			utils.HasAnySuffix(urlPath, []string{"/artifacts", "/registries"})) {
		return true
	}

	return false
}

func (r *RegistryRouter) Name() string {
	return "registry"
}
