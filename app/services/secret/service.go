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

	secretCtrl "github.com/harness/gitness/app/api/controller/secret"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/secret"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type service struct {
	secretStore    store.SecretStore
	encrypter      encrypt.Encrypter
	spacePathStore store.SpacePathStore
}

func NewService(
	secretStore store.SecretStore, encrypter encrypt.Encrypter, spacePathStore store.SpacePathStore,
) secret.Service {
	return &service{
		secretStore:    secretStore,
		encrypter:      encrypter,
		spacePathStore: spacePathStore,
	}
}

func (s *service) DecryptSecret(ctx context.Context, spacePath string, secretIdentifier string) (string, error) {
	path, err := s.spacePathStore.FindByPath(ctx, spacePath)
	if err != nil {
		log.Error().Msgf("failed to find space path: %v", err)
		return "", errors.Wrap(err, "failed to find space path")
	}
	sec, err := s.secretStore.FindByIdentifier(ctx, path.SpaceID, secretIdentifier)
	if err != nil {
		log.Error().Msgf("failed to find secret: %v", err)
		return "", errors.Wrap(err, "failed to find secret")
	}
	sec, err = secretCtrl.Dec(s.encrypter, sec)
	if err != nil {
		log.Error().Msgf("could not decrypt secret: %v", err)
		return "", errors.Wrap(err, "failed to decrypt secret")
	}
	return sec.Data, nil
}
