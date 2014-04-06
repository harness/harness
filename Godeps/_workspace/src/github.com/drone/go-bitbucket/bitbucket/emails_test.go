package bitbucket

import (
	"testing"
)

func Test_Emails(t *testing.T) {

	const dummyEmail = "dummy@localhost.com"

	// CREATE an email entry
	if _, err := client.Emails.Find(testUser, dummyEmail); err != nil {
		_, cerr := client.Emails.Create(testUser, dummyEmail)
		if cerr != nil {
			t.Error(cerr)
			return
		}
	}

	// FIND the email
	_, err := client.Emails.Find(testUser, dummyEmail)
	if err != nil {
		t.Error(err)
	}

	// LIST the email addresses
	emails, err := client.Emails.List(testUser)
	if err != nil {
		t.Error(err)
	}

	if len(emails) == 0 {
		t.Errorf("List of emails returned empty set")
	}
}
