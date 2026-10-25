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

package shield

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/harness/gitness/app/services/shield/osv"
)

func buildZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestSyncer_SyncAndLoad(t *testing.T) {
	lodash, err := os.ReadFile(filepath.FromSlash(recordsDir + "/GHSA-p6mc-m468-83gw.json"))
	if err != nil {
		t.Fatal(err)
	}
	other := `{"id":"GHSA-other","aliases":[],"affected":[{"package":{"ecosystem":"npm","name":"other"},` +
		`"ranges":[{"type":"SEMVER","events":[{"introduced":"0"},{"fixed":"2.0.0"}]}]}]}`
	archive := buildZip(t, map[string]string{"GHSA-p6mc-m468-83gw.json": string(lodash), "GHSA-other.json": other})
	epss, err := os.ReadFile(filepath.FromSlash("risk/testdata/epss.csv"))
	if err != nil {
		t.Fatal(err)
	}
	kev, err := os.ReadFile(filepath.FromSlash("risk/testdata/kev.json"))
	if err != nil {
		t.Fatal(err)
	}

	var hits, notModified atomic.Int32
	serve := func(etag, ctype string, body []byte) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			if r.Header.Get("If-None-Match") == etag {
				notModified.Add(1)
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("ETag", etag)
			w.Header().Set("Content-Type", ctype)
			_, _ = w.Write(body)
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/npm/all.zip", serve(`"osv-npm-1"`, "application/zip", archive))
	mux.HandleFunc("/Go/all.zip", http.NotFound)
	mux.HandleFunc("/epss.csv", serve(`"epss-1"`, "text/csv", epss))
	mux.HandleFunc("/kev.json", serve(`"kev-1"`, "application/json", kev))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	dir := t.TempDir()
	s := NewSyncer(SyncConfig{
		DataDir:    dir,
		Ecosystems: []string{"npm", "Go"},
		OSVBaseURL: srv.URL,
		EPSSURL:    srv.URL + "/epss.csv",
		KEVURL:     srv.URL + "/kev.json",
	}, srv.Client())

	// Go is not served: its error is reported but does not stop the other sources.
	if err = s.Sync(context.Background()); err == nil {
		t.Fatal("expected the Go archive error")
	}
	for _, p := range []string{"osv/npm.zip", "osv-npm.validators.json", "epss.validators.json", "kev.validators.json"} {
		if _, err = os.Stat(filepath.Join(dir, filepath.FromSlash(p))); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}
	if s.Scorer().EPSS == nil || s.Scorer().KEV == nil {
		t.Fatal("scorer data not loaded")
	}

	// second sync: archive revalidated with the stored ETag.
	before := notModified.Load()
	_ = s.Sync(context.Background())
	if notModified.Load() <= before {
		t.Error("expected conditional requests answered with 304")
	}

	// a new process only has the files on disk: the archive validators are reused, EPSS/KEV are refetched.
	s2 := NewSyncer(s.cfg, srv.Client())
	nm := notModified.Load()
	_ = s2.Sync(context.Background())
	if s2.Scorer().EPSS == nil || s2.Scorer().KEV == nil {
		t.Fatal("feeds must be reloaded after a restart even if the server has not changed")
	}
	if got := notModified.Load() - nm; got != 1 {
		t.Errorf("only the archive should be revalidated after a restart, got %d 304s", got)
	}

	// only the records of the scanned packages are loaded.
	db, err := s.LoadDB([]osv.Package{{Ecosystem: "npm", Name: "lodash", Version: "4.17.15", Path: "package-lock.json"}})
	if err != nil {
		t.Fatal(err)
	}
	if db.Len() != 1 {
		t.Errorf("db has %d records, want 1", db.Len())
	}
	if res := osv.Match(db, []osv.Package{{Ecosystem: "npm", Name: "lodash", Version: "4.17.15"}}); len(res.Findings) != 1 {
		t.Errorf("findings = %d", len(res.Findings))
	}

	// ecosystem without data
	if _, err = s.LoadDB([]osv.Package{{Ecosystem: "Go", Name: "x", Version: "v1.0.0"}}); err == nil {
		t.Error("expected an error for an ecosystem that is not synced")
	}
	// ecosystem not enabled is ignored
	db, err = s.LoadDB([]osv.Package{{Ecosystem: "PyPI", Name: "x", Version: "1.0"}})
	if err != nil || db.Len() != 0 {
		t.Errorf("disabled ecosystem: err=%v len=%d", err, db.Len())
	}
}
