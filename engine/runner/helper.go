package runner

import (
	"encoding/json"
	"io/ioutil"
)

// Parse parses a raw file containing a JSON encoded format of an intermediate
// representation of the pipeline.
func Parse(data []byte) (*Spec, error) {
	v := &Spec{}
	err := json.Unmarshal(data, v)
	return v, err
}

// ParseFile parses a file containing a JSON encoded format of an intermediate
// representation of the pipeline.
func ParseFile(filename string) (*Spec, error) {
	out, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return Parse(out)
}
