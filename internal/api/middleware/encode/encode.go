package encode

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/harness/gitness/types"
)

// GitPathBefore wraps an http.HandlerFunc in a layer that encodes Paths coming as part of the GIT api
// (e.g. "space1/repo.git") before executing the provided http.HandlerFunc
// The first prefix that matches the URL.Path will be used during encoding.
func GitPathBefore(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r, _ = pathTerminatedWithMarker(r, "", ".git", false)
		h.ServeHTTP(w, r)
	}
}

// TerminatedPathBefore wraps an http.HandlerFunc in a layer that encodes a terminated path (e.g. "/space1/space2/+")
// before executing the provided http.HandlerFunc. The first prefix that matches the URL.Path will
// be used during encoding.
func TerminatedPathBefore(prefixes []string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, p := range prefixes {
			// IMPORTANT: define changed separately to avoid overshadowing r
			var changed bool
			if r, changed = pathTerminatedWithMarker(r, p, "/+", false); changed {
				break
			}
		}

		h.ServeHTTP(w, r)
	}
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
func pathTerminatedWithMarker(r *http.Request, prefix string, marker string, keepMarker bool) (*http.Request, bool) {
	// In case path doesn't start with prefix - nothing to encode
	if len(r.URL.Path) < len(prefix) || r.URL.Path[0:len(prefix)] != prefix {
		return r, false
	}

	originalSubPath := r.URL.Path[len(prefix):]
	path, suffix, found := strings.Cut(originalSubPath, marker)

	// If we don't find a marker - nothing to encode
	if !found {
		return r, false
	}

	// if marker was found - convert to escaped version (skip first character in case path starts with '/')
	escapedPath := path[0:1] + strings.ReplaceAll(path[1:], types.PathSeparator, "%2F")
	if keepMarker {
		escapedPath += marker
	}
	updatedSubPath := escapedPath + suffix

	// TODO: Proper Logging
	log.Debug().Msgf(
		"[Encode] prefix: '%s', marker: '%s', original: '%s', updated: '%s'.\n",
		prefix,
		marker,
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
