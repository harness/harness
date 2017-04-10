package compiler

import (
	"github.com/docker/docker/reference"
)

// trimImage returns the short image name without tag.
func trimImage(name string) string {
	ref, err := reference.ParseNamed(name)
	if err != nil {
		return name
	}
	return reference.TrimNamed(ref).String()
}

// expandImage returns the fully qualified image name.
func expandImage(name string) string {
	ref, err := reference.ParseNamed(name)
	if err != nil {
		return name
	}
	return reference.WithDefaultTag(ref).String()
}

// matchImage returns true if the image name matches
// an image in the list. Note the image tag is not used
// in the matching logic.
func matchImage(from string, to ...string) bool {
	from = trimImage(from)
	for _, match := range to {
		if from == trimImage(match) {
			return true
		}
	}
	return false
}

// matchHostname returns true if the image hostname
// matches the specified hostname.
func matchHostname(image, hostname string) bool {
	ref, err := reference.ParseNamed(image)
	if err != nil {
		return false
	}
	return ref.Hostname() == hostname
}
