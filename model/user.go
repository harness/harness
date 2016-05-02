package model

// User represents a registered user.
//
// swagger:model user
type User struct {
	// the id for this user.
	//
	// required: true
	ID int64 `json:"id" meddler:"user_id,pk"`

	// Login is the username for this user.
	//
	// required: true
	Login string `json:"login"  meddler:"user_login"`

	// Token is the oauth2 token.
	Token string `json:"-"  meddler:"user_token"`

	// Secret is the oauth2 token secret.
	Secret string `json:"-" meddler:"user_secret"`

	// Expiry is the token and secret expriation timestamp.
	Expiry int64 `json:"-" meddler:"user_expiry"`

	// Email is the email address for this user.
	//
	// required: true
	Email string `json:"email" meddler:"user_email"`

	// the avatar url for this user.
	Avatar string `json:"avatar_url" meddler:"user_avatar"`

	// Activate indicates the user is active in the system.
	Active bool `json:"active" meddler:"user_active"`

	// Admin indicates the user is a system administrator.
	//
	// NOTE: This is sourced from the DRONE_ADMINS environment variable and is no
	// longer persisted in the database.
	Admin bool `json:"admin,omitempty" meddler:"-"`

	// Hash is a unique token used to sign tokens.
	Hash string `json:"-" meddler:"user_hash"`

	// DEPRECATED Admin indicates the user is a system administrator.
	XAdmin bool `json:"-" meddler:"user_admin"`
}
