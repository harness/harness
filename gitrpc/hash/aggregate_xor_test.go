// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package hash

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	value1 = "refs/heads/abcd:1234"
	value2 = "refs/heads/zyxw:9876"
)

var (
	hashValueEmpty, _ = hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
	hashValue1, _     = hex.DecodeString("3a00e4f6f30e7eef599350b1bc19e1469bf5c6b26c3d93839d53547f0a61060d")
	hashValue2, _     = hex.DecodeString("10111069c3abe9cec02f6bada1e1ab4233d04c7b1d4eb80f05ca2b851c3ba89d")
	hashValue1And2, _ = hex.DecodeString("2a11f49f30a5972199bc3b1c1df84a04a8258ac971732b8c98997ffa165aae90")
)

func TestXORAggregator_Empty(t *testing.T) {
	xor, _ := New(TypeSHA256, AggregationTypeXOR)

	res, err := xor.Hash(SourceFromSlice([][]byte{}))
	require.NoError(t, err, "failed to hash value1")
	require.EqualValues(t, hashValueEmpty, res)
}

func TestXORAggregator_Single(t *testing.T) {
	xor, _ := New(TypeSHA256, AggregationTypeXOR)

	res, err := xor.Hash(SourceFromSlice([][]byte{[]byte(value1)}))
	require.NoError(t, err, "failed to hash value1")
	require.EqualValues(t, hashValue1, res)

	res, err = xor.Hash(SourceFromSlice([][]byte{[]byte(value2)}))
	require.NoError(t, err, "failed to hash value2")
	require.EqualValues(t, hashValue2, res)
}

func TestXORAggregator_Multi(t *testing.T) {
	xor, _ := New(TypeSHA256, AggregationTypeXOR)

	res, err := xor.Hash(SourceFromSlice([][]byte{[]byte(value1), []byte(value2)}))
	require.NoError(t, err, "failed to hash value1 and value2")
	require.EqualValues(t, hashValue1And2, res)
}

func TestXORAggregator_MultiSame(t *testing.T) {
	xor, _ := New(TypeSHA256, AggregationTypeXOR)

	res, err := xor.Hash(SourceFromSlice([][]byte{[]byte(value1), []byte(value1)}))
	require.NoError(t, err, "failed to hash value1 and value1")
	require.EqualValues(t, hashValueEmpty, res)

	res, err = xor.Hash(SourceFromSlice([][]byte{[]byte(value2), []byte(value2)}))
	require.NoError(t, err, "failed to hash value2 and value2")
	require.EqualValues(t, hashValueEmpty, res)

	res, err = xor.Hash(SourceFromSlice([][]byte{[]byte(value1), []byte(value2), []byte(value2)}))
	require.NoError(t, err, "failed to hash value1 and value2 and value2")
	require.EqualValues(t, hashValue1, res)

	res, err = xor.Hash(SourceFromSlice([][]byte{[]byte(value1), []byte(value1), []byte(value2)}))
	require.NoError(t, err, "failed to hash value1 and value1 and value2")
	require.EqualValues(t, hashValue2, res)

	res, err = xor.Hash(SourceFromSlice([][]byte{[]byte(value1), []byte(value2), []byte(value1)}))
	require.NoError(t, err, "failed to hash value1 and value2 and value1")
	require.EqualValues(t, hashValue2, res)
}

func TestAppendMulti(t *testing.T) {
	xor, _ := New(TypeSHA256, AggregationTypeXOR)

	res, err := xor.Append(hashValue1, SourceFromSlice([][]byte{[]byte(value2)}))
	require.NoError(t, err, "failed to append value2")
	require.EqualValues(t, hashValue1And2, res)

	res, err = xor.Append(hashValue2, SourceFromSlice([][]byte{[]byte(value1)}))
	require.NoError(t, err, "failed to append value1")
	require.EqualValues(t, hashValue1And2, res)

	res, err = xor.Append(hashValue2, SourceFromSlice([][]byte{[]byte(value1)}))
	require.NoError(t, err, "failed to append value1")
	require.EqualValues(t, hashValue1And2, res)
}

func TestAppendSame(t *testing.T) {
	xor, _ := New(TypeSHA256, AggregationTypeXOR)

	res, err := xor.Append(hashValue1, SourceFromSlice([][]byte{[]byte(value1)}))
	require.NoError(t, err, "failed to append value1")
	require.EqualValues(t, hashValueEmpty, res)

	res, err = xor.Append(hashValue2, SourceFromSlice([][]byte{[]byte(value2)}))
	require.NoError(t, err, "failed to append value2")
	require.EqualValues(t, hashValueEmpty, res)

	res, err = xor.Append(hashValue1, SourceFromSlice([][]byte{[]byte(value2), []byte(value2)}))
	require.NoError(t, err, "failed to append value2 and value2")
	require.EqualValues(t, hashValue1, res)

	res, err = xor.Append(hashValue1, SourceFromSlice([][]byte{[]byte(value1), []byte(value2)}))
	require.NoError(t, err, "failed to append value1 and value2")
	require.EqualValues(t, hashValue2, res)

	res, err = xor.Append(hashValue1, SourceFromSlice([][]byte{[]byte(value2), []byte(value1)}))
	require.NoError(t, err, "failed to append value2 and value1")
	require.EqualValues(t, hashValue2, res)
}
