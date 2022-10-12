// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import "time"

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

// signature represents the Author or Committer information.
type signature struct {
	// name represents a person name. It is an arbitrary string.
	name string
	// email is an email, but it cannot be assumed to be well-formed.
	email string
	// When is the timestamp of the signature.
	when time.Time
}

type commitChangesOptions struct {
	committer *signature
	author    *signature
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
