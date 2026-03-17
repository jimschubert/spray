// Package lexer implements a lexer for the spray file format.
//
// The types and functions here are based on the text/template lexer but have
// been simplified and modified to suit the spray file format. For more details
// on this lexing approach, see "Lexical Scanning in Go" by Rob Pike:
// https://www.youtube.com/watch?v=HxaD_trXwRE or
// https://go.dev/talks/2011/lex.slide#1
package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type itemType int

const (
	// special tokens
	itemError itemType = iota
	itemUnrecognized
	itemEOF
	itemNewline

	// declarations
	itemKeywordNamespace
	itemKeywordImport
	itemKeywordType
	itemKeywordEnum
	itemKeywordModel
	itemKeywordInput
	itemKeywordAPI

	// API-related
	itemKeywordRPC
	itemKeywordStream
	itemKeywordPublish
	itemKeywordSubscribe

	// HTTP methods
	itemKeywordGET
	itemKeywordPOST
	itemKeywordPUT
	itemKeywordPATCH
	itemKeywordDELETE
	itemKeywordHEAD
	itemKeywordOPTIONS

	// decorators
	itemKeywordVersion
	itemKeywordStyle
	itemKeywordBasePath
	itemKeywordAuth
	itemKeywordBody
	itemKeywordQuery
	itemKeywordErrors
	itemKeywordSummary
	itemKeywordTag
	itemKeywordDeprecated
	itemKeywordRaw
	itemKeywordDefault
	itemKeywordPrimary
	itemKeywordUnique
	itemKeywordUpdatedAt
	itemKeywordRelation

	// API styles
	itemKeywordREST
	itemKeywordEvents

	// Auth
	itemKeywordBearer
	itemKeywordAPIKey
	itemKeywordBasic
	itemKeywordNone

	// type modifiers and literals
	itemKeywordTrue
	itemKeywordFalse
	itemKeywordNull
	itemKeywordNow
	itemKeywordUUID

	// scalars
	itemKeywordString
	itemKeywordInt
	itemKeywordFloat
	itemKeywordBoolean
	itemKeywordTimestamp
	itemKeywordDate
	itemKeywordAny

	// ident/literals
	itemIdent
	itemString
	itemInt
	itemFloat

	// other common
	itemLeftBrace
	itemRightBrace
	itemLeftParen
	itemRightParen
	itemLeftBracket
	itemRightBracket
	itemLeftAngle
	itemRightAngle
	itemColon
	itemComma
	itemDot
	itemDash
	itemArrow
	itemAt
	itemEquals
	itemQuestion
	itemSlash

	// comments
	itemComment
)

const (
	keywordNamespace  = "namespace"
	keywordImport     = "import"
	keywordType       = "type"
	keywordEnum       = "enum"
	keywordModel      = "model"
	keywordInput      = "input"
	keywordAPI        = "api"
	keywordRPC        = "rpc"
	keywordStream     = "stream"
	keywordPublish    = "publish"
	keywordSubscribe  = "subscribe"
	keywordGET        = "GET"
	keywordPOST       = "POST"
	keywordPUT        = "PUT"
	keywordPATCH      = "PATCH"
	keywordDELETE     = "DELETE"
	keywordHEAD       = "HEAD"
	keywordOPTIONS    = "OPTIONS"
	keywordVersion    = "version"
	keywordStyle      = "style"
	keywordBasePath   = "basePath"
	keywordAuth       = "auth"
	keywordBody       = "body"
	keywordQuery      = "query"
	keywordErrors     = "errors"
	keywordSummary    = "summary"
	keywordTag        = "tag"
	keywordDeprecated = "deprecated"
	keywordRaw        = "raw"
	keywordDefault    = "default"
	keywordPrimary    = "primary"
	keywordUnique     = "unique"
	keywordUpdatedAt  = "updatedAt"
	keywordRelation   = "relation"
	keywordREST       = "rest"
	keywordEvents     = "events"
	keywordBearer     = "bearer"
	keywordAPIKey     = "apiKey"
	keywordBasic      = "basic"
	keywordNone       = "none"
	keywordTrue       = "true"
	keywordFalse      = "false"
	keywordNull       = "null"
	keywordNow        = "now"
	keywordUUID       = "uuid"
	keywordString     = "string"
	keywordInt        = "int"
	keywordFloat      = "float"
	keywordBoolean    = "boolean"
	keywordTimestamp  = "timestamp"
	keywordDate       = "date"
	keywordAny        = "any"

	newline = '\n'
	eof     = -1
)

var keywords = map[string]itemType{
	keywordNamespace:  itemKeywordNamespace,
	keywordImport:     itemKeywordImport,
	keywordType:       itemKeywordType,
	keywordEnum:       itemKeywordEnum,
	keywordModel:      itemKeywordModel,
	keywordInput:      itemKeywordInput,
	keywordAPI:        itemKeywordAPI,
	keywordRPC:        itemKeywordRPC,
	keywordStream:     itemKeywordStream,
	keywordPublish:    itemKeywordPublish,
	keywordSubscribe:  itemKeywordSubscribe,
	keywordGET:        itemKeywordGET,
	keywordPOST:       itemKeywordPOST,
	keywordPUT:        itemKeywordPUT,
	keywordPATCH:      itemKeywordPATCH,
	keywordDELETE:     itemKeywordDELETE,
	keywordHEAD:       itemKeywordHEAD,
	keywordOPTIONS:    itemKeywordOPTIONS,
	keywordVersion:    itemKeywordVersion,
	keywordStyle:      itemKeywordStyle,
	keywordBasePath:   itemKeywordBasePath,
	keywordAuth:       itemKeywordAuth,
	keywordBody:       itemKeywordBody,
	keywordQuery:      itemKeywordQuery,
	keywordErrors:     itemKeywordErrors,
	keywordSummary:    itemKeywordSummary,
	keywordTag:        itemKeywordTag,
	keywordDeprecated: itemKeywordDeprecated,
	keywordRaw:        itemKeywordRaw,
	keywordDefault:    itemKeywordDefault,
	keywordPrimary:    itemKeywordPrimary,
	keywordUnique:     itemKeywordUnique,
	keywordUpdatedAt:  itemKeywordUpdatedAt,
	keywordRelation:   itemKeywordRelation,
	keywordREST:       itemKeywordREST,
	keywordEvents:     itemKeywordEvents,
	keywordBearer:     itemKeywordBearer,
	keywordAPIKey:     itemKeywordAPIKey,
	keywordBasic:      itemKeywordBasic,
	keywordNone:       itemKeywordNone,
	keywordTrue:       itemKeywordTrue,
	keywordFalse:      itemKeywordFalse,
	keywordNull:       itemKeywordNull,
	keywordNow:        itemKeywordNow,
	keywordUUID:       itemKeywordUUID,
	keywordString:     itemKeywordString,
	keywordInt:        itemKeywordInt,
	keywordFloat:      itemKeywordFloat,
	keywordBoolean:    itemKeywordBoolean,
	keywordTimestamp:  itemKeywordTimestamp,
	keywordDate:       itemKeywordDate,
	keywordAny:        itemKeywordAny,
}

type item struct {
	typ  itemType
	val  string
	pos  Pos
	line int
}

func (i item) String() string {
	switch i.typ {
	case itemError:
		return i.val
	case itemEOF:
		return "EOF"
	case itemNewline:
		return "\\n"
	case itemComment:
		return "comment"
	case itemIdent:
		return fmt.Sprintf("ident(%s)", i.val)
	case itemString:
		return fmt.Sprintf("string(%s)", i.val)
	case itemInt:
		return fmt.Sprintf("int(%s)", i.val)
	case itemFloat:
		return fmt.Sprintf("float(%s)", i.val)
	case itemUnrecognized:
		if len(i.val) > 10 {
			return fmt.Sprintf("unrecognized(%.10q...)", i.val[:10])
		}
		return fmt.Sprintf("unrecognized(%q)", i.val)
	default:
		// For all keyword and symbol tokens, just return the value
		return i.val
	}
}

// stateFn represents the state of the lexer as a function that returns the
// next state. This is based on text/template's state machine pattern.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner. This is based on text/template's
// lexer but simplified for the stencil file format.
type lexer struct {
	name      string // used only for error reports.
	input     string // the string being scanned.
	start     Pos    // start position of this item.
	pos       Pos    // current position in the input.
	line      int
	startLine int
	width     int       // width of last rune read from input.
	items     chan item // channel of scanned items.
	state     stateFn
}

// run lexes the input by executing state functions until the state is nil.
// This is the main lexing loop based on text/template's approach.
func (l *lexer) run() {
	for state := lexText; state != nil; {
		l.state = state(l)
	}
	close(l.items)
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemType) {
	l.items <- item{
		typ:  t,
		val:  l.input[l.start:l.pos],
		pos:  l.start,
		line: l.startLine,
	}
	l.start = l.pos
	l.startLine = l.line
}

func lex(name, input string) (*lexer, chan item) {
	l := &lexer{
		name:      name,
		input:     input,
		start:     0,
		pos:       0,
		line:      1,
		startLine: 1,
		width:     0,
		items:     make(chan item, 2),
		state:     lexText,
	}

	// go l.run()

	return l, l.items
}

// lexText is the initial state that scans for keywords like "namespace" at
// the start of the file. When a keyword is found, it transitions to the
// appropriate handler state. Otherwise it treats content as plain text.
func lexText(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], keywordNamespace) && l.atKeywordBoundary(len(keywordNamespace)) {
			if l.pos > l.start {
				// namespace must be the first token (ignoring whitespace/comments)
				return l.errorf("unexpected text before namespace keyword")
			}
			return lexNamespace
		}
		if l.next() == eof {
			break
		}
	}

	if l.pos > l.start {
		l.emit(itemText)
	}
	l.emit(itemEOF)
	return nil
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

// nextItem returns the next item from the input.
func (l *lexer) nextItem() item {
	for {
		select {
		case item := <-l.items:
			return item
		default:
			l.state = l.state(l)
		}
	}
}

// ignore skips over the pending input before this point, discarding it.
func (l *lexer) ignore() {
	l.start = l.pos
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

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

// errorf returns an error token and terminates the scan by passing back a
// nil pointer that will be the next state, which is expected to terminate the caller.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{
		typ:  itemError,
		val:  fmt.Sprintf(format, args...),
		pos:  l.start,
		line: l.startLine,
	}
	return nil
}

// lexNamespace consumes the namespace keyword and emits it as a token.
func lexNamespace(l *lexer) stateFn {
	l.pos += Pos(len(keywordNamespace))
	l.emit(itemNamespace)
	return lexText
}

// atKeywordBoundary checks whether a keyword at l.pos is followed by a word
// boundary. This ensures "namespace" is recognized as a keyword but
// "namespacex" is not. It returns true if the keyword is at EOF or followed
// by a non-identifier rune.
func (l *lexer) atKeywordBoundary(keywordLength int) bool {
	end := int(l.pos) + keywordLength
	if end >= len(l.input) {
		// keyword reaches EOF
		return true
	}

	nextRune, _ := utf8.DecodeRuneInString(l.input[end:])
	// keyword must be followed by a non-identifier character
	return !isIdentifierRune(nextRune)
}

// isIdentifierRune reports whether r is a valid identifier continuation, i.e., one of:
// underscore, letter, or digit.
func isIdentifierRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
