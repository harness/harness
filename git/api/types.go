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

package api

import (
	"fmt"
	"time"

	"github.com/harness/gitness/errors"
)

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

// Signature represents the Author or Committer information.
type Signature struct {
	Identity Identity
	// When is the timestamp of the Signature.
	When time.Time
}

// Decode decodes a byte array representing a signature to signature
func (s *Signature) Decode(b []byte) {
	sig, _ := NewSignatureFromCommitLine(b)
	s.Identity.Email = sig.Identity.Email
	s.Identity.Name = sig.Identity.Name
	s.When = sig.When
}

func (s *Signature) String() string {
	return fmt.Sprintf("%s <%s>", s.Identity.Name, s.Identity.Email)
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

type PathRenameDetails struct {
	OldPath         string
	NewPath         string
	CommitSHABefore string
	CommitSHAAfter  string
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
