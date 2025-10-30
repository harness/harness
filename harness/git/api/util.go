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

package api

import (
	"bytes"
	"unicode"

	"github.com/yuin/goldmark/util"
)

const (
	userPlaceholder = "sanitized-credential"
)

var schemeSep = []byte("://")

// SanitizeCredentialURLs remove all credentials in URLs (starting with "scheme://")
// for the input string: "https://user:pass@domain.com" => "https://sanitized-credential@domain.com"
func SanitizeCredentialURLs(s string) string {
	bs := util.StringToReadOnlyBytes(s)
	schemeSepPos := bytes.Index(bs, schemeSep)
	if schemeSepPos == -1 || bytes.IndexByte(bs[schemeSepPos:], '@') == -1 {
		return s // fast return if there is no URL scheme or no userinfo
	}
	out := make([]byte, 0, len(bs)+len(userPlaceholder))
	for schemeSepPos != -1 {
		schemeSepPos += 3         // skip the "://"
		sepAtPos := -1            // the possible '@' position: "https://foo@[^here]host"
		sepEndPos := schemeSepPos // the possible end position: "The https://host[^here] in log for test"
	sepLoop:
		for ; sepEndPos < len(bs); sepEndPos++ {
			c := bs[sepEndPos]
			if ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z') || ('0' <= c && c <= '9') {
				continue
			}
			switch c {
			case '@':
				sepAtPos = sepEndPos
			case '-', '.', '_', '~', '!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '=', ':', '%':
				continue // due to RFC 3986, userinfo can contain - . _ ~ ! $ & ' ( ) * + , ; = : and any percent-encoded chars
			default:
				break sepLoop // if it is an invalid char for URL (eg: space, '/', and others), stop the loop
			}
		}
		// if there is '@', and the string is like "s://u@h", then hide the "u" part
		if sepAtPos != -1 && (schemeSepPos >= 4 && unicode.IsLetter(rune(bs[schemeSepPos-4]))) &&
			sepAtPos-schemeSepPos > 0 && sepEndPos-sepAtPos > 0 {
			out = append(out, bs[:schemeSepPos]...)
			out = append(out, userPlaceholder...)
			out = append(out, bs[sepAtPos:sepEndPos]...)
		} else {
			out = append(out, bs[:sepEndPos]...)
		}
		bs = bs[sepEndPos:]
		schemeSepPos = bytes.Index(bs, schemeSep)
	}
	out = append(out, bs...)
	return util.BytesToReadOnlyString(out)
}
