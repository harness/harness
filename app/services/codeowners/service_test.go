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
	"github.com/harness/gitness/gitrpc"
)

func TestService_ParseCodeOwner(t *testing.T) {
	content1 := `**/contracts/openapi/v1/ mankrit.singh@harness.io ashish.sanodia@harness.io
	`
	content2 := `**/contracts/openapi/v1/ mankrit.singh@harness.io ashish.sanodia@harness.io
/scripts/api mankrit.singh@harness.io ashish.sanodia@harness.io`
	content3 := `# codeowner file 
**/contracts/openapi/v1/ mankrit.singh@harness.io ashish.sanodia@harness.io
#
/scripts/api mankrit.singh@harness.io ashish.sanodia@harness.io`
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
