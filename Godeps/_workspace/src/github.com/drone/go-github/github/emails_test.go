package github

import (
	"testing"
)

func Test_Emails(t *testing.T) {

	const dummyEmail = "dummy@localhost.com"

	// CREATE an email entry
	if err := client.Emails.Create(dummyEmail); err != nil {
		t.Error(err)
		return
	}

	// DELETE the email
	defer client.Emails.Delete(dummyEmail)

	// FIND the email
	if _, err := client.Emails.Find(dummyEmail); err != nil {
		t.Error(err)
		return
	}

	// LIST the email addresses
	emails, err := client.Emails.List()
	if err != nil {
		t.Error(err)
		return
	}

	if len(emails) == 0 {
		t.Errorf("List of emails returned empty set")
	}
}
