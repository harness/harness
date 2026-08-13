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

import "net/http"

// userContentCSP neutralizes active content (SVG, HTML, XML) that is served from
// user-controlled bytes. The sandbox directive is the load bearing part: on top level
// navigation the browser assigns the document a unique opaque origin and disables
// scripting, so a committed payload can neither execute nor reach the app origin's
// storage. Subresource loads (<img src=...>) ignore the response CSP, so inline
// rendering of images in markdown and file previews is unaffected.
const userContentCSP = "default-src 'none'; style-src 'unsafe-inline'; sandbox; frame-ancestors 'none'"

// UserContentSecurityHeaders sets the security headers required on any response that
// streams user-controlled bytes back to the browser.
func UserContentSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", userContentCSP)
}
