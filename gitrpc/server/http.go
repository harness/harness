// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	gitnesshttp "github.com/harness/gitness/http"

	"code.gitea.io/gitea/modules/git"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

const (
	PathParamRepoUID = "repoUID"
)

var (
	safeGitProtocolHeader = regexp.MustCompile(`^[0-9a-zA-Z]+=[0-9a-zA-Z]+(:[0-9a-zA-Z]+=[0-9a-zA-Z]+)*$`)
)

// HTTPServer exposes the gitrpc rest api.
type HTTPServer struct {
	*gitnesshttp.Server
}

func NewHTTPServer(config Config) (*HTTPServer, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration is invalid: %w", err)
	}

	reposRoot := filepath.Join(config.GitRoot, repoSubdirName)

	return &HTTPServer{
		gitnesshttp.NewServer(
			gitnesshttp.Config{
				Addr: config.HTTP.Bind,
			},
			handleHTTP(reposRoot),
		),
	}, nil
}

func handleHTTP(reposRoot string) http.Handler {
	r := chi.NewRouter()

	// Apply common api middleware.
	r.Use(middleware.NoCache)
	r.Use(middleware.Recoverer)

	// configure logging middleware.
	log := log.Logger.With().Logger()
	r.Use(hlog.NewHandler(log))
	r.Use(hlog.URLHandler("http.url"))
	r.Use(hlog.MethodHandler("http.method"))
	r.Use(HLogRequestIDHandler())
	r.Use(HLogAccessLogHandler())

	r.Route(fmt.Sprintf("/{%s}", PathParamRepoUID), func(r chi.Router) {
		r.Get("/info/refs", handleHTTPInfoRefs(reposRoot))
		r.Handle("/git-upload-pack", handleHTTPUploadPack(reposRoot))

		// push is not supported
		r.Post("/git-receive-pack", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotImplemented)
			_, _ = w.Write([]byte("receive pack is not supported by this endpoint"))
		})
	})

	return r
}

func handleHTTPInfoRefs(reposRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// Clients MUST NOT reuse or revalidate a cached response.
		// Servers MUST include sufficient Cache-Control headers to prevent caching of the response.
		// https://git-scm.com/docs/http-protocol
		setHeaderNoCache(w)

		repoUID := chi.URLParam(r, PathParamRepoUID)
		repoPath := getFullPathForRepo(reposRoot, repoUID)
		gitProtocol := r.Header.Get("Git-Protocol")
		service := getServiceType(r)

		log.Ctx(ctx).Trace().Msgf(
			"handleHTTPInfoRefs for git service: '%s', protocol: '%s', path: '%s'",
			service,
			gitProtocol,
			repoPath,
		)

		w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service))

		// NOTE: Don't include os.Environ() as we don't have control over it - define everything explicitly
		environ := []string{}
		if gitProtocol != "" {
			environ = append(environ, "GIT_PROTOCOL="+gitProtocol)
		}

		stdOut := &bytes.Buffer{}
		if err := git.NewCommand(ctx, service, "--stateless-rpc", "--advertise-refs", ".").
			Run(&git.RunOpts{
				Env:    environ,
				Dir:    repoPath,
				Stdout: stdOut,
			}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			log.Ctx(ctx).Error().Err(err).Msgf("failed running git command")

			return
		}
		if _, err := w.Write(packetWrite("# service=git-" + service + "\n")); err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			log.Ctx(ctx).Error().Err(err).Msgf("failed writing packet line")

			return
		}

		if _, err := w.Write([]byte("0000")); err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			log.Ctx(ctx).Error().Err(err).Msgf("failed writing end of response")

			return
		}

		if _, err := io.Copy(w, stdOut); err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			log.Ctx(ctx).Warn().Err(err).Msgf("failed copying response body")

			return
		}
	}
}

func handleHTTPUploadPack(reposRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		const service = "upload-pack"
		repoUID := chi.URLParam(r, PathParamRepoUID)
		repoPath := getFullPathForRepo(reposRoot, repoUID)
		gitProtocol := r.Header.Get("Git-Protocol")

		log.Ctx(ctx).Trace().Msgf(
			"handleHTTPUploadPack for git service: '%s', protocol: '%s', path: '%s'",
			service,
			gitProtocol,
			repoPath,
		)

		w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", service))

		var err error
		reqBody := r.Body

		// Handle GZIP.
		if r.Header.Get("Content-Encoding") == "gzip" {
			reqBody, err = gzip.NewReader(reqBody)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Ctx(ctx).Error().Err(err).Msgf("failed gziping response body")
				return
			}
		}

		// NOTE: Don't include os.Environ() as we don't have control over it - define everything explicitly
		environ := []string{}
		// set this for allow pre-receive and post-receive execute
		environ = append(environ, "SSH_ORIGINAL_COMMAND="+service)
		if gitProtocol != "" && safeGitProtocolHeader.MatchString(gitProtocol) {
			environ = append(environ, "GIT_PROTOCOL="+gitProtocol)
		}

		var (
			stderr bytes.Buffer
		)
		cmd := git.NewCommand(ctx, service, "--stateless-rpc", repoPath)
		cmd.SetDescription(fmt.Sprintf("%s %s %s [repo_path: %s]", git.GitExecutable, service, "--stateless-rpc", repoPath))
		err = cmd.Run(&git.RunOpts{
			Dir:               repoPath,
			Env:               environ,
			Stdout:            w,
			Stdin:             reqBody,
			Stderr:            &stderr,
			UseContextTimeout: true,
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("Failed to serve RPC(%s) in %s: %v - %s", service, repoPath, err, stderr.String())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func setHeaderNoCache(w http.ResponseWriter) {
	w.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
}

func getServiceType(r *http.Request) string {
	serviceType := r.URL.Query().Get("service")
	if !strings.HasPrefix(serviceType, "git-") {
		return ""
	}
	return strings.Replace(serviceType, "git-", "", 1)
}

// getFullPathForRepo returns the full path of a repo given the root dir of repos and the uid of the repo.
// NOTE: Split repos into subfolders using their prefix to distribute repos across a set of folders.
// TODO: Use common function between grpc and git server
func getFullPathForRepo(reposRoot, uid string) string {
	// ASSUMPTION: repoUID is of lenth at least 4 - otherwise we have trouble either way.
	return filepath.Join(
		reposRoot,                            // root folder
		uid[0:2],                             // first subfolder
		uid[2:4],                             // second subfolder
		fmt.Sprintf("%s.%s", uid[4:], "git"), // remainder with .git
	)
}

func packetWrite(str string) []byte {
	s := strconv.FormatInt(int64(len(str)+4), 16)
	if len(s)%4 != 0 {
		s = strings.Repeat("0", 4-len(s)%4) + s
	}
	return []byte(s + str)
}
