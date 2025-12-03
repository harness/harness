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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// format string for the link header value.
var linkf = `<%s>; rel="%s"`

// Pagination writes the pagination and link headers to the http.Response.
func Pagination(r *http.Request, w http.ResponseWriter, page, size, total int) {
	var (
		last = pagelen(size, total)
	)

	// Add information that doesn't require total
	PaginationNoTotal(r, w, page, size, page >= last)

	// add information that requires total
	uri := getPaginationBaseURL(r, page, size)
	params := uri.Query()

	// update the page query parameter and re-encode
	params.Set("page", strconv.Itoa(last))
	uri.RawQuery = params.Encode()

	// write the page total to the header.
	w.Header().Set("x-total", strconv.Itoa(total))
	w.Header().Set("x-total-pages", strconv.Itoa(last))
	w.Header().Add("Link", fmt.Sprintf(linkf, uri.String(), "last"))
}

// PaginationNoTotal writes the pagination and link headers to the http.Response when total is unknown.
func PaginationNoTotal(r *http.Request, w http.ResponseWriter, page int, size int, isLastPage bool) {
	var (
		next = page + 1
		prev = max(page-1, 1)
	)

	// write basic headers
	w.Header().Set("x-page", strconv.Itoa(page))
	w.Header().Set("x-per-page", strconv.Itoa(size))

	// write headers based on relative location of current page
	uri := getPaginationBaseURL(r, page, size)
	params := uri.Query()

	if !isLastPage {
		// update the page query parameter and re-encode
		params.Set("page", strconv.Itoa(next))
		uri.RawQuery = params.Encode()

		// write the next page to the header.
		w.Header().Set("x-next-page", strconv.Itoa(next))
		w.Header().Add("Link", fmt.Sprintf(linkf, uri.String(), "next"))
	}

	if page > 1 {
		// update the page query parameter and re-encode.
		params.Set("page", strconv.Itoa(prev))
		uri.RawQuery = params.Encode()

		// write the previous page to the header.
		w.Header().Set("x-prev-page", strconv.Itoa(prev))
		w.Header().Add("Link", fmt.Sprintf(linkf, uri.String(), "prev"))
	}
}

// PaginationLimit writes the x-total header.
func PaginationLimit(_ *http.Request, w http.ResponseWriter, total int) {
	w.Header().Set("x-total", strconv.Itoa(total))
}

func getPaginationBaseURL(r *http.Request, page int, size int) url.URL {
	uri := *r.URL

	// parse the existing query parameters and
	// sanize parameter list.
	params := uri.Query()
	params.Del("access_token")
	params.Del("token")
	params.Set("page", strconv.Itoa(page))
	params.Set("limit", strconv.Itoa(size))

	return uri
}

// NoCache writes the required headers to communicate to the client no caching shall be done.
func NoCache(w http.ResponseWriter) {
	w.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
}
