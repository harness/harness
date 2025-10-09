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
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/harness/gitness/git/api"
)

func TestGetFileDiffRequestsFromQuery(t *testing.T) {
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name      string
		args      args
		wantFiles api.FileDiffRequests
	}{
		{
			name: "full range",
			args: args{
				r: &http.Request{
					URL: &url.URL{
						Path:     "/diff",
						RawQuery: "path=file.txt&range=1:20",
					},
				},
			},
			wantFiles: api.FileDiffRequests{
				{
					Path:      "file.txt",
					StartLine: 1,
					EndLine:   20,
				},
			},
		},
		{
			name: "start range",
			args: args{
				r: &http.Request{
					URL: &url.URL{
						Path:     "/diff",
						RawQuery: "path=file.txt&range=1",
					},
				},
			},
			wantFiles: api.FileDiffRequests{
				{
					Path:      "file.txt",
					StartLine: 1,
				},
			},
		},
		{
			name: "end range",
			args: args{
				r: &http.Request{
					URL: &url.URL{
						Path:     "/diff",
						RawQuery: "path=file.txt&range=:20",
					},
				},
			},
			wantFiles: api.FileDiffRequests{
				{
					Path:    "file.txt",
					EndLine: 20,
				},
			},
		},
		{
			name: "multi path",
			args: args{
				r: &http.Request{
					URL: &url.URL{
						Path:     "/diff",
						RawQuery: "path=file.txt&range=:20&path=file1.txt&range=&path=file2.txt&range=1:15",
					},
				},
			},
			wantFiles: api.FileDiffRequests{
				{
					Path:    "file.txt",
					EndLine: 20,
				},
				{
					Path: "file1.txt",
				},
				{
					Path:      "file2.txt",
					StartLine: 1,
					EndLine:   15,
				},
			},
		},
		{
			name: "multi path without some range",
			args: args{
				r: &http.Request{
					URL: &url.URL{
						Path:     "/diff",
						RawQuery: "path=file.txt&range=:20&path=file1.txt&path=file2.txt&range=1:15",
					},
				},
			},
			wantFiles: api.FileDiffRequests{
				{
					Path:    "file.txt",
					EndLine: 20,
				},
				{
					Path:      "file1.txt",
					StartLine: 1,
					EndLine:   15,
				},
				{
					Path: "file2.txt",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotFiles := GetFileDiffFromQuery(tt.args.r); !reflect.DeepEqual(gotFiles, tt.wantFiles) {
				t.Errorf("GetFileDiffFromQuery() = %v, want %v", gotFiles, tt.wantFiles)
			}
		})
	}
}
