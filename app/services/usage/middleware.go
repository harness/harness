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

package usage

import (
	"net/http"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/paths"

	"github.com/rs/zerolog/log"
)

func Middleware(intf Sender, trackUpload bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ref, err := request.GetRepoRefFromPath(r)
			if err != nil {
				log.Ctx(r.Context()).Warn().Err(err).Msg("unable to get space ref")
				next.ServeHTTP(w, r)
				return
			}
			rootSpace, _, err := paths.DisectRoot(ref)
			if err != nil {
				log.Ctx(r.Context()).Warn().Err(err).Msg("unable to get root space")
				next.ServeHTTP(w, r)
				return
			}

			writer := newWriter(w)
			reader := newReader(r.Body)

			if trackUpload {
				r.Body = reader
			}

			next.ServeHTTP(writer, r)

			// send usage metrics
			m := Metric{
				SpaceRef: rootSpace,
				Bandwidth: Bandwidth{
					Out: writer.n,
					In:  reader.n,
				},
			}

			err = intf.Send(r.Context(), m)
			if err != nil {
				log.Ctx(r.Context()).Warn().Err(err).Msg("unable to send usage metric")
				return
			}
		})
	}
}
