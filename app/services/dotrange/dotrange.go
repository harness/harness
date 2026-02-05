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

package dotrange

import (
	"strings"

	"github.com/harness/gitness/app/api/usererror"
)

const dotRangeUpstreamMarker = "upstream:"

type DotRange struct {
	BaseRef      string
	BaseUpstream bool
	HeadRef      string
	HeadUpstream bool
	MergeBase    bool
}

func (r DotRange) String() string {
	sb := strings.Builder{}

	if r.BaseUpstream {
		sb.WriteString(dotRangeUpstreamMarker)
	}
	sb.WriteString(r.BaseRef)

	sb.WriteString("..")
	if r.MergeBase {
		sb.WriteByte('.')
	}

	if r.HeadUpstream {
		sb.WriteString(dotRangeUpstreamMarker)
	}
	sb.WriteString(r.HeadRef)

	return sb.String()
}

func ParsePath(path string) (DotRange, error) {
	mergeBase := true
	parts := strings.SplitN(path, "...", 2)
	if len(parts) != 2 {
		mergeBase = false
		parts = strings.SplitN(path, "..", 2)
		if len(parts) != 2 {
			return DotRange{}, usererror.BadRequestf("Invalid format %q", path)
		}
	}

	dotRange, err := Make(parts[0], parts[1], mergeBase)
	if err != nil {
		return DotRange{}, err
	}

	return dotRange, nil
}

func Make(base, head string, mergeBase bool) (DotRange, error) {
	dotRange := DotRange{
		BaseRef:   base,
		HeadRef:   head,
		MergeBase: mergeBase,
	}

	dotRange.BaseRef, dotRange.BaseUpstream = strings.CutPrefix(dotRange.BaseRef, dotRangeUpstreamMarker)
	dotRange.HeadRef, dotRange.HeadUpstream = strings.CutPrefix(dotRange.HeadRef, dotRangeUpstreamMarker)

	if dotRange.BaseUpstream && dotRange.HeadUpstream {
		return DotRange{}, usererror.BadRequestf("Only one upstream reference is allowed: %q", dotRange.String())
	}

	return dotRange, nil
}
