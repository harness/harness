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
	"fmt"
	"io"
	"time"

	"github.com/harness/gitness/errors"
)

const NilSHA = "0000000000000000000000000000000000000000"

type CloneRepoOptions struct {
	Timeout       time.Duration
	Mirror        bool
	Bare          bool
	Quiet         bool
	Branch        string
	Shared        bool
	NoCheckout    bool
	Depth         int
	Filter        string
	SkipTLSVerify bool
}

type SortOrder int

const (
	SortOrderDefault SortOrder = iota
	SortOrderAsc
	SortOrderDesc
)

type GitObjectType string

const (
	GitObjectTypeCommit GitObjectType = "commit"
	gitObjectTypeTree   GitObjectType = "tree"
	gitObjectTypeBlob   GitObjectType = "Blob"
	GitObjectTypeTag    GitObjectType = "tag"
)

func ParseGitObjectType(t string) (GitObjectType, error) {
	switch t {
	case string(GitObjectTypeCommit):
		return GitObjectTypeCommit, nil
	case string(gitObjectTypeBlob):
		return gitObjectTypeBlob, nil
	case string(gitObjectTypeTree):
		return gitObjectTypeTree, nil
	case string(GitObjectTypeTag):
		return GitObjectTypeTag, nil
	default:
		return gitObjectTypeBlob, fmt.Errorf("unknown git object type '%s'", t)
	}
}

// GitReferenceField represents the different fields available When listing references.
// For the full list, see https://git-scm.com/docs/git-for-each-ref#_field_names
type GitReferenceField string

const (
	GitReferenceFieldRefName     GitReferenceField = "refname"
	GitReferenceFieldObjectType  GitReferenceField = "objecttype"
	GitReferenceFieldObjectName  GitReferenceField = "objectname"
	GitReferenceFieldCreatorDate GitReferenceField = "creatordate"
)

func ParseGitReferenceField(f string) (GitReferenceField, error) {
	switch f {
	case string(GitReferenceFieldCreatorDate):
		return GitReferenceFieldCreatorDate, nil
	case string(GitReferenceFieldRefName):
		return GitReferenceFieldRefName, nil
	case string(GitReferenceFieldObjectName):
		return GitReferenceFieldObjectName, nil
	case string(GitReferenceFieldObjectType):
		return GitReferenceFieldObjectType, nil
	default:
		return GitReferenceFieldRefName, fmt.Errorf("unknown git reference field '%s'", f)
	}
}

type WalkInstruction int

const (
	WalkInstructionStop WalkInstruction = iota
	WalkInstructionHandle
	WalkInstructionSkip
)

type WalkReferencesEntry map[GitReferenceField]string

// TODO: can be generic (so other walk methods can use the same)
type WalkReferencesInstructor func(WalkReferencesEntry) (WalkInstruction, error)

// TODO: can be generic (so other walk methods can use the same)
type WalkReferencesHandler func(WalkReferencesEntry) error

type WalkReferencesOptions struct {
	// Patterns are the patterns used to pre-filter the references of the repo.
	// OPTIONAL. By default all references are walked.
	Patterns []string

	// Fields indicates the fields that are passed to the instructor & handler
	// OPTIONAL. Default fields are:
	// - GitReferenceFieldRefName
	// - GitReferenceFieldObjectName
	Fields []GitReferenceField

	// Instructor indicates on how to handle the reference.
	// OPTIONAL. By default all references are handled.
	// NOTE: once walkInstructionStop is returned, the walking stops.
	Instructor WalkReferencesInstructor

	// Sort indicates the field by which the references should be sorted.
	// OPTIONAL. By default GitReferenceFieldRefName is used.
	Sort GitReferenceField

	// Order indicates the Order (asc or desc) of the sorted output
	Order SortOrder

	// MaxWalkDistance is the maximum number of nodes that are iterated over before the walking stops.
	// OPTIONAL. A value of <= 0 will walk all references.
	// WARNING: Skipped elements count towards the walking distance
	MaxWalkDistance int32
}

type Commit struct {
	SHA        string          `json:"sha"`
	ParentSHAs []string        `json:"parent_shas,omitempty"`
	Title      string          `json:"title"`
	Message    string          `json:"message,omitempty"`
	Author     Signature       `json:"author"`
	Committer  Signature       `json:"committer"`
	FileStats  CommitFileStats `json:"file_stats,omitempty"`
}

type CommitFileStats struct {
	Added    []string
	Modified []string
	Removed  []string
}

type Branch struct {
	Name   string
	SHA    string
	Commit *Commit
}

type BranchFilter struct {
	Query         string
	Page          int32
	PageSize      int32
	Sort          GitReferenceField
	Order         SortOrder
	IncludeCommit bool
}

type Tag struct {
	Sha        string
	Name       string
	TargetSha  string
	TargetType GitObjectType
	Title      string
	Message    string
	Tagger     Signature
}

type CreateTagOptions struct {
	// Message is the optional message the tag will be created with - if the message is empty
	// the tag will be lightweight, otherwise it'll be annotated.
	Message string

	// Tagger is the information used in case the tag is annotated (Message is provided).
	Tagger Signature
}

// Signature represents the Author or Committer information.
type Signature struct {
	Identity Identity
	// When is the timestamp of the Signature.
	When time.Time
}

type Identity struct {
	Name  string
	Email string
}

func (i Identity) String() string {
	return fmt.Sprintf("%s <%s>", i.Name, i.Email)
}

func (i *Identity) Validate() error {
	if i.Name == "" {
		return errors.InvalidArgument("identity name is mandatory")
	}

	if i.Email == "" {
		return errors.InvalidArgument("identity email is mandatory")
	}

	return nil
}

type CommitChangesOptions struct {
	Committer Signature
	Author    Signature
	Message   string
}

type PushOptions struct {
	Remote         string
	Branch         string
	Force          bool
	ForceWithLease string
	Env            []string
	Timeout        time.Duration
	Mirror         bool
}

type TreeNodeWithCommit struct {
	TreeNode
	Commit *Commit
}

type TreeNode struct {
	NodeType TreeNodeType
	Mode     TreeNodeMode
	Sha      string
	Name     string
	Path     string
}

// TreeNodeType specifies the different types of nodes in a git tree.
// IMPORTANT: has to be consistent with rpc.TreeNodeType (proto).
type TreeNodeType int

const (
	TreeNodeTypeTree TreeNodeType = iota
	TreeNodeTypeBlob
	TreeNodeTypeCommit
)

// TreeNodeMode specifies the different modes of a node in a git tree.
// IMPORTANT: has to be consistent with rpc.TreeNodeMode (proto).
type TreeNodeMode int

const (
	TreeNodeModeFile TreeNodeMode = iota
	TreeNodeModeSymlink
	TreeNodeModeExec
	TreeNodeModeTree
	TreeNodeModeCommit
)

type Submodule struct {
	Name string
	URL  string
}

type BlobReader struct {
	SHA string
	// Size is the actual size of the blob.
	Size int64
	// ContentSize is the total number of bytes returned by the Content Reader.
	ContentSize int64
	// Content contains the (partial) content of the blob.
	Content io.ReadCloser
}

// CommitDivergenceRequest contains the refs for which the converging commits should be counted.
type CommitDivergenceRequest struct {
	// From is the ref from which the counting of the diverging commits starts.
	From string
	// To is the ref at which the counting of the diverging commits ends.
	To string
}

// CommitDivergence contains the information of the count of converging commits between two refs.
type CommitDivergence struct {
	// Ahead is the count of commits the 'From' ref is ahead of the 'To' ref.
	Ahead int32
	// Behind is the count of commits the 'From' ref is behind the 'To' ref.
	Behind int32
}

type PullRequest struct {
	BaseRepoPath string
	HeadRepoPath string

	BaseBranch string
	HeadBranch string
}

type DiffShortStat struct {
	Files     int
	Additions int
	Deletions int
}

type DiffFileHeader struct {
	OldFileName string
	NewFileName string
	Extensions  map[string]string
}

type DiffFileHunkHeaders struct {
	FileHeader   DiffFileHeader
	HunksHeaders []HunkHeader
}

type DiffCutParams struct {
	LineStart    int
	LineStartNew bool
	LineEnd      int
	LineEndNew   bool
	BeforeLines  int
	AfterLines   int
	LineLimit    int
}

type BlameReader interface {
	NextPart() (*BlamePart, error)
}

type BlamePart struct {
	Commit *Commit  `json:"commit"`
	Lines  []string `json:"lines"`
}

type PathRenameDetails struct {
	OldPath         string
	NewPath         string
	CommitSHABefore string
	CommitSHAAfter  string
}

type CommitFilter struct {
	Path      string
	AfterRef  string
	Since     int64
	Until     int64
	Committer string
}

type TempRepository struct {
	Path    string
	BaseSHA string
	HeadSHA string
}

type PathDetails struct {
	Path       string
	LastCommit *Commit
}

type FileContent struct {
	Path    string
	Content []byte
}

type MergeResult struct {
	ConflictFiles []string
}

// ObjectCount represents the parsed information from the `git count-objects -v` command.
// For field meanings, see https://git-scm.com/docs/git-count-objects#_options.
type ObjectCount struct {
	Count         int
	Size          int64
	InPack        int
	Packs         int
	SizePack      int64
	PrunePackable int
	Garbage       int
	SizeGarbage   int64
}

type FileDiffRequest struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"-"` // warning: changes are possible and this field may not exist in the future
}

type FileDiffRequests []FileDiffRequest
