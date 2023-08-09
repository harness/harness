// Copyright 2023 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package types

type Secret struct {
	ID          int64  `db:"secret_id"              json:"id"`
	Description string `db:"secret_description"     json:"description"`
	SpaceID     int64  `db:"secret_space_id"        json:"space_id"`
	UID         string `db:"secret_uid"             json:"uid"`
	Data        string `db:"secret_data"            json:"data"`
	Created     int64  `db:"secret_created"         json:"created"`
	Updated     int64  `db:"secret_updated"         json:"updated"`
	Version     int64  `db:"secret_version"         json:"version"`
}

// Copy makes a copy of the secret without the value.
func (s *Secret) CopyWithoutData() *Secret {
	return &Secret{
		ID:          s.ID,
		Description: s.Description,
		UID:         s.UID,
		SpaceID:     s.SpaceID,
		Created:     s.Created,
		Updated:     s.Updated,
		Version:     s.Version,
	}
}
