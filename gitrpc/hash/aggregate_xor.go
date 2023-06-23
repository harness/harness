// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
