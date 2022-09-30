// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package translator

import (
	"net/http"

	"github.com/harness/gitness/internal/api/middleware/encode"
)

var (
	terminatedPathPrefixesAPI = []string{"/v1/spaces/", "/v1/repos/"}
)

var _ RequestTranslator = (*TerminatedPathTranslator)(nil)

// TerminatedPathTranslator translates encoded paths.
// For example:
//   - /space1/space2/+ -> /space1%2Fspace2
//   - /space1/rep1.git -> /space1%2Frepo1
//
// Note: paths are terminated after initial routing.
type TerminatedPathTranslator struct{}

func NewTerminatedPathTranslator() *TerminatedPathTranslator {
	return &TerminatedPathTranslator{}
}

// TranslatePreRouting is called before any routing decisions are made.
func (t *TerminatedPathTranslator) TranslatePreRouting(r *http.Request) (*http.Request, error) {
	return r, nil
}

// TranslateGit is called for a git related request.
func (t *TerminatedPathTranslator) TranslateGit(r *http.Request) (*http.Request, error) {
	return r, encode.GitPath(r)
}

// TranslateAPI is called for an API related request.
func (t *TerminatedPathTranslator) TranslateAPI(r *http.Request) (*http.Request, error) {
	return r, encode.TerminatedPath(terminatedPathPrefixesAPI, r)
}

// TranslateWeb is called for an web related request.
func (t *TerminatedPathTranslator) TranslateWeb(r *http.Request) (*http.Request, error) {
	return r, nil
}
