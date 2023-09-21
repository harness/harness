// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/harness/gitness/gitrpc/server"
)

// system stores high level system components.
type system struct {
	grpcServer *server.GRPCServer
	httpServer *server.HTTPServer
}

// newSystem returns a new system structure.
func newSystem(grpcServer *server.GRPCServer, httpServer *server.HTTPServer) *system {
	return &system{
		grpcServer: grpcServer,
		httpServer: httpServer,
	}
}
