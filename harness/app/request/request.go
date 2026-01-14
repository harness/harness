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

	l := len(oldPrefix)

	if r.URL.RawPath != "" {
		if len(r.URL.RawPath) < l || r.URL.RawPath[0:l] != oldPrefix {
			return fmt.Errorf("raw path '%s' doesn't contain prefix '%s'", r.URL.RawPath, oldPrefix)
		}

		r.URL.RawPath = newPrefix + r.URL.RawPath[l:]
		r.URL.Path = url.PathEscape(r.URL.RawPath)
	} else {
		if len(r.URL.Path) < l || r.URL.Path[0:l] != oldPrefix {
			return fmt.Errorf("path '%s' doesn't contain prefix '%s'", r.URL.Path, oldPrefix)
		}

		r.URL.Path = newPrefix + r.URL.Path[l:]
	}

	return nil
}
