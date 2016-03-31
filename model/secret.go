package model

type Secret struct {
	// the id for this secret.
	ID int64 `json:"id" meddler:"secret_id,pk"`

	// the foreign key for this secret.
	RepoID int64 `json:"-" meddler:"secret_repo_id"`

	// the name of the secret which will be used as the
	// environment variable name at runtime.
	Name string `json:"name" meddler:"secret_name"`

	// the value of the secret which will be provided to
	// the runtime environment as a named environment variable.
	Value string `json:"value" meddler:"secret_value"`

	// the secret is restricted to this list of images.
	Images []string `json:"image,omitempty" meddler:"secret_images,json"`

	// the secret is restricted to this list of events.
	Events []string `json:"event,omitempty" meddler:"secret_events,json"`
}

func (s *Secret) Validate() error {
	return nil
}
