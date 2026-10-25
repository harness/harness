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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/harness/gitness/app/services/shield/osv"
	"github.com/harness/gitness/app/services/shield/risk"
	"github.com/harness/gitness/job"

	"github.com/rs/zerolog/log"
)

const (
	// JobTypeSync is the type of the vulnerability data sync job.
	JobTypeSync = "shield-vuln-sync"
	// JobTypeScan is the type of the scan job.
	JobTypeScan = "shield-scan"

	syncJobTimeout = 30 * time.Minute
)

// SyncConfig configures the vulnerability data sync.
type SyncConfig struct {
	// DataDir holds the downloaded data.
	DataDir string
	// Ecosystems are the OSV ecosystems to sync (e.g. Go, npm, PyPI).
	Ecosystems []string
	// Interval is the refresh interval of the recurring job.
	Interval time.Duration
	// URLs; empty means the public defaults.
	OSVBaseURL string
	EPSSURL    string
	KEVURL     string
}

// Syncer downloads and serves the local vulnerability data (OSV archives, EPSS and KEV).
type Syncer struct {
	cfg  SyncConfig
	osv  *osv.Fetcher
	epss *risk.EPSSFetcher
	kev  *risk.KEVFetcher

	mu         sync.RWMutex
	epssData   *risk.EPSS
	kevData    *risk.KEV
	epssVal    risk.Validators
	kevVal     risk.Validators
	syncMu     sync.Mutex
	lastResult string
}

var _ VulnData = (*Syncer)(nil)

// NewSyncer creates a syncer. The http client may be nil.
func NewSyncer(cfg SyncConfig, client *http.Client) *Syncer {
	const ua = "harness-shield"
	return &Syncer{
		cfg:  cfg,
		osv:  &osv.Fetcher{BaseURL: cfg.OSVBaseURL, Client: client, UserAgent: ua},
		epss: &risk.EPSSFetcher{URL: cfg.EPSSURL, Client: client, UserAgent: ua},
		kev:  &risk.KEVFetcher{URL: cfg.KEVURL, Client: client, UserAgent: ua},
	}
}

func (s *Syncer) osvDir() string { return filepath.Join(s.cfg.DataDir, "osv") }

func (s *Syncer) archivePath(ecosystem string) string {
	return filepath.Join(s.osvDir(), ecosystem+".zip")
}

func (s *Syncer) validatorsPath(name string) string {
	return filepath.Join(s.cfg.DataDir, name+".validators.json")
}

func readValidators(path string, v any) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, v)
}

func writeValidators(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Sync refreshes the OSV archives of the configured ecosystems and the EPSS/KEV feeds.
// Failures of one source do not stop the others; the joined error is returned.
func (s *Syncer) Sync(ctx context.Context) error {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	if err := os.MkdirAll(s.osvDir(), 0o750); err != nil {
		return fmt.Errorf("failed to create shield data dir: %w", err)
	}

	var errs []error
	updated := 0
	for _, eco := range s.cfg.Ecosystems {
		changed, err := s.syncOSV(ctx, eco)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if changed {
			updated++
		}
	}
	if err := s.syncEPSS(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := s.syncKEV(ctx); err != nil {
		errs = append(errs, err)
	}
	s.lastResult = fmt.Sprintf("osv archives updated: %d/%d", updated, len(s.cfg.Ecosystems))
	return errors.Join(errs...)
}

func (s *Syncer) syncOSV(ctx context.Context, eco string) (bool, error) {
	valPath := s.validatorsPath("osv-" + eco)
	dest := s.archivePath(eco)

	var prev osv.Validators
	// validators are only valid together with the archive they describe.
	if _, err := os.Stat(dest); err == nil {
		readValidators(valPath, &prev)
	}
	res, err := s.osv.FetchToFile(ctx, eco, prev, dest)
	if err != nil {
		return false, fmt.Errorf("osv %s: %w", eco, err)
	}
	if res.NotModified {
		return false, nil
	}
	if err = writeValidators(valPath, res.Validators); err != nil {
		return true, fmt.Errorf("osv %s: failed to store validators: %w", eco, err)
	}
	return true, nil
}

func (s *Syncer) syncEPSS(ctx context.Context) error {
	s.mu.RLock()
	prev, have := s.epssVal, s.epssData != nil
	s.mu.RUnlock()
	if !have {
		// the parsed data lives in memory only: a stored validator would answer 304 without data.
		prev = risk.Validators{}
	}
	data, v, err := s.epss.Fetch(ctx, prev)
	if err != nil {
		return fmt.Errorf("epss: %w", err)
	}
	if data == nil {
		return nil
	}
	s.mu.Lock()
	s.epssData, s.epssVal = data, v
	s.mu.Unlock()
	_ = writeValidators(s.validatorsPath("epss"), v)
	return nil
}

func (s *Syncer) syncKEV(ctx context.Context) error {
	s.mu.RLock()
	prev, have := s.kevVal, s.kevData != nil
	s.mu.RUnlock()
	if !have {
		prev = risk.Validators{}
	}
	data, v, err := s.kev.Fetch(ctx, prev)
	if err != nil {
		return fmt.Errorf("kev: %w", err)
	}
	if data == nil {
		return nil
	}
	s.mu.Lock()
	s.kevData, s.kevVal = data, v
	s.mu.Unlock()
	_ = writeValidators(s.validatorsPath("kev"), v)
	return nil
}

// Scorer implements VulnData.
func (s *Syncer) Scorer() *risk.Scorer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &risk.Scorer{EPSS: s.epssData, KEV: s.kevData}
}

// LoadDB implements VulnData: it loads the archives of the ecosystems of pkgs, keeping only the
// records of the packages that are used.
func (s *Syncer) LoadDB(pkgs []osv.Package) (*osv.DB, error) {
	enabled := make(map[string]struct{}, len(s.cfg.Ecosystems))
	for _, e := range s.cfg.Ecosystems {
		enabled[e] = struct{}{}
	}
	needed := map[string]map[string]struct{}{}
	for _, p := range pkgs {
		if _, ok := enabled[p.Ecosystem]; !ok {
			continue
		}
		if needed[p.Ecosystem] == nil {
			needed[p.Ecosystem] = map[string]struct{}{}
		}
		needed[p.Ecosystem][p.Name] = struct{}{}
	}

	db := osv.NewDB()
	for eco, names := range needed {
		path := s.archivePath(eco)
		f, err := os.Open(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("vulnerability data for %s is not synced yet", eco)
			}
			return nil, err
		}
		st, err := f.Stat()
		if err == nil {
			err = db.LoadZip(f, st.Size(), osv.WithPackageFilter(func(_, name string) bool {
				_, ok := names[name]
				return ok
			}))
		}
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to load %s vulnerability data: %w", eco, err)
		}
	}
	return db, nil
}

// Handle implements job.Handler for the sync job.
func (s *Syncer) Handle(ctx context.Context, _ string, _ job.ProgressReporter) (string, error) {
	if err := s.Sync(ctx); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("shield vulnerability data sync finished with errors")
		return s.lastResult, err
	}
	return s.lastResult, nil
}
