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
	"github.com/harness/gitness/gitrpc"
)

func TestService_ParseCodeOwner(t *testing.T) {
	content1 := "**/contracts/openapi/v1/ mankrit.singh@harness.io ashish.sanodia@harness.io\n"
	content2 := "**/contracts/openapi/v1/ mankrit.singh@harness.io ashish.sanodia@harness.io\n" +
		"/scripts/api mankrit.singh@harness.io ashish.sanodia@harness.io"
	content3 := "# codeowner file \n**/contracts/openapi/v1/ mankrit.singh@harness.io ashish.sanodia@harness.io\n" +
		"#\n/scripts/api mankrit.singh@harness.io ashish.sanodia@harness.io"
	type fields struct {
		repoStore store.RepoStore
		git       gitrpc.Interface
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
	pattern1 := [1]string{"*"}
	pattern2 := [1]string{"**"}
	pattern3 := [1]string{"abc/xyz"}
	pattern4 := [2]string{"abc/xyz", "*"}
	pattern5 := [2]string{"abc/xyz", "**"}
	pattern6 := [1]string{"doc/frotz"}
	pattern7 := [1]string{"?ilename"}
	pattern8 := [1]string{"**/foo"}
	pattern9 := [1]string{"foo/**"}
	pattern10 := [1]string{"a/**/b"}
	pattern11 := [1]string{"foo/*"}
	pattern12 := [1]string{"*.txt"}
	pattern13 := [1]string{"/scripts/"}

	type args struct {
		ctx    context.Context
		slice  []string
		target string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test * pattern",
			args: args{
				ctx:    nil,
				slice:  pattern1[:],
				target: "random",
			},
			want: true,
		},
		{
			name: "Test ** pattern",
			args: args{
				ctx:    nil,
				slice:  pattern2[:],
				target: "random/xyz",
			},
			want: true,
		},
		{
			name: "Test ** pattern on fixed path",
			args: args{
				ctx:    nil,
				slice:  pattern1[:],
				target: "abhinav/path",
			},
			want: false,
		},
		{
			name: "Test abc/xyz pattern",
			args: args{
				ctx:    nil,
				slice:  pattern3[:],
				target: "abc/xyz",
			},
			want: true,
		},
		{
			name: "Test abc/xyz pattern negative",
			args: args{
				ctx:    nil,
				slice:  pattern3[:],
				target: "abc/xy",
			},
			want: false,
		},
		{
			name: "Test incorrect pattern negative",
			args: args{
				ctx:    nil,
				slice:  pattern4[:],
				target: "random/path",
			},
			want: false,
		},
		{
			name: "Test * pattern with bigger slice",
			args: args{
				ctx:    nil,
				slice:  pattern4[:],
				target: "random",
			},
			want: true,
		},
		{
			name: "Test file path with **",
			args: args{
				ctx:    nil,
				slice:  pattern5[:],
				target: "path/to/file",
			},
			want: true,
		},
		{
			name: "Test / pattern",
			args: args{
				ctx:    nil,
				slice:  pattern6[:],
				target: "doc/frotz",
			},
			want: true,
		},
		{
			name: "Test ? pattern",
			args: args{
				ctx:    nil,
				slice:  pattern7[:],
				target: "filename",
			},
			want: true,
		},
		{
			name: "Test /** pattern",
			args: args{
				ctx:    nil,
				slice:  pattern8[:],
				target: "foo",
			},
			want: true,
		},
		{
			name: "Test /** pattern with slash",
			args: args{
				ctx:    nil,
				slice:  pattern8[:],
				target: "foo/bar",
			},
			want: false,
		},
		{
			name: "Test **/ with deep nesting",
			args: args{
				ctx:    nil,
				slice:  pattern8[:],
				target: "path/to/foo",
			},
			want: true,
		},
		{
			name: "Test **/ pattern",
			args: args{
				ctx:    nil,
				slice:  pattern9[:],
				target: "foo/bar",
			},
			want: true,
		},
		{
			name: "Test a/**/b pattern",
			args: args{
				ctx:    nil,
				slice:  pattern10[:],
				target: "a/x/y/b",
			},
			want: true,
		},
		{
			name: "Test /* pattern positive",
			args: args{
				ctx:    nil,
				slice:  pattern11[:],
				target: "foo/getting-started.md",
			},
			want: true,
		},
		{
			name: "Test /* pattern negative",
			args: args{
				ctx:    nil,
				slice:  pattern11[:],
				target: "foo/build-app/troubleshooting.md",
			},
			want: false,
		},
		{
			name: "Test * for files",
			args: args{
				ctx:    nil,
				slice:  pattern12[:],
				target: "foo.txt",
			},
			want: true,
		},
		{
			name: "Test /a/",
			args: args{
				ctx:    nil,
				slice:  pattern13[:],
				target: "/scripts/filename.txt",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := contains(tt.args.slice, tt.args.target); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
