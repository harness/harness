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

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"
)

// A Server defines parameters for running an HTTP server.
type Server struct {
	Acme    bool
	Addr    string
	Cert    string
	Key     string
	Host    string
	Handler http.Handler
}

// ListenAndServe initializes a server to respond to HTTP network requests.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if s.Acme {
		return s.listenAndServeAcme(ctx)
	} else if s.Key != "" {
		return s.listenAndServeTLS(ctx)
	}
	return s.listenAndServe(ctx)
}

func (s *Server) listenAndServe(ctx context.Context) error {
	var g errgroup.Group
	s1 := &http.Server{
		Addr:              s.Addr,
		ReadHeaderTimeout: 2 * time.Second,
		Handler:           s.Handler,
	}
	g.Go(func() error {
		<-ctx.Done()
		return s1.Shutdown(ctx)
	})
	g.Go(func() error {
		return s1.ListenAndServe()
	})
	return g.Wait()
}

func (s *Server) listenAndServeTLS(ctx context.Context) error {
	var g errgroup.Group
	s1 := &http.Server{
		Addr:              ":http",
		ReadHeaderTimeout: 2 * time.Second,
		Handler:           http.HandlerFunc(redirect),
	}
	s2 := &http.Server{
		Addr:              ":https",
		ReadHeaderTimeout: 2 * time.Second,
		Handler:           s.Handler,
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
	g.Go(func() error {
		<-ctx.Done()
		if err := s1.Shutdown(ctx); err != nil {
			return err
		}
		if err := s2.Shutdown(ctx); err != nil {
			return err
		}
		return nil
	})
	return g.Wait()
}

func (s Server) listenAndServeAcme(ctx context.Context) error {
	var g errgroup.Group
	m := &autocert.Manager{
		Cache:      autocert.DirCache(".cache"),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.Host),
	}
	s1 := &http.Server{
		Addr:              ":http",
		ReadHeaderTimeout: 2 * time.Second,
		Handler:           m.HTTPHandler(nil),
	}
	s2 := &http.Server{
		Addr:              ":https",
		Handler:           s.Handler,
		ReadHeaderTimeout: 2 * time.Second,
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
	g.Go(func() error {
		<-ctx.Done()
		if err := s1.Shutdown(ctx); err != nil {
			return err
		}
		if err := s2.Shutdown(ctx); err != nil {
			return err
		}
		return nil
	})
	return g.Wait()
}

func redirect(w http.ResponseWriter, req *http.Request) {
	target := "https://" + req.Host + req.URL.Path
	http.Redirect(w, req, target, http.StatusTemporaryRedirect)
}
