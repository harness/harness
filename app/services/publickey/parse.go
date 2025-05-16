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

package publickey

import (
	"fmt"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type KeyInfo interface {
	Matches(s string) bool
	Fingerprint() string
	Type() string
	Scheme() enum.PublicKeyScheme
}

func ParseString(keyData string) (KeyInfo, *types.Identity, string, error) {
	if len(keyData) == 0 {
		return nil, nil, "", errors.InvalidArgument("empty key")
	}

	const pgpHeader = "-----BEGIN PGP PUBLIC KEY BLOCK-----"
	const pgpFooter = "-----END PGP PUBLIC KEY BLOCK-----"

	if strings.HasPrefix(keyData, pgpHeader) && strings.HasSuffix(keyData, pgpFooter) {
		key, err := parsePGP(strings.NewReader(keyData))
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to parse PGP key: %w", err)
		}

		return key, &key.Identity, "", nil
	}

	key, comment, err := parseSSH([]byte(keyData))
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to parse SSH key: %w", err)
	}

	return key, nil, comment, nil
}
