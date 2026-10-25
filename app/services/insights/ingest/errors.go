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

// Package ingest implements the transport-independent part of the Insights
// error tracker ingestion: Sentry DSN handling, envelope and legacy store
// payload parsing, the event model, deterministic fingerprinting and
// sanitization. It has no database or HTTP routing dependencies.
package ingest

import "errors"

var (
	// ErrInvalidDSN is returned when a DSN cannot be parsed.
	ErrInvalidDSN = errors.New("insights: invalid dsn")
	// ErrMissingKey is returned when the request carries no public key.
	ErrMissingKey = errors.New("insights: missing public key")
	// ErrProjectMismatch is returned when the project of a DSN differs from the request path.
	ErrProjectMismatch = errors.New("insights: dsn project does not match request project")
	// ErrTooLarge is returned when the compressed or decompressed body exceeds the limits.
	ErrTooLarge = errors.New("insights: payload too large")
	// ErrTooManyItems is returned when an envelope has more items than allowed.
	ErrTooManyItems = errors.New("insights: too many envelope items")
	// ErrInvalidEnvelope is returned for malformed envelopes.
	ErrInvalidEnvelope = errors.New("insights: invalid envelope")
	// ErrInvalidEvent is returned for events that are not valid JSON objects.
	ErrInvalidEvent = errors.New("insights: invalid event")
	// ErrInvalidEncoding is returned for undecodable or unsupported content encodings.
	ErrInvalidEncoding = errors.New("insights: invalid content encoding")
)
