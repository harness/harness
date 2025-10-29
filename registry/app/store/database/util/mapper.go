//  Copyright 2023 Harness, Inc.
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

package util

import (
	"encoding/hex"
	"reflect"
	"strconv"
	"strings"

	"github.com/harness/gitness/registry/utils"
)

const ID = ""
const separator = "^_"

func StringToArr(s string) []string {
	return StringToArrByDelimiter(s, separator)
}

func ArrToString(arr []string) string {
	return ArrToStringByDelimiter(arr, separator)
}

func Int64ArrToString(arr []int64) string {
	return Int64ArrToStringByDelimiter(arr, separator)
}

func StringToInt64Arr(s string) []int64 {
	return StringToInt64ArrByDelimiter(s, separator)
}

func StringToArrByDelimiter(s string, delimiter string) []string {
	var arr []string
	if utils.IsEmpty(s) {
		return arr
	}
	return strings.Split(s, delimiter)
}

func ArrToStringByDelimiter(arr []string, delimiter string) string {
	return strings.Join(arr, delimiter)
}

func Int64ArrToStringByDelimiter(arr []int64, delimiter string) string {
	var s []string
	for _, i := range arr {
		s = append(s, strconv.FormatInt(i, 10))
	}
	return strings.Join(s, delimiter)
}

func StringToInt64ArrByDelimiter(s string, delimiter string) []int64 {
	var arr []int64
	if utils.IsEmpty(s) {
		return arr
	}
	for i := range strings.SplitSeq(s, delimiter) {
		j, _ := strconv.ParseInt(i, 10, 64)
		arr = append(arr, j)
	}
	return arr
}

func GetSetDBKeys(s any, ignoreKeys ...string) string {
	keys := GetDBTagsFromStruct(s)
	filteredKeys := make([]string, 0)

keysLoop:
	for _, key := range keys {
		for _, ignoreKey := range ignoreKeys {
			if key == ignoreKey {
				continue keysLoop
			}
		}
		filteredKeys = append(filteredKeys, key+" = :"+key)
	}
	return strings.Join(filteredKeys, ", ")
}

func GetDBTagsFromStruct(s any) []string {
	var tags []string
	rt := reflect.TypeOf(s)

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag != "" {
			tags = append(tags, dbTag)
		}
	}

	return tags
}

func GetHexDecodedBytes(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

func GetHexEncodedString(b []byte) string {
	return hex.EncodeToString(b)
}
