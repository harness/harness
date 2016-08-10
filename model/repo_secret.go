package model

type RepoSecret struct {
	// the id for this secret.
	ID int64 `json:"id" meddler:"secret_id,pk"`

	// the foreign key for this secret.
	RepoID int64 `json:"-" meddler:"secret_repo_id"`

	// the name of the secret which will be used as the environment variable
	// name at runtime.
	Name string `json:"name" meddler:"secret_name"`

	// the value of the secret which will be provided to the runtime environment
	// as a named environment variable.
	Value string `json:"value" meddler:"secret_value"`

	// the secret is restricted to this list of images.
	Images []string `json:"image,omitempty" meddler:"secret_images,json"`

	// the secret is restricted to this list of events.
	Events []string `json:"event,omitempty" meddler:"secret_events,json"`
}

// Secret transforms a repo secret into a simple secret.
func (s *RepoSecret) Secret() *Secret {
	return &Secret{
		Name:   s.Name,
		Value:  s.Value,
		Images: s.Images,
		Events: s.Events,
	}
}

// Clone provides a repo secrets clone without the value.
func (s *RepoSecret) Clone() *RepoSecret {
	return &RepoSecret{
		ID:     s.ID,
		Name:   s.Name,
		Images: s.Images,
		Events: s.Events,
	}
}

// Validate validates the required fields and formats.
func (s *RepoSecret) Validate() error {
	return nil
}
