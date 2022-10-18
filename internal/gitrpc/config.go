// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

// ServerConfig represents the configuration for the gitrpc server.
type ServerConfig struct {
	GitRoot string
	Bind    string
}

// ClientConfig represents the config for the gitrpc client.
type ClientConfig struct {
	Bind string
}
