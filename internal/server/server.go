// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package server implements an http server.
package server

import (
	"github.com/harness/gitness/http"
)

// Server is the http server for gitness.
type Server struct {
	*http.Server
}
