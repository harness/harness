package bitbucket

import (
	"errors"
	"os"
)

// Instance of the Bitbucket client that we'll use for our unit tests
var client *Client

var (
	// Dummy user that we'll use to run integration tests
	testUser string

	// Dummy repo that we'll use to run integration tests
	testRepo string
)

var (
	// OAuth Consumer Key registered with Bitbucket
	consumerKey string

	// OAuth Consumer Secret registered with Bitbucket
	consumerSecret string

	// A valid access token issues for the `testUser` and `consumerKey`
	accessToken string
	tokenSecret string
)

func init() {
	consumerKey = os.Getenv("BB_CONSUMER_KEY")
	consumerSecret = os.Getenv("BB_CONSUMER_SECRET")
	accessToken = os.Getenv("BB_ACCESS_TOKEN")
	tokenSecret = os.Getenv("BB_TOKEN_SECRET")
	testUser = os.Getenv("BB_USER")
	testRepo = os.Getenv("BB_REPO")

	switch {
	case len(consumerKey) == 0:
		panic(errors.New("must set the BB_CONSUMER_KEY environment variable"))
	case len(consumerSecret) == 0:
		panic(errors.New("must set the BB_CONSUMER_SECRET environment variable"))
	case len(accessToken) == 0:
		panic(errors.New("must set the BB_ACCESS_TOKEN environment variable"))
	case len(tokenSecret) == 0:
		panic(errors.New("must set the BB_TOKEN_SECRET environment variable"))
	case len(testUser) == 0:
		panic(errors.New("must set the BB_USER environment variable"))
	case len(testRepo) == 0:
		panic(errors.New("must set the BB_REPO environment variable"))
	}

	client = New(consumerKey, consumerSecret, accessToken, tokenSecret)
}
