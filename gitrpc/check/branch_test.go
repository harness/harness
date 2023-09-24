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

package check

import "testing"

func TestBranchName(t *testing.T) {
	type args struct {
		branch string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				branch: "new-branch",
			},
			wantErr: false,
		},
		{
			name: "happy path, include slash",
			args: args{
				branch: "eb/new-branch",
			},
			wantErr: false,
		},
		{
			name: "happy path, test utf-8 chars",
			args: args{
				branch: "eb/new\u2318branch",
			},
			wantErr: false,
		},
		{
			name: "branch name empty should return error",
			args: args{
				branch: "",
			},
			wantErr: true,
		},
		{
			name: "branch name starts with / should return error",
			args: args{
				branch: "/new-branch",
			},
			wantErr: true,
		},
		{
			name: "branch name contains // should return error",
			args: args{
				branch: "eb//new-branch",
			},
			wantErr: true,
		},
		{
			name: "branch name ends with / should return error",
			args: args{
				branch: "eb/new-branch/",
			},
			wantErr: true,
		},
		{
			name: "branch name starts with . should return error",
			args: args{
				branch: ".new-branch",
			},
			wantErr: true,
		},
		{
			name: "branch name contains .. should return error",
			args: args{
				branch: "new..branch",
			},
			wantErr: true,
		},
		{
			name: "branch name ends with . should return error",
			args: args{
				branch: "new-branch.",
			},
			wantErr: true,
		},
		{
			name: "branch name contains ~ should return error",
			args: args{
				branch: "new~branch",
			},
			wantErr: true,
		},
		{
			name: "branch name contains ^ should return error",
			args: args{
				branch: "^new-branch",
			},
			wantErr: true,
		},
		{
			name: "branch name contains : should return error",
			args: args{
				branch: "new:branch",
			},
			wantErr: true,
		},
		{
			name: "branch name contains control char should return error",
			args: args{
				branch: "new\x08branch",
			},
			wantErr: true,
		},
		{
			name: "branch name ends with .lock should return error",
			args: args{
				branch: "new-branch.lock",
			},
			wantErr: true,
		},
		{
			name: "branch name starts with ? should return error",
			args: args{
				branch: "?new-branch",
			},
			wantErr: true,
		},
		{
			name: "branch name contains ? should return error",
			args: args{
				branch: "new?branch",
			},
			wantErr: true,
		},
		{
			name: "branch name ends with ? should return error",
			args: args{
				branch: "new-branch?",
			},
			wantErr: true,
		},
		{
			name: "branch name starts with [ should return error",
			args: args{
				branch: "[new-branch",
			},
			wantErr: true,
		},
		{
			name: "branch name contains [ should return error",
			args: args{
				branch: "new[branch",
			},
			wantErr: true,
		},
		{
			name: "branch name ends with [ should return error",
			args: args{
				branch: "new-branch[",
			},
			wantErr: true,
		},
		{
			name: "branch name starts with * should return error",
			args: args{
				branch: "*new-branch",
			},
			wantErr: true,
		},
		{
			name: "branch name contains * should return error",
			args: args{
				branch: "new*branch",
			},
			wantErr: true,
		},
		{
			name: "branch name ends with * should return error",
			args: args{
				branch: "new-branch*",
			},
			wantErr: true,
		},
		{
			name: "branch name cannot contain a sequence @{ and should return error",
			args: args{
				branch: "new-br@{anch",
			},
			wantErr: true,
		},
		{
			name: "branch name cannot be the single character @ and should return error",
			args: args{
				branch: "@",
			},
			wantErr: true,
		},
		{
			name: "branch name cannot contain \\ and should return error",
			args: args{
				branch: "new-br\\anch",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := BranchName(tt.args.branch); (err != nil) != tt.wantErr {
				t.Errorf("validateBranchName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
