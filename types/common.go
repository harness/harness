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

package types

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/harness/gitness/errors"
)

type TagPartType string

const (
	TagPartTypeKey   TagPartType = "key"
	TagPartTypeValue TagPartType = "value"
)

func SanitizeTag(text *string, typ TagPartType, requireNonEmpty bool) error {
	if text == nil {
		return nil
	}

	*text = strings.TrimSpace(*text)

	if requireNonEmpty && len(*text) == 0 {
		return errors.InvalidArgumentf("%s must be a non-empty string", typ)
	}

	const maxTagLength = 50
	if utf8.RuneCountInString(*text) > maxTagLength {
		return errors.InvalidArgumentf("%s can have at most %d characters", typ, maxTagLength)
	}

	for _, ch := range *text {
		if unicode.IsControl(ch) {
			return errors.InvalidArgumentf("%s cannot contain control characters", typ)
		}
	}

	return nil
}
