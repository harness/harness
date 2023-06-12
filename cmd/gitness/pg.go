// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

//go:build pq
// +build pq

package main

import (
	"github.com/harness/gitness/cli"

	_ "github.com/lib/pq"
)

func main() {
	cli.Command()
}
