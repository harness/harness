package handler

import (
	"net/http"

	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
)

// Display the dashboard for a specific user
func UserShow(w http.ResponseWriter, r *http.Request, u *User) error {
	// list of repositories owned by User
	repos, err := database.ListRepos(u.ID)
	if err != nil {
		return err
	}
	// list of user team accounts
	teams, err := database.ListTeams(u.ID)
	if err != nil {
		return err
	}
	// list of recent commits
	commits, err := database.ListCommitsUser(u.ID)
	if err != nil {
		return err
	}

	data := struct {
		User    *User
		Repos   []*Repo
		Teams   []*Team
		Commits []*RepoCommit
	}{u, repos, teams, commits}
	return RenderTemplate(w, "user_dashboard.html", &data)
}

// return an HTML form for editing a user
func UserEdit(w http.ResponseWriter, r *http.Request, u *User) error {
	return RenderTemplate(w, "user_profile.html", struct{ User *User }{u})
}

// return an HTML form for editing a user password
func UserPass(w http.ResponseWriter, r *http.Request, u *User) error {
	return RenderTemplate(w, "user_password.html", struct{ User *User }{u})
}

// return an HTML form for deleting a user.
func UserDeleteConfirm(w http.ResponseWriter, r *http.Request, u *User) error {
	return RenderTemplate(w, "user_delete.html", struct{ User *User }{u})
}

// update a specific user
func UserUpdate(w http.ResponseWriter, r *http.Request, u *User) error {
	// set the name and email from the form data
	u.Name = r.FormValue("name")
	u.SetEmail(r.FormValue("email"))

	if err := u.Validate(); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}
	if err := database.SaveUser(u); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}

	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

// update a specific user's password
func UserPassUpdate(w http.ResponseWriter, r *http.Request, u *User) error {
	// set the name and email from the form data
	pass := r.FormValue("password")
	if err := u.SetPassword(pass); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}
	// save the updated password to the database
	if err := database.SaveUser(u); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}
	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

// delete a specific user.
func UserDelete(w http.ResponseWriter, r *http.Request, u *User) error {
	// the user must confirm their password before deleting
	password := r.FormValue("password")
	if err := u.ComparePassword(password); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}
	// TODO we need to delete all repos, builds, commits, branches, etc
	// TODO we should transfer ownership of all team-owned projects to the team owner
	// delete the account
	if err := database.DeleteUser(u.ID); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}

	Logout(w, r)
	return nil
}

// Display a list of all Teams for the currently authenticated User.
func UserTeams(w http.ResponseWriter, r *http.Request, u *User) error {
	teams, err := database.ListTeams(u.ID)
	if err != nil {
		return err
	}
	data := struct {
		User  *User
		Teams []*Team
	}{u, teams}
	return RenderTemplate(w, "user_teams.html", &data)
}
