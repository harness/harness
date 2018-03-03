// Copyright 2018 Drone.IO Inc
// Use of this software is governed by the Drone Enterpise License
// that can be found in the LICENSE file.

package vault

import (
	"os"
	"reflect"
	"testing"

	"github.com/kr/pretty"
)

func TestVaultGet(t *testing.T) {
	client, closer := TestServer(t, nil)
	defer closer()

	_, err := client.Logical().Write("secret/testing/drone/a", map[string]interface{}{
		"value": "hello",
		"image": "golang",
		"event": "push,pull_request",
		"repo":  "octocat/hello-world,github/*",
	})
	if err != nil {
		t.Error(err)
		return
	}

	plugin := vault{client: client}
	secret, err := plugin.get("secret/testing/drone/a")
	if err != nil {
		t.Error(err)
		return
	}

	if got, want := secret.Value, "hello"; got != want {
		t.Errorf("Expect secret value %s, got %s", want, got)
	}

	secret, err = plugin.get("secret/testing/drone/404")
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
	got := parseVaultSecret(data)
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
