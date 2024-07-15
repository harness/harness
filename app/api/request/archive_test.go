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

package request

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/harness/gitness/git/api"

	"github.com/go-chi/chi"
)

func TestParseArchiveParams(t *testing.T) {
	req := func(param string) *http.Request {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("*", param)
		ctx := context.WithValue(context.Background(), chi.RouteCtxKey, rctx)
		r, err := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}

	type args struct {
		r *http.Request
	}
	tests := []struct {
		name         string
		args         args
		wantParams   api.ArchiveParams
		wantFilename string
		wantErr      bool
	}{
		{
			name: "git archive flag is empty returns error",
			args: args{
				r: req("refs/heads/main"),
			},
			wantParams: api.ArchiveParams{},
			wantErr:    true,
		},
		{
			name: "git archive flag is unknown returns error",
			args: args{
				r: req("refs/heads/main.7z"),
			},
			wantParams: api.ArchiveParams{},
			wantErr:    true,
		},
		{
			name: "git archive flag format 'tar'",
			args: args{
				r: req("refs/heads/main.tar"),
			},
			wantParams: api.ArchiveParams{
				Format:  api.ArchiveFormatTar,
				Treeish: "refs/heads/main",
			},
			wantFilename: "main.tar",
		},
		{
			name: "git archive flag format 'zip'",
			args: args{
				r: req("refs/heads/main.zip"),
			},
			wantParams: api.ArchiveParams{
				Format:  api.ArchiveFormatZip,
				Treeish: "refs/heads/main",
			},
			wantFilename: "main.zip",
		},
		{
			name: "git archive flag format 'gz'",
			args: args{
				r: req("refs/heads/main.tar.gz"),
			},
			wantParams: api.ArchiveParams{
				Format:  api.ArchiveFormatTarGz,
				Treeish: "refs/heads/main",
			},
			wantFilename: "main.tar.gz",
		},
		{
			name: "git archive flag format 'tgz'",
			args: args{
				r: req("refs/heads/main.tgz"),
			},
			wantParams: api.ArchiveParams{
				Format:  api.ArchiveFormatTgz,
				Treeish: "refs/heads/main",
			},
			wantFilename: "main.tgz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotFilename, err := ParseArchiveParams(tt.args.r)
			if !tt.wantErr && err != nil {
				t.Errorf("ParseArchiveParams() expected error but err was nil")
			}
			if !reflect.DeepEqual(got, tt.wantParams) {
				t.Errorf("ParseArchiveParams() expected = %v, got %v", tt.wantParams, got)
			}
			if gotFilename != tt.wantFilename {
				t.Errorf("ParseArchiveParams() expected filename = %v, got %v", tt.wantFilename, gotFilename)
			}
		})
	}
}

func TestExt(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test file without ext",
			args: args{
				path: "./testdata/test",
			},
			want: "",
		},
		{
			name: "test file ext tar",
			args: args{
				path: "./testdata/test.tar",
			},
			want: "tar",
		},
		{
			name: "test file ext zip",
			args: args{
				path: "./testdata/test.zip",
			},
			want: "zip",
		},
		{
			name: "test file ext tar.gz",
			args: args{
				path: "./testdata/test.tar.gz",
			},
			want: "tar.gz",
		},
		{
			name: "test file ext tgz",
			args: args{
				path: "./testdata/test.tgz",
			},
			want: "tgz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Ext(tt.args.path); got != tt.want {
				t.Errorf("Ext() = %v, want %v", got, tt.want)
			}
		})
	}
}
