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

package webhook

import (
	"time"

	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

/*
 * The idea of segments is to expose similar fields using the same structure.
 * This makes consumption on webhook payloads easier as we ensure related webhooks have similar payload formats.
 * Segments are meant to be embedded, while Infos are meant to be used as fields.
 */

// BaseSegment contains base info of all payloads for webhooks.
type BaseSegment struct {
	Trigger   enum.WebhookTrigger `json:"trigger"`
	Repo      RepositoryInfo      `json:"repo"`
	Principal PrincipalInfo       `json:"principal"`
}

// ReferenceSegment contains the reference info for webhooks.
type ReferenceSegment struct {
	Ref ReferenceInfo `json:"ref"`
}

// ReferenceDetailsSegment contains extra defails for reference related payloads for webhooks.
type ReferenceDetailsSegment struct {
	SHA    string      `json:"sha"`
	Commit *CommitInfo `json:"commit,omitempty"`
}

// ReferenceUpdateSegment contains extra details for reference update related payloads for webhooks.
type ReferenceUpdateSegment struct {
	OldSHA string `json:"old_sha"`
	Forced bool   `json:"forced"`
}

// PullReqTargetReferenceSegment contains details for the pull req target reference for webhooks.
type PullReqTargetReferenceSegment struct {
	TargetRef ReferenceInfo `json:"target_ref"`
}

// PullReqSegment contains details for all pull req related payloads for webhooks.
type PullReqSegment struct {
	PullReq PullReqInfo `json:"pull_req"`
}

// PullReqCommentSegment contains details for all pull req comment related payloads for webhooks.
type PullReqCommentSegment struct {
	CommentInfo CommentInfo `json:"comment"`
}

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
func repositoryInfoFrom(repo *types.Repository, urlProvider url.Provider) RepositoryInfo {
	return RepositoryInfo{
		ID:            repo.ID,
		Path:          repo.Path,
		UID:           repo.UID,
		DefaultBranch: repo.DefaultBranch,
		GitURL:        urlProvider.GenerateGITCloneURL(repo.Path),
	}
}

// PullReqInfo describes the pullreq related info for a webhook payload.
// NOTE: don't use types package as we want pullreq payload to be independent from API calls.
type PullReqInfo struct {
	Number        int64             `json:"number"`
	State         enum.PullReqState `json:"state"`
	IsDraft       bool              `json:"is_draft"`
	Title         string            `json:"title"`
	SourceRepoID  int64             `json:"source_repo_id"`
	SourceBranch  string            `json:"source_branch"`
	TargetRepoID  int64             `json:"target_repo_id"`
	TargetBranch  string            `json:"target_branch"`
	MergeStrategy *enum.MergeMethod `json:"merge_strategy"`
	Author        PrincipalInfo     `json:"author"`
}

// pullReqInfoFrom gets the PullReqInfo from a types.PullReq.
func pullReqInfoFrom(pr *types.PullReq) PullReqInfo {
	return PullReqInfo{
		Number:        pr.Number,
		State:         pr.State,
		IsDraft:       pr.IsDraft,
		Title:         pr.Title,
		SourceRepoID:  pr.SourceRepoID,
		SourceBranch:  pr.SourceBranch,
		TargetRepoID:  pr.TargetRepoID,
		TargetBranch:  pr.TargetBranch,
		MergeStrategy: pr.MergeMethod,
		Author:        principalInfoFrom(&pr.Author),
	}
}

// PrincipalInfo describes the principal related info for a webhook payload.
// NOTE: don't use types package as we want webhook payload to be independent from API calls.
type PrincipalInfo struct {
	ID          int64              `json:"id"`
	UID         string             `json:"uid"`
	DisplayName string             `json:"display_name"`
	Email       string             `json:"email"`
	Type        enum.PrincipalType `json:"type"`
	Created     int64              `json:"created"`
	Updated     int64              `json:"updated"`
}

// principalInfoFrom gets the PrincipalInfo from a types.Principal.
func principalInfoFrom(principal *types.PrincipalInfo) PrincipalInfo {
	return PrincipalInfo{
		ID:          principal.ID,
		UID:         principal.UID,
		DisplayName: principal.DisplayName,
		Email:       principal.Email,
		Type:        principal.Type,
		Created:     principal.Created,
		Updated:     principal.Updated,
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

// ReferenceInfo describes a unique reference in gitness.
// It contains both the reference name as well as the repo the reference belongs to.
type ReferenceInfo struct {
	Name string         `json:"name"`
	Repo RepositoryInfo `json:"repo"`
}

type CommentInfo struct {
	Text string `json:"text"`
	ID   int64  `json:"id"`
}
