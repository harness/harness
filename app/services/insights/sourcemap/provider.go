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
	"errors"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// MapProvider finds the source map for a minified file of a release.
// Implementations return ErrNotFound (possibly wrapped) when there is none.
type MapProvider interface {
	Find(ctx context.Context, release, absPath string) (*Map, error)
}

// ArtifactStore returns the raw bytes of an uploaded artifact (a minified file
// or a .map file) keyed by release and file URL. It returns ErrNotFound
// (possibly wrapped) when the artifact does not exist.
type ArtifactStore interface {
	Artifact(ctx context.Context, release, fileURL string) ([]byte, error)
}

// ArtifactProvider is a MapProvider on top of an ArtifactStore. It resolves
// the map of a file by following its //# sourceMappingURL= comment (inline
// data: URIs are supported), falling back to "<file>.map". Parsed maps are
// cached when a Cache is set.
type ArtifactProvider struct {
	Store ArtifactStore
	Cache *Cache // optional
}

// Find implements MapProvider.
func (p *ArtifactProvider) Find(ctx context.Context, release, absPath string) (*Map, error) {
	file := stripQuery(absPath)
	key := release + "\x00" + file
	if p.Cache != nil {
		if m, ok := p.Cache.Get(key); ok {
			return m, nil
		}
	}
	data, err := p.load(ctx, release, file)
	if err != nil {
		return nil, err
	}
	m, err := Parse(data)
	if err != nil {
		return nil, err
	}
	if p.Cache != nil {
		p.Cache.Put(key, m)
	}
	return m, nil
}

func (p *ArtifactProvider) load(ctx context.Context, release, file string) ([]byte, error) {
	content, err := p.Store.Artifact(ctx, release, file)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if err == nil {
		if ref, ok := ExtractSourceMappingURL(string(content)); ok {
			if inline, ok := decodeDataURI(ref); ok {
				return inline, nil
			}
			return p.Store.Artifact(ctx, release, ResolveURL(file, ref))
		}
	}
	// No comment (or no minified file stored): conventional sibling map.
	return p.Store.Artifact(ctx, release, file+".map")
}

var sourceMappingURLRe = regexp.MustCompile(`(?m)^[ \t]*(?://|/\*)[#@][ \t]*sourceMappingURL=([^\s*]+)[ \t]*(?:\*/)?[ \t]*$`)

// ExtractSourceMappingURL returns the value of the last //# sourceMappingURL=
// (or /*# ... */) comment of a minified file.
func ExtractSourceMappingURL(content string) (string, bool) {
	// Only the tail matters; avoid regex work over megabytes of code.
	const tail = 4096
	if len(content) > tail {
		content = content[len(content)-tail:]
	}
	all := sourceMappingURLRe.FindAllStringSubmatch(content, -1)
	if len(all) == 0 {
		return "", false
	}
	return all[len(all)-1][1], true
}

// ResolveURL resolves ref (as found in a sourceMappingURL comment) against the
// URL or path of the file that contained it.
func ResolveURL(base, ref string) string {
	if u, err := url.Parse(ref); err == nil && u.IsAbs() {
		return ref
	}
	if bu, err := url.Parse(base); err == nil && bu.Scheme != "" {
		if ru, err := url.Parse(ref); err == nil {
			return bu.ResolveReference(ru).String()
		}
	}
	if strings.HasPrefix(ref, "/") {
		return ref
	}
	return path.Join(path.Dir(base), ref)
}

func decodeDataURI(ref string) ([]byte, bool) {
	rest, ok := strings.CutPrefix(ref, "data:")
	if !ok {
		return nil, false
	}
	meta, payload, ok := strings.Cut(rest, ",")
	if !ok {
		return nil, false
	}
	if strings.HasSuffix(meta, ";base64") {
		b, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			b, err = base64.RawStdEncoding.DecodeString(payload)
		}
		return b, err == nil
	}
	s, err := url.PathUnescape(payload)
	return []byte(s), err == nil
}

func stripQuery(u string) string {
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		return u[:i]
	}
	return u
}
