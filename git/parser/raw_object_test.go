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

package parser

import (
	"bytes"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/publickey"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/crypto/ssh"
)

func TestObject(t *testing.T) {
	tests := []struct {
		name string
		data string
		want ObjectRaw
	}{
		{
			name: "empty",
			data: "",
			want: ObjectRaw{
				Headers: []ObjectHeader{},
				Message: "",
			},
		},
		{
			name: "no_header",
			data: "\nline1\nline2\n",
			want: ObjectRaw{
				Headers: []ObjectHeader{},
				Message: "line1\nline2\n",
			},
		},
		{
			name: "no_body",
			data: "header 1\nheader 2\n",
			want: ObjectRaw{
				Headers: []ObjectHeader{
					{Type: "header", Value: "1\n"},
					{Type: "header", Value: "2\n"},
				},
				Message: "",
			},
		},
		{
			name: "dummy_content",
			data: "header 1\nheader 2\n\nblah blah\nblah",
			want: ObjectRaw{
				Headers: []ObjectHeader{
					{Type: "header", Value: "1\n"},
					{Type: "header", Value: "2\n"},
				},
				Message: "blah blah\nblah",
			},
		},
		{
			name: "dummy_content_multiline_header",
			data: "header-simple 1\nheader-multiline line1\n line2\nheader-three blah\n\nblah blah\nblah",
			want: ObjectRaw{
				Headers: []ObjectHeader{
					{Type: "header-simple", Value: "1\n"},
					{Type: "header-multiline", Value: "line1\nline2\n"},
					{Type: "header-three", Value: "blah\n"},
				},
				Message: "blah blah\nblah",
			},
		},
		{
			name: "simple_commit",
			data: `tree a32348a67ba786383cedddccd79944992e1656b9
parent 286d9081dfddd0b95e43f98f32984b782678fc43
author Marko Gaćeša <marko.gacesa@harness.io> 1748009627 +0200
committer Marko Gaćeša <marko.gacesa@harness.io> 1748012917 +0200

Test commit
`,
			want: ObjectRaw{
				Headers: []ObjectHeader{
					{Type: "tree", Value: "a32348a67ba786383cedddccd79944992e1656b9\n"},
					{Type: "parent", Value: "286d9081dfddd0b95e43f98f32984b782678fc43\n"},
					{Type: "author", Value: "Marko Gaćeša <marko.gacesa@harness.io> 1748009627 +0200\n"},
					{Type: "committer", Value: "Marko Gaćeša <marko.gacesa@harness.io> 1748012917 +0200\n"},
				},
				Message:       "Test commit\n",
				SignedContent: nil,
				Signature:     nil,
				SignatureType: "",
			},
		},
		{
			name: "signed_commit",
			data: `tree 1e6502c1add2beb75875d261ca28abdf6e3d9091
parent a74b6a06bcf7f0d7b902af492826c20f9835a932
author Marko Gaćeša <marko.gacesa@harness.io> 1749221807 +0200
committer Marko Gaćeša <marko.gacesa@harness.io> 1749221807 +0200
gpgsig -----BEGIN SSH SIGNATURE-----
 U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgEM1i8vha2gQ/ZXHinPejh0hS4C
 x8VV1M2uwW6tglOswAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
 AAAAQDJwTh2XHcewg3MXY8hxnH1WuSAjuQPzcjaoX0Q1x923k4y2Y2hXd/cN6l+PdGo71B
 8+HfQ6jFa7/UU4cZu4QAc=
 -----END SSH SIGNATURE-----

this is a commit message
`,
			want: ObjectRaw{
				Headers: []ObjectHeader{
					{Type: "tree", Value: "1e6502c1add2beb75875d261ca28abdf6e3d9091\n"},
					{Type: "parent", Value: "a74b6a06bcf7f0d7b902af492826c20f9835a932\n"},
					{Type: "author", Value: "Marko Gaćeša <marko.gacesa@harness.io> 1749221807 +0200\n"},
					{Type: "committer", Value: "Marko Gaćeša <marko.gacesa@harness.io> 1749221807 +0200\n"},
				},
				Message: "this is a commit message\n",
				SignedContent: []byte(`tree 1e6502c1add2beb75875d261ca28abdf6e3d9091
parent a74b6a06bcf7f0d7b902af492826c20f9835a932
author Marko Gaćeša <marko.gacesa@harness.io> 1749221807 +0200
committer Marko Gaćeša <marko.gacesa@harness.io> 1749221807 +0200

this is a commit message
`),
				Signature: []byte(`-----BEGIN SSH SIGNATURE-----
U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgEM1i8vha2gQ/ZXHinPejh0hS4C
x8VV1M2uwW6tglOswAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
AAAAQDJwTh2XHcewg3MXY8hxnH1WuSAjuQPzcjaoX0Q1x923k4y2Y2hXd/cN6l+PdGo71B
8+HfQ6jFa7/UU4cZu4QAc=
-----END SSH SIGNATURE-----
`),
				SignatureType: "SSH SIGNATURE",
			},
		},
		{
			name: "signed_tag",
			data: `object 7a56ee7136c7d4882f88db68c0e629b81a47bfc9
type commit
tag test
tagger Marko Gaćeša <marko.gacesa@harness.io> 1749035203 +0200

This is a test tag
-----BEGIN SSH SIGNATURE-----
U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgEM1i8vha2gQ/ZXHinPejh0hS4C
x8VV1M2uwW6tglOswAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
AAAAQDzfgUo2uoz/VCuv74QnweB16XS6FGmaDkefMcVpYJdz88WRG99yhmYC0ca6QYiaj4
ttNpubwUBQRPTo8z5Aows=
-----END SSH SIGNATURE-----
`,
			want: ObjectRaw{
				Headers: []ObjectHeader{
					{Type: "object", Value: "7a56ee7136c7d4882f88db68c0e629b81a47bfc9\n"},
					{Type: "type", Value: "commit\n"},
					{Type: "tag", Value: "test\n"},
					{Type: "tagger", Value: "Marko Gaćeša <marko.gacesa@harness.io> 1749035203 +0200\n"},
				},
				Message: "This is a test tag\n",
				SignedContent: []byte(`object 7a56ee7136c7d4882f88db68c0e629b81a47bfc9
type commit
tag test
tagger Marko Gaćeša <marko.gacesa@harness.io> 1749035203 +0200

This is a test tag
`),
				Signature: []byte(`-----BEGIN SSH SIGNATURE-----
U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgEM1i8vha2gQ/ZXHinPejh0hS4C
x8VV1M2uwW6tglOswAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
AAAAQDzfgUo2uoz/VCuv74QnweB16XS6FGmaDkefMcVpYJdz88WRG99yhmYC0ca6QYiaj4
ttNpubwUBQRPTo8z5Aows=
-----END SSH SIGNATURE-----
`),
				SignatureType: "SSH SIGNATURE",
			},
		},
		{
			name: "merge_commit",
			data: `tree bf7ecca7c6741453e16a0a92be5d9ccd779abcfa
parent 04617da8f3215c84ae4af39b8d734c3df2247347
parent 7077c29016c1be5465678c9ba25983937040dcb2
author Marko Gaćeša <marko.gacesa@harness.io> 1749219134 +0200
committer Marko Gaćeša <marko.gacesa@harness.io> 1749219134 +0200
mergetag object 7077c29016c1be5465678c9ba25983937040dcb2
 type commit
 tag v1.0.0
 tagger Marko Gaćeša <marko.gacesa@harness.io> 1749218976 +0200
 
 version 1
 -----BEGIN SSH SIGNATURE-----
 U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgEM1i8vha2gQ/ZXHinPejh0hS4C
 x8VV1M2uwW6tglOswAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
 AAAAQG0+9xHX8+7AnbkV//QH7ZvrDoUcm6GrqWkTwHmgSqBsMa7X8aXOtcwPwNJvXpOl8E
 prGrumXZoEXzcZMrCG5A0=
 -----END SSH SIGNATURE-----
gpgsig -----BEGIN SSH SIGNATURE-----
 U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgEM1i8vha2gQ/ZXHinPejh0hS4C
 x8VV1M2uwW6tglOswAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
 AAAAQJpnz9Dsv6VleulZSzd3/PRGTJoPsUem0Waq4EbSRB7FDewjf11LRkkqNENiivT1pT
 Rv18ZouJpO2LRIXdZpxAE=
 -----END SSH SIGNATURE-----

Merge tag 'v1.0.0' into marko

version 1
`,
			want: ObjectRaw{
				Headers: []ObjectHeader{
					{Type: "tree", Value: "bf7ecca7c6741453e16a0a92be5d9ccd779abcfa\n"},
					{Type: "parent", Value: "04617da8f3215c84ae4af39b8d734c3df2247347\n"},
					{Type: "parent", Value: "7077c29016c1be5465678c9ba25983937040dcb2\n"},
					{Type: "author", Value: "Marko Gaćeša <marko.gacesa@harness.io> 1749219134 +0200\n"},
					{Type: "committer", Value: "Marko Gaćeša <marko.gacesa@harness.io> 1749219134 +0200\n"},
					{Type: "mergetag", Value: `object 7077c29016c1be5465678c9ba25983937040dcb2
type commit
tag v1.0.0
tagger Marko Gaćeša <marko.gacesa@harness.io> 1749218976 +0200

version 1
-----BEGIN SSH SIGNATURE-----
U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgEM1i8vha2gQ/ZXHinPejh0hS4C
x8VV1M2uwW6tglOswAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
AAAAQG0+9xHX8+7AnbkV//QH7ZvrDoUcm6GrqWkTwHmgSqBsMa7X8aXOtcwPwNJvXpOl8E
prGrumXZoEXzcZMrCG5A0=
-----END SSH SIGNATURE-----
`},
				},
				Message: "Merge tag 'v1.0.0' into marko\n\nversion 1\n",
				SignedContent: []byte(`tree bf7ecca7c6741453e16a0a92be5d9ccd779abcfa
parent 04617da8f3215c84ae4af39b8d734c3df2247347
parent 7077c29016c1be5465678c9ba25983937040dcb2
author Marko Gaćeša <marko.gacesa@harness.io> 1749219134 +0200
committer Marko Gaćeša <marko.gacesa@harness.io> 1749219134 +0200
mergetag object 7077c29016c1be5465678c9ba25983937040dcb2
 type commit
 tag v1.0.0
 tagger Marko Gaćeša <marko.gacesa@harness.io> 1749218976 +0200
 
 version 1
 -----BEGIN SSH SIGNATURE-----
 U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgEM1i8vha2gQ/ZXHinPejh0hS4C
 x8VV1M2uwW6tglOswAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
 AAAAQG0+9xHX8+7AnbkV//QH7ZvrDoUcm6GrqWkTwHmgSqBsMa7X8aXOtcwPwNJvXpOl8E
 prGrumXZoEXzcZMrCG5A0=
 -----END SSH SIGNATURE-----

Merge tag 'v1.0.0' into marko

version 1
`),
				Signature: []byte(`-----BEGIN SSH SIGNATURE-----
U1NIU0lHAAAAAQAAADMAAAALc3NoLWVkMjU1MTkAAAAgEM1i8vha2gQ/ZXHinPejh0hS4C
x8VV1M2uwW6tglOswAAAADZ2l0AAAAAAAAAAZzaGE1MTIAAABTAAAAC3NzaC1lZDI1NTE5
AAAAQJpnz9Dsv6VleulZSzd3/PRGTJoPsUem0Waq4EbSRB7FDewjf11LRkkqNENiivT1pT
Rv18ZouJpO2LRIXdZpxAE=
-----END SSH SIGNATURE-----
`),
				SignatureType: "SSH SIGNATURE",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			object, err := Object([]byte(test.data))
			if err != nil {
				t.Errorf("failed: %s", err.Error())
				return
			}

			if diff := cmp.Diff(object, test.want); diff != "" {
				t.Errorf("failed:\n%s\n", diff)
			}

			if len(object.Signature) == 0 || object.SignatureType != "SSH SIGNATURE" {
				// skip testing signed content because the data doesn't contain a signature
				return
			}

			// If git object from the test contains a signature,
			// we verify if object.SignedContent is correct
			// (if it's possible to verify the signature from the content).

			block, rest := pem.Decode(object.Signature)
			if block == nil || len(rest) > 0 || block.Type != object.SignatureType {
				t.Errorf("failed to decode signature")
				return
			}

			var signature publickey.SSHSignatureBlob
			if err := ssh.Unmarshal(block.Bytes, &signature); err != nil {
				t.Errorf("failed to parse signature: %s", err.Error())
				return
			}

			sshSig := ssh.Signature{}
			if err := ssh.Unmarshal(signature.Signature, &sshSig); err != nil {
				t.Errorf("failed to unmarshal ssh signature: %s", err.Error())
				return
			}

			h, _ := publickey.SSHHash(signature.HashAlgorithm)
			h.Write(object.SignedContent)

			key, err := ssh.ParsePublicKey(signature.PublicKey) // we get the public key directly from the signature
			if err != nil {
				t.Errorf("failed to parse signature key: %s", err.Error())
				return
			}

			signedMessage := ssh.Marshal(publickey.SSHMessageWrapper{
				Namespace:     signature.Namespace,
				HashAlgorithm: signature.HashAlgorithm,
				Hash:          h.Sum(nil),
			})
			buf := bytes.NewBuffer(nil)
			buf.Write(signature.MagicPreamble[:])
			buf.Write(signedMessage)

			err = key.Verify(buf.Bytes(), &sshSig)
			if err != nil {
				t.Errorf("failed to verify signature: %s", err.Error())
				return
			}
		})
	}
}

func TestObjectNegative(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		errStr string
	}{
		{
			name:   "header_without_EOL",
			data:   "header 1\nheader 2",
			errStr: "header line must end with EOL character",
		},
		{
			name:   "header_without_value",
			data:   "header\n\nbody",
			errStr: "malformed header",
		},
		{
			name:   "header_without_type",
			data:   " 1\n",
			errStr: "malformed header",
		},
		{
			name:   "header_invalid_sig",
			data:   "gpgsig this is\n not a sig\n\nbody",
			errStr: "invalid signature header",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Object([]byte(test.data))

			if err == nil {
				t.Error("expected error but got none")
				return
			}

			if want, got := test.errStr, err.Error(); !strings.HasPrefix(got, want) {
				t.Errorf("want error message to start with %s, got %s", want, got)
			}
		})
	}
}
