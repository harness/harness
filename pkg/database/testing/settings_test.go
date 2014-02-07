package database

import (
	"testing"

	"github.com/drone/drone/pkg/database"
)

func TestGetSettings(t *testing.T) {
	Setup()
	defer Teardown()

	// even though no settings exist yet, we should
	// not see an error since we supress the msg
	settings, err := database.GetSettings()
	//if err != nil {
	//	t.Error(err)
	//}

	// add some settings
	//settings := &modelSettings{}
	settings.Scheme = "https"
	settings.Domain = "foo.com"
	settings.BitbucketKey = "bitbucketkey"
	settings.BitbucketSecret = "bitbucketsecret"
	settings.GitHubKey = "githubkey"
	settings.GitHubSecret = "githubsecret"
	settings.SmtpAddress = "noreply@foo.bar"
	settings.SmtpServer = "0.0.0.0"
	settings.SmtpUsername = "username"
	settings.SmtpPassword = "password"

	// save the updated settings
	if err := database.SaveSettings(settings); err != nil {
		t.Error(err)
	}

	// re-retrieve the settings post-save
	settings, err = database.GetSettings()
	if err != nil {
		t.Error(err)
	}

	if settings.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, settings.ID)
	}

	if settings.Scheme != "https" {
		t.Errorf("Exepected Scheme %s, got %s", "https", settings.Scheme)
	}

	if settings.Domain != "foo.com" {
		t.Errorf("Exepected Domain %s, got %s", "foo.com", settings.Domain)
	}

	// Verify caching works and is threadsafe
	settingsA, _ := database.GetSettings()
	settingsB, _ := database.GetSettings()
	settingsA.Domain = "foo.bar.baz"
	if settingsA.Domain == settingsB.Domain {
		t.Errorf("Exepected Domain ThreadSafe and unchanged")
	}
}
