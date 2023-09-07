// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package request

import (
	"net/http"
)

const (
	QueryParamAccessToken   = "access_token"
	QueryParamIncludeCookie = "include_cookie"
	CookieToken             = "token"
)

func GetAccessTokenFromQuery(r *http.Request) (string, bool) {
	return QueryParam(r, QueryParamAccessToken)
}

func GetIncludeCookieFromQueryOrDefault(r *http.Request, dflt bool) (bool, error) {
	return QueryParamAsBoolOrDefault(r, QueryParamIncludeCookie, dflt)
}

func GetTokenFromCookie(r *http.Request) (string, bool) {
	return GetCookie(r, CookieToken)
}
