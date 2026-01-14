//  Copyright 2023 Harness, Inc.
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

package types

import (
	"database/sql"
	"time"

	"github.com/opencontainers/go-digest"
)

// Manifest DTO object.
type Manifest struct {
	ID            int64
	RegistryID    int64
	TotalSize     int64
	SchemaVersion int
	MediaType     string
	MediaTypeID   int64
	ImageName     string
	ArtifactType  sql.NullString
	Digest        digest.Digest
	Payload       Payload
	Configuration *Configuration
	SubjectID     sql.NullInt64
	SubjectDigest digest.Digest
	NonConformant bool
	// NonDistributableLayers identifies whether a manifest
	// references foreign/non-distributable layers. For now, we are
	// not registering metadata about these layers,
	// but we may wish to backfill that metadata in the future by parsing
	// the manifest payload.
	NonDistributableLayers bool
	Annotations            JSONB
	CreatedAt              time.Time
	CreatedBy              int64
	UpdatedAt              time.Time
	UpdatedBy              int64
}

// Manifests is a slice of Manifest pointers.
type Manifests []*Manifest
