// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package token

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/harness/scm/internal/api/render"
	"github.com/harness/scm/internal/api/request"
	"github.com/harness/scm/internal/store"
	"github.com/harness/scm/internal/token"
	"github.com/harness/scm/types"

	"github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

// Must returns an http.HandlerFunc middleware that authenticates
// the http.Request and errors if the account cannot be authenticated.
func Must(users store.UserStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			str := extractToken(r)

			if len(str) == 0 {
				render.ErrorCode(w, errors.New("Requires authentication"), 401)
				return
			}

			var user *types.User
			parsed, err := jwt.ParseWithClaims(str, &token.Claims{}, func(token_ *jwt.Token) (interface{}, error) {
				sub := token_.Claims.(*token.Claims).Subject
				id, err := strconv.ParseInt(sub, 10, 64)
				if err != nil {
					return nil, err
				}

				user, err = users.Find(ctx, id)
				if err != nil {
					hlog.FromRequest(r).
						Error().Err(err).
						Int64("user", id).
						Msg("cannot find user")
					return nil, err
				}
				return []byte(user.Salt), nil
			})
			if err != nil {
				render.ErrorCode(w, err, 401)
				return
			}
			if parsed.Valid == false {
				render.ErrorCode(w, errors.New("Invalid token"), 401)
				return
			}
			if _, ok := parsed.Method.(*jwt.SigningMethodHMAC); !ok {
				render.ErrorCode(w, errors.New("Invalid token"), 401)
				return
			}

			// this code should be deprecated, since the jwt.ParseWithClaims
			// should fail if the token is expired. TODO remove once we have
			// proper unit tests in place.
			if claims, ok := parsed.Claims.(*token.Claims); ok {
				if claims.ExpiresAt > 0 {
					if time.Now().Unix() > claims.ExpiresAt {
						render.ErrorCode(w, errors.New("Expired token"), 401)
						return
					}
				}
			}

			log.Ctx(ctx).UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.Str("session_email", user.Email).Bool("session_admin", user.Admin)
			})

			next.ServeHTTP(w, r.WithContext(
				request.WithUser(ctx, user),
			))
		})
	}
}

func extractToken(r *http.Request) string {
	bearer := r.Header.Get("Authorization")
	if bearer == "" {
		return r.FormValue("access_token")
	}
	bearer = strings.TrimPrefix(bearer, "Bearer ")
	bearer = strings.TrimPrefix(bearer, "IdentityService ")
	return bearer
}
