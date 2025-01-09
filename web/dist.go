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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/rs/zerolog/log"
)

//go:embed dist/*
var EmbeddedUIFS embed.FS

const (
	distPath              = "dist"
	remoteEntryJS         = "remoteEntry.js"
	remoteEntryJSFullPath = "/" + remoteEntryJS
)

// Handler returns an http.HandlerFunc that servers the
// static content from the embedded file system.
//
//nolint:gocognit // refactor if required.
func Handler(uiSourceOverride string) http.HandlerFunc {
	fs, err := fs.Sub(EmbeddedUIFS, distPath)
	if err != nil {
		panic(fmt.Errorf("failed to load embedded files: %w", err))
	}

	// override UI source if provided (for local development)
	if uiSourceOverride != "" {
		log.Info().Msgf("Starting with alternate UI located at %q", uiSourceOverride)
		fs = os.DirFS(uiSourceOverride)
	}

	remoteEntryContent, remoteEntryExists, err := readRemoteEntryJSContent(fs)
	if err != nil {
		panic(fmt.Errorf("failed to read remote entry JS content: %w", err))
	}

	fileMap, err := createFileMapForDistFolder(fs)
	if err != nil {
		panic(fmt.Errorf("failed to create file map for dist folder: %w", err))
	}

	publicIndexFileExists := fileMap["index_public.html"]

	// Create an http.FileServer to serve the contents of the files subdiretory.
	handler := http.FileServer(http.FS(fs))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the file base path
		basePath := path.Base(r.URL.Path)

		// fallback to root in case the file doesn't exist so /index.html is served
		if !fileMap[basePath] {
			r.URL.Path = "/"
		} else {
			r.URL.Path = "/" + basePath
		}

		// handle public access
		if publicIndexFileExists &&
			RenderPublicAccessFrom(r.Context()) &&
			(r.URL.Path == "/" || r.URL.Path == "/index.html") {
			r.URL.Path = "/index_public.html"
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
		if remoteEntryExists && r.URL.Path == remoteEntryJSFullPath {
			http.ServeContent(w, r, r.URL.Path, time.Now(), bytes.NewReader(remoteEntryContent))
		} else {
			handler.ServeHTTP(w, r)
		}
	})
}

func readRemoteEntryJSContent(fileSystem fs.FS) ([]byte, bool, error) {
	file, err := fileSystem.Open(remoteEntryJS)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to open remoteEntry.js: %w", err)
	}
	defer file.Close()

	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read remoteEntry.js: %w", err)
	}

	enableCDN := os.Getenv("ENABLE_CDN")

	if len(enableCDN) == 0 {
		enableCDN = "false"
	}

	return bytes.Replace(buf, []byte("__ENABLE_CDN__"), []byte(enableCDN), 1), true, nil
}

func createFileMapForDistFolder(fileSystem fs.FS) (map[string]bool, error) {
	fileMap := make(map[string]bool)

	err := fs.WalkDir(fileSystem, ".", func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to read file info for %q: %w", path, err)
		}

		if path != "." {
			fileMap[path] = true
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build file map: %w", err)
	}

	return fileMap, nil
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
