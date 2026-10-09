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

package langstats

import (
	"strings"
	"testing"
)

// goSource has 2 source lines of code ("package main", "func main() {}"),
// one comment line and one blank line. The LOC metric counts only the 2
// source lines (scc Code).
const goSource = "package main\n\n// a comment\nfunc main() {}\n"

// addBlob feeds content to the analyzer the way the git layer does, via a
// streaming reader, failing the test on error.
func addBlob(t *testing.T, a *Analyzer, path, content string) {
	t.Helper()
	if err := a.AddBlob(path, int64(len(content)), strings.NewReader(content)); err != nil {
		t.Fatalf("AddBlob(%q) error: %v", path, err)
	}
}

func TestAnalyzer_KnownLanguageCountsCodeOnly(t *testing.T) {
	a := NewAnalyzer()
	addBlob(t, a, "main.go", goSource)

	stat, ok := a.Stats()["Go"]
	if !ok {
		t.Fatalf("expected a Go bucket, got languages %v", keys(a.Stats()))
	}
	if stat.Files != 1 {
		t.Errorf("Files = %d, want 1", stat.Files)
	}
	if stat.Bytes != int64(len(goSource)) {
		t.Errorf("Bytes = %d, want %d", stat.Bytes, len(goSource))
	}
	// Only source lines count - comments and blanks are excluded.
	if a.CodeLines() != 2 {
		t.Errorf("CodeLines = %d, want 2", a.CodeLines())
	}
}

func TestAnalyzer_BinaryFileIgnoredForLOC(t *testing.T) {
	// A control byte within the first bytes marks the blob as binary.
	binary := "\x00\x01\x02\x03PNG\x00garbage"

	a := NewAnalyzer()
	addBlob(t, a, "logo.png", binary)

	// Binary blobs still count towards bytes/files ...
	var files, bytes int64
	for _, s := range a.Stats() {
		files += s.Files
		bytes += s.Bytes
	}
	if files != 1 || bytes != int64(len(binary)) {
		t.Errorf("binary file not counted in byte/file breakdown: files=%d bytes=%d", files, bytes)
	}
	// ... but contribute no LOC.
	if a.CodeLines() != 0 {
		t.Errorf("CodeLines = %d, want 0 for binary file", a.CodeLines())
	}
}

func TestAnalyzer_ExtensionlessFileIgnoredForLOC(t *testing.T) {
	text := "line one\nline two\nline three\n"

	a := NewAnalyzer()
	addBlob(t, a, "LICENSE", text)

	// Extensionless files are still bucketed for the byte/file breakdown ...
	if _, ok := a.Stats()[""]; !ok {
		t.Errorf("expected an extensionless \"\" bucket, got %v", keys(a.Stats()))
	}
	// ... but are ignored for LOC.
	if a.CodeLines() != 0 {
		t.Errorf("CodeLines = %d, want 0 for extensionless file", a.CodeLines())
	}
}

func TestAnalyzer_UnknownExtensionBestEffortLineCount(t *testing.T) {
	// Unknown extension scc cannot classify -> best-effort non-blank line count.
	content := "alpha\n\nbeta\n   \ngamma\n"

	a := NewAnalyzer()
	addBlob(t, a, "data.unknownext", content)

	if _, ok := a.Stats()[Unclassified]; !ok {
		t.Errorf("expected an %q bucket, got %v", Unclassified, keys(a.Stats()))
	}
	// 3 non-blank lines (alpha, beta, gamma); the empty and whitespace-only
	// lines are skipped.
	if a.CodeLines() != 3 {
		t.Errorf("CodeLines = %d, want 3", a.CodeLines())
	}
}

func TestAnalyzer_UnknownExtensionHandlesCRLF(t *testing.T) {
	// Windows line endings (\r\n) count once per line, and blank /
	// whitespace-only lines are skipped.
	content := "alpha\r\n\r\nbeta\r\n \t\r\ngamma\r\n"

	a := NewAnalyzer()
	addBlob(t, a, "data.unknownext", content)

	if a.CodeLines() != 3 {
		t.Errorf("CodeLines = %d, want 3", a.CodeLines())
	}
}

func TestAnalyzer_LargeKnownFileCountedViaSCC(t *testing.T) {
	// A known-language file within the buffer limit is counted with scc.
	const n = 100_000
	content := strings.Repeat("x = 1\n", n) // ~600 KiB, under maxBufferedBytes

	a := NewAnalyzer()
	addBlob(t, a, "big.py", content)

	if a.CodeLines() != int64(n) {
		t.Errorf("CodeLines = %d, want %d", a.CodeLines(), n)
	}
}

func TestAnalyzer_OversizedTextFileStreamed(t *testing.T) {
	// A text file larger than maxBufferedBytes is counted by streaming (non-blank
	// lines) rather than buffered - the whole file is never held in memory.
	n := int64(maxBufferedBytes/len("x = 1\n") + 1000)
	content := strings.Repeat("x = 1\n", int(n))
	if int64(len(content)) <= maxBufferedBytes {
		t.Fatalf("test setup: content %d not above maxBufferedBytes %d", len(content), maxBufferedBytes)
	}

	a := NewAnalyzer()
	addBlob(t, a, "huge.py", content)

	if a.CodeLines() != n {
		t.Errorf("CodeLines = %d, want %d", a.CodeLines(), n)
	}
}

func TestAnalyzer_OversizedBinaryFileStreamed(t *testing.T) {
	// A binary blob larger than maxBufferedBytes is classified from its prefix,
	// counts towards bytes/files, and contributes no LOC.
	size := maxBufferedBytes + 5000
	binary := "\x00" + strings.Repeat("a", size-1)

	a := NewAnalyzer()
	addBlob(t, a, "archive.zip", binary)

	var files, bytes int64
	for _, s := range a.Stats() {
		files += s.Files
		bytes += s.Bytes
	}
	if files != 1 || bytes != int64(size) {
		t.Errorf("binary blob not counted in byte/file breakdown: files=%d bytes=%d", files, bytes)
	}
	if a.CodeLines() != 0 {
		t.Errorf("CodeLines = %d, want 0 for binary blob", a.CodeLines())
	}
}

func TestAnalyzer_TotalAcrossFiles(t *testing.T) {
	a := NewAnalyzer()
	addBlob(t, a, "main.go", goSource)             // +2 (known)
	addBlob(t, a, "notes.unknownext", "a\nb\nc\n") // +3 (unknown ext)
	addBlob(t, a, "LICENSE", "x\ny\n")             // +0 (extensionless)
	addBlob(t, a, "blob.bin", "\x00\x00\x00\x00")  // +0 (binary)

	if a.CodeLines() != 5 {
		t.Errorf("CodeLines = %d, want 5", a.CodeLines())
	}
}

func TestCountNonBlankLines(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int64
	}{
		{"empty", "", 0},
		{"single line no newline", "abc", 1},
		{"single line trailing newline", "abc\n", 1},
		{"multiple lines trailing newline", "a\nb\nc\n", 3},
		{"multiple lines no trailing newline", "a\nb\nc", 3},
		{"blank line between content", "a\n\nb\n", 2},
		{"leading and trailing blank lines", "\n\na\n\n", 1},
		{"only newlines", "\n\n\n", 0},
		{"whitespace-only lines", "   \n\t\n \f\n", 0},
		{"whitespace-only line between content", "a\n \t \nb\n", 2},
		{"content with surrounding whitespace", "  x  \n", 1},
		{"crlf line endings", "a\r\nb\r\n", 2},
		{"crlf blank line between content", "a\r\n\r\nb\r\n", 2},
		{"lone cr line endings", "a\rb\r", 2},
		{"crlf no trailing newline", "a\r\nb", 2},
		// Larger than the 32 KiB read buffer: exercises state carried across reads,
		// including a \r\n split across a buffer boundary.
		{"content spanning read buffers", strings.Repeat("line\r\n", 20_000), 20_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := countNonBlankLines(strings.NewReader(tt.content))
			if err != nil {
				t.Fatalf("countNonBlankLines error: %v", err)
			}
			if got != tt.want {
				t.Errorf("countNonBlankLines(%q...) = %d, want %d", truncate(tt.content), got, tt.want)
			}
		})
	}
}

func truncate(s string) string {
	if len(s) > 32 {
		return s[:32]
	}
	return s
}

func TestIsBinary(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"empty", "", false},
		{"plain text", "package main\nfunc main() {}\n", false},
		{"text with tabs and crlf", "a\tb\r\n\tc\r\n", false},
		{"text with vertical tab and form feed", "a\vb\fc\n", false},
		{"nul byte", "abc\x00def", true},
		{"low control char", "abc\x01def", true},
		{"escape char", "abc\x1bdef", true},
		{"bell char", "abc\x07def", true},
		{"high byte alone not binary", "caf\xc3\xa9\n", false},
		{"control char beyond scan limit", strings.Repeat("a", binaryCheckLimit) + "\x00", false},
		{"control char at scan limit boundary", strings.Repeat("a", binaryCheckLimit-1) + "\x00", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBinary([]byte(tt.content)); got != tt.want {
				t.Errorf("IsBinary(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func keys(m map[string]*LangStat) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
