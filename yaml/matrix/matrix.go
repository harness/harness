package matrix

import (
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	limitTags = 10
	limitAxis = 25
)

// Matrix represents the build matrix.
type Matrix map[string][]string

// Exclude represents the exclusion matrix.
type Exclude map[string][]string

// Axis represents a single permutation of entries
// from the build matrix.
type Axis map[string]string

// String returns a string representation of an Axis as
// a comma-separated list of environment variables.
func (a Axis) String() string {
	var envs []string
	for k, v := range a {
		envs = append(envs, k+"="+v)
	}
	return strings.Join(envs, " ")
}

// Parse parses the Matrix section of the yaml file and
// returns a list of axis.
func Parse(raw string) ([]Axis, error) {
	matrix, exclude, err := parseMatrix(raw)
	if err != nil {
		return nil, err
	}

	// if not a matrix build return an array
	// with just the single axis.
	if len(matrix) == 0 {
		return nil, nil
	}

	return Subtract(Perm(matrix), Perm(exclude)), nil
}

// Perm calculates the permutations for a matrix
//
// Note that this method will cap the number of permutations
// to 25 to prevent an overly expensive calculation.
func Perm(matrix map[string][]string) []Axis {
	// calculate number of permutations and
	// extract the list of tags
	// (ie go_version, redis_version, etc)
	var perm int
	var tags []string
	for k, v := range matrix {
		perm *= len(v)
		if perm == 0 {
			perm = len(v)
		}
		tags = append(tags, k)
	}

	// structure to hold the transformed
	// result set
	axisList := []Axis{}

	// for each axis calculate the uniqe
	// set of values that should be used.
	for p := 0; p < perm; p++ {
		axis := map[string]string{}
		decr := perm
		for i, tag := range tags {
			elems := matrix[tag]
			decr = decr / len(elems)
			elem := p / decr % len(elems)
			axis[tag] = elems[elem]

			// enforce a maximum number of tags
			// in the build matrix.
			if i > limitTags {
				break
			}
		}

		// append to the list of axis.
		axisList = append(axisList, axis)

		// enforce a maximum number of axis
		// that should be calculated.
		if p > limitAxis {
			break
		}
	}

	return axisList
}

// Subtract two matrix permutations and return axes - excludes
func Subtract(axes []Axis, excludes []Axis) []Axis {
        // compute the indices of axes to be removed from matrix:
        // we remove permuatations containing all excluded key-value pairs,
        // i.e., the logical operation among the exclude-keys is AND while
        // among the exlude-values is OR, see also the test
        var r []int
        for _, excl := range excludes {
          for i, axis := range axes {
            in := make(map[string]bool)
            for ek, ev := range excl {
              for ak, av := range axis {
                if ek == ak && ev == av {
                  in[ek] = true
                }
              }
            }
            if len(in) == len(excl) {
              r = append(r,i)
            }
          }
        }
        // subtract
        w := 0
        outer: for i, x := range axes {
          for _, id := range r {
            if id == i {
              continue outer
            }
          }
          axes[w] = x
          w++
        }
        axes = axes[:w]
        return axes
}

// helper function to parse the Matrix data from
// the raw yaml file.
func parseMatrix(raw string) (Matrix, Exclude, error) {
	data := struct {
		Matrix map[string][]string
		Exclude map[string][]string
	}{}
	err := yaml.Unmarshal([]byte(raw), &data)
	return data.Matrix, data.Exclude, err
}
