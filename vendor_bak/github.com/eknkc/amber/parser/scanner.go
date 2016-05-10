package parser

import (
	"bufio"
	"container/list"
	"fmt"
	"io"
	"regexp"
)

const (
	tokEOF = -(iota + 1)
	tokDoctype
	tokComment
	tokIndent
	tokOutdent
	tokBlank
	tokId
	tokClassName
	tokTag
	tokText
	tokAttribute
	tokIf
	tokElse
	tokEach
	tokAssignment
	tokImport
	tokNamedBlock
	tokExtends
	tokMixin
	tokMixinCall
)

const (
	scnNewLine = iota
	scnLine
	scnEOF
)

type scanner struct {
	reader      *bufio.Reader
	indentStack *list.List
	stash       *list.List

	state  int32
	buffer string

	line          int
	col           int
	lastTokenLine int
	lastTokenCol  int
	lastTokenSize int

	readRaw bool
}

type token struct {
	Kind  rune
	Value string
	Data  map[string]string
}

func newScanner(r io.Reader) *scanner {
	s := new(scanner)
	s.reader = bufio.NewReader(r)
	s.indentStack = list.New()
	s.stash = list.New()
	s.state = scnNewLine
	s.line = -1
	s.col = 0

	return s
}

func (s *scanner) Pos() SourcePosition {
	return SourcePosition{s.lastTokenLine + 1, s.lastTokenCol + 1, s.lastTokenSize, ""}
}

// Returns next token found in buffer
func (s *scanner) Next() *token {
	if s.readRaw {
		s.readRaw = false
		return s.NextRaw()
	}

	s.ensureBuffer()

	if stashed := s.stash.Front(); stashed != nil {
		tok := stashed.Value.(*token)
		s.stash.Remove(stashed)
		return tok
	}

	switch s.state {
	case scnEOF:
		if outdent := s.indentStack.Back(); outdent != nil {
			s.indentStack.Remove(outdent)
			return &token{tokOutdent, "", nil}
		}

		return &token{tokEOF, "", nil}
	case scnNewLine:
		s.state = scnLine

		if tok := s.scanIndent(); tok != nil {
			return tok
		}

		return s.Next()
	case scnLine:
		if tok := s.scanMixin(); tok != nil {
			return tok
		}

		if tok := s.scanMixinCall(); tok != nil {
			return tok
		}

		if tok := s.scanDoctype(); tok != nil {
			return tok
		}

		if tok := s.scanCondition(); tok != nil {
			return tok
		}

		if tok := s.scanEach(); tok != nil {
			return tok
		}

		if tok := s.scanImport(); tok != nil {
			return tok
		}

		if tok := s.scanExtends(); tok != nil {
			return tok
		}

		if tok := s.scanBlock(); tok != nil {
			return tok
		}

		if tok := s.scanAssignment(); tok != nil {
			return tok
		}

		if tok := s.scanTag(); tok != nil {
			return tok
		}

		if tok := s.scanId(); tok != nil {
			return tok
		}

		if tok := s.scanClassName(); tok != nil {
			return tok
		}

		if tok := s.scanAttribute(); tok != nil {
			return tok
		}

		if tok := s.scanComment(); tok != nil {
			return tok
		}

		if tok := s.scanText(); tok != nil {
			return tok
		}
	}

	return nil
}

func (s *scanner) NextRaw() *token {
	result := ""
	level := 0

	for {
		s.ensureBuffer()

		switch s.state {
		case scnEOF:
			return &token{tokText, result, map[string]string{"Mode": "raw"}}
		case scnNewLine:
			s.state = scnLine

			if tok := s.scanIndent(); tok != nil {
				if tok.Kind == tokIndent {
					level++
				} else if tok.Kind == tokOutdent {
					level--
				} else {
					result = result + "\n"
					continue
				}

				if level < 0 {
					s.stash.PushBack(&token{tokOutdent, "", nil})

					if len(result) > 0 && result[len(result)-1] == '\n' {
						result = result[:len(result)-1]
					}

					return &token{tokText, result, map[string]string{"Mode": "raw"}}
				}
			}
		case scnLine:
			if len(result) > 0 {
				result = result + "\n"
			}
			for i := 0; i < level; i++ {
				result += "\t"
			}
			result = result + s.buffer
			s.consume(len(s.buffer))
		}
	}

	return nil
}

var rgxIndent = regexp.MustCompile(`^(\s+)`)

func (s *scanner) scanIndent() *token {
	if len(s.buffer) == 0 {
		return &token{tokBlank, "", nil}
	}

	var head *list.Element
	for head = s.indentStack.Front(); head != nil; head = head.Next() {
		value := head.Value.(*regexp.Regexp)

		if match := value.FindString(s.buffer); len(match) != 0 {
			s.consume(len(match))
		} else {
			break
		}
	}

	newIndent := rgxIndent.FindString(s.buffer)

	if len(newIndent) != 0 && head == nil {
		s.indentStack.PushBack(regexp.MustCompile(regexp.QuoteMeta(newIndent)))
		s.consume(len(newIndent))
		return &token{tokIndent, newIndent, nil}
	}

	if len(newIndent) == 0 && head != nil {
		for head != nil {
			next := head.Next()
			s.indentStack.Remove(head)
			if next == nil {
				return &token{tokOutdent, "", nil}
			} else {
				s.stash.PushBack(&token{tokOutdent, "", nil})
			}
			head = next
		}
	}

	if len(newIndent) != 0 && head != nil {
		panic("Mismatching indentation. Please use a coherent indent schema.")
	}

	return nil
}

var rgxDoctype = regexp.MustCompile(`^(!!!|doctype)\s*(.*)`)

func (s *scanner) scanDoctype() *token {
	if sm := rgxDoctype.FindStringSubmatch(s.buffer); len(sm) != 0 {
		if len(sm[2]) == 0 {
			sm[2] = "html"
		}

		s.consume(len(sm[0]))
		return &token{tokDoctype, sm[2], nil}
	}

	return nil
}

var rgxIf = regexp.MustCompile(`^if\s+(.+)$`)
var rgxElse = regexp.MustCompile(`^else\s*`)

func (s *scanner) scanCondition() *token {
	if sm := rgxIf.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return &token{tokIf, sm[1], nil}
	}

	if sm := rgxElse.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return &token{tokElse, "", nil}
	}

	return nil
}

var rgxEach = regexp.MustCompile(`^each\s+(\$[\w0-9\-_]*)(?:\s*,\s*(\$[\w0-9\-_]*))?\s+in\s+(.+)$`)

func (s *scanner) scanEach() *token {
	if sm := rgxEach.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return &token{tokEach, sm[3], map[string]string{"X": sm[1], "Y": sm[2]}}
	}

	return nil
}

var rgxAssignment = regexp.MustCompile(`^(\$[\w0-9\-_]*)?\s*=\s*(.+)$`)

func (s *scanner) scanAssignment() *token {
	if sm := rgxAssignment.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return &token{tokAssignment, sm[2], map[string]string{"X": sm[1]}}
	}

	return nil
}

var rgxComment = regexp.MustCompile(`^\/\/(-)?\s*(.*)$`)

func (s *scanner) scanComment() *token {
	if sm := rgxComment.FindStringSubmatch(s.buffer); len(sm) != 0 {
		mode := "embed"
		if len(sm[1]) != 0 {
			mode = "silent"
		}

		s.consume(len(sm[0]))
		return &token{tokComment, sm[2], map[string]string{"Mode": mode}}
	}

	return nil
}

var rgxId = regexp.MustCompile(`^#([\w-]+)(?:\s*\?\s*(.*)$)?`)

func (s *scanner) scanId() *token {
	if sm := rgxId.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return &token{tokId, sm[1], map[string]string{"Condition": sm[2]}}
	}

	return nil
}

var rgxClassName = regexp.MustCompile(`^\.([\w-]+)(?:\s*\?\s*(.*)$)?`)

func (s *scanner) scanClassName() *token {
	if sm := rgxClassName.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return &token{tokClassName, sm[1], map[string]string{"Condition": sm[2]}}
	}

	return nil
}

var rgxAttribute = regexp.MustCompile(`^\[([\w\-]+)\s*(?:=\s*(\"([^\"\\]*)\"|([^\]]+)))?\](?:\s*\?\s*(.*)$)?`)

func (s *scanner) scanAttribute() *token {
	if sm := rgxAttribute.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))

		if len(sm[3]) != 0 || sm[2] == "" {
			return &token{tokAttribute, sm[1], map[string]string{"Content": sm[3], "Mode": "raw", "Condition": sm[5]}}
		}

		return &token{tokAttribute, sm[1], map[string]string{"Content": sm[4], "Mode": "expression", "Condition": sm[5]}}
	}

	return nil
}

var rgxImport = regexp.MustCompile(`^import\s+([0-9a-zA-Z_\-\. \/]*)$`)

func (s *scanner) scanImport() *token {
	if sm := rgxImport.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return &token{tokImport, sm[1], nil}
	}

	return nil
}

var rgxExtends = regexp.MustCompile(`^extends\s+([0-9a-zA-Z_\-\. \/]*)$`)

func (s *scanner) scanExtends() *token {
	if sm := rgxExtends.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return &token{tokExtends, sm[1], nil}
	}

	return nil
}

var rgxBlock = regexp.MustCompile(`^block\s+(?:(append|prepend)\s+)?([0-9a-zA-Z_\-\. \/]*)$`)

func (s *scanner) scanBlock() *token {
	if sm := rgxBlock.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return &token{tokNamedBlock, sm[2], map[string]string{"Modifier": sm[1]}}
	}

	return nil
}

var rgxTag = regexp.MustCompile(`^(\w[-:\w]*)`)

func (s *scanner) scanTag() *token {
	if sm := rgxTag.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return &token{tokTag, sm[1], nil}
	}

	return nil
}

var rgxMixin = regexp.MustCompile(`^mixin ([a-zA-Z_]+\w*)(\(((\$\w*(,\s)?)*)\))?$`)

func (s *scanner) scanMixin() *token {
	if sm := rgxMixin.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return &token{tokMixin, sm[1], map[string]string{"Args": sm[3]}}
	}

	return nil
}

var rgxMixinCall = regexp.MustCompile(`^\+([A-Za-z_]+\w*)(\((.+(,\s)?)*\))?$`)

func (s *scanner) scanMixinCall() *token {
	if sm := rgxMixinCall.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return &token{tokMixinCall, sm[1], map[string]string{"Args": sm[3]}}
	}

	return nil
}

var rgxText = regexp.MustCompile(`^(\|)? ?(.*)$`)

func (s *scanner) scanText() *token {
	if sm := rgxText.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))

		mode := "inline"
		if sm[1] == "|" {
			mode = "piped"
		}

		return &token{tokText, sm[2], map[string]string{"Mode": mode}}
	}

	return nil
}

// Moves position forward, and removes beginning of s.buffer (len bytes)
func (s *scanner) consume(runes int) {
	if len(s.buffer) < runes {
		panic(fmt.Sprintf("Unable to consume %d runes from buffer.", runes))
	}

	s.lastTokenLine = s.line
	s.lastTokenCol = s.col
	s.lastTokenSize = runes

	s.buffer = s.buffer[runes:]
	s.col += runes
}

// Reads string into s.buffer
func (s *scanner) ensureBuffer() {
	if len(s.buffer) > 0 {
		return
	}

	buf, err := s.reader.ReadString('\n')

	if err != nil && err != io.EOF {
		panic(err)
	} else if err != nil && len(buf) == 0 {
		s.state = scnEOF
	} else {
		// endline "LF only" or "\n" use Unix, Linux, modern MacOS X, FreeBSD, BeOS, RISC OS
		if buf[len(buf)-1] == '\n' {
			buf = buf[:len(buf)-1]
		}
		// endline "CR+LF" or "\r\n" use internet protocols, DEC RT-11, Windows, CP/M, MS-DOS, OS/2, Symbian OS
		if len(buf) > 0 && buf[len(buf)-1] == '\r' {
			buf = buf[:len(buf)-1]
		}

		s.state = scnNewLine
		s.buffer = buf
		s.line += 1
		s.col = 0
	}
}
