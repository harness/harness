// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
