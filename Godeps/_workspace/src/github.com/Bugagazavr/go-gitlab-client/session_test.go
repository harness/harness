package gogitlab

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetSesison(t *testing.T) {
	ts, gitlab := Stub("stubs/session/index.json")
	session, err := gitlab.GetSession("john@example.com", "samplepassword")

	assert.Equal(t, err, nil)
	assert.Equal(t, session.Id, 1)
	assert.Equal(t, session.UserName, "john_smith")
	assert.Equal(t, session.Name, "John Smith")
	assert.Equal(t, session.State, "active")
	assert.Equal(t, session.AvatarURL, "http://someurl.com/avatar.png")
	assert.Equal(t, session.IsAdmin, false)
	assert.Equal(t, session.Bio, "somebio")
	assert.Equal(t, session.Skype, "someskype")
	assert.Equal(t, session.LinkedIn, "somelinkedin")
	assert.Equal(t, session.Twitter, "sometwitter")
	assert.Equal(t, session.WebsiteURL, "http://example.com")
	assert.Equal(t, session.Email, "john@example.com")
	assert.Equal(t, session.ThemeId, 1)
	assert.Equal(t, session.ColorSchemeId, 1)
	assert.Equal(t, session.ExternUid, "someuid")
	assert.Equal(t, session.Provider, "github.com")
	assert.Equal(t, session.CanCreateGroup, true)
	assert.Equal(t, session.CanCreateProject, true)
	assert.Equal(t, session.PrivateToken, "dd34asd13as")
	defer ts.Close()
}
