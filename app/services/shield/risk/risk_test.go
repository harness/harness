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
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func loadFixtures(t *testing.T) *Scorer {
	t.Helper()
	f, err := os.Open("testdata/epss.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	e, err := ParseEPSS(f)
	if err != nil {
		t.Fatal(err)
	}
	g, err := os.Open("testdata/kev.json")
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	k, err := ParseKEV(g)
	if err != nil {
		t.Fatal(err)
	}
	return &Scorer{EPSS: e, KEV: k}
}

func gzipFixture(t *testing.T) []byte {
	t.Helper()
	plain, err := os.ReadFile("testdata/epss.csv")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(plain)
	_ = zw.Close()
	return buf.Bytes()
}

func TestParseEPSS(t *testing.T) {
	plain, err := os.ReadFile("testdata/epss.csv")
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"plain": plain, "gzip": gzipFixture(t)} {
		e, err := ParseEPSS(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if e.Len() != 3 || e.ModelVersion != "v2025.03.14" || e.ScoreDate != "2026-10-24T12:55:00Z" {
			t.Errorf("%s: got len=%d model=%q date=%q", name, e.Len(), e.ModelVersion, e.ScoreDate)
		}
		s, ok := e.Lookup("cve-2021-44228")
		if !ok || s.Score != 0.94358 || s.Percentile != 0.9999 {
			t.Errorf("%s: lookup = %+v %v", name, s, ok)
		}
		if _, ok := e.Lookup("CVE-0000-0000"); ok {
			t.Errorf("%s: unexpected hit", name)
		}
	}
	if _, err := ParseEPSS(bytes.NewReader([]byte("foo,bar,baz\n"))); err == nil {
		t.Error("expected header error")
	}
}

func TestParseKEV(t *testing.T) {
	s := loadFixtures(t)
	if s.KEV.Len() != 2 || s.KEV.CatalogVersion != "2026.10.20" {
		t.Fatalf("kev = %d %q", s.KEV.Len(), s.KEV.CatalogVersion)
	}
	e, ok := s.KEV.Entry("CVE-2021-44228")
	if !ok || e.Product != "Log4j2" || e.KnownRansomwareCampaignUse != "Known" || e.DueDate != "2021-12-24" {
		t.Errorf("entry = %+v", e)
	}
	if !s.KEV.Contains("CVE-2023-1111") || s.KEV.Contains("CVE-2022-0001") {
		t.Error("membership wrong")
	}
	if _, err := ParseKEV(bytes.NewReader([]byte("{"))); err == nil {
		t.Error("expected decode error")
	}
}

func TestFetchers(t *testing.T) {
	gz := gzipFixture(t)
	kev, err := os.ReadFile("testdata/kev.json")
	if err != nil {
		t.Fatal(err)
	}
	serve := func(body []byte) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/missing" {
				http.NotFound(w, r)
				return
			}
			if r.Header.Get("If-None-Match") == `"v1"` {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("ETag", `"v1"`)
			w.Header().Set("Last-Modified", "Fri, 24 Oct 2026 12:00:00 GMT")
			_, _ = w.Write(body)
		}))
	}
	ctx := context.Background()

	es := serve(gz)
	defer es.Close()
	ef := &EPSSFetcher{URL: es.URL, UserAgent: "test"}
	e, v, err := ef.Fetch(ctx, Validators{})
	if err != nil || e.Len() != 3 || v.ETag != `"v1"` || v.LastModified == "" {
		t.Fatalf("epss fetch: %v %v %+v", err, e.Len(), v)
	}
	e2, v2, err := ef.Fetch(ctx, v)
	if err != nil || e2 != nil || v2 != v {
		t.Fatalf("epss 304: %v %v %+v", err, e2, v2)
	}
	if _, _, err := (&EPSSFetcher{URL: es.URL + "/missing"}).Fetch(ctx, Validators{}); err == nil {
		t.Error("expected status error")
	}

	ks := serve(kev)
	defer ks.Close()
	kf := &KEVFetcher{URL: ks.URL}
	k, v, err := kf.Fetch(ctx, Validators{})
	if err != nil || k.Len() != 2 || v.ETag == "" {
		t.Fatalf("kev fetch: %v %v", err, v)
	}
	k2, _, err := kf.Fetch(ctx, v)
	if err != nil || k2 != nil {
		t.Fatalf("kev 304: %v %v", err, k2)
	}
}

func TestScore(t *testing.T) {
	s := loadFixtures(t)
	tests := []struct {
		name string
		in   Input
		want int
		chk  func(*testing.T, Breakdown)
	}{
		{
			name: "reachable, no epss/kev data",
			in:   Input{CVSSScore: 10, HasCVSS: true, Reachability: ReachabilityReachable},
			want: 100, // 13 / 10, clamped
		},
		{
			name: "log4shell worst case",
			in: Input{CVSSScore: 10, HasCVSS: true, CVEs: []string{"CVE-2021-44228"},
				Reachability: ReachabilityReachable},
			want: 100, // 10*1.4999*1.5*1.3 = 29.248, clamped
			chk: func(t *testing.T, b Breakdown) {
				if !b.InKEV || b.KEVFactor != 1.5 || b.EPSSCVE != "CVE-2021-44228" {
					t.Errorf("breakdown = %+v", b)
				}
			},
		},
		{
			name: "unknown reachability, epss low, no KEV",
			in:   Input{CVSSScore: 7, HasCVSS: true, CVEs: []string{"CVE-2022-0001"}},
			want: 42, // 7*0.595 = 4.165
		},
		{
			name: "max epss among aliases + kev via alias",
			in: Input{CVSSScore: 5, HasCVSS: true, CVEs: []string{"CVE-2022-0001", "CVE-2023-1111"},
				Reachability: ReachabilityUnknown},
			want: 98, // 5*1.3*1.5 = 9.75
			chk: func(t *testing.T, b Breakdown) {
				if b.EPSSCVE != "CVE-2023-1111" || b.KEVCVE != "CVE-2023-1111" {
					t.Errorf("breakdown = %+v", b)
				}
			},
		},
		{
			name: "not reachable high confidence",
			in: Input{CVSSScore: 10, HasCVSS: true, Reachability: ReachabilityNotReachable,
				Confidence: 0.9},
			want: 20, // 2
		},
		{
			name: "not reachable low confidence ignored",
			in: Input{CVSSScore: 10, HasCVSS: true, Reachability: ReachabilityNotReachable,
				Confidence: 0.5},
			want: 100, // 10
		},
		{
			name: "not reachable custom threshold",
			in: Input{CVSSScore: 10, HasCVSS: true, Reachability: ReachabilityNotReachable,
				Confidence: 0.5, HighConfidence: 0.5},
			want: 20,
		},
		{name: "exposure internal", in: Input{CVSSScore: 10, HasCVSS: true, Exposure: ExposureInternal}, want: 80},
		{name: "exposure dev only", in: Input{CVSSScore: 10, HasCVSS: true, Exposure: ExposureDevOnly}, want: 50},
		{name: "fallback critical", in: Input{Severity: "CRITICAL"}, want: 90},
		{name: "fallback high", in: Input{Severity: "high"}, want: 75},
		{name: "fallback medium", in: Input{Severity: "medium"}, want: 50},
		{name: "fallback low", in: Input{Severity: "low"}, want: 25},
		{name: "fallback unknown label", in: Input{Severity: "weird"}, want: 50},
		{name: "fallback empty", in: Input{}, want: 50},
		{name: "cvss zero is honored", in: Input{HasCVSS: true, CVSSScore: 0, Severity: "critical"}, want: 0},
		{
			name: "clamp above",
			in: Input{HasCVSS: true, CVSSScore: 99, CVEs: []string{"CVE-2021-44228"},
				Reachability: ReachabilityReachable},
			want: 100,
		},
		{name: "clamp below", in: Input{HasCVSS: true, CVSSScore: -5}, want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, b := s.Score(tc.in)
			if got != tc.want || b.Score != got {
				t.Fatalf("score = %d (breakdown %d), want %d; %+v", got, b.Score, tc.want, b)
			}
			if tc.chk != nil {
				tc.chk(t, b)
			}
		})
	}
}

func TestScoreNoData(t *testing.T) {
	var s *Scorer
	got, b := s.Score(Input{HasCVSS: true, CVSSScore: 10, CVEs: []string{"CVE-2021-44228"}})
	if got != 100 || b.EPSSFactor != 1 || b.InKEV || b.CVSSSource != "cvss" {
		t.Errorf("got %d %+v", got, b)
	}
	_, b = (&Scorer{}).Score(Input{Severity: "high"})
	if b.CVSSSource != "severity:high" || b.Exposure != ExposurePublic || b.Reachability != ReachabilityUnknown {
		t.Errorf("breakdown = %+v", b)
	}
}

func TestEPSSFactor(t *testing.T) {
	for in, want := range map[float64]float64{-1: 0.5, 0: 0.5, 0.5: 1.0, 1: 1.5, 3: 1.5} {
		if got := EPSSFactor(in); got != want {
			t.Errorf("EPSSFactor(%v) = %v, want %v", in, got, want)
		}
	}
}
