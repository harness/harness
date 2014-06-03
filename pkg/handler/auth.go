package handler

import (
	"log"
	"net/http"

	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
	"github.com/drone/go-bitbucket/bitbucket"
	"github.com/drone/go-bitbucket/oauth1"
	"github.com/drone/go-github/github"
	"github.com/drone/go-github/oauth2"
	oauth1_stash "github.com/reinbach/go-stash/oauth1"
)

// Create the User session.
func Authorize(w http.ResponseWriter, r *http.Request) error {
	// extract form data
	username := r.FormValue("username")
	password := r.FormValue("password")
	returnTo := r.FormValue("return_to")

	// get the user from the database
	user, err := database.GetUserEmail(username)
	if err != nil {
		return RenderTemplate(w, "login_error.html", nil)
	}

	// verify the password
	if err := user.ComparePassword(password); err != nil {
		return RenderTemplate(w, "login_error.html", nil)
	}

	// add the user to the session object
	SetCookie(w, r, "_sess", username)

	// where should we send the user to?
	if len(returnTo) == 0 {
		returnTo = "/dashboard"
	}

	// redirect to the homepage
	http.Redirect(w, r, returnTo, http.StatusSeeOther)
	return nil
}

func LinkGithub(w http.ResponseWriter, r *http.Request, u *User) error {

	// get settings from database
	settings := database.SettingsMust()

	// github OAuth2 Data
	var oauth = oauth2.Client{
		RedirectURL:      settings.URL().String() + "/auth/login/github",
		AccessTokenURL:   "https://" + settings.GitHubDomain + "/login/oauth/access_token",
		AuthorizationURL: "https://" + settings.GitHubDomain + "/login/oauth/authorize",
		ClientId:         settings.GitHubKey,
		ClientSecret:     settings.GitHubSecret,
	}

	// get the OAuth code
	code := r.FormValue("code")
	if len(code) == 0 {
		scope := "repo,repo:status,user:email"
		state := "FqB4EbagQ2o"
		redirect := oauth.AuthorizeRedirect(scope, state)
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return nil
	}

	// exchange code for an auth token
	token, err := oauth.GrantToken(code)
	if err != nil {
		log.Println("Error granting GitHub authorization token")
		return err
	}

	// create the client
	client := github.New(token.AccessToken)
	client.ApiUrl = settings.GitHubApiUrl

	// get the user information
	githubUser, err := client.Users.Current()
	if err != nil {
		log.Println("Error retrieving currently authenticated GitHub user")
		return err
	}

	// save the github token to the user account
	u.GithubToken = token.AccessToken
	u.GithubLogin = githubUser.Login
	if err := database.SaveUser(u); err != nil {
		log.Println("Error persisting user's GitHub auth token to the database")
		return err
	}

	http.Redirect(w, r, "/new/github.com", http.StatusSeeOther)
	return nil
}

func LinkBitbucket(w http.ResponseWriter, r *http.Request, u *User) error {

	// get settings from database
	settings := database.SettingsMust()

	// bitbucket oauth1 consumer
	var consumer = oauth1.Consumer{
		RequestTokenURL:  "https://bitbucket.org/api/1.0/oauth/request_token/",
		AuthorizationURL: "https://bitbucket.org/!api/1.0/oauth/authenticate",
		AccessTokenURL:   "https://bitbucket.org/api/1.0/oauth/access_token/",
		CallbackURL:      settings.URL().String() + "/auth/login/bitbucket",
		ConsumerKey:      settings.BitbucketKey,
		ConsumerSecret:   settings.BitbucketSecret,
	}

	// get the oauth verifier
	verifier := r.FormValue("oauth_verifier")
	if len(verifier) == 0 {
		// Generate a Request Token
		requestToken, err := consumer.RequestToken()
		if err != nil {
			return err
		}

		// add the request token as a signed cookie
		SetCookie(w, r, "bitbucket_token", requestToken.Encode())

		url, _ := consumer.AuthorizeRedirect(requestToken)
		http.Redirect(w, r, url, http.StatusSeeOther)
		return nil
	}

	// remove bitbucket token data once before redirecting
	// back to the application.
	defer DelCookie(w, r, "bitbucket_token")

	// get the tokens from the request
	requestTokenStr := GetCookie(r, "bitbucket_token")
	requestToken, err := oauth1.ParseRequestTokenStr(requestTokenStr)
	if err != nil {
		return err
	}

	// exchange for an access token
	accessToken, err := consumer.AuthorizeToken(requestToken, verifier)
	if err != nil {
		return err
	}

	// create the Bitbucket client
	client := bitbucket.New(
		settings.BitbucketKey,
		settings.BitbucketSecret,
		accessToken.Token(),
		accessToken.Secret(),
	)

	// get the currently authenticated Bitbucket User
	user, err := client.Users.Current()
	if err != nil {
		return err
	}

	// update the user account
	u.BitbucketLogin = user.User.Username
	u.BitbucketToken = accessToken.Token()
	u.BitbucketSecret = accessToken.Secret()
	if err := database.SaveUser(u); err != nil {
		return err
	}

	http.Redirect(w, r, "/new/bitbucket.org", http.StatusSeeOther)
	return nil
}

func LinkStash(w http.ResponseWriter, r *http.Request, u *User) error {

	// get settings from database
	settings := database.SettingsMust()

	// stash oauth1 consumer
	var consumer = oauth1_stash.Consumer{
		RequestTokenURL:       settings.StashDomain + "/plugins/servlet/oauth/request-token",
		AuthorizationURL:      settings.StashDomain + "/plugins/servlet/oauth/authorize",
		AccessTokenURL:        settings.StashDomain + "/plugins/servlet/oauth/access-token",
		CallbackURL:           settings.URL().String() + "/auth/login/stash",
		ConsumerKey:           settings.StashKey,
		ConsumerPrivateKeyPem: settings.StashPrivateKey,
	}

	// get the oauth verifier
	verifier := r.FormValue("oauth_verifier")
	if len(verifier) == 0 {
		// Generate a Request Token
		requestToken, err := consumer.RequestToken()
		if err != nil {
			return err
		}

		// add the request token as a signed cookie
		SetCookie(w, r, "stash_token", requestToken.Encode())

		url, _ := consumer.AuthorizeRedirect(requestToken)
		http.Redirect(w, r, url, http.StatusSeeOther)
		return nil
	}

	// remove stash token data once before redirecting
	// back to the application.
	defer DelCookie(w, r, "stash_token")

	// get the tokens from the request
	requestTokenStr := GetCookie(r, "stash_token")
	requestToken, err := oauth1_stash.ParseRequestTokenStr(requestTokenStr)
	if err != nil {
		return err
	}

	// exchange for an access token
	accessToken, err := consumer.AuthorizeToken(requestToken, verifier)
	if err != nil {
		return err
	}

	// create the Stash client
	// client := stash.New(
	// 	settings.StashKey,
	// 	settings.StashSecret,
	// 	accessToken.Token(),
	// 	accessToken.Secret(),
	// 	settings.StashPrivateKey,
	// )

	// get the currently authenticated Stash User
	// user, err := client.Users.Current()
	// if err != nil {
	// 	return err
	// }

	// update the user account
	//u.StashLogin = user.Username
	u.StashToken = accessToken.Token()
	u.StashSecret = accessToken.Secret()
	if err := database.SaveUser(u); err != nil {
		return err
	}

	http.Redirect(w, r, "/new/stash", http.StatusSeeOther)
	return nil
}
