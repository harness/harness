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
	if strings.HasPrefix(req.URL.Path, RegistryMount) || strings.HasPrefix(req.URL.Path, "/v2/") ||
		strings.HasPrefix(req.URL.Path, "/registry/") ||
		(strings.HasPrefix(req.URL.Path, APIMount+"/v1/spaces/") &&
			(strings.HasSuffix(req.URL.Path, "/artifacts") ||
				strings.HasSuffix(req.URL.Path, "/registries"))) {
		return true
	}

	return false
}

func (r *RegistryRouter) Name() string {
	return "registry"
}
