package web

import "time"

// Options defines website handler options.
type Options struct {
	sync time.Duration
	path string
	docs string
}

// Option configures the website handler.
type Option func(*Options)

// WithSync configures the website hanlder with the duration value
// used to determine if the user account requires synchronization.
func WithSync(d time.Duration) Option {
	return func(o *Options) {
		o.sync = d
	}
}

// WithDir configures the website hanlder with the directory value
// used to serve the website from the local filesystem.
func WithDir(s string) Option {
	return func(o *Options) {
		o.path = s
	}
}

// WithDocs configures the website hanlder with the documentation
// website address, which should be included in the user interface.
func WithDocs(s string) Option {
	return func(o *Options) {
		o.docs = s
	}
}
