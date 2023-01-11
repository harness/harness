// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"time"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types"
)

// RepositoryInfo describes the repo related info for a webhook payload.
// NOTE: don't use types package as we want webhook payload to be independent from API calls.
type RepositoryInfo struct {
	ID            int64  `json:"id"`
	Path          string `json:"path"`
	UID           string `json:"uid"`
	DefaultBranch string `json:"default_branch"`
	GitURL        string `json:"git_url"`
}

// repositoryInfoFrom gets the RespositoryInfo from a types.Repository.
func repositoryInfoFrom(repo types.Repository, urlProvider *url.Provider) RepositoryInfo {
	return RepositoryInfo{
		ID:            repo.ID,
		Path:          repo.Path,
		UID:           repo.UID,
		DefaultBranch: repo.DefaultBranch,
		GitURL:        urlProvider.GenerateRepoCloneURL(repo.Path),
	}
}

// PrincipalInfo describes the principal related info for a webhook payload.
// NOTE: don't use types package as we want webhook payload to be independent from API calls.
type PrincipalInfo struct {
	ID          int64  `json:"id"`
	UID         string `json:"uid"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

// principalInfoFrom gets the PrincipalInfo from a types.Principal.
func principalInfoFrom(principal types.Principal) PrincipalInfo {
	return PrincipalInfo{
		ID:          principal.ID,
		UID:         principal.UID,
		DisplayName: principal.DisplayName,
		Email:       principal.Email,
	}
}

// CommitInfo describes the commit related info for a webhook payload.
// NOTE: don't use types package as we want webhook payload to be independent from API calls.
type CommitInfo struct {
	SHA       string        `json:"sha"`
	Message   string        `json:"message"`
	Author    SignatureInfo `json:"author"`
	Committer SignatureInfo `json:"committer"`
}

// commitInfoFrom gets the CommitInfo from a gitrpc.Commit.
func commitInfoFrom(commit gitrpc.Commit) CommitInfo {
	return CommitInfo{
		SHA:       commit.SHA,
		Message:   commit.Message,
		Author:    signatureInfoFrom(commit.Author),
		Committer: signatureInfoFrom(commit.Committer),
	}
}

// SignatureInfo describes the commit signature related info for a webhook payload.
// NOTE: don't use types package as we want webhook payload to be independent from API calls.
type SignatureInfo struct {
	Identity IdentityInfo `json:"identity"`
	When     time.Time    `json:"when"`
}

// signatureInfoFrom gets the SignatureInfo from a gitrpc.Signature.
func signatureInfoFrom(signature gitrpc.Signature) SignatureInfo {
	return SignatureInfo{
		Identity: identityInfoFrom(signature.Identity),
		When:     signature.When,
	}
}

// IdentityInfo describes the signature identity related info for a webhook payload.
// NOTE: don't use types package as we want webhook payload to be independent from API calls.
type IdentityInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// identityInfoFrom gets the IdentityInfo from a gitrpc.Identity.
func identityInfoFrom(identity gitrpc.Identity) IdentityInfo {
	return IdentityInfo{
		Name:  identity.Name,
		Email: identity.Email,
	}
}
