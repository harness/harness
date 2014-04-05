// Package authcookie implements creation and verification of signed
// authentication cookies.
//
// Cookie is a Base64 encoded (using URLEncoding, from RFC 4648) string, which
// consists of concatenation of expiration time, login, and signature:
//
// 	expiration time || login || signature
//
// where expiration time is the number of seconds since Unix epoch UTC
// indicating when this cookie must expire (4 bytes, big-endian, uint32), login
// is a byte string of arbitrary length (at least 1 byte, not null-terminated),
// and signature is 32 bytes of HMAC-SHA256(expiration_time || login, k), where
// k = HMAC-SHA256(expiration_time || login, secret key).
//
// Example:
//
//	secret := []byte("my secret key")
//
//	// Generate cookie valid for 24 hours for user "bender"
//	cookie := authcookie.NewSinceNow("bender", 24 * time.Hour, secret)
//
//	// cookie is now:
//	// Tajh02JlbmRlcskYMxowgwPj5QZ94jaxhDoh3n0Yp4hgGtUpeO0YbMTY
//	// send it to user's browser..
//	
//	// To authenticate a user later, receive cookie and:
//	login := authcookie.Login(cookie, secret)
//	if login != "" {
//		// access for login granted
//	} else {
//		// access denied
//	}
//
// Note that login and expiration time are not encrypted, they are only signed
// and Base64 encoded.
//
// For safety, the maximum length of base64-decoded cookie is limited to 1024
// bytes.
package authcookie

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"time"
)

const (
	decodedMinLength = 4 /*expiration*/ + 1 /*login*/ + 32 /*signature*/
	decodedMaxLength = 1024                                /* maximum decoded length, for safety */
)

// MinLength is the minimum allowed length of cookie string.
//
// It is useful for avoiding DoS attacks with too long cookies: before passing
// a cookie to Parse or Login functions, check that it has length less than the
// [maximum login length allowed in your application] + MinLength.
var MinLength = base64.URLEncoding.EncodedLen(decodedMinLength)

func getSignature(b []byte, secret []byte) []byte {
	keym := hmac.New(sha256.New, secret)
	keym.Write(b)
	m := hmac.New(sha256.New, keym.Sum(nil))
	m.Write(b)
	return m.Sum(nil)
}

var (
	ErrMalformedCookie = errors.New("malformed cookie")
	ErrWrongSignature  = errors.New("wrong cookie signature")
)

// New returns a signed authentication cookie for the given login,
// expiration time, and secret key.
// If the login is empty, the function returns an empty string.
func New(login string, expires time.Time, secret []byte) string {
	if login == "" {
		return ""
	}
	llen := len(login)
	b := make([]byte, llen+4+32)
	// Put expiration time.
	binary.BigEndian.PutUint32(b, uint32(expires.Unix()))
	// Put login.
	copy(b[4:], []byte(login))
	// Calculate and put signature.
	sig := getSignature([]byte(b[:4+llen]), secret)
	copy(b[4+llen:], sig)
	// Base64-encode.
	return base64.URLEncoding.EncodeToString(b)
}

// NewSinceNow returns a signed authetication cookie for the given login,
// duration since current time, and secret key.
func NewSinceNow(login string, dur time.Duration, secret []byte) string {
	return New(login, time.Now().Add(dur), secret)
}

// Parse verifies the given cookie with the secret key and returns login and
// expiration time extracted from the cookie. If the cookie fails verification
// or is not well-formed, the function returns an error.
//
// Callers must: 
//
// 1. Check for the returned error and deny access if it's present.
//
// 2. Check the returned expiration time and deny access if it's in the past.
//
func Parse(cookie string, secret []byte) (login string, expires time.Time, err error) {
	blen := base64.URLEncoding.DecodedLen(len(cookie))
	// Avoid allocation if cookie is too short or too long.
	if blen < decodedMinLength || blen > decodedMaxLength {
		err = ErrMalformedCookie
		return
	}
	b, err := base64.URLEncoding.DecodeString(cookie)
	if err != nil {
		return
	}
	// Decoded length may be different from max length, which
	// we allocated, so check it, and set new length for b.
	blen = len(b)
	if blen < decodedMinLength {
		err = ErrMalformedCookie
		return
	}
	b = b[:blen]

	sig := b[blen-32:]
	data := b[:blen-32]

	realSig := getSignature(data, secret)
	if subtle.ConstantTimeCompare(realSig, sig) != 1 {
		err = ErrWrongSignature
		return
	}
	expires = time.Unix(int64(binary.BigEndian.Uint32(data[:4])), 0)
	login = string(data[4:])
	return
}

// Login returns a valid login extracted from the given cookie and verified
// using the given secret key.  If verification fails or the cookie expired,
// the function returns an empty string.
func Login(cookie string, secret []byte) string {
	l, exp, err := Parse(cookie, secret)
	if err != nil || exp.Before(time.Now()) {
		return ""
	}
	return l
}
