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

package risk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// KEVEntry is one vulnerability of the CISA KEV catalog.
type KEVEntry struct {
	CVEID                      string `json:"cveID"`
	VendorProject              string `json:"vendorProject"`
	Product                    string `json:"product"`
	DateAdded                  string `json:"dateAdded"`
	DueDate                    string `json:"dueDate"`
	KnownRansomwareCampaignUse string `json:"knownRansomwareCampaignUse"`
}

// KEV is the set of known exploited CVEs.
type KEV struct {
	CatalogVersion string
	DateReleased   string
	entries        map[string]KEVEntry
}

// Len returns the number of CVEs in the catalog.
func (k *KEV) Len() int {
	if k == nil {
		return 0
	}
	return len(k.entries)
}

// Contains reports whether the CVE is in the catalog (case-insensitive).
func (k *KEV) Contains(cve string) bool {
	_, ok := k.Entry(cve)
	return ok
}

// Entry returns the catalog entry of the CVE.
func (k *KEV) Entry(cve string) (KEVEntry, bool) {
	if k == nil {
		return KEVEntry{}, false
	}
	e, ok := k.entries[strings.ToUpper(strings.TrimSpace(cve))]
	return e, ok
}

// ParseKEV parses the CISA KEV JSON catalog.
func ParseKEV(r io.Reader) (*KEV, error) {
	var doc struct {
		CatalogVersion  string     `json:"catalogVersion"`
		DateReleased    string     `json:"dateReleased"`
		Vulnerabilities []KEVEntry `json:"vulnerabilities"`
	}
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("kev: decode: %w", err)
	}
	k := &KEV{
		CatalogVersion: doc.CatalogVersion,
		DateReleased:   doc.DateReleased,
		entries:        make(map[string]KEVEntry, len(doc.Vulnerabilities)),
	}
	for _, v := range doc.Vulnerabilities {
		id := strings.ToUpper(strings.TrimSpace(v.CVEID))
		if id == "" {
			continue
		}
		v.CVEID = id
		k.entries[id] = v
	}
	return k, nil
}

// KEVFetcher downloads and parses the KEV catalog.
type KEVFetcher struct {
	// URL defaults to DefaultKEVURL.
	URL       string
	Client    *http.Client
	UserAgent string
}

// Fetch downloads the catalog using prev validators. When the server answers
// 304 it returns (nil, prev, nil): keep using the previously parsed data.
func (f *KEVFetcher) Fetch(ctx context.Context, prev Validators) (*KEV, Validators, error) {
	url := f.URL
	if url == "" {
		url = DefaultKEVURL
	}
	body, v, err := fetchBody(ctx, f.Client, url, f.UserAgent, prev)
	if err != nil || body == nil {
		return nil, v, err
	}
	k, err := ParseKEV(bytes.NewReader(body))
	if err != nil {
		return nil, prev, err
	}
	return k, v, nil
}
