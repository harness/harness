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

package osv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultBaseURL is the public OSV data bucket.
const DefaultBaseURL = "https://osv-vulnerabilities.storage.googleapis.com"

// Validators are the HTTP cache validators of a previously downloaded
// archive. Persist them between syncs.
type Validators struct {
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
}

// FetchResult describes the outcome of a fetch.
type FetchResult struct {
	// NotModified is true when the server answered 304; nothing was written.
	NotModified bool
	Validators  Validators
	// Size is the number of bytes written (0 when NotModified).
	Size int64
}

// Fetcher downloads OSV ecosystem archives (<base>/<ecosystem>/all.zip).
type Fetcher struct {
	// BaseURL defaults to DefaultBaseURL.
	BaseURL string
	// Client defaults to a client with a 10 minute timeout.
	Client *http.Client
	// UserAgent is sent on every request when set.
	UserAgent string
}

func (f *Fetcher) client() *http.Client {
	if f.Client != nil {
		return f.Client
	}
	return &http.Client{Timeout: 10 * time.Minute}
}

// ArchiveURL returns the download URL for an ecosystem.
func (f *Fetcher) ArchiveURL(ecosystem string) string {
	base := strings.TrimRight(f.BaseURL, "/")
	if base == "" {
		base = DefaultBaseURL
	}
	return base + "/" + url.PathEscape(ecosystem) + "/all.zip"
}

// Fetch downloads the archive of an ecosystem into w. When prev carries
// validators they are sent as If-None-Match / If-Modified-Since, and a 304
// response yields NotModified without writing anything.
func (f *Fetcher) Fetch(ctx context.Context, ecosystem string, prev Validators, w io.Writer) (*FetchResult, error) {
	if ecosystem == "" {
		return nil, fmt.Errorf("osv fetch: empty ecosystem")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.ArchiveURL(ecosystem), nil)
	if err != nil {
		return nil, err
	}
	if prev.ETag != "" {
		req.Header.Set("If-None-Match", prev.ETag)
	}
	if prev.LastModified != "" {
		req.Header.Set("If-Modified-Since", prev.LastModified)
	}
	if f.UserAgent != "" {
		req.Header.Set("User-Agent", f.UserAgent)
	}
	resp, err := f.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("osv fetch %s: %w", ecosystem, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		return &FetchResult{NotModified: true, Validators: prev}, nil
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("osv fetch %s: unexpected status %s", ecosystem, resp.Status)
	}
	n, err := io.Copy(w, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("osv fetch %s: read body: %w", ecosystem, err)
	}
	return &FetchResult{
		Validators: Validators{
			ETag:         resp.Header.Get("ETag"),
			LastModified: resp.Header.Get("Last-Modified"),
		},
		Size: n,
	}, nil
}

// FetchToFile downloads the archive to destPath atomically (temporary file
// in the same directory, then rename). On NotModified destPath is untouched.
func (f *Fetcher) FetchToFile(ctx context.Context, ecosystem string, prev Validators, destPath string) (*FetchResult, error) {
	dir := filepath.Dir(destPath)
	tmp, err := os.CreateTemp(dir, ".osv-*.zip.tmp")
	if err != nil {
		return nil, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename

	res, err := f.Fetch(ctx, ecosystem, prev, tmp)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}
	if res.NotModified {
		return res, nil
	}
	if err := os.Rename(tmpName, destPath); err != nil {
		return nil, err
	}
	return res, nil
}
