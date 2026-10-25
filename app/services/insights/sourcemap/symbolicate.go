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
	"errors"
	"fmt"
	"strings"

	"github.com/harness/gitness/app/services/insights/ingest"
)

const (
	// ContextLines is the number of pre/post context lines filled in.
	ContextLines = 5

	// VarsKey is the key in Frame.Vars under which the original (minified)
	// location is kept. Its presence also marks a frame as symbolicated.
	VarsKey = "_sourcemap"
)

// Symbolicate rewrites a minified frame in place using the map provided for
// release and frame.AbsPath.
//
// frame.Lineno is 1-based and frame.Colno is 1-based (Sentry JS convention;
// 0 means unknown and is treated as the first column). The rewritten Colno is
// 1-based as well. Function is replaced by the original name when the map has
// one for the position, otherwise it is kept.
//
// The minified location is preserved in frame.Vars[VarsKey] (the Frame type
// has no free-form field other than Vars): minified_filename,
// minified_abs_path, minified_function, minified_lineno, minified_colno and
// minified_context_line.
//
// It returns true when the frame was rewritten. A missing map (ErrNotFound), an
// unmapped position or an already symbolicated frame leave the frame untouched
// and return (false, nil); any other provider error is returned.
func Symbolicate(
	ctx context.Context,
	frame *ingest.Frame,
	release string,
	provider MapProvider,
) (bool, error) {
	if frame == nil || provider == nil || frame.AbsPath == "" || frame.Lineno <= 0 {
		return false, nil
	}
	if _, done := frame.Vars[VarsKey]; done {
		return false, nil
	}
	m, err := provider.Find(ctx, release, frame.AbsPath)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("find source map for %q: %w", frame.AbsPath, err)
	}
	if m == nil {
		return false, nil
	}
	col := int(frame.Colno) - 1
	if col < 0 {
		col = 0
	}
	mp, ok := m.Lookup(int(frame.Lineno), col)
	if !ok {
		return false, nil
	}

	orig := map[string]any{
		"minified_filename": frame.Filename,
		"minified_abs_path": frame.AbsPath,
		"minified_lineno":   int(frame.Lineno),
		"minified_colno":    int(frame.Colno),
	}
	if frame.Function != "" {
		orig["minified_function"] = frame.Function
	}
	if frame.ContextLine != "" {
		orig["minified_context_line"] = frame.ContextLine
	}
	if frame.Vars == nil {
		frame.Vars = map[string]any{}
	}
	frame.Vars[VarsKey] = orig

	frame.AbsPath = mp.Source
	frame.Filename = displayName(mp.Source)
	frame.Lineno = ingest.FlexInt(mp.Line)
	frame.Colno = ingest.FlexInt(mp.Column + 1)
	if mp.Name != "" {
		frame.Function = mp.Name
	}
	if isLibrary(mp.Source) {
		f := false
		frame.InApp = &f
	}

	// The minified context is meaningless for the original location.
	frame.ContextLine, frame.PreContext, frame.PostContext = "", nil, nil
	if content, ok := m.SourceContent(mp.Source); ok {
		frame.ContextLine, frame.PreContext, frame.PostContext = contextLines(content, mp.Line)
	}
	return true, nil
}

// isLibrary reports whether an original path is third-party / bundler code.
func isLibrary(source string) bool {
	return strings.Contains(source, "node_modules/") || strings.Contains(source, "webpack/bootstrap")
}

// displayName strips bundler URL prefixes ("webpack://app/./src/a.js" ->
// "src/a.js") to produce a repo-relative-looking filename.
func displayName(source string) string {
	s := source
	if _, rest, ok := strings.Cut(s, "://"); ok {
		// webpack://namespace/./path -> path ; other schemes keep the path part.
		if strings.HasPrefix(source, "webpack") {
			if _, p, ok := strings.Cut(rest, "/"); ok {
				s = p
			} else {
				s = rest
			}
		} else {
			s = rest
		}
	}
	for strings.HasPrefix(s, "./") {
		s = s[2:]
	}
	return s
}

// contextLines extracts the 1-based line and up to ContextLines lines around it.
func contextLines(content string, line int) (string, []string, []string) {
	lines := strings.Split(content, "\n")
	if line < 1 || line > len(lines) {
		return "", nil, nil
	}
	clean := func(s string) string { return strings.TrimSuffix(s, "\r") }
	i := line - 1
	var pre, post []string
	for j := max(0, i-ContextLines); j < i; j++ {
		pre = append(pre, clean(lines[j]))
	}
	for j := i + 1; j < len(lines) && j <= i+ContextLines; j++ {
		post = append(post, clean(lines[j]))
	}
	return clean(lines[i]), pre, post
}
