package lexer

import "fmt"

// Pos is the position of a token in the input. It is an integer offset from the start of the input.
type Pos int

func (p Pos) Position() Pos {
	return p
}

// item represents a token or text fragment returned from the lexer.
type item struct {
	typ  itemType
	val  string
	pos  Pos
	line int
}

// String is the fmt.Stringer interface
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

// itemType is the set of lexical tokens
type itemType int

//nolint:unused
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

	// Decorator keywords (not in global keywords map - lexed as identifiers, recognized in parseDecorator)
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
	itemKeywordVoid

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
	keywordNamespace = "namespace"
	keywordImport    = "import"
	keywordType      = "type"
	keywordEnum      = "enum"
	keywordModel     = "model"
	keywordInput     = "input"
	keywordAPI       = "api"
	keywordRPC       = "rpc"
	keywordStream    = "stream"
	keywordPublish   = "publish"
	keywordSubscribe = "subscribe"
	keywordGET       = "GET"
	keywordPOST      = "POST"
	keywordPUT       = "PUT"
	keywordPATCH     = "PATCH"
	keywordDELETE    = "DELETE"
	keywordHEAD      = "HEAD"
	keywordOPTIONS   = "OPTIONS"
	keywordVersion   = "version"
	keywordStyle     = "style"
	keywordBasePath  = "basePath"
	keywordAuth      = "auth"
	keywordBody      = "body"
	keywordQuery     = "query"
	keywordErrors    = "errors"
	keywordSummary   = "summary"
	keywordTag       = "tag"
	// keywordDeprecated = "deprecated"
	// keywordRaw        = "raw"
	// keywordDefault    = "default"
	// keywordPrimary    = "primary"
	// keywordUnique     = "unique"
	// keywordUpdatedAt  = "updatedAt"
	// keywordRelation   = "relation"
	keywordREST      = "rest"
	keywordEvents    = "events"
	keywordBearer    = "bearer"
	keywordAPIKey    = "apiKey"
	keywordBasic     = "basic"
	keywordNone      = "none"
	keywordTrue      = "true"
	keywordFalse     = "false"
	keywordNull      = "null"
	keywordNow       = "now"
	keywordUUID      = "uuid"
	keywordString    = "string"
	keywordInt       = "int"
	keywordFloat     = "float"
	keywordBoolean   = "boolean"
	keywordTimestamp = "timestamp"
	keywordDate      = "date"
	keywordAny       = "any"
	keywordVoid      = "void"

	newline        = '\n'
	carriageReturn = '\r'
	eof            = -1

	digits = "0123456789"

	// human readable token descriptions
	itemIdentString  = "identifier"
	itemStringString = "string"
	itemIntString    = "integer"
	itemFloatString  = "float"

	itemLeftBraceString    = "{"
	itemRightBraceString   = "}"
	itemLeftParenString    = "("
	itemRightParenString   = ")"
	itemLeftBracketString  = "["
	itemRightBracketString = "]"
	itemLeftAngleString    = "<"
	itemRightAngleString   = ">"
	itemColonString        = ":"
	itemCommaString        = ","
	itemDotString          = "."
	itemDashString         = "-"
	itemArrowString        = "->"
	itemAtString           = "@"
	itemEqualsString       = "="
	itemQuestionString     = "?"
	itemSlashString        = "/"
)

var keywords = map[string]itemType{
	keywordNamespace: itemKeywordNamespace,
	keywordImport:    itemKeywordImport,
	keywordType:      itemKeywordType,
	keywordEnum:      itemKeywordEnum,
	keywordModel:     itemKeywordModel,
	keywordInput:     itemKeywordInput,
	keywordAPI:       itemKeywordAPI,
	keywordRPC:       itemKeywordRPC,
	keywordStream:    itemKeywordStream,
	keywordPublish:   itemKeywordPublish,
	keywordSubscribe: itemKeywordSubscribe,
	keywordGET:       itemKeywordGET,
	keywordPOST:      itemKeywordPOST,
	keywordPUT:       itemKeywordPUT,
	keywordPATCH:     itemKeywordPATCH,
	keywordDELETE:    itemKeywordDELETE,
	keywordHEAD:      itemKeywordHEAD,
	keywordOPTIONS:   itemKeywordOPTIONS,
	// Decorator-only keywords are NOT included here - they're only recognized after @
	// Model field decorators: keywordDeprecated, keywordRaw, keywordDefault, keywordPrimary,
	//   keywordUnique, keywordUpdatedAt, keywordRelation
	// API/route decorators: keywordVersion, keywordStyle, keywordBasePath, keywordAuth,
	//   keywordBody, keywordQuery, keywordErrors, keywordSummary, keywordTag
	// These are all handled in parseDecorator()
	keywordREST:      itemKeywordREST,
	keywordEvents:    itemKeywordEvents,
	keywordBearer:    itemKeywordBearer,
	keywordAPIKey:    itemKeywordAPIKey,
	keywordBasic:     itemKeywordBasic,
	keywordNone:      itemKeywordNone,
	keywordTrue:      itemKeywordTrue,
	keywordFalse:     itemKeywordFalse,
	keywordNull:      itemKeywordNull,
	keywordNow:       itemKeywordNow,
	keywordUUID:      itemKeywordUUID,
	keywordString:    itemKeywordString,
	keywordInt:       itemKeywordInt,
	keywordFloat:     itemKeywordFloat,
	keywordBoolean:   itemKeywordBoolean,
	keywordTimestamp: itemKeywordTimestamp,
	keywordDate:      itemKeywordDate,
	keywordAny:       itemKeywordAny,
	keywordVoid:      itemKeywordVoid,
}

var symbolsDescriptions = func() map[itemType]string {
	sym := map[itemType]string{
		itemIdent:        itemIdentString,
		itemString:       itemStringString,
		itemInt:          itemIntString,
		itemFloat:        itemFloatString,
		itemLeftBrace:    itemLeftBraceString,
		itemRightBrace:   itemRightBraceString,
		itemLeftParen:    itemLeftParenString,
		itemRightParen:   itemRightParenString,
		itemLeftBracket:  itemLeftBracketString,
		itemRightBracket: itemRightBracketString,
		itemLeftAngle:    itemLeftAngleString,
		itemRightAngle:   itemRightAngleString,
		itemColon:        itemColonString,
		itemComma:        itemCommaString,
		itemDot:          itemDotString,
		itemDash:         itemDashString,
		itemArrow:        itemArrowString,
		itemAt:           itemAtString,
		itemEquals:       itemEqualsString,
		itemQuestion:     itemQuestionString,
		itemSlash:        itemSlashString,
	}

	for key := range keywords {
		sym[keywords[key]] = key
	}

	return sym
}()
