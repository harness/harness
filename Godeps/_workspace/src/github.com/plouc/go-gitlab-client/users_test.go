package gogitlab

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUsers(t *testing.T) {
	ts, gitlab := Stub("stubs/users/index.json")
	users, err := gitlab.Users()

	assert.Equal(t, err, nil)
	assert.Equal(t, len(users), 2)
	defer ts.Close()
}

func TestUser(t *testing.T) {
	ts, gitlab := Stub("stubs/users/show.json")
	user, err := gitlab.User("plouc")

	assert.Equal(t, err, nil)
	assert.IsType(t, new(User), user)
	assert.Equal(t, user.Id, 6)
	assert.Equal(t, user.Username, "plouc")
	assert.Equal(t, user.Name, "Raphaël Benitte")
	assert.Equal(t, user.Bio, "")
	assert.Equal(t, user.Skype, "")
	assert.Equal(t, user.LinkedIn, "")
	assert.Equal(t, user.Twitter, "")
	assert.Equal(t, user.ThemeId, 2)
	assert.Equal(t, user.State, "active")
	assert.Equal(t, user.CreatedAt, "2001-01-01T00:00:00Z")
	assert.Equal(t, user.ExternUid, "uid=plouc")
	assert.Equal(t, user.Provider, "ldap")
	defer ts.Close()
}

func TestDeleteUser(t *testing.T) {
	ts, gitlab := Stub("")
	err := gitlab.DeleteUser("1")

	assert.Equal(t, err, nil)
	defer ts.Close()
}

func TestCurrentUser(t *testing.T) {
	ts, gitlab := Stub("stubs/users/current.json")
	user, err := gitlab.CurrentUser()

	assert.Equal(t, err, nil)
	assert.Equal(t, user.Username, "john_smith")
	defer ts.Close()
}
