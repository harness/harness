// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package http

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
)

const (
	// DefaultReadHeaderTimeout defines the default timeout for reading headers.
	DefaultReadHeaderTimeout = 2 * time.Second
)

// Config defines the config of an http server.
// TODO: expose via options?
type Config struct {
	Acme              bool
	Addr              string
	Cert              string
	Key               string
	Host              string
	ReadHeaderTimeout time.Duration
}

// Server is a wrapper around http.Server that exposes different async ListenAndServe methods
// that return corresponding ShutdownFunctions.
type Server struct {
	config  Config
	handler http.Handler
}

// ShutdownFunction defines a function that is called to shutdown the server.
type ShutdownFunction func(context.Context) error

func NewServer(config Config, handler http.Handler) *Server {
	if config.ReadHeaderTimeout == 0 {
		config.ReadHeaderTimeout = DefaultReadHeaderTimeout
	}

	return &Server{
		config:  config,
		handler: handler,
	}
}

// ListenAndServe initializes a server to respond to HTTP network requests.
func (s *Server) ListenAndServe() (*errgroup.Group, ShutdownFunction) {
	if s.config.Acme {
		return s.listenAndServeAcme()
	} else if s.config.Key != "" {
		return s.listenAndServeTLS()
	}
	return s.listenAndServe()
}

func (s *Server) listenAndServe() (*errgroup.Group, ShutdownFunction) {
	var g errgroup.Group
	s1 := &http.Server{
		Addr:              s.config.Addr,
		ReadHeaderTimeout: s.config.ReadHeaderTimeout,
		Handler:           s.handler,
	}
	g.Go(func() error {
		return s1.ListenAndServe()
	})

	return &g, s1.Shutdown
}

func (s *Server) listenAndServeTLS() (*errgroup.Group, ShutdownFunction) {
	var g errgroup.Group
	s1 := &http.Server{
		Addr:              ":http",
		ReadHeaderTimeout: s.config.ReadHeaderTimeout,
		Handler:           http.HandlerFunc(redirect),
	}
	s2 := &http.Server{
		Addr:              ":https",
		ReadHeaderTimeout: s.config.ReadHeaderTimeout,
		Handler:           s.handler,
	}
	g.Go(func() error {
		return s1.ListenAndServe()
	})
	g.Go(func() error {
		return s2.ListenAndServeTLS(
			s.config.Cert,
			s.config.Key,
		)
	})

	return &g, func(ctx context.Context) error {
		var sg errgroup.Group
		sg.Go(func() error {
			return s1.Shutdown(ctx)
		})
		sg.Go(func() error {
			return s2.Shutdown(ctx)
		})
		return sg.Wait()
	}
}

func (s Server) listenAndServeAcme() (*errgroup.Group, ShutdownFunction) {
	var g errgroup.Group
	m := &autocert.Manager{
		Cache:      autocert.DirCache(".cache"),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.config.Host),
	}
	s1 := &http.Server{
		Addr:              ":http",
		ReadHeaderTimeout: s.config.ReadHeaderTimeout,
		Handler:           m.HTTPHandler(nil),
	}
	s2 := &http.Server{
		Addr:              ":https",
		Handler:           s.handler,
		ReadHeaderTimeout: s.config.ReadHeaderTimeout,
		TLSConfig: &tls.Config{
			MinVersion:     tls.VersionTLS12,
			GetCertificate: m.GetCertificate,
			NextProtos:     []string{"h2", "http/1.1"},
		},
	}
	g.Go(func() error {
		return s1.ListenAndServe()
	})
	g.Go(func() error {
		return s2.ListenAndServeTLS("", "")
	})

	return &g, func(ctx context.Context) error {
		var sg errgroup.Group
		sg.Go(func() error {
			return s1.Shutdown(ctx)
		})
		sg.Go(func() error {
			return s2.Shutdown(ctx)
		})
		return sg.Wait()
	}
}

func redirect(w http.ResponseWriter, req *http.Request) {
	target := "https://" + req.Host + req.URL.Path
	http.Redirect(w, req, target, http.StatusTemporaryRedirect)
}
