// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"fmt"
	"time"
)

type cloneRepoOptions struct {
	timeout       time.Duration
	mirror        bool
	bare          bool
	quiet         bool
	branch        string
	shared        bool
	noCheckout    bool
	depth         int
	filter        string
	skipTLSVerify bool
}

type sortOrder int

const (
	sortOrderDefault sortOrder = iota
	sortOrderAsc
	sortOrderDesc
)

type gitObjectType string

const (
	gitObjectTypeCommit gitObjectType = "commit"
	gitObjectTypeTree   gitObjectType = "tree"
	gitObjectTypeBlob   gitObjectType = "blob"
	gitObjectTypeTag    gitObjectType = "tag"
)

func parseGitObjectType(t string) (gitObjectType, error) {
	switch t {
	case string(gitObjectTypeCommit):
		return gitObjectTypeCommit, nil
	case string(gitObjectTypeBlob):
		return gitObjectTypeBlob, nil
	case string(gitObjectTypeTree):
		return gitObjectTypeTree, nil
	case string(gitObjectTypeTag):
		return gitObjectTypeTag, nil
	default:
		return gitObjectTypeBlob, fmt.Errorf("unknown git object type '%s'", t)
	}
}

// gitReferenceField represents the different fields available when listing references.
// For the full list, see https://git-scm.com/docs/git-for-each-ref#_field_names
type gitReferenceField string

const (
	gitReferenceFieldRefName     gitReferenceField = "refname"
	gitReferenceFieldObjectType  gitReferenceField = "objecttype"
	gitReferenceFieldObjectName  gitReferenceField = "objectname"
	gitReferenceFieldCreatorDate gitReferenceField = "creatordate"
)

func parseGitReferenceField(f string) (gitReferenceField, error) {
	switch f {
	case string(gitReferenceFieldCreatorDate):
		return gitReferenceFieldCreatorDate, nil
	case string(gitReferenceFieldRefName):
		return gitReferenceFieldRefName, nil
	case string(gitReferenceFieldObjectName):
		return gitReferenceFieldObjectName, nil
	case string(gitReferenceFieldObjectType):
		return gitReferenceFieldObjectType, nil
	default:
		return gitReferenceFieldRefName, fmt.Errorf("unknown git reference field '%s'", f)
	}
}

type walkInstruction int

const (
	walkInstructionStop walkInstruction = iota
	walkInstructionHandle
	walkInstructionSkip
)

type walkReferencesEntry map[gitReferenceField]string

// TODO: can be generic (so other walk methods can use the same)
type walkReferencesInstructor func(walkReferencesEntry) (walkInstruction, error)

// TODO: can be generic (so other walk methods can use the same)
type walkReferencesHandler func(walkReferencesEntry) error

type walkReferencesOptions struct {
	// patterns are the patterns used to pre-filter the references of the repo.
	// OPTIONAL. By default all references are walked.
	patterns []string

	// fields indicates the fields that are passed to the instructor & handler
	// OPTIONAL. Default fields are:
	// - gitReferenceFieldRefName
	// - gitReferenceFieldObjectName
	fields []gitReferenceField

	// instructor indicates on how to handle the reference.
	// OPTIONAL. By default all references are handled.
	// NOTE: once walkInstructionStop is returned, the walking stops.
	instructor walkReferencesInstructor

	// sort indicates the field by which the references should be sorted.
	// OPTIONAL. By default gitReferenceFieldRefName is used.
	sort gitReferenceField

	// order indicates the order (asc or desc) of the sorted output
	order sortOrder

	// maxWalkDistance is the maximum number of nodes that are iterated over before the walking stops.
	// OPTIONAL. A value of <= 0 will walk all references.
	// WARNING: Skipped elements count towards the walking distance
	maxWalkDistance int32
}

type commit struct {
	sha       string
	title     string
	message   string
	author    signature
	committer signature
}

type tag struct {
	sha        string
	name       string
	targetSha  string
	targetType gitObjectType
	title      string
	message    string
	tagger     signature
}

// signature represents the Author or Committer information.
type signature struct {
	identity identity
	// When is the timestamp of the signature.
	when time.Time
}

type identity struct {
	name  string
	email string
}

type commitChangesOptions struct {
	committer signature
	author    signature
	message   string
}

type pushOptions struct {
	remote  string
	branch  string
	force   bool
	mirror  bool
	env     []string
	timeout time.Duration
}

type treeNodeWithCommit struct {
	treeNode
	commit *commit
}

type treeNode struct {
	nodeType treeNodeType
	mode     treeNodeMode
	sha      string
	name     string
	path     string
}

// treeNodeType specifies the different types of nodes in a git tree.
// IMPORTANT: has to be consistent with rpc.TreeNodeType (proto).
type treeNodeType int

const (
	treeNodeTypeTree treeNodeType = iota
	treeNodeTypeBlob
	treeNodeTypeCommit
)

// treeNodeType specifies the different modes of a node in a git tree.
// IMPORTANT: has to be consistent with rpc.TreeNodeMode (proto).
type treeNodeMode int

const (
	treeNodeModeFile treeNodeMode = iota
	treeNodeModeSymlink
	treeNodeModeExec
	treeNodeModeTree
	treeNodeModeCommit
)

type submodule struct {
	name string
	url  string
}

type blob struct {
	size int64
	// content contains the content of the blob
	// NOTE: can be only partial content - compare len(.content) with .size
	content []byte
}
