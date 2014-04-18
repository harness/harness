package notify

import "github.com/drone/drone/pkg/mail"

type Email struct {
	Recipients []string `yaml:"recipients,omitempty"`
	Success    string   `yaml:"on_success"`
	Failure    string   `yaml:"on_failure"`
}

// Send will send an email, either success or failure,
// based on the Commit Status.
func (e *Email) Send(context *Context) error {
	switch {
	case context.Commit.Status == "Success" && e.Success != "never":
		return e.sendSuccess(context)
	case context.Commit.Status == "Failure" && e.Failure != "never":
		return e.sendFailure(context)
	}

	return nil
}

// sendFailure sends email notifications to the list of
// recipients indicating the build failed.
func (e *Email) sendFailure(context *Context) error {
	// loop through and email recipients
	for _, email := range e.Recipients {
		if err := mail.SendFailure(context.Repo.Name, context.Commit.HashShort(), email, context); err != nil {
			return err
		}
	}
	return nil
}

// sendSuccess sends email notifications to the list of
// recipients indicating the build was a success.
func (e *Email) sendSuccess(context *Context) error {
	// loop through and email recipients
	for _, email := range e.Recipients {
		if err := mail.SendSuccess(context.Repo.Name, context.Commit.HashShort(), email, context); err != nil {
			return err
		}
	}
	return nil
}
