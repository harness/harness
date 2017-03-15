package linter

// Option configures a linting option.
type Option func(*Linter)

// WithTrusted adds the trusted option to the linter.
func WithTrusted(trusted bool) Option {
	return func(linter *Linter) {
		linter.trusted = trusted
	}
}
