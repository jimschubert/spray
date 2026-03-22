// Package lexer implements a lexer for the spray file format.
//
// The types and functions here are based on the text/template lexer but have
// been simplified and modified to suit the spray file format. For more details
// on this lexing approach, see "Lexical Scanning in Go" by Rob Pike:
// https://www.youtube.com/watch?v=HxaD_trXwRE or
// https://go.dev/talks/2011/lex.slide#1
//
// The lexer operates on a single input string without copying. It uses position
// indices (start and pos) to track token boundaries within the input string.
// Token values are string slices of the original input (l.input[l.start:l.pos]),
// which avoids allocations for token data. This makes the lexer memory-efficient
// even for large input files.
//
// The lexer accumulates all tokens in a slice during a single synchronous pass.
// Call lex() to create a lexer and get the items slice, then use nextItem() to
// iterate through the tokens.
package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// keywordsByFirstChar buckets keywords by their first character for efficient lookup.
// This reduces average keyword matching from O(keyword_count) to O(keywords_per_bucket).
var keywordsByFirstChar = func() map[byte][]string {
	m := make(map[byte][]string)
	for keyword := range keywords {
		first := keyword[0]
		m[first] = append(m[first], keyword)
	}
	return m
}()

// stateFn represents the state of the lexer as a function that returns the
// next state. This is based on text/template's state machine pattern.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner. This is based on text/template's
// lexer but simplified for the spray file format.
type lexer struct {
	name      string // used only for error reports.
	input     string // the string being scanned.
	start     Pos    // start position of this item.
	pos       Pos    // current position in the input.
	line      int
	startLine int
	width     int    // width of last rune read from input.
	items     []item // accumulated items
	index     int    // current position in items slice for iteration
	state     stateFn
}

// run lexes the input by executing state functions until the state is nil.
// This is the main lexing loop based on text/template's approach.
func (l *lexer) run() {
	for l.state = lexText; l.state != nil; {
		l.state = l.state(l)
	}
}

// emit appends an item to the items slice.
func (l *lexer) emit(t itemType) {
	l.items = append(l.items, item{
		typ:  t,
		val:  l.input[l.start:l.pos],
		pos:  l.start,
		line: l.startLine,
	})
	l.start = l.pos
	l.startLine = l.line
}

// lex creates and returns a new lexer for the given input. Call l.run() to
// lex the input synchronously, then use l.nextItem() to iterate over tokens.
func lex(name, input string) *lexer {
	return &lexer{
		name:      name,
		input:     input,
		start:     0,
		pos:       0,
		line:      1,
		startLine: 1,
		width:     0,
		items:     make([]item, 0),
		index:     0,
		state:     lexText,
	}
}

// lexText is the initial state that scans for keywords, operators, and other tokens.
func lexText(l *lexer) stateFn {
	for {
		l.skipWhitespace()

		// start of next token
		l.start = l.pos

		r := l.peek()

		if r == newline || r == carriageReturn {
			if r == carriageReturn {
				l.next() // consumes \r
				if l.peek() == newline {
					l.next() // consumes \n
				}
			} else {
				l.next() // consumes \n
			}
			l.emit(itemNewline)
			continue
		}

		if r == eof {
			l.emit(itemEOF)
			return nil
		}

		if r == '#' {
			return lexComment
		}

		if keywordToken := l.tryMatchKeyword(); keywordToken != itemError {
			l.emit(keywordToken)
			continue
		}

		if operatorToken := l.tryMatchSymbol(); operatorToken != itemError {
			l.emit(operatorToken)
			continue
		}

		if r == '"' {
			return lexString
		}

		if r >= '0' && r <= '9' {
			return lexNumber
		}

		if isIdentifierRune(r) {
			return lexIdentifier
		}

		l.next()
		return l.errorf("unexpected character %q", r)
	}
}

// skipWhitespace skips over spaces without emitting tokens
func (l *lexer) skipWhitespace() {
	for {
		r := l.peek()
		if r != ' ' && r != '\t' {
			break
		}
		l.next()
	}
}

// tryMatchKeyword attempts to match any keyword at the current position.
// Keywords are bucketed by first character for efficient lookup. If a keyword
// matches, it advances the position and returns the corresponding token type.
// Otherwise returns itemError.
func (l *lexer) tryMatchKeyword() itemType {
	r := l.peek()
	if r == eof {
		return itemError
	}

	bucket, ok := keywordsByFirstChar[byte(r)]
	if !ok {
		return itemError
	}

	for _, keyword := range bucket {
		if strings.HasPrefix(l.input[int(l.pos):], keyword) &&
			l.isWordBoundary(len(keyword)) {
			l.pos += Pos(len(keyword))
			return keywords[keyword]
		}
	}

	return itemError
}

// tryMatchSymbol attempts to match an operator or delimiter at the current
// position. If one matches, it advances the position and returns the token type.
// Otherwise returns itemError.
func (l *lexer) tryMatchSymbol() itemType {
	r := l.peek()
	switch r {
	case '{':
		l.next()
		return itemLeftBrace
	case '}':
		l.next()
		return itemRightBrace
	case '(':
		l.next()
		return itemLeftParen
	case ')':
		l.next()
		return itemRightParen
	case '[':
		l.next()
		return itemLeftBracket
	case ']':
		l.next()
		return itemRightBracket
	case '<':
		l.next()
		return itemLeftAngle
	case '>':
		l.next()
		return itemRightAngle
	case ':':
		l.next()
		return itemColon
	case ',':
		l.next()
		return itemComma
	case '.':
		l.next()
		return itemDot
	case '@':
		l.next()
		return itemAt
	case '=':
		l.next()
		return itemEquals
	case '?':
		l.next()
		return itemQuestion
	case '/':
		l.next()
		return itemSlash
	case '-':
		l.next() // consumes -
		next := l.peek()
		// special case for -> operator
		if next == '>' {
			l.next() // consumes >
			return itemArrow
		}
		return itemDash
	}

	return itemError
}

// lexComment consumes a comment starting with # and emits it as a token.
func lexComment(l *lexer) stateFn {
	l.next() // consumes '#'
	for {
		r := l.peek()
		if r == newline || r == carriageReturn || r == eof {
			// a comment _can_ just be # on a line by itself
			break
		}
		l.next()
	}
	l.emit(itemComment)
	return lexText
}

// lexString consumes a string literal and emits it as a token.
// Escape characters (\n, \t, \r, \\, \", \/) are passed through to the parserState for validation.
func lexString(l *lexer) stateFn {
	l.next() // consumes open quote
	for {
		r := l.next()
		if r == eof {
			return l.errorf("unterminated string literal")
		}
		if r == '"' {
			l.emit(itemString)
			return lexText
		}
		// escapes char (validation will be done in the parserState)
		if r == '\\' {
			next := l.next()
			if next == eof {
				return l.errorf("unterminated string literal (backslash at end)")
			}
		}
	}
}

// lexNumber consumes a number (int or float) and emits it as a token.
func lexNumber(l *lexer) stateFn {
	// consume integer part
	for {
		r := l.next()
		if r == eof || !strings.ContainsRune(digits, r) {
			break
		}
	}
	l.backup()

	// check for decimal point
	if l.peek() == '.' && int(l.pos)+1 < len(l.input) {
		r, _ := utf8.DecodeRuneInString(l.input[int(l.pos)+1:])
		if strings.ContainsRune(digits, r) {
			l.next() // consumes dot
			for {
				r := l.next()
				if r == eof || !strings.ContainsRune(digits, r) {
					break
				}
			}
			l.backup()
			l.emit(itemFloat)
			return lexText
		}
	}

	l.emit(itemInt)
	return lexText
}

// lexIdentifier consumes an identifier and emits it as a token.
func lexIdentifier(l *lexer) stateFn {
	for {
		r := l.next()
		if r == eof || !isIdentifierRune(r) {
			break
		}
	}
	l.backup()
	l.emit(itemIdent)
	return lexText
}

// next returns the next rune in the input, updating position and line count.
func (l *lexer) next() (rune rune) {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	rune, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += Pos(l.width)
	if rune == newline {
		l.line++
	}
	return rune
}

// nextItem returns the next item from the items slice.
func (l *lexer) nextItem() item {
	if l.index < len(l.items) {
		item := l.items[l.index]
		l.index++
		return item
	}
	return item{typ: itemEOF}
}

// backup steps back one rune and adjusts line count if needed.
// Can be called only once per call of next.
func (l *lexer) backup() {
	l.pos -= Pos(l.width)
	if l.width > 0 {
		r, _ := utf8.DecodeRuneInString(l.input[l.pos : l.pos+Pos(l.width)])
		if r == newline {
			l.line--
		}
	}
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// errorf returns an error token and terminates the scan by passing back a
// nil pointer that will be the next state, terminating l.run.
func (l *lexer) errorf(format string, args ...any) stateFn {
	l.items = append(l.items, item{
		typ:  itemError,
		val:  fmt.Sprintf(format, args...),
		pos:  l.start,
		line: l.startLine,
	})
	return nil
}

// isWordBoundary checks whether a token at l.pos is followed by a word boundary, functionally similar to regex \b.
// So, "namespace" is valid but "namespacex" is not. Returns true if the token is at EOF or the next char is not one of:
// letter, digit, or underscore.
func (l *lexer) isWordBoundary(tokenLength int) bool {
	end := int(l.pos) + tokenLength
	if end >= len(l.input) {
		// token reaches EOF
		return true
	}

	nextRune, _ := utf8.DecodeRuneInString(l.input[end:])
	// token must be followed by a non-identifier character
	return !isIdentifierRune(nextRune)
}

// isIdentifierRune reports whether r is a valid identifier continuation, i.e., one of:
// underscore, letter, or digit.
func isIdentifierRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
