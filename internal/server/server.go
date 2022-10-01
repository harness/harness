// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package server implements an http server.
package server

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/harness/gitness/internal/router"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
)

const (
	// ReadHeaderTimeout defines the max time the server waits for request headers.
	ReadHeaderTimeout = 2 * time.Second
)

// A Server defines parameters for running an HTTP server.
type Server struct {
	Acme   bool
	Addr   string
	Cert   string
	Key    string
	Host   string
	router *router.Router
}

// ShutdownFunction defines a function that is called to shutdown the server.
type ShutdownFunction func(context.Context) error

// ListenAndServe initializes a server to respond to HTTP network requests.
func (s *Server) ListenAndServe() (*errgroup.Group, ShutdownFunction) {
	if s.Acme {
		return s.listenAndServeAcme()
	} else if s.Key != "" {
		return s.listenAndServeTLS()
	}
	return s.listenAndServe()
}

func (s *Server) listenAndServe() (*errgroup.Group, ShutdownFunction) {
	var g errgroup.Group
	s1 := &http.Server{
		Addr:              s.Addr,
		ReadHeaderTimeout: ReadHeaderTimeout,
		Handler:           s.router,
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
		ReadHeaderTimeout: ReadHeaderTimeout,
		Handler:           http.HandlerFunc(redirect),
	}
	s2 := &http.Server{
		Addr:              ":https",
		ReadHeaderTimeout: ReadHeaderTimeout,
		Handler:           s.router,
	}
	g.Go(func() error {
		return s1.ListenAndServe()
	})
	g.Go(func() error {
		return s2.ListenAndServeTLS(
			s.Cert,
			s.Key,
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
		HostPolicy: autocert.HostWhitelist(s.Host),
	}
	s1 := &http.Server{
		Addr:              ":http",
		ReadHeaderTimeout: ReadHeaderTimeout,
		Handler:           m.HTTPHandler(nil),
	}
	s2 := &http.Server{
		Addr:              ":https",
		Handler:           s.router,
		ReadHeaderTimeout: ReadHeaderTimeout,
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
