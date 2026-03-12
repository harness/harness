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

import (
	"strings"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func Test_getStartedTime(t *testing.T) {
	type args struct {
		in    *ReportInput
		check types.Check
		now   int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "nothing to pending",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusPending},
				check: types.Check{},
				now:   1234,
			},
			want: 0,
		},

		{
			name: "nothing to running",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusRunning},
				check: types.Check{},
				now:   1234,
			},
			want: 1234,
		},
		{
			name: "nothing to completed",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusSuccess},
				check: types.Check{},
				now:   1234,
			},
			want: 1234,
		},
		{
			name: "nothing to completed1",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusSuccess, Started: 1},
				check: types.Check{},
				now:   1234,
			},
			want: 1,
		},
		{
			name: "pending to pending",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusPending, Started: 1},
				check: types.Check{Status: enum.CheckStatusPending, Started: 0},
				now:   1234,
			},
			want: 1,
		},
		{
			name: "pending to pending1",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusPending},
				check: types.Check{Status: enum.CheckStatusPending, Started: 0},
				now:   1234,
			},
			want: 0,
		},
		{
			name: "pending to running",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusRunning},
				check: types.Check{Status: enum.CheckStatusPending, Started: 0},
				now:   1234,
			},
			want: 1234,
		},
		{
			name: "pending to running1",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusRunning, Started: 1},
				check: types.Check{Status: enum.CheckStatusPending, Started: 0},
				now:   1234,
			},
			want: 1,
		},
		{
			name: "pending to completed",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusSuccess, Started: 1},
				check: types.Check{Status: enum.CheckStatusPending, Started: 0},
				now:   1234,
			},
			want: 1,
		},
		{
			name: "pending to completed1",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusSuccess},
				check: types.Check{Status: enum.CheckStatusPending, Started: 0},
				now:   1234,
			},
			want: 1234,
		},
		{
			name: "running to pending",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusPending, Started: 1},
				check: types.Check{Status: enum.CheckStatusRunning, Started: 9876},
				now:   1234,
			},
			want: 1,
		},
		{
			name: "running to pending1",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusPending},
				check: types.Check{Status: enum.CheckStatusRunning, Started: 9876},
				now:   1234,
			},
			want: 0,
		},
		{
			name: "running to running",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusRunning},
				check: types.Check{Status: enum.CheckStatusRunning, Started: 9876},
				now:   1234,
			},
			want: 9876,
		},
		{
			name: "running to running1",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusRunning, Started: 1},
				check: types.Check{Status: enum.CheckStatusRunning, Started: 9876},
				now:   1234,
			},
			want: 1,
		},
		{
			name: "running to completed",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusSuccess},
				check: types.Check{Status: enum.CheckStatusRunning, Started: 9876},
				now:   1234,
			},
			want: 9876,
		},
		{
			name: "running to completed",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusSuccess, Started: 1},
				check: types.Check{Status: enum.CheckStatusRunning, Started: 9876},
				now:   1234,
			},
			want: 1,
		},
		{
			name: "completed to pending",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusPending},
				check: types.Check{Status: enum.CheckStatusSuccess, Started: 9876},
				now:   1234,
			},
			want: 0,
		},
		{
			name: "completed to running",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusRunning},
				check: types.Check{Status: enum.CheckStatusSuccess, Started: 9876},
				now:   1234,
			},
			want: 1234,
		},
		{
			name: "completed to completed",
			args: args{
				in:    &ReportInput{Status: enum.CheckStatusSuccess},
				check: types.Check{Status: enum.CheckStatusSuccess, Started: 9876},
				now:   1234,
			},
			want: 1234,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getStartTime(tt.args.in, tt.args.check, tt.args.now); got != tt.want {
				t.Errorf("getStartTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_Sanitize_SummaryAndLink(t *testing.T) {
	noopSanitizer := func(_ *ReportInput, _ *auth.Session) error { return nil }
	sanitizers := map[enum.CheckPayloadKind]func(in *ReportInput, s *auth.Session) error{
		enum.CheckPayloadKindEmpty: noopSanitizer,
	}
	session := &auth.Session{}

	tests := []struct {
		name    string
		input   *ReportInput
		wantErr bool
		errMsg  string
	}{
		{
			name: "summary at max length is valid",
			input: &ReportInput{
				Identifier: "check1",
				Status:     enum.CheckStatusSuccess,
				Summary:    strings.Repeat("a", 2048),
				Payload:    types.CheckPayload{Kind: enum.CheckPayloadKindEmpty},
			},
			wantErr: false,
		},
		{
			name: "summary exceeding max length is rejected",
			input: &ReportInput{
				Identifier: "check1",
				Status:     enum.CheckStatusSuccess,
				Summary:    strings.Repeat("a", 2049),
				Payload:    types.CheckPayload{Kind: enum.CheckPayloadKindEmpty},
			},
			wantErr: true,
			errMsg:  "Summary can be at most",
		},
		{
			name: "empty summary is valid",
			input: &ReportInput{
				Identifier: "check1",
				Status:     enum.CheckStatusSuccess,
				Payload:    types.CheckPayload{Kind: enum.CheckPayloadKindEmpty},
			},
			wantErr: false,
		},
		{
			name: "link at max length is valid",
			input: &ReportInput{
				Identifier: "check1",
				Status:     enum.CheckStatusSuccess,
				Link:       strings.Repeat("a", 2048),
				Payload:    types.CheckPayload{Kind: enum.CheckPayloadKindEmpty},
			},
			wantErr: false,
		},
		{
			name: "link exceeding max length is rejected",
			input: &ReportInput{
				Identifier: "check1",
				Status:     enum.CheckStatusSuccess,
				Link:       strings.Repeat("a", 2049),
				Payload:    types.CheckPayload{Kind: enum.CheckPayloadKindEmpty},
			},
			wantErr: true,
			errMsg:  "Link can be at most",
		},
		{
			name: "empty link is valid",
			input: &ReportInput{
				Identifier: "check1",
				Status:     enum.CheckStatusSuccess,
				Payload:    types.CheckPayload{Kind: enum.CheckPayloadKindEmpty},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Sanitize(sanitizers, session)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Sanitize() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Sanitize() error = %q, want it to contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}
