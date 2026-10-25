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
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetcher(t *testing.T) {
	const etag = `"abc123"`
	const lastMod = "Wed, 01 Jan 2025 00:00:00 GMT"
	body := []byte("zip-bytes")
	var gotPath, gotUA string
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		gotPath, gotUA = r.URL.EscapedPath(), r.UserAgent()
		if r.Header.Get("If-None-Match") == etag || r.Header.Get("If-Modified-Since") == lastMod {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if r.URL.Path == "/Missing/all.zip" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("ETag", etag)
		w.Header().Set("Last-Modified", lastMod)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	f := &Fetcher{BaseURL: srv.URL + "/", Client: srv.Client(), UserAgent: "shield-test"}
	ctx := t.Context()

	var buf bytes.Buffer
	res, err := f.Fetch(ctx, "Go", Validators{}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/Go/all.zip" || gotUA != "shield-test" {
		t.Errorf("path=%q ua=%q", gotPath, gotUA)
	}
	if res.NotModified || res.Size != int64(len(body)) || !bytes.Equal(buf.Bytes(), body) {
		t.Errorf("unexpected result %+v body=%q", res, buf.Bytes())
	}
	if res.Validators.ETag != etag || res.Validators.LastModified != lastMod {
		t.Errorf("validators %+v", res.Validators)
	}

	// Conditional requests: by ETag, then by Last-Modified only.
	for _, v := range []Validators{{ETag: etag}, {LastModified: lastMod}} {
		buf.Reset()
		res, err = f.Fetch(ctx, "Go", v, &buf)
		if err != nil {
			t.Fatal(err)
		}
		if !res.NotModified || buf.Len() != 0 || res.Validators != v {
			t.Errorf("expected 304 for %+v, got %+v (%d bytes)", v, res, buf.Len())
		}
	}

	// Stale validators lead to a full download.
	buf.Reset()
	res, err = f.Fetch(ctx, "Go", Validators{ETag: `"old"`}, &buf)
	if err != nil || res.NotModified || buf.Len() == 0 {
		t.Errorf("stale validators: %+v %v", res, err)
	}

	// Ecosystem names are path-escaped.
	if _, err = f.Fetch(ctx, "Rocky Linux", Validators{}, &buf); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/Rocky%20Linux/all.zip" {
		t.Errorf("escaped path %q", gotPath)
	}

	// Errors.
	if _, err = f.Fetch(ctx, "Missing", Validators{}, &buf); err == nil {
		t.Error("expected error on 404")
	}
	if _, err = f.Fetch(ctx, "", Validators{}, &buf); err == nil {
		t.Error("expected error on empty ecosystem")
	}
	if got := (&Fetcher{}).ArchiveURL("npm"); got != DefaultBaseURL+"/npm/all.zip" {
		t.Errorf("default URL %q", got)
	}
	_ = hits
}

func TestFetchToFile(t *testing.T) {
	var fail bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == `"v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if fail {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write([]byte("payload"))
	}))
	defer srv.Close()
	f := &Fetcher{BaseURL: srv.URL}
	dir := t.TempDir()
	dest := filepath.Join(dir, "go.zip")

	res, err := f.FetchToFile(t.Context(), "Go", Validators{}, dest)
	if err != nil || res.Validators.ETag != `"v1"` {
		t.Fatalf("%+v %v", res, err)
	}
	if b, _ := os.ReadFile(dest); string(b) != "payload" {
		t.Fatalf("content %q", b)
	}

	// 304 leaves the file alone and leaves no temp files.
	res, err = f.FetchToFile(t.Context(), "Go", res.Validators, dest)
	if err != nil || !res.NotModified {
		t.Fatalf("%+v %v", res, err)
	}
	// Failure keeps the previous file.
	fail = true
	if _, err = f.FetchToFile(t.Context(), "Go", Validators{}, dest); err == nil {
		t.Fatal("expected error")
	}
	if b, _ := os.ReadFile(dest); string(b) != "payload" {
		t.Fatalf("previous file clobbered: %q", b)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("leftover files: %v", entries)
	}
}
