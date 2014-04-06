package bitbucket

import (
	"testing"
)

func Test_Keys(t *testing.T) {

	// Test Public key that we'll add to the account	
	public := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCkRHDtJljvvZudiXxLt+JoHEQ4olLX6vZrVkm4gRVZEC7llKs9lXubHAwzIm+odIWZnoqNKjh0tSQYd5UAlSsrzn9YVvp0Lc2eJo0N1AWuyMzb9na+lfhT/YdM3Htkm14v7OZNdX4fqff/gCuLBIv9Bc9XH0jfEliOmfaDMQsbzcDi4usRoXBrJQQiu6M0A9FF0ruBdpKp0q08XSteGh5cMn1LvOS+vLrkHXi3bOXWvv7YXoVoI5OTUQGJjxmEehRssYiMfwD58cv7v2+PMLR3atGVCnoxxu/zMkXQlBKmEyN9VS7Cr8WOoZcNsCd9C6CCrbP5HZnjiE8F0R9d1zjP test@localhost"
	title := "test@localhost"

	// create a new public key
	key, err := client.Keys.Create(testUser, public, title)
	if err != nil {
		t.Error(err)
		return
	}

	// cleanup after ourselves & delete this dummy key
	defer client.Keys.Delete(testUser, key.Id)

	// Get the new key we recently created
	find, err := client.Keys.Find(testUser, key.Id)
	if title != find.Label {
		t.Errorf("key label [%v]; want [%v]", find.Label, title)
	}

	// Get a list of SSH keys for the user
	keys, err := client.Keys.List(testUser)
	if err != nil {
		t.Error(err)
	}

	if len(keys) == 0 {
		t.Errorf("List of keys returned empty set")
	}

}
