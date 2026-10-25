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
	"io"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

const (
	maxImportLines     = 3
	maxImportLineChars = 120
)

// ecosystemExts are the source file extensions searched for imports per OSV ecosystem.
var ecosystemExts = map[string][]string{
	"Go":   {".go"},
	"npm":  {".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs"},
	"PyPI": {".py"},
}

type sourceFile struct {
	path    string
	content string
}

// importIndex searches the source files of one commit for imports of packages. Files are read once
// per ecosystem and bounded by file count and bytes.
type importIndex struct {
	git    GitReader
	repo   *types.RepositoryCore
	commit string
	cfg    TriageConfig

	paths    []string
	listed   bool
	files    map[string][]sourceFile
	loadedOK map[string]bool
}

func newImportIndex(g GitReader, repo *types.RepositoryCore, commit string, cfg TriageConfig) *importIndex {
	return &importIndex{
		git: g, repo: repo, commit: commit, cfg: cfg,
		files: map[string][]sourceFile{}, loadedOK: map[string]bool{},
	}
}

// usages returns how many files import the package (-1 if unknown) and up to maxImportLines short
// import lines.
func (x *importIndex) usages(ctx context.Context, ecosystem, pkg string) (int, []string) {
	re := importRegexp(ecosystem, pkg)
	if re == nil {
		return -1, nil
	}
	files, ok := x.load(ctx, ecosystem)
	if !ok {
		return -1, nil
	}
	count := 0
	var lines []string
	for _, f := range files {
		loc := re.FindStringIndex(f.content)
		if loc == nil {
			continue
		}
		count++
		if len(lines) < maxImportLines {
			lines = append(lines, f.path+": "+shortLine(f.content, loc[0]))
		}
	}
	return count, lines
}

func shortLine(content string, at int) string {
	start := strings.LastIndexByte(content[:at], '\n') + 1
	end := strings.IndexByte(content[at:], '\n')
	if end < 0 {
		end = len(content) - at
	}
	line := strings.TrimSpace(content[start : at+end])
	if len(line) > maxImportLineChars {
		line = line[:maxImportLineChars]
	}
	return line
}

func importRegexp(ecosystem, pkg string) *regexp.Regexp {
	q := regexp.QuoteMeta(pkg)
	switch ecosystem {
	case "Go":
		return regexp.MustCompile(`"` + q + `(/[^"]*)?"`)
	case "npm":
		return regexp.MustCompile("['\"`]" + q + "(/[^'\"`]*)?['\"`]")
	case "PyPI":
		mod := regexp.QuoteMeta(strings.ReplaceAll(strings.ToLower(pkg), "-", "_"))
		return regexp.MustCompile(`(?mi)^\s*(import|from)\s+` + mod + `(\.|\s|,|$)`)
	}
	return nil
}

func (x *importIndex) load(ctx context.Context, ecosystem string) ([]sourceFile, bool) {
	if x.loadedOK[ecosystem] {
		return x.files[ecosystem], true
	}
	exts := ecosystemExts[ecosystem]
	if len(exts) == 0 {
		return nil, false
	}
	read := git.ReadParams{RepoUID: x.repo.GitUID}
	if !x.listed {
		out, err := x.git.ListPaths(ctx, &git.ListPathsParams{ReadParams: read, GitREF: x.commit})
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("shield triage: failed to list paths")
			return nil, false
		}
		x.paths = append([]string(nil), out.Files...)
		sort.Strings(x.paths)
		x.listed = true
	}

	var files []sourceFile
	var total int64
	for _, p := range x.paths {
		if len(files) >= x.cfg.MaxFiles || total >= x.cfg.MaxTotalBytes {
			break
		}
		if !hasExt(p, exts) || skipDir(p) {
			continue
		}
		content, err := x.read(ctx, read, p)
		if err != nil {
			continue
		}
		total += int64(len(content))
		files = append(files, sourceFile{path: p, content: content})
	}
	x.files[ecosystem], x.loadedOK[ecosystem] = files, true
	return files, true
}

func (x *importIndex) read(ctx context.Context, read git.ReadParams, p string) (string, error) {
	node, err := x.git.GetTreeNode(ctx, &git.GetTreeNodeParams{ReadParams: read, GitREF: x.commit, Path: p})
	if err != nil {
		return "", err
	}
	blob, err := x.git.GetBlob(ctx, &git.GetBlobParams{
		ReadParams: read, SHA: node.Node.SHA, SizeLimit: x.cfg.MaxFileBytes,
	})
	if err != nil {
		return "", err
	}
	defer blob.Content.Close()
	b, err := io.ReadAll(io.LimitReader(blob.Content, x.cfg.MaxFileBytes))
	return string(b), err
}

func hasExt(p string, exts []string) bool {
	e := path.Ext(p)
	for _, x := range exts {
		if e == x {
			return true
		}
	}
	return false
}

func skipDir(p string) bool {
	for _, seg := range strings.Split(p, "/") {
		if seg == "node_modules" || seg == "vendor" || seg == "dist" {
			return true
		}
	}
	return false
}
