package github

import (
	"testing"
)

func Test_Orgs(t *testing.T) {

	// Get the currently authenticated user
	_, err := client.Orgs.List()
	if err != nil {
		t.Error(err)
		return
	}
}
