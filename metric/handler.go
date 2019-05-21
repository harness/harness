// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package metric

import (
	"errors"
	"net/http"

	"github.com/drone/drone/core"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// errInvalidToken is returned when the prometheus token is invalid.
var errInvalidToken = errors.New("Invalid or missing prometheus token")

// errAccessDenied is returned when the authorized user does not
// have access to the metrics endpoint.
var errAccessDenied = errors.New("Access denied")

// Server is an http Metrics server.
type Server struct {
	metrics   http.Handler
	session   core.Session
	anonymous bool
}

// NewServer returns a new metrics server.
func NewServer(session core.Session, anonymous bool) *Server {
	return &Server{
		metrics:   promhttp.Handler(),
		session:   session,
		anonymous: anonymous,
	}
}

// ServeHTTP responds to an http.Request and writes system
// metrics to the response body in plain text format.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, _ := s.session.Get(r)
	switch {
	case !s.anonymous && user == nil:
		http.Error(w, errInvalidToken.Error(), 401)
	case !s.anonymous && !user.Admin && !user.Machine:
		http.Error(w, errAccessDenied.Error(), 403)
	default:
		s.metrics.ServeHTTP(w, r)
	}
}
