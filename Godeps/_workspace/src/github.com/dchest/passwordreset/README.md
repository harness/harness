Package passwordreset
=====================

	import "github.com/dchest/passwordreset"

Package passwordreset implements creation and verification of secure tokens
useful for implementation of "reset forgotten password" feature in web
applications.

This package generates and verifies signed one-time tokens that can be
embedded in a link sent to users when they initiate the password reset
procedure. When a user changes their password, or when the expiry time
passes, the token becomes invalid.

Secure token format:

	expiration time || login || signature

where expiration time is the number of seconds since Unix epoch UTC
indicating when this token must expire (4 bytes, big-endian, uint32), login
is a byte string of arbitrary length (at least 1 byte, not null-terminated),
and signature is 32 bytes of HMAC-SHA256(expiration_time || login, k), where
k = HMAC-SHA256(expiration_time || login, userkey), where userkey =
HMAC-SHA256(password value, secret key), where password value is any piece
of information derived from user's password, which will change once the user
changes their password (for example, a hash of the password), and secret key
is an application-specific secret key.

Password value is used to make tokens one-time, that is, once a user changes
their password, the token which they used to do a reset, becomes invalid.

Usage example:

Your application must have a strong secret key for password reset purposes.
This key will be used to generate and verify password reset tokens.  (If you
already have a secret key, for example, for authcookie package, it's better
not to reuse it, just use a different one.)

	secret := []byte("assume we have a long randomly generated secret key here")

Create a function that will query your users database and return some
password-related value for the given login.  A password-related value means
some value that will change once a user changes their password, for example:
a password hash, a random salt used to generate it, or time of password
creation.  This value, mixed with app-specific secret key, will be used as a
key for password reset token, thus it will be kept secret.

	func getPasswordHash(login string) ([]byte, error) {
		// return password hash for the login,
		// or an error if there's no such user
	}

When a user initiates password reset (by entering their login, and maybe
answering a secret question), generate a reset token:

	pwdval, err := getPasswordHash(login)
	if err != nil {
		// user doesn't exists, abort
		return
	}
	// Generate reset token that expires in 12 hours
	token := passwordreset.NewToken(login, 12 * time.Hour, pwdval, secret)

Send a link with this token to the user by email, for example:
https://www.example.com/reset?token=Talo3mRjaGVzdITUAGOXYZwCMq7EtHfYH4ILcBgKaoWXDHTJOIlBUfcr

Once a user clicks this link, read a token from it, then verify this token
by passing it to VerifyToken function along with the getPasswordHash
function, and an app-specific secret key:

	login, err := passwordreset.VerifyToken(token, getPasswordHash, secret)
	if err != nil {
		// verification failed, don't allow password reset
		return
	}
	// OK, reset password for login (e.g. allow to change it)

If verification succeeded, allow to change password for the returned login.


Variables
---------

	var (
	    ErrMalformedToken = errors.New("malformed token")
	    ErrExpiredToken   = errors.New("token expired")
	    ErrWrongSignature = errors.New("wrong token signature")
	)


	var MinTokenLength = authcookie.MinLength

MinTokenLength is the minimum allowed length of token string.

It is useful for avoiding DoS attacks with very long tokens: before passing
a token to VerifyToken function, check that it has length less than [the
maximum login length allowed in your application] + MinTokenLength.


Functions
---------

### func NewToken

	func NewToken(login string, dur time.Duration, pwdval, secret []byte) string
	
NewToken returns a new password reset token for the given login, which expires
after the given time duration since now, signed by the key generated from the
given password value (which can be any value that will be changed once a user
resets their password, such as password hash or salt used to generate it), and
the given secret key.

### func VerifyToken

	func VerifyToken(token string, pwdvalFn func(string) ([]byte, error), secret []byte) (login string, err os.Error)
	
VerifyToken verifies the given token with the password value returned by the
given function and the given secret key, and returns login extracted from
the valid token. If the token is not valid, the function returns an error.

Function pwdvalFn must return the current password value for the login it
receives in arguments, or an error. If it returns an error, VerifyToken
returns the same error.

