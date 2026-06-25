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
	"github.com/harness/gitness/app/services/importer"
)

// Kind identifies the SCM event family carried by Event.
type Kind string

const (
	KindPullRequest Kind = "pull_request"
	KindCheck       Kind = "check_run"
)

// Provider identifies the upstream SCM. Alias of importer.ProviderType so
// values flow into types.LinkedPullReq / types.LinkedRepo without casts.
type Provider = importer.ProviderType

const (
	ProviderGitHub = importer.ProviderTypeGitHub
)

// Event is the broker-agnostic event consumed by the Dispatcher. Adapters
// living outside this package translate platform-specific webhook envelopes
// into Event values; the dispatcher and handlers stay free of transport
// coupling.
type Event struct {
	Provider   Provider
	AccountID  string
	DeliveryID string // provider-side unique delivery id, used for logging
	Payload    Payload
}

// Payload is the SCM-agnostic interface every event payload satisfies.
// Adding a new event family means adding a new Payload implementation; the
// dispatcher and handler registry do not change. RepoProviderID is the
// routing key.
type Payload interface {
	Kind() Kind
	RepoProviderID() string
}

// User identifies a provider-side actor.
type User struct {
	Login   string
	Avatar  string
	HTMLURL string
}

// Repository is the minimal provider-side repo identity used for routing.
type Repository struct {
	ProviderID string
}
