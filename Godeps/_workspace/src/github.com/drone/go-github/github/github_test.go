package github

import (
	"errors"
	"os"
)

// Instance of the Github client that we'll use for our unit tests
var client *Client

var (
	// Dummy user that we'll use to run integration tests
	testUser string

	// Dummy repo that we'll use to run integration tests
	testRepo string

	// Valid OAuth token, issued for the `testUser` that we can
	// use for integration testing.
	testToken string
)

func init() {

	testUser = os.Getenv("GH_USER")
	testRepo = os.Getenv("GH_REPO")
	testToken = os.Getenv("GH_TOKEN")

	switch {
	case len(testUser) == 0:
		panic(errors.New("must set the GH_USER environment variable"))
	case len(testRepo) == 0:
		panic(errors.New("must set the GH_REPO environment variable"))
	case len(testToken) == 0:
		panic(errors.New("must set the GH_TOKEN environment variable"))
	}

	client = New(testToken)
}
