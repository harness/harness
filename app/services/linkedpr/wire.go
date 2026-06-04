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

package linkedpr

import (
	"github.com/harness/gitness/app/store"

	"github.com/google/wire"
)

// WireSet wires only the gitness-side orchestrator (Dispatcher,
// AuthorResolver). Concrete handlers (e.g. handlers/github.WireSet), the
// Transport, and the Handlers map are platform-bound and must be provided by
// the embedding application's WireSet.
var WireSet = wire.NewSet(
	ProvideDispatcher,
	ProvideAuthorResolver,
)

// ProvideAuthorResolver returns the system-principal default; override via wire.
func ProvideAuthorResolver() AuthorResolver {
	return &SystemPrincipalResolver{PrincipalID: 0}
}

// ProvideDispatcher accepts the wider store.LinkedRepoStore so wire can resolve
// it via the existing ProvideLinkRepoStore provider.
func ProvideDispatcher(
	linkedRepoStore store.LinkedRepoStore,
	registry Handlers,
) *Dispatcher {
	return NewDispatcher(linkedRepoStore, registry)
}
