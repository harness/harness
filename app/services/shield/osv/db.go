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
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Range types defined by the OSV schema.
const (
	RangeSemver    = "SEMVER"
	RangeEcosystem = "ECOSYSTEM"
	RangeGit       = "GIT"
)

// Record is an OSV vulnerability record (the subset of the schema used).
type Record struct {
	ID               string         `json:"id"`
	Aliases          []string       `json:"aliases"`
	Related          []string       `json:"related"`
	Summary          string         `json:"summary"`
	Details          string         `json:"details"`
	Modified         string         `json:"modified"`
	Published        string         `json:"published"`
	Withdrawn        string         `json:"withdrawn"`
	Severity         []Severity     `json:"severity"`
	Affected         []Affected     `json:"affected"`
	DatabaseSpecific map[string]any `json:"database_specific"`
}

// Severity is an OSV severity entry. For CVSS_V3/CVSS_V4 Score holds the
// vector string.
type Severity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

// Affected describes the affected versions of one package.
type Affected struct {
	Package          AffectedPackage `json:"package"`
	Severity         []Severity      `json:"severity"`
	Ranges           []Range         `json:"ranges"`
	Versions         []string        `json:"versions"`
	DatabaseSpecific map[string]any  `json:"database_specific"`
}

// AffectedPackage identifies a package in an ecosystem.
type AffectedPackage struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
	Purl      string `json:"purl"`
}

// Range is a list of version events of a given type.
type Range struct {
	Type   string  `json:"type"`
	Repo   string  `json:"repo"`
	Events []Event `json:"events"`
}

// Event is one of introduced, fixed, last_affected, limit.
type Event struct {
	Introduced   string `json:"introduced,omitempty"`
	Fixed        string `json:"fixed,omitempty"`
	LastAffected string `json:"last_affected,omitempty"`
	Limit        string `json:"limit,omitempty"`
}

// entry points into a record's affected list.
type entry struct {
	rec *Record
	idx int
}

// DB is an in-memory OSV database indexed by ecosystem and package name.
// It is safe for concurrent readers once loading has finished.
type DB struct {
	index   map[string][]entry
	records map[string]*Record
}

// LoadOption configures loading.
type LoadOption func(*loadOpts)

type loadOpts struct {
	keep func(ecosystem, name string) bool
}

// WithPackageFilter keeps only affected entries for which keep returns
// true, which bounds memory when loading large ecosystems (npm). Records
// with no kept entries are dropped.
func WithPackageFilter(keep func(ecosystem, name string) bool) LoadOption {
	return func(o *loadOpts) { o.keep = keep }
}

// NewDB returns an empty database.
func NewDB() *DB {
	return &DB{index: map[string][]entry{}, records: map[string]*Record{}}
}

func indexKey(ecosystem, name string) string {
	if ecosystem == EcosystemPyPI {
		name = NormalizePyPIName(name)
	}
	return ecosystem + "\x00" + name
}

// Len returns the number of indexed records.
func (d *DB) Len() int { return len(d.records) }

// Record returns the record with the given id.
func (d *DB) Record(id string) (*Record, bool) {
	r, ok := d.records[id]
	return r, ok
}

// Add indexes a record. A record with the same id replaces the previous
// one (the newer "modified" wins by call order). Withdrawn records are
// stored but not indexed.
func (d *DB) Add(rec *Record) { d.add(rec, nil) }

func (d *DB) add(rec *Record, o *loadOpts) {
	if rec == nil || rec.ID == "" {
		return
	}
	if o != nil && o.keep != nil {
		kept := rec.Affected[:0:0]
		for _, a := range rec.Affected {
			if o.keep(a.Package.Ecosystem, a.Package.Name) {
				kept = append(kept, a)
			}
		}
		if len(kept) == 0 {
			return
		}
		rec.Affected = kept
	}
	if old, ok := d.records[rec.ID]; ok {
		d.remove(old)
	}
	d.records[rec.ID] = rec
	if rec.Withdrawn != "" {
		return
	}
	for i, a := range rec.Affected {
		k := indexKey(a.Package.Ecosystem, a.Package.Name)
		d.index[k] = append(d.index[k], entry{rec: rec, idx: i})
	}
}

func (d *DB) remove(rec *Record) {
	for _, a := range rec.Affected {
		k := indexKey(a.Package.Ecosystem, a.Package.Name)
		es := d.index[k]
		out := es[:0]
		for _, e := range es {
			if e.rec != rec {
				out = append(out, e)
			}
		}
		if len(out) == 0 {
			delete(d.index, k)
		} else {
			d.index[k] = out
		}
	}
	delete(d.records, rec.ID)
}

// Lookup returns the affected entries for a package.
func (d *DB) lookup(ecosystem, name string) []entry {
	return d.index[indexKey(ecosystem, name)]
}

// AddJSON decodes one record from r and adds it.
func (d *DB) AddJSON(r io.Reader, opts ...LoadOption) error {
	o := &loadOpts{}
	for _, f := range opts {
		f(o)
	}
	return d.addJSON(r, o)
}

func (d *DB) addJSON(r io.Reader, o *loadOpts) error {
	var rec Record
	if err := json.NewDecoder(r).Decode(&rec); err != nil {
		return err
	}
	if rec.ID == "" {
		return fmt.Errorf("record has no id")
	}
	d.add(&rec, o)
	return nil
}

// LoadDir loads every *.json file under dir (recursively). Files that fail
// to decode abort the load with an error naming the file.
func LoadDir(dir string, opts ...LoadOption) (*DB, error) {
	o := &loadOpts{}
	for _, f := range opts {
		f(o)
	}
	db := NewDB()
	err := filepath.WalkDir(dir, func(p string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if de.IsDir() || !strings.EqualFold(filepath.Ext(p), ".json") {
			return nil
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := db.addJSON(f, o); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// LoadZip loads every *.json entry of an OSV ecosystem all.zip.
func LoadZip(r io.ReaderAt, size int64, opts ...LoadOption) (*DB, error) {
	db := NewDB()
	if err := db.LoadZip(r, size, opts...); err != nil {
		return nil, err
	}
	return db, nil
}

// LoadZip adds the records of a zip archive to the database, so several
// ecosystem archives can be combined in one DB.
func (d *DB) LoadZip(r io.ReaderAt, size int64, opts ...LoadOption) error {
	o := &loadOpts{}
	for _, f := range opts {
		f(o)
	}
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() || !strings.EqualFold(filepath.Ext(zf.Name), ".json") {
			continue
		}
		if err := d.loadZipFile(zf, o); err != nil {
			return fmt.Errorf("%s: %w", zf.Name, err)
		}
	}
	return nil
}

func (d *DB) loadZipFile(zf *zip.File, o *loadOpts) error {
	rc, err := zf.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	return d.addJSON(rc, o)
}

// LoadZipFile is LoadZip for a file on disk.
func LoadZipFile(path string, opts ...LoadOption) (*DB, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return LoadZip(f, st.Size(), opts...)
}
