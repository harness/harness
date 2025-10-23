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

package secret

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

var (
	// errSecretRequiresParent if the user tries to create a secret without a parent space.
	errSecretRequiresParent = usererror.BadRequest(
		"Parent space required - standalone secret are not supported.")
)

type CreateInput struct {
	Description string `json:"description"`
	SpaceRef    string `json:"space_ref"` // Ref of the parent space
	// TODO [CODE-1363]: remove after identifier migration.
	UID        string `json:"uid" deprecated:"true"`
	Identifier string `json:"identifier"`
	Data       string `json:"data"`
}

func (c *Controller) Create(ctx context.Context, session *auth.Session, in *CreateInput) (*types.Secret, error) {
	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	parentSpace, err := c.spaceFinder.FindByRef(ctx, in.SpaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find parent by ref: %w", err)
	}

	err = apiauth.CheckSecret(
		ctx,
		c.authorizer,
		session,
		parentSpace.Path,
		"",
		enum.PermissionSecretEdit,
	)
	if err != nil {
		return nil, err
	}

	var secret *types.Secret
	now := time.Now().UnixMilli()
	secret = &types.Secret{
		CreatedBy:   session.Principal.ID,
		Description: in.Description,
		Data:        in.Data,
		SpaceID:     parentSpace.ID,
		Identifier:  in.Identifier,
		Created:     now,
		Updated:     now,
		Version:     0,
	}
	secret, err = enc(c.encrypter, secret)
	if err != nil {
		return nil, fmt.Errorf("could not encrypt secret: %w", err)
	}
	err = c.secretStore.Create(ctx, secret)
	if err != nil {
		return nil, fmt.Errorf("secret creation failed: %w", err)
	}

	return secret, nil
}

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == "" {
		in.Identifier = in.UID
	}

	parentRefAsID, err := strconv.ParseInt(in.SpaceRef, 10, 64)

	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(in.SpaceRef)) == 0) {
		return errSecretRequiresParent
	}

	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}

	in.Description = strings.TrimSpace(in.Description)
	return check.Description(in.Description)
}

// helper function returns the same secret with encrypted data.
func enc(encrypt encrypt.Encrypter, secret *types.Secret) (*types.Secret, error) {
	if secret == nil {
		return nil, fmt.Errorf("cannot encrypt a nil secret")
	}
	s := *secret
	ciphertext, err := encrypt.Encrypt(secret.Data)
	if err != nil {
		return nil, err
	}
	s.Data = string(ciphertext)
	return &s, nil
}

// helper function returns the same secret with decrypted data.
func Dec(encrypt encrypt.Encrypter, secret *types.Secret) (*types.Secret, error) {
	if secret == nil {
		return nil, fmt.Errorf("cannot decrypt a nil secret")
	}
	s := *secret
	plaintext, err := encrypt.Decrypt([]byte(secret.Data))
	if err != nil {
		return nil, err
	}
	s.Data = plaintext
	return &s, nil
}
