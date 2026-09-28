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

package render

import (
	"net/http"
	"strings"
)

// userContentCSP neutralizes active content (SVG, HTML, XML) that is served from
// user-controlled bytes. The sandbox directive is the load bearing part: on top level
// navigation the browser assigns the document a unique opaque origin and disables
// scripting, so a committed payload can neither execute nor reach the app origin's
// storage. Subresource loads (<img src=...>) ignore the response CSP, so inline
// rendering of images in markdown and file previews is unaffected.
const userContentCSP = "default-src 'none'; style-src 'unsafe-inline'; sandbox; frame-ancestors 'none'"

// navigationRenderableTypes are the media types the Synack recommended fix for CODE-6295 names
// (text/html, image/svg+xml, application/xhtml+xml) plus every other type a browser renders as
// an active document, neutralized to text/plain unconditionally - there is no exemption for how
// the request reached the raw endpoint. Any other "+xml" type is treated the same way, because
// an XML document can carry an xml-stylesheet processing instruction and execute script
// through XSLT.
var navigationRenderableTypes = map[string]struct{}{
	"text/html":       {},
	"image/svg+xml":   {},
	"text/xml":        {},
	"application/xml": {},
	"text/xsl":        {},
}

// UserContentSecurityHeaders sets the security headers required on any response that
// streams user-controlled bytes back to the browser.
func UserContentSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", userContentCSP)
}

// NeutralizeRenderableContentType rewrites contentType to text/plain if the browser would
// otherwise execute it as a document on the app origin, so that opening the URL shows the raw
// markup as text instead of parsing it. This is a second layer behind UserContentSecurityHeaders:
// the sandbox CSP already blocks execution, but a response that never becomes a document cannot
// be affected by a future weakening of that policy. Non-renderable types (including an empty
// contentType) are returned unchanged.
func NeutralizeRenderableContentType(contentType string) string {
	if !isNavigationRenderable(contentType) {
		return contentType
	}

	return "text/plain; charset=utf-8"
}

// isNavigationRenderable strips any parameters off contentType and reports whether the
// remaining media type renders as an active document.
func isNavigationRenderable(contentType string) bool {
	mediaType, _, _ := strings.Cut(contentType, ";")
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))

	if _, ok := navigationRenderableTypes[mediaType]; ok {
		return true
	}

	return strings.HasSuffix(mediaType, "+xml")
}
