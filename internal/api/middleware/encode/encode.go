// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package encode

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog/hlog"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/request"
	"github.com/harness/gitness/types"
)

const (
	EncodedPathSeparator = "%252F"
)

// GitPathBefore wraps an http.HandlerFunc in a layer that encodes a path coming
// as part of the GIT api (e.g. "space1/repo.git") before executing the provided http.HandlerFunc.
func GitPathBefore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := pathTerminatedWithMarker(r, "", ".git", false)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// TerminatedPathBefore wraps an http.HandlerFunc in a layer that encodes a terminated path (e.g. "/space1/space2/+")
// before executing the provided http.HandlerFunc. The first prefix that matches the URL.Path will
// be used during encoding (prefix is ignored during encoding).
func TerminatedPathBefore(prefixes []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, p := range prefixes {
			changed, err := pathTerminatedWithMarker(r, p, "/+", false)
			if err != nil {
				render.TranslatedUserError(w, err)
				return
			}

			// first prefix that leads to success we can stop
			if changed {
				break
			}
		}

		next.ServeHTTP(w, r)
	})
}

// pathTerminatedWithMarker function encodes a path followed by a custom marker and returns a request with an
// updated URL.Path.
// A non-empty prefix can be provided to encode encode only after the prefix.
// It allows our Rest API to handle paths of the form "/spaces/space1/space2/+/authToken"
//
// Examples:
// Prefix: "" Path: "/space1/space2/+" => "/space1%2Fspace2"
// Prefix: "" Path: "/space1/space2.git" => "/space1%2Fspace2"
// Prefix: "/spaces" Path: "/spaces/space1/space2/+/authToken" => "/spaces/space1%2Fspace2/authToken".
func pathTerminatedWithMarker(r *http.Request, prefix string, marker string, keepMarker bool) (bool, error) {
	// In case path doesn't start with prefix - nothing to encode
	if len(r.URL.Path) < len(prefix) || r.URL.Path[0:len(prefix)] != prefix {
		return false, nil
	}

	originalSubPath := r.URL.Path[len(prefix):]
	path, _, found := strings.Cut(originalSubPath, marker)

	// If we don't find a marker - nothing to encode
	if !found {
		return false, nil
	}

	// if marker was found - convert to escaped version (skip first character in case path starts with '/').
	// Since replacePrefix unescapes the strings, we have to double escape.
	escapedPath := path[0:1] + strings.ReplaceAll(path[1:], types.PathSeparator, EncodedPathSeparator)
	if keepMarker {
		escapedPath += marker
	}

	prefixWithPath := prefix + path + marker
	prefixWithEscapedPath := prefix + escapedPath

	hlog.FromRequest(r).Trace().Msgf(
		"[Encode] prefix: '%s', marker: '%s', original: '%s', escaped: '%s'.\n",
		prefix,
		marker,
		prefixWithPath,
		prefixWithEscapedPath)

	err := request.ReplacePrefix(r, prefixWithPath, prefixWithEscapedPath)
	if err != nil {
		return false, err
	}

	return true, nil
}
