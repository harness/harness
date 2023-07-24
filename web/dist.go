//go:build !proxy
// +build !proxy

// Package dist embeds the static web server content.
package web

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
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

		// Disable caching and sniffing via HTTP headers for UI main entry resources
		if r.URL.Path == "/" || r.URL.Path == "/remoteEntry.js" || r.URL.Path == "/index.html" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate, max-age=0")
			w.Header().Set("pragma", "no-cache")
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}

		if r.URL.Path == "/remoteEntry.js" {
			if readerRemoteEntry, errFetch := fetchRemoteEntryJS(fs); errFetch == nil {
				http.ServeContent(w, r, r.URL.Path, time.Now(), readerRemoteEntry)
			} else {
				log.Error().Msgf("Failed to fetch remoteEntry.js %v", err)
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			}
		} else {
			// and finally serve the file.
			handler.ServeHTTP(w, r)
		}
	})
}

var remoteEntryContent *bytes.Reader

func fetchRemoteEntryJS(fs fs.FS) (*bytes.Reader, error) {
	if remoteEntryContent == nil {
		path := "remoteEntry.js"

		file, err := fs.Open(path)
		if err != nil {
			log.Error().Msgf("Failed to open file %v", path)
			return nil, err
		}

		buf, err := io.ReadAll(file)
		if err != nil {
			log.Error().Msgf("Failed to read file %v", path)
			return nil, err
		}

		enableCDN := os.Getenv("ENABLE_CDN")
		if len(enableCDN) == 0 {
			enableCDN = "false"
		}

		modBuf := bytes.Replace(buf, []byte("__ENABLE_CDN__"), []byte(enableCDN), 1)

		remoteEntryContent = bytes.NewReader(modBuf)
	}

	return remoteEntryContent, nil
}
