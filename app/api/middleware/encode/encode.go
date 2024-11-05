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

package encode

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/request"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/hlog"
	"golang.org/x/exp/slices"
)

const (
	EncodedPathSeparator = "%2F"
)

// GitPathBefore wraps an http.HandlerFunc in a layer that encodes a path coming
// as part of the GIT api (e.g. "space1/repo.git") before executing the provided http.HandlerFunc.
func GitPathBefore(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ok, err := pathTerminatedWithMarker(r, "", ".git", "")
			if err != nil {
				render.TranslatedUserError(ctx, w, err)
				return
			}
			if !ok {
				if _, err = processGitRequest(r); err != nil {
					render.TranslatedUserError(ctx, w, err)
					return
				}
			}

			next.ServeHTTP(w, r)
		},
	)
}

func processGitRequest(r *http.Request) (bool, error) {
	const infoRefsPath = "/info/refs"
	const uploadPack = "git-upload-pack"
	const uploadPackPath = "/" + uploadPack
	const receivePack = "git-receive-pack"
	const receivePackPath = "/" + receivePack
	const serviceParam = "service"

	allowedServices := []string{
		uploadPack,
		receivePack,
	}

	urlPath := r.URL.Path
	if r.URL.RawPath != "" {
		urlPath = r.URL.RawPath
	}

	switch r.Method {
	case http.MethodGet:
		// check if request is coming from git client
		if strings.HasSuffix(urlPath, infoRefsPath) && r.URL.Query().Has(serviceParam) {
			service := r.URL.Query().Get(serviceParam)
			if !slices.Contains(allowedServices, service) {
				return false, errors.InvalidArgument("git request allows only %v service, got: %s",
					allowedServices, service)
			}
			return pathTerminatedWithMarkerAndURL(r, "", infoRefsPath, infoRefsPath, urlPath)
		}
	case http.MethodPost:
		if strings.HasSuffix(urlPath, uploadPackPath) {
			return pathTerminatedWithMarkerAndURL(r, "", uploadPackPath, uploadPackPath, urlPath)
		}

		if strings.HasSuffix(urlPath, receivePackPath) {
			return pathTerminatedWithMarkerAndURL(r, "", receivePackPath, receivePackPath, urlPath)
		}
	}

	// no other APIs are called by git - just treat it as a full repo path.
	return pathTerminatedWithMarkerAndURL(r, "", "", "", urlPath)
}

// TerminatedPathBefore wraps an http.HandlerFunc in a layer that encodes a terminated path (e.g. "/space1/space2/+")
// before executing the provided http.HandlerFunc. The first prefix that matches the URL.Path will
// be used during encoding (prefix is ignored during encoding).
func TerminatedPathBefore(prefixes []string, next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			for _, p := range prefixes {
				changed, err := pathTerminatedWithMarker(r, p, "/+", "")
				if err != nil {
					render.TranslatedUserError(ctx, w, err)
					return
				}

				// first prefix that leads to success we can stop
				if changed {
					break
				}
			}

			next.ServeHTTP(w, r)
		},
	)
}

// TerminatedRegexPathBefore is similar to TerminatedPathBefore but supports regex prefixes.
func TerminatedRegexPathBefore(regexPrefixes []string, next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			for _, p := range regexPrefixes {
				changed, err := regexPathTerminatedWithMarker(r, p, "/+", "")
				if err != nil {
					render.TranslatedUserError(ctx, w, err)
					return
				}

				// first prefix that leads to success we can stop
				if changed {
					break
				}
			}

			next.ServeHTTP(w, r)
		},
	)
}

// pathTerminatedWithMarker function encodes a path followed by a custom marker and returns a request with an
// updated URL.Path.
// A non-empty prefix can be provided to encode only after the prefix.
// It allows our Rest API to handle paths of the form "/spaces/space1/space2/+/authToken"
//
// Examples:
// Prefix: "" Path: "/space1/space2/+" => "/space1%2Fspace2"
// Prefix: "" Path: "/space1/.gitness.git" => "/space1%2F.gitness"
// Prefix: "/spaces" Path: "/spaces/space1/space2/+/authToken" => "/spaces/space1%2Fspace2/authToken".
func pathTerminatedWithMarker(r *http.Request, prefix string, marker string, markerReplacement string) (bool, error) {
	// r.URL.Path contains URL-decoded URI path. r.URL.RawPath contains raw URI path. But, if the URI don't contain
	// any character that needs encoding, the r.URL.RawPath is empty (otherwise it would be equal to r.URL.Path).

	// We work with r.URL.RawPath it is exist, or with r.URL.Path if the r.URL.RawPath is empty.
	urlPath := r.URL.Path
	if r.URL.RawPath != "" {
		urlPath = r.URL.RawPath
	}

	return pathTerminatedWithMarkerAndURL(r, prefix, marker, markerReplacement, urlPath)
}

func pathTerminatedWithMarkerAndURL(
	r *http.Request, prefix string, marker string, markerReplacement string, urlPath string,
) (bool, error) {
	// In case path doesn't start with prefix - nothing to encode
	if len(urlPath) < len(prefix) || urlPath[0:len(prefix)] != prefix {
		return false, nil
	}

	originalSubPath := urlPath[len(prefix):]
	path, found := cutOutTerminatedPath(originalSubPath, marker)
	if !found {
		// If we don't find a marker - nothing to encode
		return false, nil
	}

	// if marker was found - convert to escaped version (skip first character in case path starts with '/').
	escapedPath := path[0:1] + strings.ReplaceAll(path[1:], types.PathSeparatorAsString, EncodedPathSeparator)

	prefixWithPath := prefix + path + marker
	prefixWithEscapedPath := prefix + escapedPath + markerReplacement

	hlog.FromRequest(r).Trace().Msgf(
		"[Encode] prefix: '%s', marker: '%s', original: '%s', escaped: '%s'.\n",
		prefix,
		marker,
		prefixWithPath,
		prefixWithEscapedPath,
	)

	err := request.ReplacePrefix(r, prefixWithPath, prefixWithEscapedPath)
	if err != nil {
		return false, err
	}

	return true, nil
}

// regexPathTerminatedWithMarker is similar to pathTerminatedWithMarker but with regex prefix support.
//
// Example:
// 1. Path: "/registry/app1%2Fremote2/artifact/foo/bar/+/summary"
// Prefix: => "^/registry/([^/]+)/artifact/" Marker: => "/+" MarkerReplacement: => ""
// ==> "/registry/app1%2Fremote2/artifact/foo%2Fbar/summary"
//
// 2. Path: "/registry/abc/artifact/foo/bar/+/summary"
// Prefix: => "^/registry/([^/]+)/artifact/" Marker: => "/+" MarkerReplacement: => ""
// ==> "/registry/abc/artifact/foo%2Fbar/summary"

func regexPathTerminatedWithMarker(
	r *http.Request,
	regexPrefix string,
	marker string,
	markerReplacement string,
) (bool, error) {
	prefixPattern := regexp.MustCompile(regexPrefix)

	matches := prefixPattern.FindStringSubmatch(r.URL.Path)
	// In case path doesn't start with prefix - nothing to encode
	if len(matches) == 0 {
		return false, nil
	}

	// We only care about the first match as we provide prefix
	prefix := matches[0]

	urlPath := r.URL.Path
	if r.URL.RawPath != "" {
		urlPath = r.URL.RawPath
	}
	return pathTerminatedWithMarkerAndURL(r, prefix, marker, markerReplacement, urlPath)
}

// cutOutTerminatedPath cuts out the resource path terminated with the provided marker (path segment suffix).
// e.g. subPath: "/space1/space2/+/authToken", marker: "/+" => "/space1/space2"
// e.g. subPath: "/space1/space2.git", marker: ".git" => "/space1/space2"
// e.g. subPath: "/space1/space2.git/", marker: ".git" => "/space1/space2".
func cutOutTerminatedPath(subPath string, marker string) (string, bool) {
	// if subpath ends with the marker, just remove the marker.
	if strings.HasSuffix(subPath, marker) {
		return subPath[:len(subPath)-len(marker)], true
	}

	// ensure we only look for path segment suffixes when looking for the marker.
	if !strings.HasSuffix(marker, "/") {
		marker += "/"
	}
	if path, _, found := strings.Cut(subPath, marker); found {
		return path, true
	}

	return "", false
}
