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

package codeowners

import (
	"reflect"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
)

func TestService_ParseCodeOwner(t *testing.T) {
	type fields struct {
		repoStore store.RepoStore
		git       git.Interface
		Config    Config
	}
	type args struct {
		codeOwnersContent string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []Entry
		wantErr bool
	}{
		{
			name: "Code owners Single",
			args: args{`**/contracts/openapi/v1/ user1@harness.io user2@harness.io`},
			want: []Entry{
				{
					LineNumber: 1,
					Pattern:    "**/contracts/openapi/v1/",
					Owners:     []string{"user1@harness.io", "user2@harness.io"},
				},
			},
		},
		{
			name: "Code owners Multiple",
			args: args{`
**/contracts/openapi/v1/ user1@harness.io user2@harness.io
/scripts/api user3@harness.io user4@harness.io
			`},
			want: []Entry{
				{
					LineNumber: 2,
					Pattern:    "**/contracts/openapi/v1/",
					Owners:     []string{"user1@harness.io", "user2@harness.io"},
				},
				{
					LineNumber: 3,
					Pattern:    "/scripts/api",
					Owners:     []string{"user3@harness.io", "user4@harness.io"},
				},
			},
		},
		{
			name: " Code owners with full line comments",
			args: args{`
# codeowner file
**/contracts/openapi/v1/ user1@harness.io user2@harness.io
#
/scripts/api user1@harness.io user2@harness.io
			`},
			want: []Entry{
				{
					LineNumber: 3,
					Pattern:    "**/contracts/openapi/v1/",
					Owners:     []string{"user1@harness.io", "user2@harness.io"},
				},
				{
					LineNumber: 5,
					Pattern:    "/scripts/api",
					Owners:     []string{"user1@harness.io", "user2@harness.io"},
				},
			},
		},
		{
			name: " Code owners with reset",
			args: args{`
* user1@harness.io
/scripts/api
			`},
			want: []Entry{
				{
					LineNumber: 2,
					Pattern:    "*",
					Owners:     []string{"user1@harness.io"},
				},
				{
					LineNumber: 3,
					Pattern:    "/scripts/api",
					Owners:     []string{},
				},
			},
		},
		{
			name: " Code owners with escaped characters in pattern",
			args: args{`
# escaped escape character
\\ user1@harness.io
# escaped control characters (are unescaped)
\ \	\# user2@harness.io
# escaped special pattern syntax characters (stay escaped)
\*\?\[\]\{\}\-\!\^  user3@harness.io
# mix of escapes
\\\ \*\\\\\? user4@harness.io
			`},
			want: []Entry{
				{

					LineNumber: 3,
					Pattern:    "\\\\",
					Owners:     []string{"user1@harness.io"},
				},
				{

					LineNumber: 5,
					Pattern:    " 	#",
					Owners:     []string{"user2@harness.io"},
				},
				{
					LineNumber: 7,
					Pattern:    "\\*\\?\\[\\]\\{\\}\\-\\!\\^",
					Owners:     []string{"user3@harness.io"},
				},
				{
					LineNumber: 9,
					Pattern:    "\\\\ \\*\\\\\\\\\\?",
					Owners:     []string{"user4@harness.io"},
				},
			},
		},
		{
			name: " Code owners with multiple spaces as divider",
			args: args{`
*    user1@harness.io       user2@harness.io
			`},
			want: []Entry{
				{
					LineNumber: 2,
					Pattern:    "*",
					Owners:     []string{"user1@harness.io", "user2@harness.io"},
				},
			},
		},
		{
			name: " Code owners with invalid escaping standalone '\\'",
			args: args{`
\
			`},
			wantErr: true,
		},
		{
			name: " Code owners with invalid escaping unsupported char",
			args: args{`
\a
			`},
			wantErr: true,
		},
		{
			name: " Code owners with utf8",
			args: args{`
D∆NCE user@h∆rness.io
			`},
			want: []Entry{
				{
					LineNumber: 2,
					Pattern:    "D∆NCE",
					Owners:     []string{"user@h∆rness.io"},
				},
			},
		},
		{
			name: " Code owners with tabs and spaces",
			args: args{`
a\		user1@harness.io	user2@harness.io  	 user3@harness.io

			`},
			want: []Entry{
				{
					LineNumber: 2,
					Pattern:    "a	",
					Owners:     []string{"user1@harness.io", "user2@harness.io", "user3@harness.io"},
				},
			},
		},
		{
			name: " Code owners with inline comments",
			args: args{`
a #user1@harness.io
b	  # user1@harness.io
c #
d#
e# user1@harness.io
f user1@harness.io#user2@harness.io
g   user1@harness.io	#  user2@harness.io
			`},
			want: []Entry{
				{
					LineNumber: 2,
					Pattern:    "a",
					Owners:     []string{},
				},
				{
					LineNumber: 3,
					Pattern:    "b",
					Owners:     []string{},
				},
				{
					LineNumber: 4,
					Pattern:    "c",
					Owners:     []string{},
				},
				{
					LineNumber: 5,
					Pattern:    "d",
					Owners:     []string{},
				},
				{
					LineNumber: 6,
					Pattern:    "e",
					Owners:     []string{},
				},
				{
					LineNumber: 7,
					Pattern:    "f",
					Owners:     []string{"user1@harness.io"},
				},
				{
					LineNumber: 8,
					Pattern:    "g",
					Owners:     []string{"user1@harness.io"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				repoStore: tt.fields.repoStore,
				git:       tt.fields.git,
				config:    tt.fields.Config,
			}
			got, err := s.parseCodeOwner(tt.args.codeOwnersContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCodeOwner() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCodeOwner() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_match(t *testing.T) {
	type args struct {
		pattern            string
		matchingTargets    []string
		nonMatchingTargets []string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "root",
			args: args{
				pattern:            "/",
				matchingTargets:    []string{"a.txt", "x/a.txt", "x/y/a.txt"},
				nonMatchingTargets: []string{},
			},
		},
		{
			name: "file exact match",
			args: args{
				pattern:            "/a.txt",
				matchingTargets:    []string{"a.txt", "a.txt/b.go", "a.txt/b.go/c.ar"},
				nonMatchingTargets: []string{"a.txt2", "b.txt", "a.go", "x/a.txt"},
			},
		},
		{
			name: "file exact match with directory",
			args: args{
				pattern:            "/x/a.txt",
				matchingTargets:    []string{"x/a.txt", "x/a.txt/b.txt", "x/a.txt/b.go/c.ar"},
				nonMatchingTargets: []string{"a.txt", "x/a.txt2", "x/b.txt", "x/a.go"},
			},
		},
		{
			name: "file relative match",
			args: args{
				pattern:            "a.txt",
				matchingTargets:    []string{"a.txt", "x/a.txt", "x/y/a.txt", "x/y/a.txt/b.go/c.ar"},
				nonMatchingTargets: []string{"a.txt2", "b.txt", "a.go", "x/a.txt2"},
			},
		},
		{
			name: "file relative match with directory",
			args: args{
				pattern:            "x/a.txt",
				matchingTargets:    []string{"x/a.txt", "x/a.txt/b.go", "x/a.txt/b.go/c.ar"},
				nonMatchingTargets: []string{"a.txt2", "b.txt", "a.go", "x/a.txt2", "v/x/a.txt", "y/a.txt"},
			},
		},
		{
			name: "folder exact match",
			args: args{
				pattern:            "/x/",
				matchingTargets:    []string{"x/a.txt", "x/b.go", "x/y/a.txt"},
				nonMatchingTargets: []string{"x", "a.txt", "y/a.txt", "w/x/a.txt"},
			},
		},
		{
			name: "folder relative match",
			args: args{
				pattern:            "x/",
				matchingTargets:    []string{"x/a.txt", "x/b.txt", "w/x/a.txt", "w/x/y/a.txt"},
				nonMatchingTargets: []string{"x", "w/x", "a.txt", "y/a.txt"},
			},
		},
		{
			name: "match-all",
			args: args{
				pattern:            "*",
				matchingTargets:    []string{"a", "a.txt", "x/a.txt", "x/y/a.txt"},
				nonMatchingTargets: []string{},
			},
		},
		{
			name: "match-all in relative dir",
			args: args{
				pattern:            "x/*",
				matchingTargets:    []string{"x/a.txt"},
				nonMatchingTargets: []string{"x", "y/a.txt", "w/x/b.go", "x/a.txt/b.txt"},
			},
		},
		{
			name: "match-all in absolute dir",
			args: args{
				pattern:            "/x/*",
				matchingTargets:    []string{"x/a.txt", "x/b.go"},
				nonMatchingTargets: []string{"x", "y/a.txt", "w/x/a.txt", "x/a.txt/b.go"},
			},
		},
		{
			name: "file match-all type",
			args: args{
				pattern:            "*.txt",
				matchingTargets:    []string{"a.txt", "x/a.txt", "x/y/a.txt", "x/y/a.txt/b.go", "x/y/a.txt/c.ar"},
				nonMatchingTargets: []string{"a.txt2", "a.go"},
			},
		},
		{
			name: "file match-all type in root folder",
			args: args{
				pattern:            "/*.txt",
				matchingTargets:    []string{"a.txt", "a.txt/b.go", "a.txt/b.go/c.ar"},
				nonMatchingTargets: []string{"a.txt2", "a.go", "x/a.txt", "x/y/a.txt"},
			},
		},
		{
			name: "file match-all type in absolute sub folder",
			args: args{
				pattern:            "/x/*.txt",
				matchingTargets:    []string{"x/a.txt", "x/a.txt/b.go", "x/a.txt/b.go/c.ar"},
				nonMatchingTargets: []string{"a.txt", "x/a.txt2", "x/a.go", "w/x/a.txt", "y/a.txt"},
			},
		},
		{
			name: "file match-all types in relative sub folder",
			args: args{
				pattern:            "x/*.txt",
				matchingTargets:    []string{"x/a.txt", "x/a.txt/b.go", "x/a.txt/b.go/c.ar"},
				nonMatchingTargets: []string{"a.txt", "x/a.txt2", "x/a.go", "w/x/a.txt", "y/a.txt"},
			},
		},
		{
			name: "inner match-all",
			args: args{
				pattern:            "/x/*/a.txt",
				matchingTargets:    []string{"x/y/a.txt", "x/y/a.txt/b.go", "x/y/a.txt/b.go/c.ar"},
				nonMatchingTargets: []string{"a.txt", "x/a.txt", "w/x/y/a.txt", "x/y/z/a.txt"},
			},
		},
		{
			name: "escaped match-all",
			args: args{
				pattern:            "\\*",
				matchingTargets:    []string{"*", "x/y/*", "x/y/*/b.go/c.ar"},
				nonMatchingTargets: []string{"a.txt"},
			},
		},
		/*
			TODO: Fix bug in doublestar library, currently doesn't match `a.` ...
				{
					name: "trailing match-all on string",
					args: args{
						pattern:            "a.*",
						matchingTargets:    []string{"a.", "a.txt", "x/a.txt", "x/a.txt/b.go", "x/a.txt/b.go/c.ar"},
						nonMatchingTargets: []string{"atxt", "b.txt"},
					},
				},
		*/
		{
			name: "globstar",
			args: args{
				pattern:            "**",
				matchingTargets:    []string{"a", "a.txt", "x/a.txt", "x/y/a.txt"},
				nonMatchingTargets: []string{},
			},
		},
		{
			name: "trailing globstar absolute path",
			args: args{
				pattern:            "/x/**",
				matchingTargets:    []string{"x/a.txt", "x/b.txt", "x/y/a.txt"},
				nonMatchingTargets: []string{"a.txt", "x", "y/a.txt", "w/x/a.txt"},
			},
		},
		{
			name: "trailing globstar relative path",
			args: args{
				pattern:            "x/**",
				matchingTargets:    []string{"x/a.txt", "x/b.txt", "x/y/a.txt"},
				nonMatchingTargets: []string{"a.txt", "x", "y/a.txt", "w/x/a.txt"},
			},
		},
		/*
			TODO: Fix bug in doublestar library, currently doesn't match `a.` ...
				{
					name: "trailing globstar on string",
					args: args{
						pattern:            "a.**",
						matchingTargets:    []string{"a.", "a.txt", "x/a.txt", "x/a.txt/b.go", "x/a.txt/b.go/c.ar"},
						nonMatchingTargets: []string{"atxt", "b.txt"},
					},
				},
		*/
		{
			name: "leading globstar",
			args: args{
				pattern:            "**/a.txt",
				matchingTargets:    []string{"a.txt", "x/a.txt", "x/y/a.txt", "x/y/a.txt/b.go", "x/y/a.txt/b.go/c.ar"},
				nonMatchingTargets: []string{"b.txt", "a.txt2"},
			},
		},
		{
			name: "surrounding globstar",
			args: args{
				pattern:            "**/x/**",
				matchingTargets:    []string{"x/a.txt", "w/x/a.txt", "x/y/a.txt", "w/x/y/a.txt"},
				nonMatchingTargets: []string{"a.txt", "x", "w/x"},
			},
		},
		{
			name: "inner globstar",
			args: args{
				pattern: "/x/**/a.txt",
				matchingTargets: []string{
					"x/a.txt", "x/y/a.txt", "x/y/z/a.txt", "x/y/z/a.txt/b.go", "x/y/z/a.txt/b.go/c.ar"},
				nonMatchingTargets: []string{"a.txt", "w/x/a.txt", "y/a.txt"},
			},
		},
		{
			name: "multi-inner globstar",
			args: args{
				pattern: "/x/**/z/**/a.txt",
				matchingTargets: []string{
					"x/z/a.txt",
					"x/y/z/l/a.txt",
					"x/y/yy/z/l/ll/a.txt",
					"x/y/yy/z/l/ll/a.txt/b.go",
					"x/y/yy/z/l/ll/a.txt/b.go/c.ar",
				},
				nonMatchingTargets: []string{"a.txt", "x/a.txt", "z/a.txt", "w/x/a.txt", "y/a.txt"},
			},
		},
		{
			name: "dirty globstar",
			args: args{
				pattern:            "/a**.txt",
				matchingTargets:    []string{"a.txt", "abc.txt", "a.txt/b.go", "a.txt/b.go/c.ar"},
				nonMatchingTargets: []string{"a/b/.txt"},
			},
		},
		{
			name: "escaped globstar",
			args: args{
				pattern:            "\\*\\*",
				matchingTargets:    []string{"**", "x/**", "**/y", "x/**/y"},
				nonMatchingTargets: []string{"x"},
			},
		},
		{
			name: "partially escaped globstar",
			args: args{
				pattern:            "*\\*",
				matchingTargets:    []string{"*", "**", "a*", "x/*", "x/a*", "*/y", "a*/", "x/*/y", "x/a*/y"},
				nonMatchingTargets: []string{"x"},
			},
		},
		{
			name: "single wildchar",
			args: args{
				pattern:            "/a.?xt",
				matchingTargets:    []string{"a.txt", "a.xxt/b.go", "a.xxt/b.go/c.ar"},
				nonMatchingTargets: []string{"x/a.txt", "z/a.txt", "w/x/a.txt", "y/a.txt", "a./xt"},
			},
		},
		{
			name: "escaped single wildchar",
			args: args{
				pattern:            "/a.\\?xt",
				matchingTargets:    []string{"a.?xt", "a.?xt/b.go", "a.?xt/b.go/c.ar"},
				nonMatchingTargets: []string{"a.\\?xt", "a.txt", "x/a.?xt"},
			},
		},
		{
			name: "class",
			args: args{
				pattern:            "/[abc].txt",
				matchingTargets:    []string{"a.txt", "b.txt", "c.txt"},
				nonMatchingTargets: []string{"[a-c].txt", "d.txt", "A.txt"},
			},
		},
		{
			name: "range class",
			args: args{
				pattern:            "/[a-c].txt",
				matchingTargets:    []string{"a.txt", "b.txt", "c.txt"},
				nonMatchingTargets: []string{"[a-c].txt", "d.txt", "A.txt"},
			},
		},
		{
			name: "escaped class",
			args: args{
				pattern:            "/\\[a-c\\].txt",
				matchingTargets:    []string{"[a-c].txt"},
				nonMatchingTargets: []string{"\\[a-c\\].txt", "a.txt", "b.txt", "c.txt"},
			},
		},
		{
			name: "class escaped control chars",
			args: args{
				pattern:            "/[\\!\\^\\-a-c].txt",
				matchingTargets:    []string{"a.txt", "b.txt", "c.txt", "^.txt", "!.txt", "-.txt"},
				nonMatchingTargets: []string{"d.txt", "[\\!\\^\\-a-c].txt", "[!^-a-c].txt"},
			},
		},
		{
			name: "inverted class ^",
			args: args{
				pattern:            "/[^a-c].txt",
				matchingTargets:    []string{"d.txt", "B.txt"},
				nonMatchingTargets: []string{"a.txt", "b.txt", "c.txt", "[^a-c].txt", "[a-c].txt"},
			},
		},
		{
			name: "escaped inverted class ^",
			args: args{
				pattern:            "/\\[^a-c\\].txt",
				matchingTargets:    []string{"[^a-c].txt"},
				nonMatchingTargets: []string{"\\[^a-c\\].txt", "a.txt", "b.txt", "c.txt", "d.txt", "[a-c].txt"},
			},
		},
		{
			name: "inverted class !",
			args: args{
				pattern:            "/[!a-c].txt",
				matchingTargets:    []string{"d.txt", "B.txt"},
				nonMatchingTargets: []string{"a.txt", "b.txt", "c.txt", "[!a-c].txt", "[a-c].txt"},
			},
		},
		{
			name: "escaped inverted class !",
			args: args{
				pattern:            "/\\[!a-c\\].txt",
				matchingTargets:    []string{"[!a-c].txt"},
				nonMatchingTargets: []string{"\\[!a-c\\].txt", "a.txt", "b.txt", "c.txt", "d.txt", "[a-c].txt"},
			},
		},
		{
			name: "alternate matches",
			args: args{
				pattern: "/{a,b,[c-d],e?,f\\*}.txt",
				matchingTargets: []string{
					"a.txt", "b.txt", "c.txt", "d.txt", "e2.txt", "f*.txt", "a.txt/b.go", "a.txt/b.go/c.ar"},
				nonMatchingTargets: []string{
					"{a,b,[c-d],e?,f\\*}.txt", "{a,b,[c-d],e?,f*}.txt", "e.txt", "f.txt", "g.txt", "ab.txt"},
			},
		},
		{
			name: "space",
			args: args{
				pattern:            "/a b.txt",
				matchingTargets:    []string{"a b.txt"},
				nonMatchingTargets: []string{"a.txt", "b.txt", "ab.txt", "a  b.txt"},
			},
		},
		{
			name: "tab",
			args: args{
				pattern:            "/a	b.txt",
				matchingTargets:    []string{"a	b.txt"},
				nonMatchingTargets: []string{"a.txt", "b.txt", "ab.txt", "a b.txt", "a		b.txt"},
			},
		},
		{
			// Note: it's debatable which behavior is correct - for now keep doublestar default behavior on this.
			// Keeping UT to ensure we don't accidentally change behavior.
			name: "escaped backslash",
			args: args{
				pattern:            "/a\\\\/b.txt",
				matchingTargets:    []string{"a\\/b.txt"},
				nonMatchingTargets: []string{"a\\\\/b.txt", "a/b.txt", "a/b.txt/c.ar"},
			},
		},
		{
			// Note: it's debatable which behavior is correct - for now keep doublestar default behavior on this.
			// Keeping UT to ensure we don't accidentally change behavior.
			name: "escaped path separator",
			args: args{
				pattern:            "/a\\/b.txt",
				matchingTargets:    []string{"a/b.txt", "a/b.txt/c.ar"},
				nonMatchingTargets: []string{"a\\/b.txt"},
			},
		},
	}
	testMatch := func(pattern string, target string, want bool) {
		got, err := match(pattern, target)
		if err != nil {
			t.Errorf("failed with error: %s", err)
		} else if got != want {
			t.Errorf("match(%q, %q) = %t but wanted %t)", pattern, target, got, want)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, target := range tt.args.matchingTargets {
				if len(target) > 0 && target[0] == '/' {
					t.Errorf("target shouldn't start with leading '/'")
				}

				testMatch(tt.args.pattern, target, true)
				testMatch(tt.args.pattern, "/"+target, true)
			}
			for _, target := range tt.args.nonMatchingTargets {
				if len(target) > 0 && target[0] == '/' {
					t.Errorf("target shouldn't start with leading '/'")
				}

				testMatch(tt.args.pattern, target, false)
				testMatch(tt.args.pattern, "/"+target, false)
			}
		})
	}
}
