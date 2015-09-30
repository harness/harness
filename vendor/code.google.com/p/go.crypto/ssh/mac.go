// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

// Message authentication support

import (
	"crypto/hmac"
	"crypto/sha1"
	"hash"
)

type macMode struct {
	keySize int
	new     func(key []byte) hash.Hash
}

// truncatingMAC wraps around a hash.Hash and truncates the output digest to
// a given size.
type truncatingMAC struct {
	length int
	hmac   hash.Hash
}

func (t truncatingMAC) Write(data []byte) (int, error) {
	return t.hmac.Write(data)
}

func (t truncatingMAC) Sum(in []byte) []byte {
	out := t.hmac.Sum(in)
	return out[:len(in)+t.length]
}

func (t truncatingMAC) Reset() {
	t.hmac.Reset()
}

func (t truncatingMAC) Size() int {
	return t.length
}

func (t truncatingMAC) BlockSize() int { return t.hmac.BlockSize() }

// Specifies a default set of MAC algorithms and a preference order.
// This is based on RFC 4253, section 6.4, with the removal of the
// hmac-md5 variants as they have reached the end of their useful life.
var DefaultMACOrder = []string{"hmac-sha1", "hmac-sha1-96"}

var macModes = map[string]*macMode{
	"hmac-sha1": {20, func(key []byte) hash.Hash {
		return hmac.New(sha1.New, key)
	}},
	"hmac-sha1-96": {20, func(key []byte) hash.Hash {
		return truncatingMAC{12, hmac.New(sha1.New, key)}
	}},
}
