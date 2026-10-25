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
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// NPMLockParser reads package-lock.json and npm-shrinkwrap.json with
// lockfileVersion 2 or 3 (the "packages" map). Version 1 lockfiles, which
// only have a nested "dependencies" tree, return ErrUnsupportedFormat.
// Linked workspace packages and entries without a version are skipped.
type NPMLockParser struct{}

func (NPMLockParser) Ecosystem() string { return EcosystemNPM }

func (NPMLockParser) Handles(filePath string) bool {
	b := baseName(filePath)
	return b == "package-lock.json" || b == "npm-shrinkwrap.json"
}

type npmLock struct {
	LockfileVersion int                   `json:"lockfileVersion"`
	Packages        map[string]npmLockPkg `json:"packages"`
}

type npmLockPkg struct {
	Name                 string            `json:"name"`
	Version              string            `json:"version"`
	Link                 bool              `json:"link"`
	Resolved             string            `json:"resolved"`
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
}

const nodeModules = "node_modules/"

func (NPMLockParser) Parse(filePath string, r io.Reader) (*ParseResult, error) {
	var lock npmLock
	if err := json.NewDecoder(r).Decode(&lock); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if len(lock.Packages) == 0 {
		return nil, fmt.Errorf("%w: lockfileVersion %d has no \"packages\" map", ErrUnsupportedFormat, lock.LockfileVersion)
	}
	direct := map[string]bool{}
	if root, ok := lock.Packages[""]; ok {
		for _, m := range []map[string]string{
			root.Dependencies, root.DevDependencies, root.OptionalDependencies, root.PeerDependencies,
		} {
			for n := range m {
				direct[n] = true
			}
		}
	}
	var pkgs []Package
	for key, p := range lock.Packages {
		if key == "" || p.Link || p.Version == "" {
			continue
		}
		i := strings.LastIndex(key, nodeModules)
		if i < 0 {
			continue // workspace member such as "packages/foo"
		}
		short := key[i+len(nodeModules):]
		name := short
		if p.Name != "" {
			// Aliased install (npm:real@x): "name" holds the real package name.
			name = p.Name
		}
		pkgs = append(pkgs, Package{
			Ecosystem: EcosystemNPM,
			Name:      name,
			Version:   p.Version,
			Path:      filePath,
			Direct:    key == nodeModules+short && direct[short],
		})
	}
	return &ParseResult{Packages: dedupePackages(pkgs)}, nil
}
