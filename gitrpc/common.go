// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"errors"

	"github.com/harness/gitness/gitrpc/rpc"
)

const NilSHA = "0000000000000000000000000000000000000000"

// ReadParams contains the base parameters for read operations.
type ReadParams struct {
	RepoUID string
}

func (p ReadParams) Validate() error {
	if p.RepoUID == "" {
		return errors.New("repository id cannot be empty")
	}
	return nil
}

// WriteParams contains the base parameters for write operations.
type WriteParams struct {
	RepoUID string
	Actor   Identity
	EnvVars map[string]string
}

func mapToRPCReadRequest(p ReadParams) *rpc.ReadRequest {
	return &rpc.ReadRequest{
		RepoUid: p.RepoUID,
	}
}

func mapToRPCWriteRequest(p WriteParams) *rpc.WriteRequest {
	out := &rpc.WriteRequest{
		RepoUid: p.RepoUID,
		Actor: &rpc.Identity{
			Name:  p.Actor.Name,
			Email: p.Actor.Email,
		},
		EnvVars: make([]*rpc.EnvVar, len(p.EnvVars)),
	}

	i := 0
	for k, v := range p.EnvVars {
		out.EnvVars[i] = &rpc.EnvVar{
			Name:  k,
			Value: v,
		}
		i++
	}

	return out
}
