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
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// HandleGitServicePack handles the service pack part of git's smart http protocol (receive-/upload-pack).
func HandleGitServicePack(
	service enum.GitServiceType,
	repoCtrl *repo.Controller,
	urlProvider url.Provider,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			pktError(ctx, w, err)
			return
		}

		contentEncoding := request.GetContentEncodingFromHeadersOrDefault(r, "")
		gitProtocol := request.GetGitProtocolFromHeadersOrDefault(r, "")

		// Handle GZIP.
		dataReader := r.Body
		if contentEncoding == "gzip" {
			gzipReader, err := gzip.NewReader(dataReader)
			if err != nil {
				pktError(ctx, w, fmt.Errorf("failed to create new gzip reader: %w", err))
				return
			}
			defer func() {
				if cErr := gzipReader.Close(); cErr != nil {
					log.Ctx(ctx).Warn().Err(cErr).Msg("failed to close the gzip reader")
				}
			}()
			dataReader = gzipReader
		}

		// Clients MUST NOT reuse or revalidate a cached response.
		// Servers MUST include sufficient Cache-Control headers to prevent caching of the response.
		// https://git-scm.com/docs/http-protocol
		render.NoCache(w)
		w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", service))

		start := time.Now()
		cw := &countingWriter{w: w}

		err = repoCtrl.GitServicePack(ctx, session, repoRef, api.ServicePackOptions{
			Service:      service,
			StatelessRPC: true,
			Stdout:       cw,
			Stdin:        dataReader,
			Protocol:     gitProtocol,
		})
		if errors.Is(err, apiauth.ErrUnauthorized) {
			render.GitBasicAuth(ctx, w, urlProvider)
			return
		}
		if err != nil {
			if cw.n > 0 {
				// The response is already streaming: the status is sent and an ERR pkt-line would end up
				// in the middle of the pack, so only record what happened.
				logStreamFailure(ctx, err, service, cw.n, time.Since(start))
				return
			}
			pktError(ctx, w, err)
			return
		}
	}
}

// countingWriter counts the bytes written to the underlying writer.
type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

// logStreamFailure logs a service pack that failed after data was already sent to the client.
func logStreamFailure(
	ctx context.Context,
	err error,
	service enum.GitServiceType,
	written int64,
	elapsed time.Duration,
) {
	level := zerolog.ErrorLevel
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		level = zerolog.WarnLevel
	}
	log.Ctx(ctx).WithLevel(level).Err(err).
		Str("service", string(service)).
		Int64("bytes_written", written).
		Dur("elapsed", elapsed).
		Msg("service pack failed after the response started streaming")
}
