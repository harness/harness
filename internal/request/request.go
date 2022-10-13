// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package router provides http handlers for serving the
// web applicationa and API endpoints.
package request

import (
	"fmt"
	"net/http"
	"net/url"
)

// ReplacePrefix replaces the path of the request.
// IMPORTANT:
//   - both prefix are unescaped for path, and used as is for RawPath!
//   - only called by top level handler!!
func ReplacePrefix(r *http.Request, oldPrefix string, newPrefix string) error {
	/*
	 * According to official documentation, we can change anything in the request but the body:
	 *    https://pkg.go.dev/net/http#Handler
	 *
	 * ASSUMPTION:
	 *		This is called by a top level handler (no router or middleware above it)
	 *		Therefore, we don't have to worry about getting any routing metadata out of sync.
	 *
	 * This is different to returning a shallow clone with updated URL, which is what
	 * http.StripPrefix or earlier versions of request.WithContext are doing:
	 * 		https://cs.opensource.google/go/go/+/refs/tags/go1.19:src/net/http/server.go;l=2138
	 *		https://cs.opensource.google/go/go/+/refs/tags/go1.18:src/net/http/request.go;l=355
	 *
	 * http.StripPrefix initially changed the path only, but that was updated because of official recommendations:
	 * 		https://github.com/golang/go/issues/18952
	 */
	unOldPrefix, err := url.PathUnescape(oldPrefix)
	if err != nil {
		return fmt.Errorf("failed to unescape old prefix '%s'", oldPrefix)
	}
	unNewPrefix, err := url.PathUnescape(newPrefix)
	if err != nil {
		return fmt.Errorf("failed to unescape new prefix '%s'", newPrefix)
	}

	unl := len(unOldPrefix)
	if len(r.URL.Path) < unl || r.URL.Path[0:unl] != unOldPrefix {
		return fmt.Errorf("path '%s' doesn't contain prefix '%s'", r.URL.Path, unOldPrefix)
	}

	// only change RawPath if it exists
	if r.URL.RawPath != "" {
		l := len(oldPrefix)
		if len(r.URL.RawPath) < l || r.URL.RawPath[0:l] != oldPrefix {
			return fmt.Errorf("raw path '%s' doesn't contain prefix '%s'", r.URL.RawPath, oldPrefix)
		}

		r.URL.RawPath = newPrefix + r.URL.RawPath[l:]
	}

	r.URL.Path = unNewPrefix + r.URL.Path[unl:]

	return nil
}
