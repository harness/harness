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

package suspect

import (
	"net/url"
	"path"
	"strings"
)

// NormalizePath strips SDK specific decorations (schemes, query strings, "/app/", "./")
// from a frame path and returns a slash-separated path without leading slash.
func NormalizePath(p string) string {
	p = strings.TrimSpace(p)
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	p = strings.ReplaceAll(p, "\\", "/")
	for _, scheme := range []string{"webpack-internal://", "webpack://", "file://", "app:///", "app://"} {
		if strings.HasPrefix(p, scheme) {
			p = strings.TrimPrefix(p, scheme)
			// webpack://<namespace>/./src/x.js -> drop the namespace segment
			if strings.HasPrefix(scheme, "webpack") {
				if i := strings.Index(p, "/"); i >= 0 {
					p = p[i+1:]
				}
			}
			break
		}
	}
	if u, err := url.PathUnescape(p); err == nil {
		p = u
	}
	p = strings.TrimPrefix(p, "/app/")
	p = strings.TrimPrefix(p, "./")
	p = path.Clean("/" + p)
	return strings.TrimPrefix(p, "/")
}

func stripExt(p string) string {
	return strings.TrimSuffix(p, path.Ext(p))
}

// pathIndex matches normalized frame paths against the release tree.
type pathIndex struct {
	byBase map[string][]string
	byStem map[string][]string
}

func newPathIndex(paths []string) *pathIndex {
	idx := &pathIndex{byBase: map[string][]string{}, byStem: map[string][]string{}}
	for _, p := range paths {
		b := path.Base(p)
		idx.byBase[b] = append(idx.byBase[b], p)
		s := stripExt(b)
		idx.byStem[s] = append(idx.byStem[s], p)
	}
	return idx
}

// match returns the repository path for the frame path: the longest segment
// suffix with at least one match wins, and it must be unique.
func (x *pathIndex) match(framePath string, stem bool) (string, bool) {
	norm := NormalizePath(framePath)
	if norm == "" || norm == "." {
		return "", false
	}
	if stem {
		norm = stripExt(norm)
	}
	segs := strings.Split(norm, "/")
	last := segs[len(segs)-1]
	cands := x.byBase[last]
	if stem {
		cands = x.byStem[last]
	}
	for k := len(segs); k >= 1; k-- {
		suffix := strings.Join(segs[len(segs)-k:], "/")
		var found []string
		for _, c := range cands {
			cc := c
			if stem {
				cc = stripExt(c)
			}
			if cc == suffix || strings.HasSuffix(cc, "/"+suffix) {
				found = append(found, c)
			}
		}
		if len(found) == 1 {
			return found[0], true
		}
		if len(found) > 1 {
			return "", false // ambiguous at the longest matching suffix
		}
	}
	return "", false
}

func (x *pathIndex) mapFrame(f Frame) (string, bool) {
	for _, p := range []string{f.Filename, f.AbsPath} {
		if p == "" {
			continue
		}
		if m, ok := x.match(p, false); ok {
			return m, true
		}
	}
	if f.Filename == "" && f.AbsPath == "" && f.Module != "" {
		return x.match(strings.ReplaceAll(f.Module, ".", "/"), true)
	}
	return "", false
}
