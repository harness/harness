// Copyright 2018 Drone.IO Inc
// Use of this software is governed by the Drone Enterpise License
// that can be found in the LICENSE file.

package vault

import (
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/vault/api"
	"github.com/kr/pretty"
)

// Use the following snippet to spin up a local vault
// server for integration testing:
//
//    docker run --cap-add=IPC_LOCK -e 'VAULT_DEV_ROOT_TOKEN_ID=dummy' -p 8200:8200 vault
//    export VAULT_ADDR=http://127.0.0.1:8200
//    export VAULT_TOKEN=dummy

func TestVaultGet(t *testing.T) {
	if os.Getenv("VAULT_TOKEN") == "" {
		t.SkipNow()
		return
	}

	client, err := api.NewClient(nil)
	if err != nil {
		t.Error(err)
		return
	}

	_, err = client.Logical().Write("secret/testing/drone/a", map[string]interface{}{
		"value": "hello",
		"fr":    "bonjour",
		"image": "golang",
		"event": "push,pull_request",
		"repo":  "octocat/hello-world,github/*",
	})
	if err != nil {
		t.Error(err)
		return
	}

	plugin := vault{client: client}
	secret, err := plugin.get("secret/testing/drone/a", "value")
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := secret.Value, "hello"; got != want {
		t.Errorf("Expect secret value %s, got %s", want, got)
	}

	secret, err = plugin.get("secret/testing/drone/a", "fr")
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := secret.Value, "bonjour"; got != want {
		t.Errorf("Expect secret value %s, got %s", want, got)
	}

	secret, err = plugin.get("secret/testing/drone/404", "value")
	if err != nil {
		t.Errorf("Expect silent failure when secret does not exist, got %s", err)
	}
	if secret != nil {
		t.Errorf("Expect nil secret when path does not exist")
	}
}

func TestVaultSecretParse(t *testing.T) {
	data := map[string]interface{}{
		"value": "password",
		"event": "push,tag",
		"image": "plugins/s3,plugins/ec2",
		"repo":  "octocat/hello-world,github/*",
	}
	want := vaultSecret{
		Value: "password",
		Event: []string{"push", "tag"},
		Image: []string{"plugins/s3", "plugins/ec2"},
		Repo:  []string{"octocat/hello-world", "github/*"},
	}
	got := parseVaultSecret(data, "value")
	if !reflect.DeepEqual(want, *got) {
		t.Errorf("Failed read Secret.Data")
		pretty.Fdiff(os.Stderr, want, got)
	}
}

func TestVaultSecretMatch(t *testing.T) {
	secret := vaultSecret{
		Repo: []string{"octocat/hello-world", "github/*"},
	}
	if secret.Match("octocat/*") {
		t.Errorf("Expect octocat/* does not match")
	}
	if !secret.Match("octocat/hello-world") {
		t.Errorf("Expect octocat/hello-world does match")
	}
	if !secret.Match("github/hello-world") {
		t.Errorf("Expect github/hello-world does match wildcard")
	}
}
