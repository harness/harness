package gravatar

import (
	"crypto/md5"
	"fmt"
	"strings"
)

// helper function to create a Gravatar Hash
// for the given Email address.
func Generate(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	hash := md5.New()
	hash.Write([]byte(email))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
