// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"time"
)

type cloneRepoOption struct {
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

type referenceSortOption int

const (
	referenceSortOptionDefault referenceSortOption = iota
	referenceSortOptionName
	referenceSortOptionDate
)

type reference struct {
	name   string
	target string
}

type commit struct {
	sha       string
	title     string
	message   string
	author    signature
	committer signature
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
