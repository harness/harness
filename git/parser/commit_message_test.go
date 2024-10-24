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

import "testing"

func TestCleanUpWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		exp   string
	}{
		{
			name: "remove_trailing_spaces_in_lines",
			input: "" +
				"ABC   \n" +
				"\t\t\n" +
				"DEF\t\n",
			exp: "" +
				"ABC\n" +
				"\n" +
				"DEF\n",
		},
		{
			name: "add_eof_to_the_last_line",
			input: "" +
				"ABC\n" +
				"DEF",
			exp: "" +
				"ABC\n" +
				"DEF\n",
		},
		{
			name: "remove_consecutive_empty_lines",
			input: "" +
				"ABC\n" +
				"\n" +
				"\t\t\n" +
				"\n" +
				"DEF\n",
			exp: "" +
				"ABC\n" +
				"\n" +
				"DEF\n",
		},
		{
			name: "remove_empty_lines_from_the_message_bottom",
			input: "" +
				"ABC\n" +
				"\n" +
				"DEF\n" +
				"\n" +
				"\n" +
				"\n",
			exp: "" +
				"ABC\n" +
				"\n" +
				"DEF\n",
		},
		{
			name: "remove_empty_lines_from_the_message_top",
			input: "" +
				"\n" +
				"\n" +
				"ABC\n" +
				"\n" +
				"DEF\n" +
				"\n",
			exp: "" +
				"ABC\n" +
				"\n" +
				"DEF\n",
		},
		{
			name: "multi_line_body",
			input: "" +
				"ABC\n" +
				"DEF\n" +
				"\n" +
				"GHI\n" +
				"JKL\n" +
				"\n" +
				"NMO\n",
			exp: "" +
				"ABC\n" +
				"DEF\n" +
				"\n" +
				"GHI\n" +
				"JKL\n" +
				"\n" +
				"NMO\n",
		},
		{
			name: "complex",
			input: "" +
				"\n" +
				"subj one\n" +
				"   subj two\n" +
				"\t\t\n" +
				"  \n" +
				"  body one\n" +
				"body two\n" +
				" \t \n" +
				"   body three\n" +
				" \n" +
				"    ",
			exp: "" +
				"subj one\n" +
				"   subj two\n" +
				"\n" +
				"  body one\n" +
				"body two\n" +
				"\n" +
				"   body three\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cleaned := CleanUpWhitespace(test.input)
			if want, got := test.exp, cleaned; want != got {
				t.Errorf("want=%q, got=%q", want, got)
			}
		})
	}
}

func TestExtractSubject(t *testing.T) {
	tests := []struct {
		name  string
		input string
		exp   string
	}{
		{
			name: "join_lines",
			input: "" +
				"ABC\n" +
				"DEF\n",
			exp: "ABC DEF",
		},
		{
			name: "stop_after_empty",
			input: "" +
				"ABC\n" +
				"DEF\n" +
				"\n" +
				"GHI\n",
			exp: "ABC DEF",
		},
		{
			name: "ignore_extra_whitespace",
			input: "" +
				"\t\n" +
				" ABC  \n" +
				"\tDEF   \n" +
				"\t\t\n" +
				"GHI",
			exp: "ABC DEF",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			subject := ExtractSubject(test.input)
			if want, got := test.exp, subject; want != got {
				t.Errorf("want=%q, got=%q", want, got)
			}
		})
	}
}

func TestSplitMessage(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expSubject string
		expBody    string
	}{
		{
			name: "remove_trailing_spaces_in_lines",
			input: "" +
				"ABC   \n" +
				"\t\t\n" +
				"DEF\n",
			expSubject: "ABC",
			expBody:    "DEF\n",
		},
		{
			name: "add_eof_to_the_last_line",
			input: "" +
				"ABC\n" +
				"DEF",
			expSubject: "ABC DEF",
			expBody:    "",
		},
		{
			name: "add_eof_to_the_last_line_of_body",
			input: "" +
				"ABC\n" +
				"DEF\n" +
				"\n" +
				"GHI",
			expSubject: "ABC DEF",
			expBody:    "GHI\n",
		},
		{
			name: "remove_consecutive_empty_lines",
			input: "" +
				"ABC\n" +
				"\n" +
				"\t\t\n" +
				"\n" +
				"DEF\n",
			expSubject: "ABC",
			expBody:    "DEF\n",
		},
		{
			name: "multi_line_body",
			input: "" +
				"ABC\n" +
				"\n" +
				"DEF\n" +
				"GHI\n",
			expSubject: "ABC",
			expBody:    "DEF\nGHI\n",
		},
		{
			name: "remove_empty_lines_from_the_message_bottom",
			input: "" +
				"ABC\n" +
				"\n" +
				"DEF\n" +
				"\n" +
				"\n" +
				"\n",
			expSubject: "ABC",
			expBody:    "DEF\n",
		},
		{
			name: "remove_empty_lines_from_the_message_top",
			input: "" +
				"\n" +
				"\n" +
				"ABC\n" +
				"\n" +
				"DEF\n" +
				"\n",
			expSubject: "ABC",
			expBody:    "DEF\n",
		},
		{
			name: "complex",
			input: "" +
				"\n" +
				"subj one\n" +
				"   subj two\n" +
				"\t\t\n" +
				"  \n" +
				"  body one\n" +
				"body two\n" +
				" \t \n" +
				"   body three\n" +
				" \n" +
				"    ",
			expSubject: "subj one subj two",
			expBody: "" +
				"  body one\n" +
				"body two\n" +
				"\n" +
				"   body three\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			subject, body := SplitMessage(test.input)

			if want, got := test.expSubject, subject; want != got {
				t.Errorf("subject: want=%q, got=%q", want, got)
			}

			if want, got := test.expBody, body; want != got {
				t.Errorf("body: want=%q, got=%q", want, got)
			}
		})
	}
}
