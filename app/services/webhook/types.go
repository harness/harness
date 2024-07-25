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
	"context"
	"encoding/json"
	"time"

	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
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

// ReferenceDetailsSegment contains extra details for reference related payloads for webhooks.
type ReferenceDetailsSegment struct {
	SHA string `json:"sha"`

	HeadCommit *CommitInfo `json:"head_commit,omitempty"`

	Commits           *[]CommitInfo `json:"commits,omitempty"`
	TotalCommitsCount int           `json:"total_commits_count,omitempty"`

	// Deprecated
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

// PullReqUpdateSegment contains details what has been updated in the pull request.
type PullReqUpdateSegment struct {
	TitleChanged       bool   `json:"title_changed"`
	TitleOld           string `json:"title_old"`
	TitleNew           string `json:"title_new"`
	DescriptionChanged bool   `json:"description_changed"`
	DescriptionOld     string `json:"description_old"`
	DescriptionNew     string `json:"description_new"`
}

// RepositoryInfo describes the repo related info for a webhook payload.
// NOTE: don't use types package as we want webhook payload to be independent from API calls.
type RepositoryInfo struct {
	ID            int64  `json:"id"`
	Path          string `json:"path"`
	Identifier    string `json:"identifier"`
	DefaultBranch string `json:"default_branch"`
	GitURL        string `json:"git_url"`
	GitSSHURL     string `json:"git_ssh_url"`
}

// TODO [CODE-1363]: remove after identifier migration.
func (r RepositoryInfo) MarshalJSON() ([]byte, error) {
	// alias allows us to embed the original object while avoiding an infinite loop of marshaling.
	type alias RepositoryInfo
	return json.Marshal(&struct {
		alias
		UID string `json:"uid"`
	}{
		alias: (alias)(r),
		UID:   r.Identifier,
	})
}

// repositoryInfoFrom gets the RespositoryInfo from a types.Repository.
func repositoryInfoFrom(ctx context.Context, repo *types.Repository, urlProvider url.Provider) RepositoryInfo {
	return RepositoryInfo{
		ID:            repo.ID,
		Path:          repo.Path,
		Identifier:    repo.Identifier,
		DefaultBranch: repo.DefaultBranch,
		GitURL:        urlProvider.GenerateGITCloneURL(ctx, repo.Path),
		GitSSHURL:     urlProvider.GenerateGITCloneSSHURL(ctx, repo.Path),
	}
}

// PullReqInfo describes the pullreq related info for a webhook payload.
// NOTE: don't use types package as we want pullreq payload to be independent from API calls.
type PullReqInfo struct {
	Number        int64             `json:"number"`
	State         enum.PullReqState `json:"state"`
	IsDraft       bool              `json:"is_draft"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	SourceRepoID  int64             `json:"source_repo_id"`
	SourceBranch  string            `json:"source_branch"`
	TargetRepoID  int64             `json:"target_repo_id"`
	TargetBranch  string            `json:"target_branch"`
	MergeStrategy *enum.MergeMethod `json:"merge_strategy,omitempty"`
	Author        PrincipalInfo     `json:"author"`
	PrURL         string            `json:"pr_url"`
}

// pullReqInfoFrom gets the PullReqInfo from a types.PullReq.
func pullReqInfoFrom(
	ctx context.Context,
	pr *types.PullReq,
	repo *types.Repository,
	urlProvider url.Provider,
) PullReqInfo {
	return PullReqInfo{
		Number:        pr.Number,
		State:         pr.State,
		IsDraft:       pr.IsDraft,
		Title:         pr.Title,
		Description:   pr.Description,
		SourceRepoID:  pr.SourceRepoID,
		SourceBranch:  pr.SourceBranch,
		TargetRepoID:  pr.TargetRepoID,
		TargetBranch:  pr.TargetBranch,
		MergeStrategy: pr.MergeMethod,
		Author:        principalInfoFrom(&pr.Author),
		PrURL:         urlProvider.GenerateUIPRURL(ctx, repo.Path, pr.Number),
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

	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Modified []string `json:"modified"`
}

// commitInfoFrom gets the CommitInfo from a git.Commit.
func commitInfoFrom(commit git.Commit) CommitInfo {
	added := []string{}
	removed := []string{}
	modified := []string{}

	for _, stat := range commit.FileStats {
		switch {
		case stat.Status == gitenum.FileDiffStatusModified:
			modified = append(modified, stat.Path)
		case stat.Status == gitenum.FileDiffStatusRenamed:
			added = append(added, stat.Path)
			removed = append(removed, stat.OldPath)
		case stat.Status == gitenum.FileDiffStatusDeleted:
			removed = append(removed, stat.Path)
		case stat.Status == gitenum.FileDiffStatusAdded || stat.Status == gitenum.FileDiffStatusCopied:
			added = append(added, stat.Path)
		case stat.Status == gitenum.FileDiffStatusUndefined:
		default:
			log.Warn().Msgf("unknown status %q for path %q", stat.Status, stat.Path)
		}
	}

	return CommitInfo{
		SHA:       commit.SHA.String(),
		Message:   commit.Message,
		Author:    signatureInfoFrom(commit.Author),
		Committer: signatureInfoFrom(commit.Committer),
		Added:     added,
		Removed:   removed,
		Modified:  modified,
	}
}

// commitsInfoFrom gets the ExtendedCommitInfo from a []git.Commit.
func commitsInfoFrom(commits []git.Commit) []CommitInfo {
	commitsInfo := make([]CommitInfo, len(commits))
	for i, commit := range commits {
		commitsInfo[i] = commitInfoFrom(commit)
	}
	return commitsInfo
}

// SignatureInfo describes the commit signature related info for a webhook payload.
// NOTE: don't use types package as we want webhook payload to be independent from API calls.
type SignatureInfo struct {
	Identity IdentityInfo `json:"identity"`
	When     time.Time    `json:"when"`
}

// signatureInfoFrom gets the SignatureInfo from a git.Signature.
func signatureInfoFrom(signature git.Signature) SignatureInfo {
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

// identityInfoFrom gets the IdentityInfo from a git.Identity.
func identityInfoFrom(identity git.Identity) IdentityInfo {
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
	ID       int64  `json:"id"`
	ParentID *int64 `json:"parent_id,omitempty"`
	Text     string `json:"text"`
}
