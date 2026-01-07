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

package types

import (
	"encoding/json"

	"github.com/harness/gitness/types/enum"
)

type PublicKey struct {
	// ID of the key. Frontend doesn't need it.
	ID int64 `json:"-"`

	// PrincipalID who owns the key.
	// Not returned in API response because the API always returns keys for the same user.
	PrincipalID int64 `json:"-"`

	Created int64 `json:"created"`

	// Verified holds the timestamp when the key was successfully used to access the system.
	Verified *int64 `json:"verified"`

	Identifier string `json:"identifier"`

	// Usage holds the allowed usage for the key - authorization or signature verification.
	Usage enum.PublicKeyUsage `json:"usage"`

	// Fingerprint is a short hash sum of the key. Useful for quick key comparison.
	// The value is indexed in the database.
	Fingerprint string `json:"fingerprint"`

	// Content holds the original uploaded public key data.
	Content string `json:"-"`

	Comment string `json:"comment"`

	// Type of the key - the algorithm used to generate the key.
	Type string `json:"type"`

	// Scheme indicates if it's SSH or PGP key.
	Scheme enum.PublicKeyScheme `json:"scheme"`

	// ValidFrom and ValidTo are validity period for the key.
	// If they have valid values, the key should NOT be used outside the period.
	ValidFrom *int64 `json:"valid_from"`
	ValidTo   *int64 `json:"valid_to"`

	// RevocationReason is the reason why the key has been revoked.
	// If a key has a RevocationReason it should also have ValidTo timestamp set.
	RevocationReason *enum.RevocationReason `json:"revocation_reason"`

	// Metadata holds additional key metadata info for the UI (for PGP keys).
	Metadata json.RawMessage `json:"metadata"`
}

type PublicKeyFilter struct {
	ListQueryFilter
	Sort  enum.PublicKeySort
	Order enum.Order

	Usages  []enum.PublicKeyUsage
	Schemes []enum.PublicKeyScheme
}
