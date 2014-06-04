package handler

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
)

// ErrorHandler wraps the default http.HandleFunc to handle an
// error as the return value.
type ErrorHandler func(w http.ResponseWriter, r *http.Request) error

func (h ErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		log.Print(err)
	}
}

// UserHandler wraps the default http.HandlerFunc to include
// the currently authenticated User in the method signature,
// in addition to handling an error as the return value.
type UserHandler func(w http.ResponseWriter, r *http.Request, user *User) error

func (h UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, err := readUser(r)
	if err != nil {
		redirectLogin(w, r)
		return
	}

	if err = h(w, r, user); err != nil {
		log.Print(err)
		RenderError(w, err, http.StatusBadRequest)
	}
}

// AdminHandler wraps the default http.HandlerFunc to include
// the currently authenticated User in the method signature,
// in addition to handling an error as the return value. It also
// verifies the user has Administrative privileges.
type AdminHandler func(w http.ResponseWriter, r *http.Request, user *User) error

func (h AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, err := readUser(r)
	if err != nil {
		redirectLogin(w, r)
		return
	}

	// User MUST have administrative privileges in order
	// to execute the handler.
	if user.Admin == false {
		RenderNotFound(w)
		return
	}

	if err = h(w, r, user); err != nil {
		log.Print(err)
		RenderError(w, err, http.StatusBadRequest)
	}
}

// RepoHandler wraps the default http.HandlerFunc to include
// the currently authenticated User and requested Repository
// in the method signature, in addition to handling an error
// as the return value.
type RepoHandler func(w http.ResponseWriter, r *http.Request, user *User, repo *Repo) error

func (h RepoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// repository name from the URL parameters
	hostParam := r.FormValue(":host")
	userParam := r.FormValue(":owner")
	nameParam := r.FormValue(":name")
	repoName := fmt.Sprintf("%s/%s/%s", hostParam, userParam, nameParam)

	repo, err := database.GetRepoSlug(repoName)
	if err != nil || repo == nil {
		RenderNotFound(w)
		return
	}

	// retrieve the user from the database
	user, err := readUser(r)

	// if the user is not found, we can still
	// serve the page assuming the repository
	// is public.
	switch {
	case err != nil && repo.Private == true:
		redirectLogin(w, r)
		return
	case err != nil && repo.Private == false:
		h(w, r, nil, repo)
		return
	}

	// The User must own the repository OR be a member
	// of the Team that owns the repository OR the repo
	// must not be private.
	if repo.Private && user.ID != repo.UserID {
		if member, _ := database.IsMember(user.ID, repo.TeamID); !member {
			RenderNotFound(w)
			return
		}
	}

	if err = h(w, r, user, repo); err != nil {
		log.Print(err)
		RenderError(w, err, http.StatusBadRequest)
	}
}

// RepoHandler wraps the default http.HandlerFunc to include
// the currently authenticated User and requested Repository
// in the method signature, in addition to handling an error
// as the return value.
type RepoAdminHandler func(w http.ResponseWriter, r *http.Request, user *User, repo *Repo) error

func (h RepoAdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, err := readUser(r)
	if err != nil {
		redirectLogin(w, r)
		return
	}

	// repository name from the URL parameters
	hostParam := r.FormValue(":host")
	userParam := r.FormValue(":owner")
	nameParam := r.FormValue(":name")
	repoName := fmt.Sprintf("%s/%s/%s", hostParam, userParam, nameParam)

	repo, err := database.GetRepoSlug(repoName)
	if err != nil {
		RenderNotFound(w)
		return
	}

	// The User must own the repository OR be a member
	// of the Team that owns the repository.
	if admin, _ := database.IsRepoAdmin(user, repo); admin == false {
		RenderNotFound(w)
		return
	}

	if err = h(w, r, user, repo); err != nil {
		log.Print(err)
		RenderError(w, err, http.StatusBadRequest)
	}
}

// helper function that reads the currently authenticated
// user from the given http.Request.
func readUser(r *http.Request) (*User, error) {
	username := GetCookie(r, "_sess")
	if len(username) == 0 {
		return nil, fmt.Errorf("No user session")
	}

	// get the user from the database
	user, err := database.GetUserEmail(username)
	if err != nil || user == nil || user.ID == 0 {
		return nil, err
	}

	return user, nil
}

// helper function that retrieves the repository based
// on the URL parameters
func readRepo(r *http.Request) (*Repo, error) {
	// get the repo data from the URL parameters
	hostParam := r.FormValue(":host")
	userParam := r.FormValue(":owner")
	nameParam := r.FormValue(":slug")
	repoSlug := fmt.Sprintf("%s/%s/%s", hostParam, userParam, nameParam)

	// get the repo from the database
	return database.GetRepoSlug(repoSlug)
}

// helper function that sends the user to the login page.
func redirectLogin(w http.ResponseWriter, r *http.Request) {
	v := url.Values{}
	v.Add("return_to", r.URL.String())
	http.Redirect(w, r, "/login?"+v.Encode(), http.StatusSeeOther)
}

func renderNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	RenderTemplate(w, "404.amber", nil)
}

func renderBadRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	RenderTemplate(w, "500.amber", nil)
}
