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

package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"regexp"
	"strings"
)

// DefaultPlaceholder in a custom fingerprint is replaced by the default components.
const DefaultPlaceholder = "{{ default }}"

// Fingerprint sources.
const (
	SourceCustom     = "custom"
	SourceStacktrace = "stacktrace"
	SourceException  = "exception"
	SourceMessage    = "message"
	SourceEmpty      = "empty"
)

// Fingerprint is the grouping key of an event plus the data it came from.
type Fingerprint struct {
	// Hash is a stable hex digest (32 chars) of Components.
	Hash string `json:"hash"`
	// Source says which rule produced the components.
	Source string `json:"source"`
	// Components are the human readable inputs of the hash, for debugging
	// and explanations.
	Components []string `json:"components"`
}

// ComputeFingerprint groups an event deterministically.
//
//  1. A custom fingerprint wins; "{{ default }}" expands to the default components.
//  2. Otherwise: the exception types of the chain plus the normalized frames
//     of the outermost (last) exception that has a stacktrace. In-app frames
//     are used when there are any, else all frames. Line numbers are ignored.
//  3. Exceptions without stacktrace: types plus the normalized exception value.
//  4. No exception: the normalized message template.
func ComputeFingerprint(ev *Event) Fingerprint {
	def, src := defaultComponents(ev)
	if len(ev.Fingerprint) > 0 {
		var comps []string
		for _, c := range ev.Fingerprint {
			if strings.TrimSpace(c) == DefaultPlaceholder || strings.ReplaceAll(c, " ", "") == "{{default}}" {
				comps = append(comps, def...)
			} else {
				comps = append(comps, c)
			}
		}
		return finish(comps, SourceCustom)
	}
	return finish(def, src)
}

func finish(comps []string, src string) Fingerprint {
	if len(comps) == 0 {
		src = SourceEmpty
		comps = []string{"<no-data>"}
	}
	h := sha256.Sum256([]byte(strings.Join(comps, "\x00")))
	return Fingerprint{Hash: hex.EncodeToString(h[:16]), Source: src, Components: comps}
}

func defaultComponents(ev *Event) ([]string, string) {
	values := ev.ExceptionValues()
	if len(values) > 0 {
		var comps []string
		for _, v := range values {
			t := v.Type
			if v.Module != "" {
				t = v.Module + "." + v.Type
			}
			comps = append(comps, "type:"+t)
		}
		// the outermost exception with a stacktrace
		for i := len(values) - 1; i >= 0; i-- {
			st := values[i].Stacktrace
			if st == nil || len(st.Frames) == 0 {
				continue
			}
			if fc := frameComponents(st.Frames); len(fc) > 0 {
				return append(comps, fc...), SourceStacktrace
			}
		}
		last := values[len(values)-1]
		if v := normalizeMessage(last.Value); v != "" {
			comps = append(comps, "value:"+v)
		}
		return comps, SourceException
	}
	if m := normalizeMessage(ev.MessageTemplate()); m != "" {
		return []string{"message:" + m}, SourceMessage
	}
	if m := normalizeMessage(ev.FormattedMessage()); m != "" {
		return []string{"message:" + m}, SourceMessage
	}
	return nil, SourceEmpty
}

func frameComponents(frames []Frame) []string {
	var sel []Frame
	for _, f := range frames {
		if f.InApp != nil && *f.InApp {
			sel = append(sel, f)
		}
	}
	if len(sel) == 0 {
		sel = frames
	}
	var out []string
	for _, f := range sel {
		loc := normalizeModule(f)
		fn := normalizeFunction(f.Function)
		if loc == "" && fn == "" {
			continue
		}
		out = append(out, "frame:"+loc+"|"+fn)
	}
	return out
}

// normalizeModule prefers the module and otherwise reduces the filename to
// its last two path segments, dropping hosts, query strings and content
// hashes so that bundles and deploy paths do not split groups.
func normalizeModule(f Frame) string {
	if f.Module != "" {
		return f.Module
	}
	p := f.Filename
	if p == "" {
		p = f.AbsPath
	}
	if p == "" {
		return ""
	}
	if strings.Contains(p, "://") {
		if u, err := url.Parse(p); err == nil {
			p = u.Path
		}
	}
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	p = strings.ReplaceAll(p, "\\", "/")
	segs := strings.Split(strings.Trim(p, "/"), "/")
	if len(segs) > 2 {
		segs = segs[len(segs)-2:]
	}
	p = strings.Join(segs, "/")
	return fileHashRe.ReplaceAllString(p, "$1")
}

var (
	fileHashRe = regexp.MustCompile(`[.-][0-9a-fA-F]{6,}(\.[A-Za-z0-9]+)$`)
	// drop extension chunk hashes like app.3f9a1c.chunk.js
	addrRe      = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	lambdaNumRe = regexp.MustCompile(`(?i)(lambda[_$]*)\d+|\$\$Lambda\$\d+(/0x[0-9a-fA-F]+)?|\$\d+`)
	goFuncRe    = regexp.MustCompile(`\.(func|gowrap|deferwrap)\d+(\.\d+)*`)
	goSuffixRe  = regexp.MustCompile(`-fm$|\.\(\*?[^)]*\)\.[^.]*-fm$`)
	anonNumRe   = regexp.MustCompile(`(?i)(anonymous|closure)[#_ ]*\d+`)
	uuidRe      = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	hexRe       = regexp.MustCompile(`(?i)\b(0x[0-9a-f]+|[0-9a-f]{8,})\b`)
	numRe       = regexp.MustCompile(`\d+(\.\d+)?`)
	spaceRe     = regexp.MustCompile(`\s+`)
)

// normalizeFunction strips what changes between builds and runs: addresses,
// lambda and closure numbering, Go generated suffixes.
func normalizeFunction(fn string) string {
	fn = strings.TrimSpace(fn)
	if fn == "" {
		return ""
	}
	fn = addrRe.ReplaceAllString(fn, "")
	fn = goFuncRe.ReplaceAllString(fn, "")
	fn = goSuffixRe.ReplaceAllString(fn, "")
	fn = lambdaNumRe.ReplaceAllString(fn, "${1}")
	fn = anonNumRe.ReplaceAllString(fn, "${1}")
	return strings.TrimSpace(fn)
}

// normalizeMessage replaces volatile tokens (uuids, hex, numbers) so that
// "user 42 not found" and "user 43 not found" group together.
func normalizeMessage(m string) string {
	m = strings.TrimSpace(m)
	if m == "" {
		return ""
	}
	m = uuidRe.ReplaceAllString(m, "<uuid>")
	m = hexRe.ReplaceAllString(m, "<hex>")
	m = numRe.ReplaceAllString(m, "<num>")
	return spaceRe.ReplaceAllString(m, " ")
}
