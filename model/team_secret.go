package model

type TeamSecret struct {
	// the id for this secret.
	ID int64 `json:"id" meddler:"team_secret_id,pk"`

	// the foreign key for this secret.
	Key string `json:"-" meddler:"team_secret_key"`

	// the name of the secret which will be used as the environment variable
	// name at runtime.
	Name string `json:"name" meddler:"team_secret_name"`

	// the value of the secret which will be provided to the runtime environment
	// as a named environment variable.
	Value string `json:"value" meddler:"team_secret_value"`

	// the secret is restricted to this list of images.
	Images []string `json:"image,omitempty" meddler:"team_secret_images,json"`

	// the secret is restricted to this list of events.
	Events []string `json:"event,omitempty" meddler:"team_secret_events,json"`

	// whether the secret requires verification
	SkipVerify bool `json:"skip_verify" meddler:"team_secret_skip_verify"`

	// whether the secret should be concealed in the build log
	Conceal bool `json:"conceal" meddler:"team_secret_conceal"`
}

// Secret transforms a repo secret into a simple secret.
func (s *TeamSecret) Secret() *Secret {
	return &Secret{
		Name:       s.Name,
		Value:      s.Value,
		Images:     s.Images,
		Events:     s.Events,
		SkipVerify: s.SkipVerify,
		Conceal:    s.Conceal,
	}
}

// Clone provides a repo secrets clone without the value.
func (s *TeamSecret) Clone() *TeamSecret {
	return &TeamSecret{
		ID:         s.ID,
		Name:       s.Name,
		Images:     s.Images,
		Events:     s.Events,
		SkipVerify: s.SkipVerify,
		Conceal:    s.Conceal,
	}
}

// Validate validates the required fields and formats.
func (s *TeamSecret) Validate() error {
	return nil
}
