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
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultEPSSURL is the public FIRST EPSS daily scores file.
const DefaultEPSSURL = "https://epss.cyentia.com/epss_scores-current.csv.gz"

// DefaultKEVURL is the public CISA Known Exploited Vulnerabilities catalog.
const DefaultKEVURL = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"

// EPSSScore is one EPSS entry: the probability of exploitation in the next
// 30 days (0..1) and its percentile among all scored CVEs (0..1).
type EPSSScore struct {
	Score      float64
	Percentile float64
}

// EPSS is a lookup of EPSS scores by upper-case CVE id.
type EPSS struct {
	// ModelVersion and ScoreDate come from the "#model_version:...,score_date:..."
	// header comment, if present.
	ModelVersion string
	ScoreDate    string
	scores       map[string]EPSSScore
}

// Len returns the number of CVEs.
func (e *EPSS) Len() int {
	if e == nil {
		return 0
	}
	return len(e.scores)
}

// Lookup returns the entry of the CVE (case-insensitive).
func (e *EPSS) Lookup(cve string) (EPSSScore, bool) {
	if e == nil {
		return EPSSScore{}, false
	}
	s, ok := e.scores[strings.ToUpper(strings.TrimSpace(cve))]
	return s, ok
}

// ParseEPSS parses the FIRST EPSS CSV, gzip-compressed or plain (detected by
// the gzip magic number): an optional "#model_version:...,score_date:..."
// comment line, the "cve,epss,percentile" header and one row per CVE.
// Malformed rows are skipped; a file with a bad header is an error.
func ParseEPSS(r io.Reader) (*EPSS, error) {
	br := bufio.NewReader(r)
	magic, err := br.Peek(2)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("epss: read: %w", err)
	}
	var src io.Reader = br
	if len(magic) == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		zr, zerr := gzip.NewReader(br)
		if zerr != nil {
			return nil, fmt.Errorf("epss: gzip: %w", zerr)
		}
		defer zr.Close()
		src = zr
	}

	e := &EPSS{scores: map[string]EPSSScore{}}
	sc := bufio.NewScanner(src)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	headerSeen := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			for _, kv := range strings.Split(strings.TrimPrefix(line, "#"), ",") {
				k, v, ok := strings.Cut(kv, ":")
				if !ok {
					continue
				}
				switch strings.TrimSpace(k) {
				case "model_version":
					e.ModelVersion = strings.TrimSpace(v)
				case "score_date":
					e.ScoreDate = strings.TrimSpace(v)
				}
			}
			continue
		}
		fields := strings.Split(line, ",")
		if !headerSeen {
			if len(fields) < 3 || !strings.EqualFold(strings.TrimSpace(fields[0]), "cve") {
				return nil, fmt.Errorf("epss: unexpected header %q", line)
			}
			headerSeen = true
			continue
		}
		if len(fields) < 3 {
			continue
		}
		score, err1 := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
		pct, err2 := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
		if err1 != nil || err2 != nil {
			continue
		}
		e.scores[strings.ToUpper(strings.TrimSpace(fields[0]))] = EPSSScore{
			Score:      clamp(score, 0, 1),
			Percentile: clamp(pct, 0, 1),
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("epss: read: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("epss: missing header")
	}
	return e, nil
}

// Validators are the HTTP cache validators of a previous download.
type Validators struct {
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
}

// fetchBody GETs url with conditional headers. It returns the body (nil when
// the server answered 304) and the new validators.
func fetchBody(
	ctx context.Context,
	client *http.Client,
	url, userAgent string,
	prev Validators,
) ([]byte, Validators, error) {
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, prev, err
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	if prev.ETag != "" {
		req.Header.Set("If-None-Match", prev.ETag)
	}
	if prev.LastModified != "" {
		req.Header.Set("If-Modified-Since", prev.LastModified)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, prev, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNotModified:
		return nil, prev, nil
	case http.StatusOK:
	default:
		return nil, prev, fmt.Errorf("GET %s: unexpected status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, prev, fmt.Errorf("GET %s: read body: %w", url, err)
	}
	return body, Validators{
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
	}, nil
}

// EPSSFetcher downloads and parses the EPSS file.
type EPSSFetcher struct {
	// URL defaults to DefaultEPSSURL.
	URL       string
	Client    *http.Client
	UserAgent string
}

// Fetch downloads the file using prev validators. When the server answers
// 304 it returns (nil, prev, nil): keep using the previously parsed data.
func (f *EPSSFetcher) Fetch(ctx context.Context, prev Validators) (*EPSS, Validators, error) {
	url := f.URL
	if url == "" {
		url = DefaultEPSSURL
	}
	body, v, err := fetchBody(ctx, f.Client, url, f.UserAgent, prev)
	if err != nil || body == nil {
		return nil, v, err
	}
	e, err := ParseEPSS(bytes.NewReader(body))
	if err != nil {
		return nil, prev, err
	}
	return e, v, nil
}
