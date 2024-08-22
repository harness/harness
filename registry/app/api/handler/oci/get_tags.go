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

package oci

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg/docker"

	"github.com/rs/zerolog/log"
)

func (h *Handler) GetTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	info, err := h.getRegistryInfo(r, false)
	if err != nil {
		handleErrors(ctx, []error{err}, w)
		return
	}
	errorsList := make(errcode.Errors, 0)

	q := r.URL.Query()
	lastEntry := q.Get("last")
	maxEntries, err := strconv.Atoi(q.Get("n"))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to parse max entries %s", q.Get("n"))
		maxEntries = docker.DefaultMaximumReturnedEntries
	}

	if maxEntries <= 0 {
		maxEntries = docker.DefaultMaximumReturnedEntries
	}

	rs, tags, err := h.Controller.GetTags(ctx, lastEntry, maxEntries, r.URL.String(), info)
	log.Ctx(ctx).Debug().Msgf("GetTags: %v %s", rs, tags)

	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to list tags")
		handleErrors(ctx, errorsList, w)
		return
	}
	rs.WriteHeadersToResponse(w)
	enc := json.NewEncoder(w)
	if err := enc.Encode(
		docker.TagsAPIResponse{
			Name: info.RegIdentifier,
			Tags: tags,
		},
	); err != nil {
		errorsList = append(errorsList, errcode.ErrCodeUnknown.WithDetail(err))
	}
	handleErrors(ctx, errorsList, w)
}
