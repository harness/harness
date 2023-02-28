// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

// ContentEncodingType describes the encoding of content.
type ContentEncodingType string

const (
	// ContentEncodingTypeUTF8 describes utf-8 encoded content.
	ContentEncodingTypeUTF8 ContentEncodingType = "utf8"

	// ContentEncodingTypeBase64 describes base64 encoded content.
	ContentEncodingTypeBase64 ContentEncodingType = "base64"
)

func (ContentEncodingType) Enum() []interface{} { return toInterfaceSlice(contentEncodingTypes) }

var contentEncodingTypes = sortEnum([]ContentEncodingType{
	ContentEncodingTypeUTF8,
	ContentEncodingTypeBase64,
})
