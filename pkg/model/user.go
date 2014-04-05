package model

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"code.google.com/p/go.crypto/bcrypt"
)

var (
	ErrInvalidUserName = errors.New("Invalid User Name")
	ErrInvalidPassword = errors.New("Invalid Password")
	ErrInvalidEmail    = errors.New("Invalid Email Address")
)

// Gravatar URL pattern
var GravatarPattern = "https://gravatar.com/avatar/%s?s=%v&d=identicon"

// Simple regular expression used to verify that an email
// address matches the expected standard format.
var RegexpEmail = regexp.MustCompile(`^[^@]+@[^@.]+\.[^@.]+`)

type User struct {
	ID       int64     `meddler:"id,pk"            json:"id"`
	Email    string    `meddler:"email"            json:"email"`
	Password string    `meddler:"password"         json:"-"`
	Token    string    `meddler:"token"            json:"-"`
	Name     string    `meddler:"name"             json:"name"`
	Gravatar string    `meddler:"gravatar"         json:"gravatar"`
	Created  time.Time `meddler:"created,utctime"  json:"created"`
	Updated  time.Time `meddler:"updated,utctime"  json:"updated"`
	Admin    bool      `meddler:"admin"            json:"-"`

	// GitHub OAuth2 token for accessing public repositories.
	GithubLogin string `meddler:"github_login" json:"-"`
	GithubToken string `meddler:"github_token" json:"-"`

	// Bitbucket OAuth1.0a token and token secret.
	BitbucketLogin  string `meddler:"bitbucket_login"  json:"-"`
	BitbucketToken  string `meddler:"bitbucket_token"  json:"-"`
	BitbucketSecret string `meddler:"bitbucket_secret" json:"-"`

	GitlabToken string `meddler:"gitlab_token" json:"-"`
}

// Creates a new User from the given Name and Email.
func NewUser(name, email string) *User {
	user := User{}
	user.Name = name
	user.Token = createToken()
	user.SetEmail(email)
	return &user
}

// Returns the Gravatar Image URL.
func (u *User) Image() string      { return fmt.Sprintf(GravatarPattern, u.Gravatar, 42) }
func (u *User) ImageSmall() string { return fmt.Sprintf(GravatarPattern, u.Gravatar, 32) }
func (u *User) ImageLarge() string { return fmt.Sprintf(GravatarPattern, u.Gravatar, 160) }

// Set the email address and calculate the
// Gravatar hash.
func (u *User) SetEmail(email string) {
	u.Email = email
	u.Gravatar = createGravatar(email)
}

// Set the password and hash with bcrypt
func (u *User) SetPassword(password string) error {
	// validate the password is an appropriate size
	switch {
	case len(password) < 6:
		return ErrInvalidPassword
	case len(password) > 256:
		return ErrInvalidPassword
	}

	// convert the password to a hash
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	// update the user
	u.Password = string(b)
	return nil
}

// Validate verifies all required fields are correctly populated.
func (u *User) Validate() error {
	switch {
	case len(u.Name) == 0:
		return ErrInvalidUserName
	case len(u.Name) >= 255:
		return ErrInvalidUserName
	case len(u.Email) == 0:
		return ErrInvalidEmail
	case len(u.Email) >= 255:
		return ErrInvalidEmail
	case RegexpEmail.MatchString(u.Email) == false:
		return ErrInvalidEmail
	default:
		return nil
	}
}

// ComparePassword compares the supplied password to
// the user password stored in the database.
func (u *User) ComparePassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}
