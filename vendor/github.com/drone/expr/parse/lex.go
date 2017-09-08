package parse

import (
	"unicode"
	"unicode/utf8"
)

// token is a lexical token.
type token uint

// list of lexical tokens.
const (
	// special tokens
	tokenIllegal token = iota
	tokenEOF

	// identifiers and basic type literals
	tokenIdent
	tokenText
	tokenReal
	tokenInteger

	// operators and delimiters
	tokenEq     // ==
	tokenLt     // <
	tokenLte    // <=
	tokenGt     // >
	tokenGte    // >=
	tokenNeq    // !=
	tokenComma  // ,
	tokenLparen // (
	tokenRparen // )

	// keywords
	tokenNot
	tokenAnd
	tokenOr
	tokenIn
	tokenGlob
	tokenRegexp
	tokenTrue
	tokenFalse
)

// lexer implements a lexical scanner that reads unicode characters
// and tokens from a byte buffer.
type lexer struct {
	buf   []byte
	pos   int
	start int
	width int
}

// scan reads the next token or Unicode character from source and
// returns it. It returns EOF at the end of the source.
func (l *lexer) scan() token {
	l.start = l.pos
	l.skipWhitespace()

	r := l.read()
	switch {
	case isIdent(r):
		l.unread()
		return l.scanIdent()
	case isQuote(r):
		l.unread()
		return l.scanQuote()
	case isNumeric(r):
		l.unread()
		return l.scanNumber()
	case isCompare(r):
		l.unread()
		return l.scanCompare()
	}

	switch r {
	case eof:
		return tokenEOF
	case '(':
		return tokenLparen
	case ')':
		return tokenRparen
	case ',':
		return tokenComma
	}

	return tokenIllegal
}

// peek reads the next token or Unicode character from source and
// returns it without advancing the scanner.
func (l *lexer) peek() token {
	var (
		pos   = l.pos
		start = l.start
		width = l.width
	)
	tok := l.scan()
	l.pos = pos
	l.start = start
	l.width = width
	return tok
}

// bytes returns the bytes corresponding to the most recently scanned
// token. Valid after calling Scan().
func (l *lexer) bytes() []byte {
	return l.buf[l.start:l.pos]
}

// string returns the string corresponding to the most recently scanned
// token. Valid after calling Scan().
func (l *lexer) string() string {
	return string(l.bytes())
}

// init initializes a scanner with a new buffer.
func (l *lexer) init(buf []byte) {
	l.buf = buf
	l.pos = 0
	l.start = 0
	l.width = 0
}

func (l *lexer) scanIdent() token {
	for {
		if r := l.read(); r == eof {
			break
		} else if !isIdent(r) {
			l.unread()
			break
		}
	}

	ident := l.bytes()
	switch string(ident) {
	case "NOT", "not":
		return tokenNot
	case "AND", "and":
		return tokenAnd
	case "OR", "or":
		return tokenOr
	case "IN", "in":
		return tokenIn
	case "GLOB", "glob":
		return tokenGlob
	case "REGEXP", "regexp":
		return tokenRegexp
	case "TRUE", "true":
		return tokenTrue
	case "FALSE", "false":
		return tokenFalse
	}

	return tokenIdent
}

func (l *lexer) scanQuote() (tok token) {
	l.read() // consume first quote

	for {
		if r := l.read(); r == eof {
			return tokenIllegal
		} else if isQuote(r) {
			break
		}
	}
	return tokenText
}

func (l *lexer) scanNumber() token {
	for {
		if r := l.read(); r == eof {
			break
		} else if !isNumeric(r) {
			l.unread()
			break
		}
	}
	return tokenInteger
}

func (l *lexer) scanCompare() (tok token) {
	switch l.read() {
	case '=':
		tok = tokenEq
	case '!':
		tok = tokenNeq
	case '>':
		tok = tokenGt
	case '<':
		tok = tokenLt
	}

	r := l.read()
	switch {
	case tok == tokenGt && r == '=':
		tok = tokenGte
	case tok == tokenLt && r == '=':
		tok = tokenLte
	case tok == tokenEq && r == '=':
		tok = tokenEq
	case tok == tokenNeq && r == '=':
		tok = tokenNeq
	case tok == tokenNeq && r != '=':
		tok = tokenIllegal
	default:
		l.unread()
	}
	return
}

func (l *lexer) skipWhitespace() {
	for {
		if r := l.read(); r == eof {
			break
		} else if !isWhitespace(r) {
			l.unread()
			break
		}
	}
	l.ignore()
}

func (l *lexer) read() rune {
	if l.pos >= len(l.buf) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRune(l.buf[l.pos:])
	l.width = w
	l.pos += l.width
	return r
}

func (l *lexer) unread() {
	l.pos -= l.width
}

func (l *lexer) ignore() {
	l.start = l.pos
}

// eof rune sent when end of file is reached
var eof = rune(0)

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
}

func isNumeric(r rune) bool {
	return unicode.IsDigit(r) || r == '.'
}

func isQuote(r rune) bool {
	return r == '\''
}

func isCompare(r rune) bool {
	return r == '=' || r == '!' || r == '>' || r == '<'
}

func isIdent(r rune) bool {
	return unicode.IsLetter(r) || r == '_' || r == '-'
}
