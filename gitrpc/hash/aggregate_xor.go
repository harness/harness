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
	"errors"
	"fmt"
	"io"
)

// xorAggregator is an implementation of the Aggregator interface
// that aggregates hashes by XORing them.
type xorAggregator struct {
	hfm      hashFactoryMethod
	hashSize int
}

func (a *xorAggregator) Empty() []byte {
	return make([]byte, a.hashSize)
}

func (a *xorAggregator) Hash(source Source) ([]byte, error) {
	return a.append(a.Empty(), source)
}

func (a *xorAggregator) Append(hash []byte, source Source) ([]byte, error) {
	// copy value to ensure we don't modify the original hash array
	hashCopy := make([]byte, len(hash))
	copy(hashCopy, hash)

	return a.append(hashCopy, source)
}

func (a *xorAggregator) append(hash []byte, source Source) ([]byte, error) {
	if len(hash) != a.hashSize {
		return nil, fmt.Errorf(
			"hash is of invalid length %d, aggregator works with hashes of length %d",
			len(hash),
			a.hashSize,
		)
	}
	// create new hasher to allow asynchronous usage
	hasher := a.hfm()

	v, err := source.Next()
	for err == nil {
		// calculate hash of the value
		hasher.Reset()
		hasher.Write(v)
		vHash := hasher.Sum(nil)

		// combine the hash with the current hash
		hash = xorInPlace(hash, vHash)

		v, err = source.Next()
	}
	if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("failed getting the next element from source: %w", err)
	}

	return hash, nil
}

// xorInPlace XORs the provided byte arrays in place.
// If one slice is shorter, 0s will be used as replacement elements.
// WARNING: The method will taint the passed arrays!
func xorInPlace(a, b []byte) []byte {
	// ensure len(a) >= len(b)
	if len(b) > len(a) {
		a, b = b, a
	}

	// xor all values from a with b (or 0)
	for i := 0; i < len(a); i++ {
		var bi byte
		if i < len(b) {
			bi = b[i]
		}

		a[i] ^= bi
	}

	return a
}
