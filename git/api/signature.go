package api

import (
	"fmt"
	"github.com/harness/gitness/errors"
	"time"
)

// Signature represents the Author or Committer information.
type Signature struct {
	Identity Identity
	// When is the timestamp of the Signature.
	When time.Time
}

// Decode decodes a byte array representing a signature to signature
func (s *Signature) Decode(b []byte) {
	sig, _ := NewSignatureFromCommitLine(b)
	s.Identity.Email = sig.Identity.Email
	s.Identity.Name = sig.Identity.Name
	s.When = sig.When
}

func (s *Signature) String() string {
	return fmt.Sprintf("%s <%s>", s.Identity.Name, s.Identity.Email)
}

type Identity struct {
	Name  string
	Email string
}

func (i Identity) String() string {
	return fmt.Sprintf("%s <%s>", i.Name, i.Email)
}

func (i *Identity) Validate() error {
	if i.Name == "" {
		return errors.InvalidArgument("identity name is mandatory")
	}

	if i.Email == "" {
		return errors.InvalidArgument("identity email is mandatory")
	}

	return nil
}
