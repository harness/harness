// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authn

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/token"
	"github.com/harness/gitness/types"
	"github.com/rs/zerolog/hlog"
)

var _ Authenticator = (*TokenAuthenticator)(nil)

type TokenAuthenticator struct {
	users store.UserStore
}

func NewTokenAuthenticator(userStore store.UserStore) Authenticator {
	return &TokenAuthenticator{
		users: userStore,
	}
}

func (a *TokenAuthenticator) Authenticate(r *http.Request) (*types.User, error) {
	ctx := r.Context()
	str := extractToken(r)

	if len(str) == 0 {
		return nil, nil
	}

	var user *types.User
	parsed, err := jwt.ParseWithClaims(str, &token.Claims{}, func(token_ *jwt.Token) (interface{}, error) {
		sub := token_.Claims.(*token.Claims).Subject
		id, err := strconv.ParseInt(sub, 10, 64)
		if err != nil {
			return nil, err
		}

		user, err = a.users.Find(ctx, id)
		if err != nil {
			hlog.FromRequest(r).
				Error().Err(err).
				Int64("user", id).
				Msg("cannot find user")
			return nil, fmt.Errorf("Failed to get user info: %s", err)
		}
		return []byte(user.Salt), nil
	})
	if err != nil {
		return nil, err
	}
	if parsed.Valid == false {
		return nil, errors.New("Invalid token")
	}

	if _, ok := parsed.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, errors.New("Invalid token")
	}

	// this code should be deprecated, since the jwt.ParseWithClaims
	// should fail if the token is expired. TODO remove once we have
	// proper unit tests in place.
	if claims, ok := parsed.Claims.(*token.Claims); ok {
		if claims.ExpiresAt > 0 {
			if time.Now().Unix() > claims.ExpiresAt {
				return nil, errors.New("Expired token")
			}
		}
	}

	return user, nil
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
