package handler

import (
	"net/http"

	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
	"github.com/drone/go-github/github"
	"github.com/drone/go-github/oauth2"
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
		return err
	}

	// create the client
	client := github.New(token.AccessToken)
	client.ApiUrl = settings.GitHubApiUrl

	// get the user information
	githubUser, err := client.Users.Current()
	if err != nil {
		return err
	}

	// save the github token to the user account
	u.GithubToken = token.AccessToken
	u.GithubLogin = githubUser.Login
	if err := database.SaveUser(u); err != nil {
		return err
	}

	http.Redirect(w, r, "/new/github.com", http.StatusSeeOther)
	return nil
}
