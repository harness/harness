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

package types

import (
	"time"

	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types/enum"
)

const NilSHA = "0000000000000000000000000000000000000000"

// PaginationFilter stores pagination query parameters.
type PaginationFilter struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// CommitFilter stores commit query parameters.
type CommitFilter struct {
	PaginationFilter
	After        string  `json:"after"`
	Path         string  `json:"path"`
	Since        int64   `json:"since"`
	Until        int64   `json:"until"`
	Committer    string  `json:"committer"`
	CommitterIDs []int64 `json:"committer_ids"`
	Author       string  `json:"author"`
	AuthorIDs    []int64 `json:"author_ids"`
	IncludeStats bool    `json:"include_stats"`
}

type BranchMetadataOptions struct {
	IncludeChecks   bool `json:"include_checks"`
	IncludeRules    bool `json:"include_rules"`
	IncludePullReqs bool `json:"include_pullreqs"`
	MaxDivergence   int  `json:"max_divergence"`
}

// BranchFilter stores branch query parameters.
type BranchFilter struct {
	Query         string                `json:"query"`
	Sort          enum.BranchSortOption `json:"sort"`
	Order         enum.Order            `json:"order"`
	Page          int                   `json:"page"`
	Size          int                   `json:"size"`
	IncludeCommit bool                  `json:"include_commit"`
	BranchMetadataOptions
}

// TagFilter stores commit tag query parameters.
type TagFilter struct {
	Query string             `json:"query"`
	Sort  enum.TagSortOption `json:"sort"`
	Order enum.Order         `json:"order"`
	Page  int                `json:"page"`
	Size  int                `json:"size"`
}

type ChangeStats struct {
	Insertions int64 `json:"insertions"`
	Deletions  int64 `json:"deletions"`
	Changes    int64 `json:"changes"`
}

type CommitFileStats struct {
	Path    string                 `json:"path"`
	OldPath string                 `json:"old_path,omitempty"`
	Status  gitenum.FileDiffStatus `json:"status"`
	ChangeStats
}

type CommitStats struct {
	Total ChangeStats       `json:"total,omitempty"`
	Files []CommitFileStats `json:"files,omitempty"`
}

type Commit struct {
	SHA        sha.SHA      `json:"sha"`
	TreeSHA    sha.SHA      `json:"-"`
	ParentSHAs []sha.SHA    `json:"parent_shas,omitempty"`
	Title      string       `json:"title"`
	Message    string       `json:"message"`
	Author     Signature    `json:"author"`
	Committer  Signature    `json:"committer"`
	SignedData *SignedData  `json:"-"`
	Stats      *CommitStats `json:"stats,omitempty"`

	Signature *GitSignatureResult `json:"signature"`
}

func (c *Commit) GetSHA() sha.SHA                      { return c.SHA }
func (c *Commit) SetSignature(sig *GitSignatureResult) { c.Signature = sig }
func (c *Commit) GetSigner() *Signature                { return &c.Committer }
func (c *Commit) GetSignedData() *SignedData           { return c.SignedData }

type CommitTag struct {
	Name        string      `json:"name"`
	SHA         sha.SHA     `json:"sha"`
	IsAnnotated bool        `json:"is_annotated"`
	Title       string      `json:"title,omitempty"`
	Message     string      `json:"message,omitempty"`
	Tagger      *Signature  `json:"tagger,omitempty"`
	SignedData  *SignedData `json:"-"`
	Commit      *Commit     `json:"commit,omitempty"`

	Signature *GitSignatureResult `json:"signature"`
}

func (t *CommitTag) GetSHA() sha.SHA                      { return t.SHA }
func (t *CommitTag) SetSignature(sig *GitSignatureResult) { t.Signature = sig }
func (t *CommitTag) GetSigner() *Signature                { return t.Tagger }
func (t *CommitTag) GetSignedData() *SignedData           { return t.SignedData }

type Signature struct {
	Identity Identity  `json:"identity"`
	When     time.Time `json:"when"`
}

type Identity struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type SignedData struct {
	Type          string
	Signature     []byte
	SignedContent []byte
}

type RenameDetails struct {
	OldPath         string `json:"old_path"`
	NewPath         string `json:"new_path"`
	CommitShaBefore string `json:"commit_sha_before"`
	CommitShaAfter  string `json:"commit_sha_after"`
}

type ListCommitResponse struct {
	Commits       []*Commit       `json:"commits"`
	RenameDetails []RenameDetails `json:"rename_details"`
	TotalCommits  int             `json:"total_commits,omitempty"`
}

type GitSignatureResult struct {
	RepoID     int64   `json:"-"`
	ObjectSHA  sha.SHA `json:"-"`
	ObjectTime int64   `json:"-"`

	// Created is the timestamp when the signature was first verified.
	Created int64 `json:"created,omitempty"`

	// Updated is the timestamp when result has been updated (i.e. because of key revocation).
	Updated int64 `json:"updated,omitempty"`

	// Result is the result of the signature verification.
	Result enum.GitSignatureResult `json:"result"`

	// PrincipalID is owner of the key with which signature has been checked.
	PrincipalID int64 `json:"-"`

	KeyScheme enum.PublicKeyScheme `json:"key_scheme,omitempty"`

	// KeyID is the ID of the key with which signature has been checked.
	KeyID string `json:"key_id,omitempty"`

	// KeyFingerprint is the fingerprint of the key with which signature has been checked.
	KeyFingerprint string `json:"key_fingerprint,omitempty"`
}
