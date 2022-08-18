// Copyright 2021 Harness, Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE.md file.

//go:build !proxy
// +build !proxy

// Package dist embeds the static web server content.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path/filepath"
)

//go:embed dist/*
var UI embed.FS

// Handler returns an http.HandlerFunc that servers the
// static content from the embedded file system.
func Handler() http.HandlerFunc {
	// Load the files subdirectory
	fs, err := fs.Sub(UI, "dist")
	if err != nil {
		panic(err)
	}
	// Create an http.FileServer to serve the
	// contents of the files subdiretory.
	handler := http.FileServer(http.FS(fs))

	// Create an http.HandlerFunc that wraps the
	// http.FileServer to always load the index.html
	// file if a directory path is being requested.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// because this is a single page application,
		// we need to always load the index.html file
		// in the root of the project, unless the path
		// points to a file with an extension (css, js, etc)
		if filepath.Ext(r.URL.Path) == "" {
			// HACK: alter the path to point to the
			// root of the project.
			r.URL.Path = "/"
		}
		// and finally server the file.
		handler.ServeHTTP(w, r)
	})
}
