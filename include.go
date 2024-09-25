package harness

import (
	"embed"
)

//go:embed */**
var _ embed.FS
