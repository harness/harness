package gogitlab

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetUserKeys(t *testing.T) {
	ts, gitlab := Stub("stubs/public_keys/index.json")
	keys, err := gitlab.UserKeys()

	assert.Equal(t, err, nil)
	assert.Equal(t, len(keys), 2)
	defer ts.Close()
}

func TestGetUserKey(t *testing.T) {
	ts, gitlab := Stub("stubs/public_keys/show.json")
	key, err := gitlab.UserKey("1")

	assert.Equal(t, err, nil)
	assert.IsType(t, new(PublicKey), key)
	assert.Equal(t, key.Title, "Public key")
	assert.Equal(t, key.Key, "ssh-rsa AAAAB3NzaC1yc2EAAAABJQAAAIEAiPWx6WM4lhHNedGfBpPJNPpZ7yKu+dnn1SJejgt4596k6YjzGGphH2TUxwKzxcKDKKezwkpfnxPkSMkuEspGRt/aZZ9wa++Oi7Qkr8prgHc4soW6NUlfDzpvZK2H5E7eQaSeP3SAwGmQKUFHCddNaP0L+hM7zhFNzjFvpaMgJw0=")
	defer ts.Close()
}

func TestAddKey(t *testing.T) {
	ts, gitlab := Stub("")
	err := gitlab.AddKey("Public key", "stubbed key")

	assert.Equal(t, err, nil)
	defer ts.Close()
}

func TestAddUserKey(t *testing.T) {
	ts, gitlab := Stub("")
	err := gitlab.AddUserKey("1", "Public key", "stubbed key")

	assert.Equal(t, err, nil)
	defer ts.Close()
}

func TestDeleteKey(t *testing.T) {
	ts, gitlab := Stub("")
	err := gitlab.DeleteKey("1")

	assert.Equal(t, err, nil)
	defer ts.Close()
}
