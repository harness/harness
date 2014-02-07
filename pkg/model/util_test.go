package model

import (
	"testing"
)

func Test_createSlug(t *testing.T) {
	strings := map[string]string{
		"John Tyler":        "john-tyler",
		"James K. Polk":     "james-k-polk",
		"George H. W. Bush": "george-h-w-bush",
		"François Hollande": "francois-hollande",
		"dàzǒngtǒng":        "dazongtong",
		"大總統":               "大總統",
	}

	for k, v := range strings {
		if slug := createSlug(k); slug != v {
			t.Errorf("Expected Slug %s for string %s, got %s", v, k, slug)
		}
	}
}
