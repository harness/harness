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

package repo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

// HandleRaw returns the raw content of a file.
func HandleRaw(repoCtrl *repo.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		session, _ := request.AuthSessionFrom(ctx)

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		gitRef := request.GetGitRefFromQueryOrDefault(r, "")
		path := request.GetOptionalRemainderFromPath(r)

		resp, err := repoCtrl.Raw(ctx, session, repoRef, gitRef, path)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		defer func() {
			if err := resp.Data.Close(); err != nil {
				log.Ctx(ctx).Warn().Err(err).Msgf("failed to close blob content reader.")
			}
		}()

		ifNoneMatch, ok := request.GetIfNoneMatchFromHeader(r)
		if ok && ifNoneMatch == resp.SHA.String() {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Add("Content-Length", fmt.Sprint(resp.Size))
		w.Header().Add(request.HeaderETag, resp.SHA.String())

		// http package hasnt implemented svg mime type detection
		// https://github.com/golang/go/blob/master/src/net/http/sniff.go#L66
		if resp.Size > 0 {
			buf := make([]byte, 512) // 512 bytes is standard for MIME detection
			n, err := io.ReadFull(resp.Data, buf)
			if err == nil || err == io.EOF || err == io.ErrUnexpectedEOF {
				contentType := detectContentType(buf[:n])
				w.Header().Set("Content-Type", contentType)

				resp.Data = &types.MultiReadCloser{
					Reader:    io.MultiReader(bytes.NewReader(buf[:n]), resp.Data),
					CloseFunc: resp.Data.Close,
				}
			}
		}

		render.Reader(ctx, w, http.StatusOK, resp.Data)
	}
}

// xmlPrefixRegex is used to detect XML declarations in a case-insensitive way.
var xmlPrefixRegex = regexp.MustCompile(`(?i)^<\?xml`)

// svgPrefixRegex is used to detect SVG tag openers in a case-insensitive way.
var svgPrefixRegex = regexp.MustCompile(`(?i)^<svg`)

// svgTagRegex is used to detect SVG tags anywhere in the content.
var svgTagRegex = regexp.MustCompile(`(?i)<svg`)

// detectContentType enhances Go's standard http.DetectContentType with SVG support
// following the WHATWG MIME Sniffing Standard https://mimesniff.spec.whatwg.org/
func detectContentType(data []byte) string {
	if len(data) > 5 {
		if xmlPrefixRegex.Match(data) || svgPrefixRegex.Match(data) {
			if svgTagRegex.Match(data) {
				return "image/svg+xml"
			}
		}
	}

	return http.DetectContentType(data)
}
