package pipeline

import (
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/cncd/pipeline/pipeline/backend"
)

// Parse parses the pipeline config from an io.Reader.
func Parse(r io.Reader) (*backend.Config, error) {
	cfg := new(backend.Config)
	err := json.NewDecoder(r).Decode(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// ParseFile parses the pipeline config from a file.
func ParseFile(path string) (*backend.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// ParseString parses the pipeline config from a string.
func ParseString(s string) (*backend.Config, error) {
	return Parse(
		strings.NewReader(s),
	)
}
