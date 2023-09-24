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

package database

import (
	"testing"
)

func TestOffset(t *testing.T) {
	tests := []struct {
		page int
		size int
		want uint64
	}{
		{
			page: 0,
			size: 10,
			want: 0,
		},
		{
			page: 1,
			size: 10,
			want: 0,
		},
		{
			page: 2,
			size: 10,
			want: 10,
		},
		{
			page: 3,
			size: 10,
			want: 20,
		},
		{
			page: 4,
			size: 100,
			want: 300,
		},
		{
			page: 4,
			size: 0, // unset, expect default 100
			want: 300,
		},
	}

	for _, test := range tests {
		got, want := Offset(test.page, test.size), test.want
		if got != want {
			t.Errorf("Got %d want %d for page %d, size %d", got, want, test.page, test.size)
		}
	}
}

func TestLimit(t *testing.T) {
	tests := []struct {
		size int
		want uint64
	}{
		{
			size: 0,
			want: 100,
		},
		{
			size: 10,
			want: 10,
		},
	}

	for _, test := range tests {
		got, want := Limit(test.size), test.want
		if got != want {
			t.Errorf("Got %d want %d for size %d", got, want, test.size)
		}
	}
}
