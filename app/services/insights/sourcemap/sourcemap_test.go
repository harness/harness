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

package sourcemap

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/insights/ingest"
)

const b64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

func encVLQ(v int) string {
	u := v << 1
	if v < 0 {
		u = (-v << 1) | 1
	}
	var sb strings.Builder
	for {
		d := u & 31
		u >>= 5
		if u > 0 {
			d |= 32
		}
		sb.WriteByte(b64[d])
		if u == 0 {
			return sb.String()
		}
	}
}

// encodeMappings builds a mappings string from absolute segments per generated
// line: {genCol, src, srcLine0, srcCol[, name]}.
func encodeMappings(lines [][][]int) string {
	var out []string
	var src, sl, sc, nm int
	for _, segs := range lines {
		var parts []string
		prev := 0
		for _, s := range segs {
			p := encVLQ(s[0] - prev)
			prev = s[0]
			if len(s) >= 4 {
				p += encVLQ(s[1]-src) + encVLQ(s[2]-sl) + encVLQ(s[3]-sc)
				src, sl, sc = s[1], s[2], s[3]
			}
			if len(s) == 5 {
				p += encVLQ(s[4] - nm)
				nm = s[4]
			}
			parts = append(parts, p)
		}
		out = append(out, strings.Join(parts, ","))
	}
	return strings.Join(out, ";")
}

const appSource = "// header\nexport function boom() {\n  throw new Error(\"x\");\n}\n"

// tinyMap maps `function a(){throw new Error("x")}` (minified line 1) back to
// appSource; line 2 of the bundle is mapped to node_modules code.
func tinyMap(t *testing.T) []byte {
	t.Helper()
	m := map[string]any{
		"version":    3,
		"file":       "app.min.js",
		"sourceRoot": "webpack:///",
		"sources":    []string{"./src/app.js", "./node_modules/lib/index.js"},
		"sourcesContent": []any{
			appSource,
			nil,
		},
		"names": []string{"boom"},
		"mappings": encodeMappings([][][]int{
			{{0, 0, 1, 16, 0}, {13, 0, 2, 2}, {19, 0, 2, 8}},
			{{0, 1, 9, 4}},
		}),
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestDecodeVLQ(t *testing.T) {
	cases := map[string]int{"A": 0, "C": 1, "D": -1, "gB": 16, "hB": -16, "+/////D": 2147483647}
	for s, want := range cases {
		got, pos, err := decodeVLQ(s, 0)
		if err != nil || got != want || pos != len(s) {
			t.Errorf("decodeVLQ(%q) = %d,%d,%v want %d", s, got, pos, err, want)
		}
	}
	for _, bad := range []string{"", "g", "!", "gggggggggg"} {
		if _, _, err := decodeVLQ(bad, 0); !errors.Is(err, ErrInvalidVLQ) {
			t.Errorf("decodeVLQ(%q) err = %v, want ErrInvalidVLQ", bad, err)
		}
	}
}

func TestVLQRoundTrip(t *testing.T) {
	for _, v := range []int{0, 1, -1, 15, 16, -16, 1000, -123456, 1 << 30} {
		got, _, err := decodeVLQ(encVLQ(v), 0)
		if err != nil || got != v {
			t.Errorf("roundtrip %d = %d, %v", v, got, err)
		}
	}
}

func TestLookup(t *testing.T) {
	m, err := Parse(tinyMap(t))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		line, col int
		ok        bool
		want      Mapping
	}{
		{"first segment start", 1, 0, true, Mapping{"webpack:///./src/app.js", 2, 16, "boom"}},
		{"inside first segment", 1, 12, true, Mapping{"webpack:///./src/app.js", 2, 16, "boom"}},
		{"second segment edge", 1, 13, true, Mapping{"webpack:///./src/app.js", 3, 2, ""}},
		{"last segment and beyond", 1, 500, true, Mapping{"webpack:///./src/app.js", 3, 8, ""}},
		{"second line", 2, 3, true, Mapping{"webpack:///./node_modules/lib/index.js", 10, 4, ""}},
		{"line zero", 0, 0, false, Mapping{}},
		{"negative column", 1, -1, false, Mapping{}},
		{"line out of range", 3, 0, false, Mapping{}},
	}
	for _, tc := range tests {
		got, ok := m.Lookup(tc.line, tc.col)
		if ok != tc.ok || got != tc.want {
			t.Errorf("%s: Lookup(%d,%d) = %+v,%v want %+v,%v", tc.name, tc.line, tc.col, got, ok, tc.want, tc.ok)
		}
	}
}

func TestLookupBeforeFirstSegment(t *testing.T) {
	data := fmt.Sprintf(`{"version":3,"sources":["a.js"],"names":[],"mappings":%q}`,
		encodeMappings([][][]int{{{5, 0, 0, 0}}}))
	m, err := Parse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Lookup(1, 4); ok {
		t.Error("column before first segment must not resolve")
	}
	if got, ok := m.Lookup(1, 5); !ok || got.Line != 1 || got.Column != 0 {
		t.Errorf("got %+v %v", got, ok)
	}
}

func TestParseErrors(t *testing.T) {
	tests := map[string]string{
		"not json":       `nope`,
		"bad version":    `{"version":2,"sources":[],"mappings":""}`,
		"bad vlq":        `{"version":3,"sources":["a"],"names":[],"mappings":"AA!A"}`,
		"truncated vlq":  `{"version":3,"sources":["a"],"names":[],"mappings":"AAAg"}`,
		"two fields":     `{"version":3,"sources":["a"],"names":[],"mappings":"AA"}`,
		"six fields":     `{"version":3,"sources":["a"],"names":["n"],"mappings":"AAAAAA"}`,
		"source range":   `{"version":3,"sources":["a"],"names":[],"mappings":"ACAA"}`,
		"name range":     `{"version":3,"sources":["a"],"names":[],"mappings":"AAAAC"}`,
		"negative col":   `{"version":3,"sources":["a"],"names":[],"mappings":"DAAA"}`,
		"url section":    `{"version":3,"sections":[{"offset":{"line":0,"column":0},"url":"x.map"}]}`,
		"unordered":      `{"version":3,"sections":[{"offset":{"line":2,"column":0},"map":{"version":3,"sources":[],"mappings":""}},{"offset":{"line":1,"column":0},"map":{"version":3,"sources":[],"mappings":""}}]}`,
		"nested index":   `{"version":3,"sections":[{"offset":{"line":0,"column":0},"map":{"version":3,"sections":[]}}]}`,
		"negative index": `{"version":3,"sources":["a"],"names":[],"mappings":"AADA"}`,
	}
	for name, in := range tests {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	if _, err := Parse([]byte(tests["bad vlq"])); !errors.Is(err, ErrInvalidVLQ) {
		t.Errorf("bad vlq err = %v", err)
	}
}

func TestParseXSSIAndEmpty(t *testing.T) {
	m, err := Parse([]byte(")]}'\n{\"version\":3,\"sources\":[],\"names\":[],\"mappings\":\"\"}"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Lookup(1, 0); ok {
		t.Error("empty map must not resolve")
	}
}

func TestIndexMap(t *testing.T) {
	sub := func(src string, line int) string {
		return fmt.Sprintf(`{"version":3,"sources":[%q],"sourcesContent":["l1\nl2\nl3"],"names":[],"mappings":%q}`,
			src, encodeMappings([][][]int{{{0, 0, line, 0}}, {{0, 0, line + 1, 0}}}))
	}
	idx := fmt.Sprintf(`{"version":3,"file":"bundle.js","sections":[
		{"offset":{"line":0,"column":0},"map":%s},
		{"offset":{"line":10,"column":20},"map":%s}]}`, sub("a.js", 0), sub("b.js", 1))
	m, err := Parse([]byte(idx))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		line, col int
		ok        bool
		src       string
		oline     int
	}{
		{1, 0, true, "a.js", 1},
		{2, 7, true, "a.js", 2},
		{3, 0, false, "", 0},   // past section a, before section b
		{11, 19, false, "", 0}, // before b's column offset on its first line
		{11, 20, true, "b.js", 2},
		{12, 0, true, "b.js", 3},
	}
	for _, tc := range tests {
		got, ok := m.Lookup(tc.line, tc.col)
		if ok != tc.ok || got.Source != tc.src || got.Line != tc.oline {
			t.Errorf("Lookup(%d,%d) = %+v,%v", tc.line, tc.col, got, ok)
		}
	}
	if c, ok := m.SourceContent("b.js"); !ok || c != "l1\nl2\nl3" {
		t.Errorf("SourceContent = %q, %v", c, ok)
	}
}

func TestContextLines(t *testing.T) {
	var sb []string
	for i := 1; i <= 20; i++ {
		sb = append(sb, fmt.Sprintf("line%d", i))
	}
	content := strings.Join(sb, "\r\n")

	c, pre, post := contextLines(content, 10)
	if c != "line10" || len(pre) != 5 || pre[0] != "line5" || pre[4] != "line9" ||
		len(post) != 5 || post[0] != "line11" || post[4] != "line15" {
		t.Errorf("middle: %q %v %v", c, pre, post)
	}
	c, pre, post = contextLines(content, 1)
	if c != "line1" || len(pre) != 0 || len(post) != 5 {
		t.Errorf("first: %q %v %v", c, pre, post)
	}
	c, pre, post = contextLines(content, 20)
	if c != "line20" || len(pre) != 5 || len(post) != 0 {
		t.Errorf("last: %q %v %v", c, pre, post)
	}
	if c, _, _ := contextLines(content, 21); c != "" {
		t.Error("out of range line must yield no context")
	}
	if c, _, _ := contextLines(content, 0); c != "" {
		t.Error("line 0 must yield no context")
	}
}

type mapProv struct {
	m   *Map
	err error
	got [2]string
}

func (p *mapProv) Find(_ context.Context, release, absPath string) (*Map, error) {
	p.got = [2]string{release, absPath}
	return p.m, p.err
}

func TestSymbolicate(t *testing.T) {
	m, err := Parse(tinyMap(t))
	if err != nil {
		t.Fatal(err)
	}
	prov := &mapProv{m: m}
	tr := true
	f := &ingest.Frame{
		Filename: "app.min.js", AbsPath: "https://cdn.example.com/app.min.js",
		Function: "a", Lineno: 1, Colno: 14, // 1-based colno 14 == 0-based column 13
		InApp: &tr, ContextLine: "minified code",
	}
	ok, err := Symbolicate(context.Background(), f, "rel1", prov)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if prov.got != [2]string{"rel1", "https://cdn.example.com/app.min.js"} {
		t.Errorf("provider args = %v", prov.got)
	}
	if f.AbsPath != "webpack:///./src/app.js" || f.Filename != "src/app.js" ||
		f.Lineno != 3 || f.Colno != 3 || f.Function != "a" {
		t.Errorf("rewritten frame = %+v", f)
	}
	if f.ContextLine != `  throw new Error("x");` || len(f.PreContext) != 2 || f.PreContext[0] != "// header" ||
		len(f.PostContext) != 2 || f.PostContext[0] != "}" {
		t.Errorf("context = %q %q %q", f.ContextLine, f.PreContext, f.PostContext)
	}
	if f.InApp == nil || !*f.InApp {
		t.Error("in_app must stay true for app code")
	}
	orig, _ := f.Vars[VarsKey].(map[string]any)
	if orig["minified_abs_path"] != "https://cdn.example.com/app.min.js" || orig["minified_lineno"] != 1 ||
		orig["minified_colno"] != 14 || orig["minified_function"] != "a" || orig["minified_context_line"] != "minified code" {
		t.Errorf("original location lost: %v", orig)
	}

	// Idempotent.
	if ok, _ := Symbolicate(context.Background(), f, "rel1", prov); ok {
		t.Error("second symbolication must be a no-op")
	}
}

func TestSymbolicateNameAndFirstColumn(t *testing.T) {
	m, _ := Parse(tinyMap(t))
	f := &ingest.Frame{AbsPath: "app.min.js", Function: "a", Lineno: 1, Colno: 1}
	if ok, _ := Symbolicate(context.Background(), f, "r", &mapProv{m: m}); !ok {
		t.Fatal("expected symbolication")
	}
	if f.Function != "boom" || f.Lineno != 2 || f.Colno != 17 {
		t.Errorf("frame = %+v", f)
	}
	// No sourcesContent for that source in the map -> no context, and stale minified context cleared.
}

func TestSymbolicateLibraryFrame(t *testing.T) {
	m, _ := Parse(tinyMap(t))
	tr := true
	f := &ingest.Frame{AbsPath: "app.min.js", Lineno: 2, Colno: 4, InApp: &tr, ContextLine: "min"}
	if ok, _ := Symbolicate(context.Background(), f, "r", &mapProv{m: m}); !ok {
		t.Fatal("expected symbolication")
	}
	if f.InApp == nil || *f.InApp {
		t.Error("node_modules frame must be in_app=false")
	}
	if f.ContextLine != "" || f.PreContext != nil {
		t.Errorf("no sourcesContent: context must be cleared, got %q", f.ContextLine)
	}
	if !isLibrary("webpack:///webpack/bootstrap") || isLibrary("src/app.js") {
		t.Error("isLibrary mismatch")
	}
}

func TestSymbolicateSkips(t *testing.T) {
	m, _ := Parse(tinyMap(t))
	ctx := context.Background()
	notFound := &mapProv{err: fmt.Errorf("wrapped: %w", ErrNotFound)}
	f := &ingest.Frame{AbsPath: "x.js", Lineno: 1, Colno: 1}
	if ok, err := Symbolicate(ctx, f, "r", notFound); ok || err != nil {
		t.Errorf("not found: %v %v", ok, err)
	}
	boom := &mapProv{err: errors.New("db down")}
	if _, err := Symbolicate(ctx, f, "r", boom); err == nil {
		t.Error("provider failure must be returned")
	}
	for _, fr := range []*ingest.Frame{
		{AbsPath: "", Lineno: 1},
		{AbsPath: "x.js", Lineno: 0},
		{AbsPath: "x.js", Lineno: 99, Colno: 1}, // out of range
	} {
		before := *fr
		if ok, err := Symbolicate(ctx, fr, "r", &mapProv{m: m}); ok || err != nil {
			t.Errorf("%+v: ok=%v err=%v", before, ok, err)
		}
		if fr.Vars != nil || fr.Lineno != before.Lineno {
			t.Errorf("frame modified: %+v", fr)
		}
	}
}

func TestCacheLRU(t *testing.T) {
	m, _ := Parse(tinyMap(t))
	sz := m.Size()
	if sz <= 0 {
		t.Fatal("size must be positive")
	}
	c := NewCache(sz*2 + sz/2)
	c.Put("a", m)
	c.Put("b", m)
	if _, ok := c.Get("a"); !ok { // a becomes most recent
		t.Fatal("a missing")
	}
	c.Put("c", m) // evicts b
	if _, ok := c.Get("b"); ok {
		t.Error("b should have been evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Error("a should remain")
	}
	if c.Len() != 2 || c.Bytes() != 2*sz {
		t.Errorf("len=%d bytes=%d", c.Len(), c.Bytes())
	}
	c.Put("a", m) // replace, no growth
	if c.Len() != 2 || c.Bytes() != 2*sz {
		t.Errorf("after replace len=%d bytes=%d", c.Len(), c.Bytes())
	}
	small := NewCache(sz - 1)
	small.Put("x", m)
	if small.Len() != 0 {
		t.Error("oversized map must not be cached")
	}
}

type memStore struct {
	files map[string][]byte
	calls int
}

func (s *memStore) Artifact(_ context.Context, release, u string) ([]byte, error) {
	s.calls++
	if b, ok := s.files[release+"|"+u]; ok {
		return b, nil
	}
	return nil, ErrNotFound
}

func TestExtractSourceMappingURL(t *testing.T) {
	tests := map[string]string{
		"code();\n//# sourceMappingURL=app.js.map\n":                      "app.js.map",
		"code();\n//@ sourceMappingURL=old.map":                           "old.map",
		"code();\n/*# sourceMappingURL=c.map */\n":                        "c.map",
		"//# sourceMappingURL=a.map\ncode();\n//# sourceMappingURL=b.map": "b.map",
	}
	for in, want := range tests {
		if got, ok := ExtractSourceMappingURL(in); !ok || got != want {
			t.Errorf("%q = %q,%v want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "var s='//# sourceMappingURL=x';", "no comment"} {
		if got, ok := ExtractSourceMappingURL(in); ok {
			t.Errorf("%q unexpectedly = %q", in, got)
		}
	}
}

func TestResolveURL(t *testing.T) {
	tests := [][3]string{
		{"https://cdn.example.com/js/app.min.js", "app.min.js.map", "https://cdn.example.com/js/app.min.js.map"},
		{"https://cdn.example.com/js/app.min.js", "/maps/a.map", "https://cdn.example.com/maps/a.map"},
		{"https://cdn.example.com/js/app.min.js", "https://other/x.map", "https://other/x.map"},
		{"~/js/app.min.js", "../m/a.map", "~/m/a.map"},
	}
	for _, tc := range tests {
		if got := ResolveURL(tc[0], tc[1]); got != tc[2] {
			t.Errorf("ResolveURL(%q,%q) = %q want %q", tc[0], tc[1], got, tc[2])
		}
	}
}

func TestArtifactProvider(t *testing.T) {
	mapJSON := tinyMap(t)
	ctx := context.Background()
	file := "https://cdn.example.com/js/app.min.js"

	t.Run("sourceMappingURL comment", func(t *testing.T) {
		st := &memStore{files: map[string][]byte{
			"r|" + file:                              []byte("a();\n//# sourceMappingURL=../maps/app.map\n"),
			"r|https://cdn.example.com/maps/app.map": mapJSON,
		}}
		p := &ArtifactProvider{Store: st, Cache: NewCache(1 << 20)}
		m, err := p.Find(ctx, "r", file+"?v=1")
		if err != nil || m == nil {
			t.Fatalf("Find: %v", err)
		}
		calls := st.calls
		if _, err := p.Find(ctx, "r", file); err != nil || st.calls != calls {
			t.Errorf("second Find must hit the cache (calls %d -> %d, err %v)", calls, st.calls, err)
		}
	})
	t.Run("inline data uri", func(t *testing.T) {
		uri := "data:application/json;base64," + base64.StdEncoding.EncodeToString(mapJSON)
		st := &memStore{files: map[string][]byte{"r|" + file: []byte("a();\n//# sourceMappingURL=" + uri)}}
		m, err := (&ArtifactProvider{Store: st}).Find(ctx, "r", file)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := m.Lookup(1, 0); !ok {
			t.Error("inline map does not resolve")
		}
	})
	t.Run("sibling .map fallback", func(t *testing.T) {
		st := &memStore{files: map[string][]byte{"r|" + file + ".map": mapJSON}}
		if _, err := (&ArtifactProvider{Store: st}).Find(ctx, "r", file); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("not found and wrong release", func(t *testing.T) {
		st := &memStore{files: map[string][]byte{"r|" + file + ".map": mapJSON}}
		_, err := (&ArtifactProvider{Store: st}).Find(ctx, "other", file)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("invalid map", func(t *testing.T) {
		st := &memStore{files: map[string][]byte{"r|" + file + ".map": []byte("{")}}
		_, err := (&ArtifactProvider{Store: st}).Find(ctx, "r", file)
		if !errors.Is(err, ErrInvalidMap) {
			t.Errorf("err = %v", err)
		}
	})
}
