package resources

import "embed"

var (
	//go:embed gitignore
	Gitignore embed.FS

	//go:embed licence
	Licence embed.FS
)
