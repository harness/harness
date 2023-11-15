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
	"context"
	"reflect"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
)

func TestService_ParseCodeOwner(t *testing.T) {
	content1 := "**/contracts/openapi/v1/ mankrit.singh@harness.io ashish.sanodia@harness.io\n"
	content2 := "**/contracts/openapi/v1/ mankrit.singh@harness.io ashish.sanodia@harness.io\n" +
		"/scripts/api mankrit.singh@harness.io ashish.sanodia@harness.io"
	content3 := "# codeowner file \n**/contracts/openapi/v1/ mankrit.singh@harness.io ashish.sanodia@harness.io\n" +
		"#\n/scripts/api mankrit.singh@harness.io ashish.sanodia@harness.io"
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
			args: args{codeOwnersContent: content1},
			want: []Entry{{
				Pattern: "**/contracts/openapi/v1/",
				Owners:  []string{"mankrit.singh@harness.io", "ashish.sanodia@harness.io"},
			},
			},
		},
		{
			name: "Code owners Multiple",
			args: args{codeOwnersContent: content2},
			want: []Entry{{
				Pattern: "**/contracts/openapi/v1/",
				Owners:  []string{"mankrit.singh@harness.io", "ashish.sanodia@harness.io"},
			},
				{
					Pattern: "/scripts/api",
					Owners:  []string{"mankrit.singh@harness.io", "ashish.sanodia@harness.io"},
				},
			},
		},
		{
			name: "Code owners With comments",
			args: args{codeOwnersContent: content3},
			want: []Entry{{
				Pattern: "**/contracts/openapi/v1/",
				Owners:  []string{"mankrit.singh@harness.io", "ashish.sanodia@harness.io"},
			},
				{
					Pattern: "/scripts/api",
					Owners:  []string{"mankrit.singh@harness.io", "ashish.sanodia@harness.io"},
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

func Test_contains(t *testing.T) {
	target1 := []string{"random"}
	target2 := []string{"random/xyz"}
	target3 := []string{"abhinav/path"}
	target4 := []string{"abc/xyz"}
	target5 := []string{"abc/xyz", "random"}
	target6 := []string{"doc/frotz"}
	target7 := []string{"filename"}
	target8 := []string{"as/foo"}
	target9 := []string{"foo/bar"}
	target10 := []string{"a/x/y/b"}
	target11 := []string{"foo/getting-started.md"}
	target12 := []string{"foo.txt"}
	target13 := []string{"/scripts/filename.txt"}
	target14 := []string{"path/to/file.txt"}
	target15 := []string{"path/to/foo"}
	target16 := []string{"foo/build-app/troubleshooting.md"}

	type args struct {
		ctx     context.Context
		pattern string
		target  []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test * pattern",
			args: args{
				ctx:     nil,
				pattern: "*",
				target:  target1,
			},
			want: true,
		},
		{
			name: "Test ** pattern",
			args: args{
				ctx:     nil,
				pattern: "**",
				target:  target2,
			},
			want: true,
		},
		{
			name: "Test ** pattern on fixed path",
			args: args{
				ctx:     nil,
				pattern: "abc/xyz",
				target:  target3,
			},
			want: false,
		},
		{
			name: "Test abc/xyz pattern",
			args: args{
				ctx:     nil,
				pattern: "abc/xyz",
				target:  target4,
			},
			want: true,
		},
		{
			name: "Test incorrect pattern negative",
			args: args{
				ctx:     nil,
				pattern: "random/xyz",
				target:  target5,
			},
			want: false,
		},
		{
			name: "Test file path with **",
			args: args{
				ctx:     nil,
				pattern: "**",
				target:  target14,
			},
			want: true,
		},
		{
			name: "Test / pattern",
			args: args{
				ctx:     nil,
				pattern: "doc/frotz",
				target:  target6,
			},
			want: true,
		},
		{
			name: "Test ? pattern",
			args: args{
				ctx:     nil,
				pattern: "?ilename",
				target:  target7,
			},
			want: true,
		},
		{
			name: "Test /** pattern",
			args: args{
				ctx:     nil,
				pattern: "**/foo",
				target:  target8,
			},
			want: true,
		},
		{
			name: "Test **/ with deep nesting",
			args: args{
				ctx:     nil,
				pattern: "**/foo",
				target:  target15,
			},
			want: true,
		},
		{
			name: "Test **/ pattern",
			args: args{
				ctx:     nil,
				pattern: "foo/**",
				target:  target9,
			},
			want: true,
		},
		{
			name: "Test a/**/b pattern",
			args: args{
				ctx:     nil,
				pattern: "a/x/y/b",
				target:  target10,
			},
			want: true,
		},
		{
			name: "Test /* pattern positive",
			args: args{
				ctx:     nil,
				pattern: "foo/*",
				target:  target11,
			},
			want: true,
		},
		{
			name: "Test /* pattern negative",
			args: args{
				ctx:     nil,
				pattern: "foo/*",
				target:  target16,
			},
			want: false,
		},
		{
			name: "Test * for files",
			args: args{
				ctx:     nil,
				pattern: "*.txt",
				target:  target12,
			},
			want: true,
		},
		{
			name: "Test /a/",
			args: args{
				ctx:     nil,
				pattern: "/scripts/",
				target:  target13,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := contains(tt.args.pattern, tt.args.target); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
