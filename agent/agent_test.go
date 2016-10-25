package agent

import "testing"
import "github.com/drone/drone/model"

func Test_newSecretsReplacer(t *testing.T) {
	secrets := []*model.Secret{
		{Name: "SECRET",
			Value:  "secret_value",
			Images: []string{"*"},
			Events: []string{"*"},
		},
	}

	text := "This is SECRET: secret_value"
	expected := "This is SECRET: *****"
	secretsReplacer := newSecretsReplacer(secrets)
	result := secretsReplacer.Replace(text)

	if result != expected {
		t.Errorf("Wanted %q, got %q.", expected, result)
	}
}
