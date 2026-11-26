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
	"testing"
	"time"

	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestMscCheckMarshalJSON(t *testing.T) {
	c := Check{Identifier: "ci/build", Status: enum.CheckStatusSuccess}
	raw, err := json.Marshal(c)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"uid":"ci/build"`)

	result := CheckResult{Identifier: "gate", Status: enum.CheckStatusFailure}
	raw, err = json.Marshal(result)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"uid":"gate"`)
}

func TestMscConnectorAuthValidate(t *testing.T) {
	basic := &ConnectorAuth{
		AuthType: enum.ConnectorAuthTypeBasic,
		Basic:    &BasicAuthCreds{Username: "user", Password: SecretRef{Identifier: "pwd"}},
	}
	require.NoError(t, basic.Validate())

	bearer := &ConnectorAuth{
		AuthType: enum.ConnectorAuthTypeBearer,
		Bearer:   &BearerTokenCreds{Token: SecretRef{Identifier: "tok"}},
	}
	require.NoError(t, bearer.Validate())

	require.Error(t, (&ConnectorAuth{AuthType: enum.ConnectorAuthTypeBasic}).Validate())
	require.Error(t, (&ConnectorAuth{AuthType: enum.ConnectorAuthTypeBearer}).Validate())
	require.Error(t, (&ConnectorAuth{AuthType: "other"}).Validate())
}

func TestMscGithubConnectorData(t *testing.T) {
	good := &GithubConnectorData{
		APIURL: "https://api.github.com",
		Auth: &ConnectorAuth{
			AuthType: enum.ConnectorAuthTypeBearer,
			Bearer:   &BearerTokenCreds{Token: SecretRef{Identifier: "gh"}},
		},
	}
	require.NoError(t, good.Validate())
	require.Equal(t, enum.ConnectorTypeGithub, good.Type())

	require.Error(t, (&GithubConnectorData{}).Validate())
}

func TestMscGitSignatureAccessors(t *testing.T) {
	when := time.Unix(0, 0).UTC()
	sig := GitSignatureResult{}
	commit := &Commit{
		SHA:       sha.Must("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"),
		Committer: Signature{Identity: Identity{Name: "c", Email: "c@test.invalid"}, When: when},
	}
	commit.SetSignature(&sig)
	require.Equal(t, commit.SHA, commit.GetSHA())
	require.Same(t, &sig, commit.Signature)
	require.Equal(t, &commit.Committer, commit.GetSigner())

	tag := &CommitTag{Tagger: &Signature{Identity: Identity{Name: "t", Email: "t@test.invalid"}, When: when}}
	tag.SetSignature(&sig)
	require.Equal(t, tag.SHA, tag.GetSHA())
	require.Equal(t, tag.Tagger, tag.GetSigner())
}

func TestMscCodeOwnersValidationAddf(t *testing.T) {
	var v CodeOwnersValidation
	v.Addf(enum.CodeOwnerViolationCodePatternInvalid, "bad pattern %s", "docs/*")
	require.Len(t, v.Violations, 1)
	require.Contains(t, v.Violations[0].Message, "docs/*")
}

func TestMscConnectorConfigValidate(t *testing.T) {
	cfg := ConnectorConfig{
		Github: &GithubConnectorData{
			APIURL: "https://api.github.com",
			Auth: &ConnectorAuth{
				AuthType: enum.ConnectorAuthTypeBearer,
				Bearer:   &BearerTokenCreds{Token: SecretRef{Identifier: "gh"}},
			},
		},
	}
	require.NoError(t, cfg.Validate(enum.ConnectorTypeGithub))
	require.Error(t, cfg.Validate(enum.ConnectorType("other")))
}

func TestMscGetSignedData(t *testing.T) {
	data := &SignedData{Type: "pgp", Signature: []byte("sig")}
	commit := &Commit{SignedData: data}
	require.Same(t, data, commit.GetSignedData())

	tag := &CommitTag{SignedData: data}
	require.Same(t, data, tag.GetSignedData())
}
