// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package dist embeds the static web server content.
package web

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"time"
)

//go:embed dist/*
var UI embed.FS
var remoteEntryContent []byte
var fileMap map[string]bool

const distPath = "dist"
const remoteEntryJS = "remoteEntry.js"
const remoteEntryJSFullPath = "/" + remoteEntryJS

// Handler returns an http.HandlerFunc that servers the
// static content from the embedded file system.
//
//nolint:gocognit // refactor if required.
func Handler() http.HandlerFunc {
	// Load the files subdirectory
	fs, err := fs.Sub(UI, distPath)
	if err != nil {
		panic(err)
	}

	// Create an http.FileServer to serve the
	// contents of the files subdiretory.
	handler := http.FileServer(http.FS(fs))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the file base path
		basePath := path.Base(r.URL.Path)

		// If the file exists in dist/ then serve it from "/".
		// Otherwise, rewrite the request to "/" so /index.html is served
		if fileNotFoundInDist(basePath) {
			r.URL.Path = "/"
		} else {
			r.URL.Path = "/" + basePath
		}

		if RenderPublicAccessFrom(r.Context()) &&
			(r.URL.Path == "/" || r.URL.Path == "/index.html") {
			r.URL.Path = "./index_public.html"
		}

		// Disable caching and sniffing via HTTP headers for UI main entry resources
		if r.URL.Path == "/" ||
			r.URL.Path == remoteEntryJSFullPath ||
			r.URL.Path == "/index.html" ||
			r.URL.Path == "/index_public.html" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate, max-age=0")
			w.Header().Set("pragma", "no-cache")
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}

		// Serve /remoteEntry.js from memory
		if r.URL.Path == remoteEntryJSFullPath {
			http.ServeContent(w, r, r.URL.Path, time.Now(), bytes.NewReader(remoteEntryContent))
		} else {
			handler.ServeHTTP(w, r)
		}
	})
}

func init() {
	err := readRemoteEntryJSContent()
	if err != nil {
		panic(err)
	}

	err = createFileMapForDistFolder()
	if err != nil {
		panic(err)
	}
}

func readRemoteEntryJSContent() error {
	fs, err := fs.Sub(UI, distPath)

	if err != nil {
		return fmt.Errorf("failed to open /dist: %w", err)
	}

	file, err := fs.Open(remoteEntryJS)

	if err != nil {
		return fmt.Errorf("failed to open remoteEntry.js: %w", err)
	}

	defer file.Close()
	buf, err := io.ReadAll(file)

	if err != nil {
		return fmt.Errorf("failed to read remoteEntry.js: %w", err)
	}

	enableCDN := os.Getenv("ENABLE_CDN")

	if len(enableCDN) == 0 {
		enableCDN = "false"
	}

	remoteEntryContent = bytes.Replace(buf, []byte("__ENABLE_CDN__"), []byte(enableCDN), 1)

	return nil
}

func createFileMapForDistFolder() error {
	fileMap = make(map[string]bool)

	err := fs.WalkDir(UI, distPath, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to build file map for path %q: %w", path, err)
		}

		if path != distPath { // exclude "dist" from file map
			fileMap[path] = true
		}

		return nil
	})

	return err
}

func fileNotFoundInDist(path string) bool {
	return !fileMap[distPath+"/"+path]
}

// renderPublicAccessKey is the context key for storing and retrieving whether public access should be rendered.
type renderPublicAccessKey struct{}

// RenderPublicAccessFrom retrieves the public access rendering config from the context.
// If true, the UI should be rendered for public access.
func RenderPublicAccessFrom(ctx context.Context) bool {
	if v, ok := ctx.Value(renderPublicAccessKey{}).(bool); ok {
		return v
	}

	return false
}

// WithRenderPublicAccess returns a copy of parent in which the public access rendering is set to the provided value.
func WithRenderPublicAccess(parent context.Context, v bool) context.Context {
	return context.WithValue(parent, renderPublicAccessKey{}, v)
}
