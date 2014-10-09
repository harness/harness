package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/dchest/authcookie"
	"github.com/drone/drone/pkg/database"
	"github.com/drone/drone/pkg/mail"
	. "github.com/drone/drone/pkg/model"
)

// Display a list of ALL users in the system
func AdminUserList(w http.ResponseWriter, r *http.Request, u *User) error {
	users, err := database.ListUsers()
	if err != nil {
		return err
	}

	data := struct {
		User  *User
		Users []*User
	}{u, users}

	return RenderTemplate(w, "admin_users.html", &data)
}

// Invite a user to join the system
func AdminUserAdd(w http.ResponseWriter, r *http.Request, u *User) error {
	return RenderTemplate(w, "admin_users_add.html", &struct{ User *User }{u})
}

func UserInvite(w http.ResponseWriter, r *http.Request) error {
	// generate the password reset token
	email := r.FormValue("email")
	token := authcookie.New(email, time.Now().Add(12*time.Hour), secret)

	// get settings
	hostname := database.SettingsMust().URL().String()
	emailEnabled := database.SettingsMust().SmtpServer != ""

	if !emailEnabled {
		// Email is not enabled, so must let the user know the signup link
		link := fmt.Sprintf("%v/register?token=%v", hostname, token)
		return RenderText(w, link, http.StatusOK)
	}

	// send data to template
	data := struct {
		Host  string
		Email string
		Token string
	}{hostname, email, token}

	// send the email message async
	go mail.SendActivation(email, data)

	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

// Invite a user to join the system
func AdminUserInvite(w http.ResponseWriter, r *http.Request, u *User) error {
	return UserInvite(w, r)
}

// Form to edit a user
func AdminUserEdit(w http.ResponseWriter, r *http.Request, u *User) error {
	idstr := r.FormValue("id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		return err
	}

	// get the user from the database
	user, err := database.GetUser(int64(id))
	if err != nil {
		return err
	}

	data := struct {
		User     *User
		EditUser *User
	}{u, user}

	return RenderTemplate(w, "admin_users_edit.html", &data)
}

func AdminUserUpdate(w http.ResponseWriter, r *http.Request, u *User) error {
	// get the ID from the URL parameter
	idstr := r.FormValue("id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		return err
	}

	// get the user from the database
	user, err := database.GetUser(int64(id))
	if err != nil {
		return err
	}

	// update if user is administrator or not
	switch r.FormValue("Admin") {
	case "true":
		user.Admin = true
	case "false":
		user.Admin = false
	}

	// saving user
	if err := database.SaveUser(user); err != nil {
		return err
	}

	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

func AdminUserDelete(w http.ResponseWriter, r *http.Request, u *User) error {
	// get the ID from the URL parameter
	idstr := r.FormValue("id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		return err
	}

	// cannot delete self
	if u.ID == int64(id) {
		return RenderForbidden(w)
	}

	// delete the user
	if err := database.DeleteUser(int64(id)); err != nil {
		return err
	}

	http.Redirect(w, r, "/account/admin/users", http.StatusSeeOther)
	return nil
}

// Return an HTML form for the User to update the site settings.
func AdminSettings(w http.ResponseWriter, r *http.Request, u *User) error {
	// get settings from database
	settings := database.SettingsMust()

	data := struct {
		User     *User
		Settings *Settings
	}{u, settings}

	return RenderTemplate(w, "admin_settings.html", &data)
}

func AdminSettingsUpdate(w http.ResponseWriter, r *http.Request, u *User) error {
	// get settings from database
	settings := database.SettingsMust()

	// update smtp settings
	settings.Domain = r.FormValue("Domain")
	settings.Scheme = r.FormValue("Scheme")

	// update bitbucket settings
	settings.BitbucketKey = r.FormValue("BitbucketKey")
	settings.BitbucketSecret = r.FormValue("BitbucketSecret")

	// update github settings
	settings.GitHubKey = r.FormValue("GitHubKey")
	settings.GitHubSecret = r.FormValue("GitHubSecret")
	settings.GitHubDomain = r.FormValue("GitHubDomain")
	settings.GitHubApiUrl = r.FormValue("GitHubApiUrl")

	// update gitlab settings
	settings.GitlabApiUrl = r.FormValue("GitlabApiUrl")
	glUrl, err := url.Parse(settings.GitlabApiUrl)
	if err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}
	settings.GitlabDomain = glUrl.Host

	// update smtp settings
	settings.SmtpServer = r.FormValue("SmtpServer")
	settings.SmtpPort = r.FormValue("SmtpPort")
	settings.SmtpAddress = r.FormValue("SmtpAddress")
	settings.SmtpUsername = r.FormValue("SmtpUsername")
	settings.SmtpPassword = r.FormValue("SmtpPassword")

	settings.OpenInvitations = (r.FormValue("OpenInvitations") == "on")

	// validate user input
	if err := settings.Validate(); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}

	// persist changes
	if err := database.SaveSettings(settings); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}


    //send test mail to saved settings 

    if err := mail.SendTest(); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
    }

    //TODO shouldn't it be deleted, is it?
	// make sure the mail package is updated with the
	// latest client information.
	//mail.SetClient(&mail.SMTPClient{
	//	Host: settings.SmtpServer,
	//	Port: settings.SmtpPort,
	//	User: settings.SmtpUsername,
	//	Pass: settings.SmtpPassword,
	//	From: settings.SmtpAddress,
	//})

	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

func Install(w http.ResponseWriter, r *http.Request) error {
	// we can only perform the inital installation if no
	// users exist in the system
	if users, err := database.ListUsers(); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	} else if len(users) != 0 {
		// if users exist in the systsem
		// we should render a NotFound page
		return RenderNotFound(w)
	}

	return RenderTemplate(w, "install.html", true)
}

func InstallPost(w http.ResponseWriter, r *http.Request) error {
	// we can only perform the inital installation if no
	// users exist in the system
	if users, err := database.ListUsers(); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	} else if len(users) != 0 {
		// if users exist in the systsem
		// we should render a NotFound page
		return RenderNotFound(w)
	}

	// set the email and name
	user := NewUser(r.FormValue("name"), r.FormValue("email"))
	user.Admin = true

	// set the new password
	if err := user.SetPassword(r.FormValue("password")); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}

	// verify fields are correct
	if err := user.Validate(); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}

	// save to the database
	if err := database.SaveUser(user); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}

	// update settings
	settings := Settings{}
	settings.Domain = r.FormValue("Domain")
	settings.Scheme = r.FormValue("Scheme")
	settings.GitHubApiUrl = "https://api.github.com"
	settings.GitHubDomain = "github.com"
	settings.GitlabApiUrl = "https://gitlab.com"
	settings.GitlabDomain = "gitlab.com"
	database.SaveSettings(&settings)

	// add the user to the session object
	// so that he/she is loggedin
	SetCookie(w, r, "_sess", user.Email)

	// send the user to the settings page
	// to complete the configuration.
	http.Redirect(w, r, "/account/admin/settings", http.StatusSeeOther)
	return nil
}
