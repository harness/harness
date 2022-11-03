// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package server

// Config represents the configuration for the gitrpc server.
type Config struct {
	GitRoot string
	Bind    string
}
