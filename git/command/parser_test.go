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

package command

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	type args struct {
		args []string
	}
	tests := []struct {
		name string
		args args
		want *Command
	}{
		{
			name: "git version test",
			args: args{
				args: []string{
					"git",
					"version",
				},
			},
			want: &Command{
				Name: "version",
			},
		},
		{
			name: "git help test",
			args: args{
				args: []string{
					"git",
					"--help",
				},
			},
			want: &Command{
				Globals: []string{"--help"},
			},
		},
		{
			name: "diff basic test",
			args: args{
				args: []string{
					"git",
					"diff",
					"main...dev",
				},
			},
			want: &Command{
				Name: "diff",
				Args: []string{"main...dev"},
			},
		},
		{
			name: "diff path test",
			args: args{
				args: []string{
					"git",
					"diff",
					"--shortstat",
					"main...dev",
					"--",
					"file1",
					"file2",
				},
			},
			want: &Command{
				Name:  "diff",
				Flags: []string{"--shortstat"},
				Args:  []string{"main...dev"},
				PostSepArgs: []string{
					"file1",
					"file2",
				},
			},
		},
		{
			name: "diff path test",
			args: args{
				args: []string{
					"git",
					"diff",
					"--shortstat",
					"main...dev",
					"--",
				},
			},
			want: &Command{
				Name:        "diff",
				Flags:       []string{"--shortstat"},
				Args:        []string{"main...dev"},
				PostSepArgs: []string{},
			},
		},
		{
			name: "git remote basic test",
			args: args{
				args: []string{
					"git",
					"remote",
					"set-url",
					"origin",
					"http://reponame",
				},
			},
			want: &Command{
				Name:   "remote",
				Action: "set-url",
				Args:   []string{"origin", "http://reponame"},
			},
		},
		{
			name: "pack object test",
			args: args{
				args: []string{
					"git",
					"--shallow-file",
					"",
					"pack-objects",
					"--revs",
					"--thin",
					"--stdout",
					"--progress",
					"--delta-base-offset",
				},
			},
			want: &Command{
				Globals: []string{"--shallow-file", ""},
				Name:    "pack-objects",
				Flags: []string{
					"--revs",
					"--thin",
					"--stdout",
					"--progress",
					"--delta-base-offset",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Parse(tt.args.args...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
