// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

/*
 * Encodes a terminated FQSN or FQRN and returns a request with an updated URL.Path
 * The first prefix that matches the URL.Path will be used during encoding.
 * This method can be used in case there are multiple prefixes for terminated FQNs.
 */
func EncodeTerminatedFQNs(r *http.Request, prefixes []string) (*http.Request, bool) {
	for _, p := range prefixes {
		if res, changed := EncodeTerminatedFQN(r, p); changed {
			return res, true
		}
	}

	return r, false
}

/*
 * This function encodes a terminated FQSN or FQRN and returns a request with an updated URL.Path.
 * A non-empty prefix can be provided to encode encode only after the prefix.
 * It allows our Rest API to handle paths of the form "/spaces/space1/space2/+/authToken"
 *
 * Examples:
 *   Prefix: "" Path: "/space1/space2/+" => "/space1%2Fspace2"
 *   Prefix: "/spaces" Path: "/spaces/space1/space2/+/authToken" => "/spaces/space1%2Fspace2/authToken"
 */
func EncodeTerminatedFQN(r *http.Request, prefix string) (*http.Request, bool) {
	// In case path doesn't start with prefix - nothing to encode
	if len(r.URL.Path) < len(prefix) || r.URL.Path[0:len(prefix)] != prefix {
		return r, false
	}

	originalSubPath := r.URL.Path[len(prefix):]
	space, suffix, found := strings.Cut(originalSubPath, "/+")

	// If we don't find a termination sequence - nothing to encode
	if !found {
		return r, false
	}

	// if terminated FQSN was found - convert to escaped version
	escapedSpace := "/" + strings.Replace(space[1:], "/", "%2F", -1)
	updatedSubPath := escapedSpace + suffix

	// TODO: Proper Logging
	fmt.Printf(
		"Encoding terminated FQSN: prefix: '%s', original: '%s', updated: '%s'.\n",
		prefix,
		originalSubPath,
		updatedSubPath)

	/*
	 * Return shallow clone with updated URL, similar to http.StripPrefix or earlier version of request.WithContext
	 * 		https://cs.opensource.google/go/go/+/refs/tags/go1.19:src/net/http/server.go;l=2138
	 *		https://cs.opensource.google/go/go/+/refs/tags/go1.18:src/net/http/request.go;l=355
	 *
	 * http.StripPrefix initially changed the path only, but that was updated because of official recommendations:
	 * 		https://github.com/golang/go/issues/18952
	 */
	r2 := new(http.Request)
	*r2 = *r
	r2.URL = new(url.URL)
	*r2.URL = *r.URL
	r2.URL.Path = prefix + updatedSubPath
	r2.URL.RawPath = ""

	return r2, true
}
