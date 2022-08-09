// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package render

import "testing"

func Test_pagelen(t *testing.T) {
	tests := []struct {
		size, total, want int
	}{
		{25, 1, 1},
		{25, 24, 1},
		{25, 25, 1},
		{25, 26, 2},
		{25, 49, 2},
		{25, 50, 2},
		{25, 51, 3},
	}

	for _, test := range tests {
		got, want := pagelen(test.size, test.total), test.want
		if got != want {
			t.Errorf("got page length %d, want %d", got, want)
		}
	}
}
