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

package bump

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
	"golang.org/x/mod/sumdb/dirhash"
)

// DefaultGoProxy is the module proxy used when none is configured.
const DefaultGoProxy = "https://proxy.golang.org"

const (
	maxModFileSize = 1 << 20
	maxZipSize     = 256 << 20
	goModSuffix    = "/go.mod"
)

// GoProxy downloads modules from a GOPROXY compatible server.
type GoProxy struct {
	BaseURL string
	Client  *http.Client
}

// NewGoProxy creates a proxy client; an empty baseURL selects DefaultGoProxy.
func NewGoProxy(baseURL string) *GoProxy {
	if baseURL == "" {
		baseURL = DefaultGoProxy
	}
	return &GoProxy{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client:  &http.Client{Timeout: 2 * time.Minute},
	}
}

func (p *GoProxy) get(ctx context.Context, path, version, ext string, limit int64) ([]byte, error) {
	ep, err := module.EscapePath(path)
	if err != nil {
		return nil, err
	}
	ev, err := module.EscapeVersion(version)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s/@v/%s.%s", p.BaseURL, ep, ev, ext)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: status %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", url, limit)
	}
	return data, nil
}

// Mod fetches the go.mod of the module version.
func (p *GoProxy) Mod(ctx context.Context, path, version string) ([]byte, error) {
	return p.get(ctx, path, version, "mod", maxModFileSize)
}

// ZipHash fetches the module zip and returns its h1 hash.
func (p *GoProxy) ZipHash(ctx context.Context, path, version string) (string, error) {
	data, err := p.get(ctx, path, version, "zip", maxZipSize)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "bump-*.zip")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	return dirhash.HashZip(f.Name(), dirhash.Hash1)
}

func modHash(data []byte) (string, error) {
	return dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(data)), nil
	})
}

// GoModResult is the outcome of a Go module bump.
type GoModResult struct {
	GoMod []byte
	GoSum []byte
}

// BumpGoMod raises the requirement on modPath in goMod to version `to` (same major line only) and
// adds the matching go.sum lines. goSum may be nil. Hashes are computed from the proxy content and are
// NOT verified against the checksum database; CI must run with the usual go.sum verification.
//
//nolint:gocognit,gocyclo,cyclop // linear sequence of refusal checks
func BumpGoMod(
	ctx context.Context,
	proxy *GoProxy,
	goMod, goSum []byte,
	modPath, to string,
) (*GoModResult, error) {
	f, err := modfile.Parse("go.mod", goMod, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}
	if !semver.IsValid(to) {
		return nil, needsAgent("fixed version %q is not a valid module version", to)
	}
	_, pathMajor, _ := module.SplitPathVersion(modPath)
	if err = module.CheckPathMajor(to, pathMajor); err != nil {
		return nil, needsAgent("fixed version %s is not on the major line of %s", to, modPath)
	}

	current := map[string]string{}
	for _, r := range f.Require {
		current[r.Mod.Path] = r.Mod.Version
	}
	from, ok := current[modPath]
	if !ok {
		return nil, fmt.Errorf("%w: %s is not required by go.mod", ErrUnsupported, modPath)
	}
	if semver.Compare(from, to) >= 0 {
		return nil, fmt.Errorf("%w: %s is already at %s (>= %s)", ErrUnsupported, modPath, from, to)
	}
	if semver.Major(from) != semver.Major(to) {
		return nil, needsAgent("%s -> %s crosses a major version", from, to)
	}
	for _, r := range f.Replace {
		if r.Old.Path == modPath {
			return nil, needsAgent("%s is replaced in go.mod", modPath)
		}
	}

	newModData, err := proxy.Mod(ctx, modPath, to)
	if err != nil {
		return nil, err
	}
	newMod, err := modfile.ParseLax("go.mod", newModData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod of %s@%s: %w", modPath, to, err)
	}
	if newMod.Go != nil {
		if f.Go == nil {
			return nil, needsAgent("%s@%s requires go %s but go.mod has no go directive",
				modPath, to, newMod.Go.Version)
		}
		if goVersionCompare(newMod.Go.Version, f.Go.Version) > 0 {
			return nil, needsAgent("%s@%s requires go %s, project declares go %s",
				modPath, to, newMod.Go.Version, f.Go.Version)
		}
	}
	for _, r := range newMod.Require {
		have, present := current[r.Mod.Path]
		if !present || semver.Compare(have, r.Mod.Version) < 0 {
			return nil, needsAgent("%s@%s changes requirements: %s %s (project has %q)",
				modPath, to, r.Mod.Path, r.Mod.Version, have)
		}
	}

	if err = f.AddRequire(modPath, to); err != nil {
		return nil, fmt.Errorf("failed to update requirement: %w", err)
	}
	f.Cleanup()
	out, err := f.Format()
	if err != nil {
		return nil, fmt.Errorf("failed to format go.mod: %w", err)
	}

	sum, err := updateGoSum(ctx, proxy, goSum, modPath, to, newModData, newMod.Require)
	if err != nil {
		return nil, err
	}
	return &GoModResult{GoMod: out, GoSum: sum}, nil
}

func updateGoSum(
	ctx context.Context,
	proxy *GoProxy,
	goSum []byte,
	modPath, to string,
	newModData []byte,
	reqs []*modfile.Require,
) ([]byte, error) {
	lines := map[string]struct{}{}
	have := map[string]struct{}{}
	for _, l := range strings.Split(string(goSum), "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		lines[l] = struct{}{}
		if fs := strings.Fields(l); len(fs) >= 2 {
			have[fs[0]+" "+fs[1]] = struct{}{}
		}
	}
	add := func(path, version, hash string) {
		lines[path+" "+version+" "+hash] = struct{}{}
		have[path+" "+version] = struct{}{}
	}

	if _, ok := have[modPath+" "+to]; !ok {
		h, err := proxy.ZipHash(ctx, modPath, to)
		if err != nil {
			return nil, err
		}
		add(modPath, to, h)
	}
	if _, ok := have[modPath+" "+to+goModSuffix]; !ok {
		h, err := modHash(newModData)
		if err != nil {
			return nil, err
		}
		add(modPath, to+goModSuffix, h)
	}
	// the module graph of the new version needs the go.mod hashes of its requirements.
	for _, r := range reqs {
		if _, ok := have[r.Mod.Path+" "+r.Mod.Version+goModSuffix]; ok {
			continue
		}
		data, err := proxy.Mod(ctx, r.Mod.Path, r.Mod.Version)
		if err != nil {
			return nil, err
		}
		h, err := modHash(data)
		if err != nil {
			return nil, err
		}
		add(r.Mod.Path, r.Mod.Version+goModSuffix, h)
	}

	sorted := make([]string, 0, len(lines))
	for l := range lines {
		sorted = append(sorted, l)
	}
	// same order as the go command: by module path, then version, "/go.mod" entry after the zip entry.
	sort.Slice(sorted, func(i, j int) bool {
		a, b := strings.Fields(sorted[i]), strings.Fields(sorted[j])
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		va, vb := strings.TrimSuffix(a[1], goModSuffix), strings.TrimSuffix(b[1], goModSuffix)
		if c := semver.Compare(va, vb); c != 0 {
			return c < 0
		}
		return a[1] < b[1]
	})
	return []byte(strings.Join(sorted, "\n") + "\n"), nil
}

// goVersionCompare compares go directive versions such as "1.21", "1.21.3" or "1.22rc1".
func goVersionCompare(a, b string) int {
	pa, pb := goVersionParts(a), goVersionParts(b)
	for i := range pa {
		if pa[i] != pb[i] {
			if pa[i] < pb[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func goVersionParts(v string) [3]int {
	var out [3]int
	for i, s := range strings.SplitN(v, ".", 3) {
		n := 0
		for n < len(s) && s[n] >= '0' && s[n] <= '9' {
			n++
		}
		out[i], _ = strconv.Atoi(s[:n])
	}
	return out
}
