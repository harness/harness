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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/boyter/scc/processor"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/maps"
)

const Unclassified = "Unclassified"

// LangStat holds the byte/file statistics for a single language.
type LangStat struct {
	Bytes int64
	Files int64
}

// sccInit loads scc's language database exactly once (it is package-level state).
var sccInit sync.Once

func initSCC() {
	sccInit.Do(processor.ProcessConstants)
}

// maxBufferedBytes bounds how much of a blob is held in memory for scc's
// comment-aware counting. Larger blobs are counted by streaming (see AddBlob),
// so the server never loads an entire file into memory.
const maxBufferedBytes = 1 << 20 // 1 MiB

// Analyzer accumulates language statistics across a repository's files, fed one
// blob at a time via AddBlob so callers can stream rather than buffer the tree.
type Analyzer struct {
	stats                  map[string]*LangStat
	unclassifiedExtensions map[string]struct{}
	codeLines              int64
}

// NewAnalyzer returns a ready-to-use Analyzer. It lazily initializes scc.
func NewAnalyzer() *Analyzer {
	initSCC()
	return &Analyzer{
		stats:                  map[string]*LangStat{},
		unclassifiedExtensions: map[string]struct{}{},
	}
}

// AddBlob records a blob's byte/file contribution and its source lines of code,
// reading content from r without ever holding the whole blob in memory:
//   - binary and extensionless files contribute no LOC;
//   - a known language within maxBufferedBytes is counted with scc (comment and
//     blank aware), for parity with the CI LOC estimator;
//   - unknown extensions and oversized blobs are counted by streaming a
//     non-blank line count through a fixed buffer.
//
// r is expected to yield exactly the blob content (e.g. an io.LimitReader); the
// caller is responsible for draining any bytes AddBlob leaves unread.
func (a *Analyzer) AddBlob(path string, size int64, r io.Reader) error {
	ext := a.bucket(path, size)
	if ext == "" {
		return nil // extensionless: no LOC
	}

	// Read a bounded prefix to classify the blob as binary.
	prefix := make([]byte, min(binaryCheckLimit, size))
	n, err := io.ReadFull(r, prefix)
	// EOF/ErrUnexpectedEOF are tolerated (file shorter than the limit).
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("failed to read blob prefix: %w", err)
	}
	prefix = prefix[:n]

	if IsBinary(prefix) {
		return nil // binary: no LOC
	}

	filename := filepath.Base(path)
	possibleLanguages, _ := processor.DetectLanguage(filename)

	// Unknown extension (no comment rules) or oversized blob: stream a non-blank
	// line count so we never buffer the whole file.
	if len(possibleLanguages) == 0 || size > maxBufferedBytes {
		lines, err := countNonBlankLines(io.MultiReader(bytes.NewReader(prefix), r))
		if err != nil {
			return err
		}
		a.codeLines += lines
		return nil
	}

	// Known language within the buffer limit: count with scc for comment and
	// blank awareness. Bounded by maxBufferedBytes.
	rest, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read blob content: %w", err)
	}
	content := make([]byte, 0, len(prefix)+len(rest))
	content = append(content, prefix...)
	content = append(content, rest...)

	job := &processor.FileJob{
		Filename:          filename,
		PossibleLanguages: possibleLanguages,
		Content:           content,
		Bytes:             int64(len(content)),
	}
	job.Language = processor.DetermineLanguage(job.Filename, "", job.PossibleLanguages, job.Content)
	processor.CountStats(job)
	a.codeLines += job.Code
	return nil
}

// bucket adds a blob's byte/file contribution under its extension-based language
// and returns the file extension.
func (a *Analyzer) bucket(path string, size int64) string {
	ext := filepath.Ext(path)
	lang, _ := GetLanguageByExtension(ext)

	stat, ok := a.stats[lang]
	if !ok {
		stat = &LangStat{}
		a.stats[lang] = stat
	}

	stat.Bytes += size
	stat.Files++

	if lang == Unclassified {
		a.unclassifiedExtensions[ext] = struct{}{}
	}

	return ext
}

// Stats returns the accumulated per-language byte/file statistics keyed by language.
func (a *Analyzer) Stats() map[string]*LangStat {
	return a.stats
}

// CodeLines returns the repository's source-lines-of-code total.
func (a *Analyzer) CodeLines() int64 {
	return a.codeLines
}

// binaryCheckLimit bounds the binary-detection scan, matching scc's own cutoff.
const binaryCheckLimit = 10000

// IsBinary reports whether content looks binary: a control character other than
// text whitespace (tab, newline, vertical tab, form feed, CR) within the first
// binaryCheckLimit bytes.
func IsBinary(content []byte) bool {
	limit := min(len(content), binaryCheckLimit)
	for i := range limit {
		b := content[i]
		switch b {
		case '\t', '\n', '\v', '\f', '\r':
			// Text whitespace, not binary.
		default:
			if b < 0x20 {
				return true
			}
		}
	}
	return false
}

// countNonBlankLines counts non-blank lines from r in a single streaming pass
// using a fixed-size buffer, so it never holds the whole input in memory. It
// treats \n, \r and \r\n as line endings; blank and whitespace-only lines are
// not counted.
func countNonBlankLines(r io.Reader) (int64, error) {
	buf := make([]byte, 32*1024)
	var n int64
	lineHasContent := false
	for {
		read, err := r.Read(buf)
		for _, b := range buf[:read] {
			switch b {
			case '\n', '\r':
				if lineHasContent {
					n++
					lineHasContent = false
				}
			case ' ', '\t', '\v', '\f':
				// Whitespace within a line - does not make the line non-blank.
			default:
				lineHasContent = true
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("failed to read while counting lines: %w", err)
		}
	}
	if lineHasContent {
		n++
	}
	return n, nil
}

// LogUnclassified logs the set of file extensions that could not be classified.
func (a *Analyzer) LogUnclassified(ctx context.Context) {
	log.Info().Ctx(ctx).
		Strs("unclassified_extensions", maps.Keys(a.unclassifiedExtensions)).
		Msgf("Detected %d unclassified file extensions during language analysis",
			len(a.unclassifiedExtensions))
}

// GetLanguageByExtension returns the first matching language for an extension
// and whether the extension maps to a single language.
func GetLanguageByExtension(ext string) (lang string, unique bool) {
	if ext == "" {
		return "", false
	}

	langs, ok := progLangsByExt[strings.ToLower(ext)]
	if !ok || len(langs) == 0 {
		return Unclassified, false
	}

	return langs[0], len(langs) == 1
}
