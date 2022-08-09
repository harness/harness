// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import "testing"

func TestOffset(t *testing.T) {
	tests := []struct {
		page int
		size int
		want int
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
		got, want := offset(test.page, test.size), test.want
		if got != want {
			t.Errorf("Got %d want %d for page %d, size %d", got, want, test.page, test.size)
		}
	}
}

func TestLimit(t *testing.T) {
	tests := []struct {
		size int
		want int
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
		got, want := limit(test.size), test.want
		if got != want {
			t.Errorf("Got %d want %d for size %d", got, want, test.size)
		}
	}
}
