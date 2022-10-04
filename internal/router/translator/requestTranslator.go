// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package translator

import (
	"net/http"
)

// RequestTranslator is responsible for translating an incomming request
// before it's getting routed and handled.
type RequestTranslator interface {
	// TranslatePreRouting is called before any routing decisions are made.
	TranslatePreRouting(*http.Request) (*http.Request, error)

	// TranslateGit is called for a git related request.
	TranslateGit(*http.Request) (*http.Request, error)

	// TranslateAPI is called for an API related request.
	TranslateAPI(*http.Request) (*http.Request, error)

	// TranslateWeb is called for an web related request.
	TranslateWeb(*http.Request) (*http.Request, error)
}
