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

	"github.com/harness/gitness/logging"
)

type WebRouter struct {
	handler http.Handler
}

func NewWebRouter(handler http.Handler) *WebRouter {
	return &WebRouter{handler: handler}
}

func (r *WebRouter) Handle(w http.ResponseWriter, req *http.Request) {
	req = req.WithContext(logging.NewContext(req.Context(), WithLoggingRouter("web")))
	r.handler.ServeHTTP(w, req)
}

func (r *WebRouter) IsEligibleTraffic(*http.Request) bool {
	// Everything else will be routed to web (or return 404)
	return true
}

func (r *WebRouter) Name() string {
	return "web"
}
