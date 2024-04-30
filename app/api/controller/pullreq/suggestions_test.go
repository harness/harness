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

package pullreq

import (
	"reflect"
	"testing"
)

func Test_parseSuggestions(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want []suggestion
	}{
		{
			name: "test empty",
			arg:  "",
			want: []suggestion{},
		},
		{
			name: "test no code block",
			arg:  "a\nb",
			want: []suggestion{},
		},
		{
			name: "test indented code block",
			arg:  "    a\nb",
			want: []suggestion{},
		},
		{
			name: "test not enough fence markers (`)",
			arg:  "`` suggestion\nb",
			want: []suggestion{},
		},
		{
			name: "test not enough fence markers (~)",
			arg:  "~~ suggestion\nb",
			want: []suggestion{},
		},
		{
			name: "test indented fences start marker (` with space)",
			arg:  "    ``` suggestion\nb",
			want: []suggestion{},
		},
		{
			name: "test indented fence start marker (~ with tab)",
			arg:  "\t~~~ suggestion\nb",
			want: []suggestion{},
		},
		{
			name: "test indented fence end marker (` with space)",
			arg:  "``` suggestion\na\n    ```\n",
			want: []suggestion{
				{
					checkSum: "6e0f2a7504f8e96c862c0f963faea994e527bd32a1c5c2c79acbf6baf57854e7",
					code:     "a\n    ```",
				},
			},
		},
		{
			name: "test indented fence end marker (~ with tab)",
			arg:  "~~~ suggestion\na\n\t~~~\n",
			want: []suggestion{
				{
					checkSum: "f5b959e235539ff7c9d2a687a1a5d05fa0c15e325dc50c83947c9d27c9d4fddf",
					code:     "a\n\t~~~",
				},
			},
		},
		{
			name: "test fence marker with invalid char (` with `)",
			arg:  "``` suggestion `\nb",
			want: []suggestion{},
		},
		{
			name: "test wrong language (`)",
			arg:  "``` abc\nb",
			want: []suggestion{},
		},
		{
			name: "test wrong language (~)",
			arg:  "~~~ abc\nb",
			want: []suggestion{},
		},
		{
			name: "test language prefix (`)",
			arg:  "``` suggestions\nb",
			want: []suggestion{},
		},
		{
			name: "test suggestion empty without code or endmarker (`)",
			arg:  "``` suggestion",
			want: []suggestion{
				{
					checkSum: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					code:     "",
				},
			},
		},
		{
			name: "test suggestion empty without endmarker (`)",
			arg:  "``` suggestion\n",
			want: []suggestion{
				{
					checkSum: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					code:     "",
				},
			},
		},
		{
			name: "test suggestion empty with endmarker (~)",
			arg:  "~~~ suggestion\n~~~",
			want: []suggestion{
				{
					checkSum: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					code:     "",
				},
			},
		},
		{
			name: "test suggestion newline only without endmarker (`)",
			arg:  "``` suggestion\n\n",
			want: []suggestion{
				{
					checkSum: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					code:     "", // first \n is for end of header, second \n is for beginning of trailer
				},
			},
		},
		{
			name: "test suggestion newline only without endmarker (`)",
			arg:  "``` suggestion\n\n\n",
			want: []suggestion{
				{
					checkSum: "01ba4719c80b6fe911b091a7c05124b64eeece964e09c058ef8f9805daca546b",
					code:     "\n", // first \n is for end of header, second \n is for beginning of trailer
				},
			},
		},
		{
			name: "test suggestion without end and line without newline (`)",
			arg:  "``` suggestion\na",
			want: []suggestion{
				{
					checkSum: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
					code:     "a",
				},
			},
		},
		{
			name: "test suggestion without end and line with newline (~)",
			arg:  "~~~ suggestion\na\n",
			want: []suggestion{
				{
					checkSum: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
					code:     "a",
				},
			},
		},
		{
			name: "test suggestion with wrong end (`)",
			arg:  "``` suggestion\na\n~~~",
			want: []suggestion{
				{
					checkSum: "862bb949147c31270cce026205482a2fcd797047a5fdc01a0baa1bdfaf136386",
					code:     "a\n~~~",
				},
			},
		},
		{
			name: "test suggestion with wrong end (~)",
			arg:  "~~~ suggestion\na\n```\n",
			want: []suggestion{
				{
					checkSum: "83380dbcc26319cbdbb70c1ad11480c464cf731560a5d645afb727da33930611",
					code:     "a\n```",
				},
			},
		},
		{
			name: "test suggestion with not enough endmarker (`)",
			arg:  "``` suggestion\na\n``\n",
			want: []suggestion{
				{
					checkSum: "5b2b3107a7cceac969684464f39d300c8d1480b5fa300bc1b222c8e21db6c757",
					code:     "a\n``",
				},
			},
		},
		{
			name: "test suggestion with not enough endmarker (~, more than 3)",
			arg:  "~~~~ suggestion\na\n~~~",
			want: []suggestion{
				{
					checkSum: "862bb949147c31270cce026205482a2fcd797047a5fdc01a0baa1bdfaf136386",
					code:     "a\n~~~",
				},
			},
		},
		{
			name: "test suggestion with trailing invalid chars on endmarker (`)",
			arg:  "``` suggestion\na\n```a\n",
			want: []suggestion{
				{
					checkSum: "bd87ec5a4beda93c6912f0b786556f3a7c30772222dc65c104e2e60770492339",
					code:     "a\n```a",
				},
			},
		},
		{
			name: "test suggestion with trailing invalid chars on endmarker (~)",
			arg:  "~~~ suggestion\na\n~~~a",
			want: []suggestion{
				{
					checkSum: "07333da4c8a7348e4acd1a06566bb82d1a5f2a963e126158842308ec8b1d68f0",
					code:     "a\n~~~a",
				},
			},
		},
		{
			name: "test basic suggestion with text around(`)",
			arg:  "adb\n``` suggestion\na\n```\nawef\n2r3",
			want: []suggestion{
				{
					checkSum: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
					code:     "a",
				},
			},
		},
		{
			name: "test basic suggestion with text around(~)",
			arg:  "adb\n~~~ suggestion\na\n~~~\nawef\n2r3",
			want: []suggestion{
				{
					checkSum: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
					code:     "a",
				},
			},
		},
		{
			name: "test suggestion with spaces in markers (~)",
			arg:  "   ~~~   \t\tsuggestion \t\na\n   ~~~    \t    ",
			want: []suggestion{
				{
					checkSum: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
					code:     "a",
				},
			},
		},
		{
			name: "test suggestion with spaces in markers (`, more than 3)",
			arg:  "   ```` \t  suggestion   \t\t   \na\n   ````   \t     ",
			want: []suggestion{
				{
					checkSum: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
					code:     "a",
				},
			},
		},
		{
			name: "test suggestion with too many end marker chars (`)",
			arg:  "``` suggestion\na\n`````````",
			want: []suggestion{
				{
					checkSum: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
					code:     "a",
				},
			},
		},
		{
			name: "test suggestion with too many end marker chars (`)",
			arg:  "~~~~~ suggestion\na\n~~~~~~~~~~~~",
			want: []suggestion{
				{
					checkSum: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
					code:     "a",
				},
			},
		},
		{
			name: "test suggestion that contains opposite marker (`)",
			arg:  "``` suggestion\n~~~ suggestion\na\n~~~\n```",
			want: []suggestion{
				{
					checkSum: "ed25a1606bf819448bf7e76fce9dbd2897fa5a379b67be74b5819ee521455783",
					code:     "~~~ suggestion\na\n~~~",
				},
			},
		},
		{
			name: "test suggestion that contains opposite marker (~)",
			arg:  "~~~ suggestion\n``` suggestion\na\n```\n~~~",
			want: []suggestion{
				{
					checkSum: "2463ad212ec8179e1f4d2a9ac35349b02e46bcba2173f6b05c9f73dfb4ca7ed9",
					code:     "``` suggestion\na\n```",
				},
			},
		},
		{
			name: "test suggestion that contains shorter marker (`)",
			arg:  "```` suggestion\n``` suggestion\na\n```\n````",
			want: []suggestion{
				{
					checkSum: "2463ad212ec8179e1f4d2a9ac35349b02e46bcba2173f6b05c9f73dfb4ca7ed9",
					code:     "``` suggestion\na\n```",
				},
			},
		},
		{
			name: "test suggestion that contains shorter marker (~)",
			arg:  "~~~~ suggestion\n~~~ suggestion\na\n~~~\n~~~~",
			want: []suggestion{
				{
					checkSum: "ed25a1606bf819448bf7e76fce9dbd2897fa5a379b67be74b5819ee521455783",
					code:     "~~~ suggestion\na\n~~~",
				},
			},
		},
		{
			name: "test multiple suggestions same marker (`)",
			arg:  "``` suggestion\na\n```\nsomething``\n``` suggestion\nb\n```",
			want: []suggestion{
				{
					checkSum: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
					code:     "a",
				},
				{
					checkSum: "3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d",
					code:     "b",
				},
			},
		},
		{
			name: "test multiple suggestions same marker (~)",
			arg:  "~~~ suggestion\na\n~~~\nsomething~~\n~~~ suggestion\nb\n~~~",
			want: []suggestion{
				{
					checkSum: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
					code:     "a",
				},
				{
					checkSum: "3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d",
					code:     "b",
				},
			},
		},
		{
			name: "test multiple suggestions different markder (`,~)",
			arg:  "``` suggestion\na\n```\nsomething~~\n~~~ suggestion\nb\n~~~",
			want: []suggestion{
				{
					checkSum: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
					code:     "a",
				},
				{
					checkSum: "3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d",
					code:     "b",
				},
			},
		},
		{
			name: "test multiple suggestions last not ending (`,~)",
			arg:  "``` suggestion\na\n```\nsomething~~\n~~~ suggestion\nb\n~~~\n\n``` suggestion\nc",
			want: []suggestion{
				{
					checkSum: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
					code:     "a",
				},
				{
					checkSum: "3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d",
					code:     "b",
				},
				{
					checkSum: "2e7d2c03a9507ae265ecf5b5356885a53393a2029d241394997265a1a25aefc6",
					code:     "c",
				},
			},
		},

		{
			name: "test with crlf and multiple (`,~)",
			arg:  "abc\n``` suggestion\r\na\n```\r\n~~~ suggestion\nb\r\n~~~",
			want: []suggestion{
				{
					checkSum: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb",
					code:     "a",
				},
				{
					checkSum: "3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d",
					code:     "b",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSuggestions(tt.arg)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSuggestions() = %s, want %s", got, tt.want)
			}
		})
	}
}
