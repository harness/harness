// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/harness/gitness/internal/gitrpc"
	"github.com/harness/gitness/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

type CtxRepoType string

const (
	CtxRepoKey CtxRepoType = "repo"
)

func GetInfoRefs(client gitrpc.Interface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := hlog.FromRequest(r)
		repo, ok := r.Context().Value(CtxRepoKey).(*types.Repository)
		if !ok {
			ctxKeyError(w, log)
			return
		}

		// Clients MUST NOT reuse or revalidate a cached response.
		// Servers MUST include sufficient Cache-Control headers to prevent caching of the response.
		// https://git-scm.com/docs/http-protocol
		setHeaderNoCache(w)

		service := getServiceType(r)
		log.Debug().Msgf("in GetInfoRefs: git service: %v", service)
		w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service))

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if err := client.GetInfoRefs(ctx, w, &gitrpc.InfoRefsParams{
			RepoUID:     repo.GitUID,
			Service:     service,
			Options:     nil,
			GitProtocol: r.Header.Get("Git-Protocol"),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Err(err).Msgf("in GetInfoRefs: error occurred in service %v", service)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func GetUploadPack(client gitrpc.Interface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const service = "upload-pack"
		log := hlog.FromRequest(r)
		repo, ok := r.Context().Value(CtxRepoKey).(*types.Repository)
		if !ok {
			ctxKeyError(w, log)
			return
		}

		if err := serviceRPC(w, r, client, repo.GitUID, service, repo.CreatedBy); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func PostReceivePack(client gitrpc.Interface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const service = "receive-pack"
		log := hlog.FromRequest(r)
		repo, ok := r.Context().Value(CtxRepoKey).(*types.Repository)
		if !ok {
			ctxKeyError(w, log)
			return
		}

		if err := serviceRPC(w, r, client, repo.GitUID, service, repo.CreatedBy); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func serviceRPC(
	w http.ResponseWriter,
	r *http.Request,
	client gitrpc.Interface,
	repo, service string,
	principalID int64,
) error {
	ctx := r.Context()
	log := hlog.FromRequest(r)
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Err(err).Msgf("serviceRPC: Close: %v", err)
		}
	}()

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", service))

	var err error
	reqBody := r.Body

	// Handle GZIP.
	if r.Header.Get("Content-Encoding") == "gzip" {
		reqBody, err = gzip.NewReader(reqBody)
		if err != nil {
			return err
		}
	}
	return client.ServicePack(ctx, w, &gitrpc.ServicePackParams{
		RepoUID:     repo,
		Service:     service,
		Data:        reqBody,
		Options:     nil,
		PrincipalID: principalID,
		GitProtocol: r.Header.Get("Git-Protocol"),
	})
}

func setHeaderNoCache(w http.ResponseWriter) {
	w.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
}

func getServiceType(r *http.Request) string {
	serviceType := r.FormValue("service")
	if !strings.HasPrefix(serviceType, "git-") {
		return ""
	}
	return strings.Replace(serviceType, "git-", "", 1)
}

func ctxKeyError(w http.ResponseWriter, log *zerolog.Logger) {
	errMsg := "key 'repo' missing in context"
	http.Error(w, errMsg, http.StatusBadRequest)
	log.Error().Msg(errMsg)
}
