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

package account

import (
	"errors"
	"net/http"
	"time"

	"github.com/harness/gitness/types"
)

func includeTokenCookie(
	r *http.Request, w http.ResponseWriter,
	tokenResponse *types.TokenResponse,
	cookieName string,
	enforceSecure bool, // Add this parameter
) {
	cookie := newEmptyTokenCookie(r, cookieName, enforceSecure) // Include enforceSecure
	cookie.Value = tokenResponse.AccessToken
	if tokenResponse.Token.ExpiresAt != nil {
		cookie.Expires = time.UnixMilli(*tokenResponse.Token.ExpiresAt)
	}

	http.SetCookie(w, cookie)
}

func deleteTokenCookieIfPresent(r *http.Request, w http.ResponseWriter, cookieName string, enforceSecure bool) {
	// if no token is present in the cookies, nothing todo.
	// No other error type expected here - and even if there is, let's try best effort deletion.
	_, err := r.Cookie(cookieName)
	if errors.Is(err, http.ErrNoCookie) {
		return
	}

	cookie := newEmptyTokenCookie(r, cookieName, enforceSecure) // Include enforceSecure
	cookie.Value = ""
	cookie.Expires = time.UnixMilli(0) // this effectively tells the browser to delete the cookie
	
	http.SetCookie(w, cookie)
}

func newEmptyTokenCookie(r *http.Request, cookieName string, enforceSecure bool) *http.Cookie {
    isSecure := false

    // Check for the X-Forwarded-Proto header
    if proto := r.Header.Get("X-Forwarded-Proto"); proto == "https" {
        isSecure = true
    } else if r.TLS != nil || r.URL.Scheme == "https" {
        isSecure = true
    }

    // Enforce 'Secure' based on the provided flag
    if enforceSecure {
        isSecure = true
    }

    return &http.Cookie{
        Name:     cookieName,
        SameSite: http.SameSiteStrictMode,
        HttpOnly: true,
        Path:     "/",
        Domain:   r.URL.Hostname(),
        Secure:   isSecure,
    }
}
