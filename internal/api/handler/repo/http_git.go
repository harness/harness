// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"compress/gzip"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	repoctrl "github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

type CtxRepoType string

type GitAuthError struct {
	AccountID string
}

func (e GitAuthError) Error() string {
	return fmt.Sprintf("Authentication failed for account %s", e.AccountID)
}

func GetInfoRefs(client gitrpc.Interface, repoStore store.RepoStore, authorizer authz.Authorizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			http.Error(w, usererror.Translate(err).Error(), http.StatusInternalServerError)
			return
		}

		repo, err := repoStore.FindRepoFromRef(ctx, repoRef)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		accountID, _, err := paths.DisectRoot(repo.Path)
		if err != nil {
			return
		}

		if err = apiauth.CheckRepo(ctx, authorizer, session, repo, enum.PermissionRepoView, true); err != nil {
			if errors.Is(err, apiauth.ErrNotAuthenticated) {
				basicAuth(w, accountID)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Clients MUST NOT reuse or revalidate a cached response.
		// Servers MUST include sufficient Cache-Control headers to prevent caching of the response.
		// https://git-scm.com/docs/http-protocol
		setHeaderNoCache(w)

		service := getServiceType(r)
		log.Debug().Msgf("in GetInfoRefs: git service: %v", service)
		w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service))

		if err = client.GetInfoRefs(ctx, w, &gitrpc.InfoRefsParams{
			ReadParams:  repoctrl.CreateRPCReadParams(repo),
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

func GetUploadPack(client gitrpc.Interface, urlProvider *url.Provider,
	repoStore store.RepoStore, authorizer authz.Authorizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const service = "upload-pack"

		if err := serviceRPC(w, r, client, urlProvider, repoStore, authorizer, service, false,
			enum.PermissionRepoView, true); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func PostReceivePack(client gitrpc.Interface, urlProvider *url.Provider,
	repoStore store.RepoStore, authorizer authz.Authorizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const service = "receive-pack"
		if err := serviceRPC(w, r, client, urlProvider, repoStore, authorizer, service, true,
			enum.PermissionRepoEdit, false); err != nil {
			var authError *GitAuthError
			if errors.As(err, &authError) {
				basicAuth(w, authError.AccountID)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func serviceRPC(
	w http.ResponseWriter,
	r *http.Request,
	client gitrpc.Interface,
	urlProvider *url.Provider,
	repoStore store.RepoStore,
	authorizer authz.Authorizer,
	service string,
	isWriteOperation bool,
	permission enum.Permission,
	orPublic bool,
) error {
	ctx := r.Context()
	log := hlog.FromRequest(r)
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Err(err).Msgf("serviceRPC: Close: %v", err)
		}
	}()

	session, _ := request.AuthSessionFrom(ctx)
	repoRef, err := request.GetRepoRefFromPath(r)
	if err != nil {
		return err
	}

	repo, err := repoStore.FindRepoFromRef(ctx, repoRef)
	if err != nil {
		return err
	}

	accountID, _, err := paths.DisectRoot(repo.Path)
	if err != nil {
		return err
	}

	if err = apiauth.CheckRepo(ctx, authorizer, session, repo, permission, orPublic); err != nil {
		if errors.Is(err, apiauth.ErrNotAuthenticated) {
			return &GitAuthError{
				AccountID: accountID,
			}
		}
		return err
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", service))

	reqBody := r.Body

	// Handle GZIP.
	if r.Header.Get("Content-Encoding") == "gzip" {
		reqBody, err = gzip.NewReader(reqBody)
		if err != nil {
			return err
		}
	}
	params := &gitrpc.ServicePackParams{
		Service:     service,
		Data:        reqBody,
		Options:     nil,
		GitProtocol: r.Header.Get("Git-Protocol"),
	}

	// setup read/writeparams depending on whether it's a write operation
	if isWriteOperation {
		var writeParams gitrpc.WriteParams
		writeParams, err = repoctrl.CreateRPCWriteParams(ctx, urlProvider, session, repo)
		if err != nil {
			return fmt.Errorf("failed to create RPC write params: %w", err)
		}
		params.WriteParams = &writeParams
	} else {
		readParams := repoctrl.CreateRPCReadParams(repo)
		params.ReadParams = &readParams
	}

	return client.ServicePack(ctx, w, params)
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

func basicAuth(w http.ResponseWriter, accountID string) {
	w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, accountID))
	w.WriteHeader(http.StatusUnauthorized)
}
