// Source: https://github.com/goharbor/harbor

// Copyright 2016 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lib

import (
	"fmt"
	"strings"
)

// Link defines the model that describes the HTTP link header.
type Link struct {
	URL   string
	Rel   string
	Attrs map[string]string
}

// String returns the string representation of a link.
func (l *Link) String() string {
	s := fmt.Sprintf("<%s>", l.URL)
	if len(l.Rel) > 0 {
		s = fmt.Sprintf(`%s; rel="%s"`, s, l.Rel)
	}
	for key, value := range l.Attrs {
		s = fmt.Sprintf(`%s; %s="%s"`, s, key, value)
	}
	return s
}

// Links is a link object array.
type Links []*Link

// String returns the string representation of links.
func (l Links) String() string {
	var strs []string
	for _, link := range l {
		strs = append(strs, link.String())
	}
	return strings.Join(strs, " , ")
}

// ParseLinks parses the link header into Links
// e.g. <http://example.com/TheBook/chapter2>; rel="previous";
// title="previous chapter" , <http://example.com/TheBook/chapter4>; rel="next"; title="next chapter".
func ParseLinks(str string) Links {
	var links Links
	for lk := range strings.SplitSeq(str, ",") {
		link := &Link{
			Attrs: map[string]string{},
		}
		for attr := range strings.SplitSeq(lk, ";") {
			attr = strings.TrimSpace(attr)
			if len(attr) == 0 {
				continue
			}
			if attr[0] == '<' && attr[len(attr)-1] == '>' {
				link.URL = attr[1 : len(attr)-1]
				continue
			}

			parts := strings.SplitN(attr, "=", 2)
			key := parts[0]
			value := ""
			if len(parts) == 2 {
				value = strings.Trim(parts[1], `"`)
			}
			if key == "rel" {
				link.Rel = value
			} else {
				link.Attrs[key] = value
			}
		}
		if len(link.URL) == 0 {
			continue
		}
		links = append(links, link)
	}
	return links
}
