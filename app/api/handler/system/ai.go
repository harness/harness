// Copyright 2026 Harness, Inc.
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

package system

import (
	"net/http"

	"github.com/harness/gitness/app/api/controller/system"
	"github.com/harness/gitness/app/api/render"
)

// HandleAIModels returns an http.HandlerFunc that returns the free model catalog of the AI gateway.
func HandleAIModels(sysCtrl *system.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, http.StatusOK, sysCtrl.AIModels(r.Context()))
	}
}

// HandleAIQuota returns an http.HandlerFunc that returns today's quota snapshot of the AI gateway.
func HandleAIQuota(sysCtrl *system.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, http.StatusOK, sysCtrl.AIQuota(r.Context()))
	}
}
