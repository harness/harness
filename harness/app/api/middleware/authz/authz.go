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

package authz

import (
	"net/http"

	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// BlockSessionToken blocks any request that uses a session token for authentication.
// NOTE: Major use case as of now is blocking usage of session tokens with git.
func BlockSessionToken(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// only block if auth data was available and it's based on a session token.
			if session, oks := request.AuthSessionFrom(ctx); oks {
				if tokenMetadata, ok := session.Metadata.(*auth.TokenMetadata); ok &&
					tokenMetadata.TokenType == enum.TokenTypeSession {
					log.Ctx(ctx).Warn().Msg("blocking git operation - session tokens are not allowed for usage with git")

					// NOTE: Git doesn't print the error message, so just return default 401 Unauthorized.
					render.Unauthorized(ctx, w)
					return
				}
			}

			next.ServeHTTP(w, r)
		},
	)
}
