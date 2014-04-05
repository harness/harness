Package authcookie
=====================

	import "github.com/dchest/authcookie"

Package authcookie implements creation and verification of signed
authentication cookies.

Cookie is a Base64 encoded (using URLEncoding, from RFC 4648) string, which
consists of concatenation of expiration time, login, and signature:

	expiration time || login || signature

where expiration time is the number of seconds since Unix epoch UTC
indicating when this cookie must expire (4 bytes, big-endian, uint32), login
is a byte string of arbitrary length (at least 1 byte, not null-terminated),
and signature is 32 bytes of HMAC-SHA256(expiration_time || login, k), where
k = HMAC-SHA256(expiration_time || login, secret key).

Example:

	secret := []byte("my secret key")

	// Generate cookie valid for 24 hours for user "bender"
	cookie := authcookie.NewSinceNow("bender", 24 * time.Hour, secret)

	// cookie is now:
	// Tajh02JlbmRlcskYMxowgwPj5QZ94jaxhDoh3n0Yp4hgGtUpeO0YbMTY
	// send it to user's browser..

	// To authenticate a user later, receive cookie and:
	login := authcookie.Login(cookie, secret)
	if login != "" {
		// access for login granted
	} else {
		// access denied
	}

Note that login and expiration time are not encrypted, they are only signed
and Base64 encoded.


Variables
---------

	var (
	    ErrMalformedCookie = errors.New("malformed cookie")
	    ErrWrongSignature  = errors.New("wrong cookie signature")
	)


	var MinLength = base64.URLEncoding.EncodedLen(decodedMinLength)

MinLength is the minimum allowed length of cookie string.

It is useful for avoiding DoS attacks with too long cookies: before passing
a cookie to Parse or Login functions, check that it has length less than the
[maximum login length allowed in your application] + MinLength.


Functions
---------

### func Login

	func Login(cookie string, secret []byte) string
	
Login returns a valid login extracted from the given cookie and verified
using the given secret key.  If verification fails or the cookie expired,
the function returns an empty string.

### func New

	func New(login string, expires time.Time, secret []byte) string
	
New returns a signed authentication cookie for the given login,
expiration time, and secret key.
If the login is empty, the function returns an empty string.

### func NewSinceNow

	func NewSinceNow(login string, dur time.Duration, secret []byte) string
	
NewSinceNow returns a signed authetication cookie for the given login,
duration time since current time, and secret key.

### func Parse

	func Parse(cookie string, secret []byte) (login string, expires time.Time, err error)
	
Parse verifies the given cookie with the secret key and returns login and
expiration time extracted from the cookie. If the cookie fails verification
or is not well-formed, the function returns an error.

Callers must:

1. Check for the returned error and deny access if it's present.

2. Check the returned expiration time and deny access if it's in the past.
