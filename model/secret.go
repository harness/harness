package model

import (
	"path/filepath"
)

type Secret struct {
	// the name of the secret which will be used as the environment variable
	// name at runtime.
	Name string `json:"name"`

	// the value of the secret which will be provided to the runtime environment
	// as a named environment variable.
	Value string `json:"value"`

	// the secret is restricted to this list of images.
	Images []string `json:"image,omitempty"`

	// the secret is restricted to this list of events.
	Events []string `json:"event,omitempty"`

	// whether the secret requires verification
	SkipVerify bool `json:"skip_verify"`

	// whether the secret should be concealed in the build log
	Conceal bool `json:"conceal"`
}

// Match returns true if an image and event match the restricted list.
func (s *Secret) Match(image, event string) bool {
	return s.MatchImage(image) && s.MatchEvent(event)
}

// MatchImage returns true if an image matches the restricted list.
func (s *Secret) MatchImage(image string) bool {
	for _, pattern := range s.Images {
		if match, _ := filepath.Match(pattern, image); match {
			return true
		} else if pattern == "*" {
			return true
		}
	}
	return false
}

// MatchEvent returns true if an event matches the restricted list.
func (s *Secret) MatchEvent(event string) bool {
	for _, pattern := range s.Events {
		if match, _ := filepath.Match(pattern, event); match {
			return true
		}
	}
	return false
}

// Validate validates the required fields and formats.
func (s *Secret) Validate() error {
	return nil
}
