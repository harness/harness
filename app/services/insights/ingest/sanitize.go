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

import "strings"

// Filtered replaces removed values.
const Filtered = "[Filtered]"

// SanitizeOptions controls what Sanitize removes. The zero value removes
// nothing; use DefaultSanitizeOptions for storage.
type SanitizeOptions struct {
	// DropFrameVars removes local variables from stack frames.
	DropFrameVars bool
	// ScrubCookies replaces cookie values.
	ScrubCookies bool
	// ScrubHeaders replaces header values except SafeHeaders.
	ScrubHeaders bool
	// SafeHeaders are lower-case header names whose values are kept.
	SafeHeaders []string
}

// DefaultSanitizeOptions is the policy applied before storage. The AI
// gateway redactor still runs before any model call.
func DefaultSanitizeOptions() SanitizeOptions {
	return SanitizeOptions{
		DropFrameVars: true, ScrubCookies: true, ScrubHeaders: true,
		SafeHeaders: []string{"accept", "accept-encoding", "accept-language", "content-length", "content-type", "host", "user-agent"},
	}
}

// Sanitize removes sensitive data from ev in place.
func Sanitize(ev *Event, opts SanitizeOptions) {
	if opts.DropFrameVars {
		for i := range ev.Exception.valuesOrNil() {
			if st := ev.Exception.Values[i].Stacktrace; st != nil {
				for j := range st.Frames {
					st.Frames[j].Vars = nil
				}
			}
		}
	}
	r := ev.Request
	if r == nil {
		return
	}
	if opts.ScrubCookies {
		r.Cookies = scrubCookies(r.Cookies)
		for k := range r.Headers {
			if strings.EqualFold(k, "cookie") || strings.EqualFold(k, "set-cookie") {
				r.Headers[k] = Filtered
			}
		}
	}
	if opts.ScrubHeaders {
		safe := map[string]bool{}
		for _, h := range opts.SafeHeaders {
			safe[strings.ToLower(h)] = true
		}
		for k := range r.Headers {
			if !safe[strings.ToLower(k)] {
				r.Headers[k] = Filtered
			}
		}
	}
}

func (l *ExceptionList) valuesOrNil() []Exception {
	if l == nil {
		return nil
	}
	return l.Values
}

func scrubCookies(c any) any {
	switch v := c.(type) {
	case nil:
		return nil
	case map[string]any:
		for k := range v {
			v[k] = Filtered
		}
		return v
	default: // raw "a=b; c=d" string or unknown shape
		return Filtered
	}
}
