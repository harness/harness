// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hash

import (
	"crypto/sha256"
	"fmt"
	"hash"
)

// Type defines the different types of hashing that are supported.
// NOTE: package doesn't take hash.Hash as input to allow external
// callers to both calculate the hash themselves using this package, or call git to calculate the hash,
// without the caller having to know internal details on what hash.Hash implementation is used.
type Type string

const (
	// TypeSHA256 represents the sha256 hashing method.
	TypeSHA256 Type = "sha256"
)

// AggregationType defines the different types of hash aggregation types available.
type AggregationType string

const (
	// AggregationTypeXOR aggregates a list of hashes using XOR.
	// It provides commutative, self-inverse hashing, e.g.:
	// - order of elements doesn't matter
	// - two equal elements having the same hash cancel each other out.
	AggregationTypeXOR AggregationType = "xor"
)

// Aggregator is an abstraction of a component that aggregates a list of values into a single hash.
type Aggregator interface {
	// Empty returns the empty hash of an aggregator. It is returned when hashing an empty Source
	// or hashing a Source who's hash is equal to an empty source. Furthermore, the following is always true:
	// `Hash(s) == Append(Empty(), s)` FOR ALL sources s.
	Empty() []byte

	// Hash returns the hash aggregated over all elements of the provided source.
	Hash(source Source) ([]byte, error)

	// Append returns the hash that results when aggregating the existing hash
	// with the hashes of all elements of the provided source.
	// IMPORTANT: size of existing hash has to be compatible (Empty() can be used for reference).
	Append(hash []byte, source Source) ([]byte, error)
}

// New returns a new aggregator for the given aggregation and hashing type.
func New(t Type, at AggregationType) (Aggregator, error) {
	// get hash factory method to ensure we fail on object creation in case of invalid Type.
	hfm, hashSize, err := getHashFactoryMethod(t)
	if err != nil {
		return nil, err
	}

	switch at {
	case AggregationTypeXOR:
		return &xorAggregator{
			hfm:      hfm,
			hashSize: hashSize,
		}, nil
	default:
		return nil, fmt.Errorf("unknown aggregation type '%s'", at)
	}
}

// hashFactoryMethod returns a hash.Hash implementation.
type hashFactoryMethod func() hash.Hash

// getHashFactoryMethod returns the hash factory method together with the length of its generated hashes.
// NOTE: the length is needed to ensure hashes of an empty source are similar to hashes of `a <XOR> a`.
func getHashFactoryMethod(t Type) (hashFactoryMethod, int, error) {
	switch t {
	case TypeSHA256:
		return sha256.New, sha256.Size, nil
	default:
		return nil, -1, fmt.Errorf("unknown hash type '%s'", t)
	}
}
