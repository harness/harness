package handler

import (
	"crypto/rand"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/dchest/authcookie"
	"github.com/dchest/passwordreset"
	"github.com/drone/drone/pkg/database"
	"github.com/drone/drone/pkg/mail"
	. "github.com/drone/drone/pkg/model"
)

var (
	// Secret key used to sign auth cookies,
	// password reset tokens, etc.
	secret = generateRandomKey(256)
)

// GenerateRandomKey creates a random key of size length bytes
func generateRandomKey(strength int) []byte {
	k := make([]byte, strength)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}

func SetSecret(sec string) {
	secret = []byte(sec)
}

// Returns an HTML index.html page if the user is
// not currently authenticated, otherwise redirects
// the user to their personal dashboard screen
func Index(w http.ResponseWriter, r *http.Request) error {
	// is the user already authenticated then
	// redirect to the dashboard page
	if _, err := r.Cookie("_sess"); err == nil {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return nil
	}

	// otherwise redirect to the login page
	http.Redirect(w, r, "/login", http.StatusSeeOther)
	return nil
}

// Return an HTML form for the User to login.
func Login(w http.ResponseWriter, r *http.Request) error {
	var settings, _ = database.GetSettings()

	data := struct {
		Settings *Settings
	}{settings}

	return RenderTemplate(w, "login.html", &data)
}

// Terminate the User session.
func Logout(w http.ResponseWriter, r *http.Request) error {
	DelCookie(w, r, "_sess")

	http.Redirect(w, r, "/login", http.StatusSeeOther)
	return nil
}

// Return an HTML form for the User to request a password reset.
func Forgot(w http.ResponseWriter, r *http.Request) error {
	return RenderTemplate(w, "forgot.html", nil)
}

// Return an HTML form for the User to perform a password reset.
// This page must be visited from a Password Reset email that
// contains a hash to verify the User's identity.
func Reset(w http.ResponseWriter, r *http.Request) error {
	return RenderTemplate(w, "reset.html", &struct{ Error string }{""})
}

// Return an HTML form for the User to signup.
func SignUp(w http.ResponseWriter, r *http.Request) error {
	var settings, _ = database.GetSettings()

	if settings == nil || !settings.OpenInvitations {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return nil
	}

	return RenderTemplate(w, "signup.html", nil)
}

// Return an HTML form to register for a new account. This
// page must be visited from a Signup email that contains
// a hash to verify the Email address is correct.
func Register(w http.ResponseWriter, r *http.Request) error {
	return RenderTemplate(w, "register.html", &struct{ Error string }{""})
}

func ForgotPost(w http.ResponseWriter, r *http.Request) error {
	email := r.FormValue("email")

	// attempt to retrieve the user by email address
	user, err := database.GetUserEmail(email)
	if err != nil {
		log.Printf("could not find user %s to reset password. %s", email, err)
		// if we can't find the email, we still display
		// the template to the user. This prevents someone
		// from trying to guess passwords through trial & error
		return RenderTemplate(w, "forgot_sent.html", nil)
	}

	// hostname from settings
	hostname := database.SettingsMust().URL().String()

	// generate the password reset token
	token := passwordreset.NewToken(user.Email, 12*time.Hour, []byte(user.Password), secret)
	data := struct {
		Host  string
		User  *User
		Token string
	}{hostname, user, token}

	// send the email message async
	go func() {
		if err := mail.SendPassword(email, data); err != nil {
			log.Printf("error sending password reset email to %s. %s", email, err)
		}
	}()

	// render the template indicating a success
	return RenderTemplate(w, "forgot_sent.html", nil)
}

func ResetPost(w http.ResponseWriter, r *http.Request) error {
	// verify the token and extract the username
	token := r.FormValue("token")
	email, err := passwordreset.VerifyToken(token, database.GetPassEmail, secret)
	if err != nil {
		return RenderTemplate(w, "reset.html", &struct{ Error string }{"Your password reset request is expired."})
	}

	// get the user from the database
	user, err := database.GetUserEmail(email)
	if err != nil {
		return RenderTemplate(w, "reset.html", &struct{ Error string }{"Unable to locate user account."})
	}

	// get the new password
	password := r.FormValue("password")
	if err := user.SetPassword(password); err != nil {
		return RenderTemplate(w, "reset.html", &struct{ Error string }{err.Error()})
	}

	// save to the database
	if err := database.SaveUser(user); err != nil {
		return RenderTemplate(w, "reset.html", &struct{ Error string }{"Unable to update password. Please try again"})
	}

	// add the user to the session object
	SetCookie(w, r, "_sess", user.Email)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	return nil
}

func SignUpPost(w http.ResponseWriter, r *http.Request) error {
	// if self-registration is disabled we should display an
	// error message to the user.
	if !database.SettingsMust().OpenInvitations {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return nil
	}

	// generate the password reset token
	email := r.FormValue("email")
	token := authcookie.New(email, time.Now().Add(12*time.Hour), secret)

	// get the hostname from the database for use in the email
	hostname := database.SettingsMust().URL().String()

	// data used to generate the email template
	data := struct {
		Host  string
		Email string
		Token string
	}{hostname, email, token}

	// send the email message async
	go mail.SendActivation(email, data)

	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

func RegisterPost(w http.ResponseWriter, r *http.Request) error {
	// verify the token and extract the username
	token := r.FormValue("token")
	email := authcookie.Login(token, secret)
	if len(email) == 0 {
		return RenderTemplate(w, "register.html", &struct{ Error string }{"Your registration email is expired."})
	}

	// set the email and name
	user := NewUser(r.FormValue("name"), email)

	// set the new password
	password := r.FormValue("password")
	if err := user.SetPassword(password); err != nil {
		return RenderTemplate(w, "register.html", &struct{ Error string }{err.Error()})
	}

	// verify fields are correct
	if err := user.Validate(); err != nil {
		return RenderTemplate(w, "register.html", &struct{ Error string }{err.Error()})
	}

	// save to the database
	if err := database.SaveUser(user); err != nil {
		return err
	}

	// add the user to the session object
	SetCookie(w, r, "_sess", user.Email)

	// redirect the user to their dashboard
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	return nil
}
