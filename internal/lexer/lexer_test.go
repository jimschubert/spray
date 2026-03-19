package lexer

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestLexNamespace(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []struct {
			typ itemType
			val string
		}
	}{
		{
			name:  "simple namespace declaration",
			input: "namespace acme\n",
			expected: []struct {
				typ itemType
				val string
			}{
				{itemKeywordNamespace, "namespace"},
				{itemIdent, "acme"},
				{itemNewline, "\n"},
				{itemEOF, ""},
			},
		},
		{
			name:  "qualified namespace",
			input: "namespace acme.users.v1\n",
			expected: []struct {
				typ itemType
				val string
			}{
				{itemKeywordNamespace, "namespace"},
				{itemIdent, "acme"},
				{itemDot, "."},
				{itemIdent, "users"},
				{itemDot, "."},
				{itemIdent, "v1"},
				{itemNewline, "\n"},
				{itemEOF, ""},
			},
		},
		{
			name:  "namespace with comment",
			input: "namespace acme # our namespace\n",
			expected: []struct {
				typ itemType
				val string
			}{
				{itemKeywordNamespace, "namespace"},
				{itemIdent, "acme"},
				{itemComment, "# our namespace"},
				{itemNewline, "\n"},
				{itemEOF, ""},
			},
		},
		{
			name:  "keyword prefix is not namespace",
			input: "namespacex acme\n",
			expected: []struct {
				typ itemType
				val string
			}{
				{itemIdent, "namespacex"},
				{itemIdent, "acme"},
				{itemNewline, "\n"},
				{itemEOF, ""},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l := lex("test", tc.input)
			l.run()

			for _, expected := range tc.expected {
				got := l.nextItem()
				assert.Equal(t, expected.typ, got.typ)
				assert.Equal(t, expected.val, got.val)
			}
		})
	}
}

func TestLexOperators(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []itemType
	}{
		{
			name:  "braces",
			input: "{ }",
			expected: []itemType{
				itemLeftBrace, itemRightBrace, itemEOF,
			},
		},
		{
			name:  "parentheses",
			input: "( )",
			expected: []itemType{
				itemLeftParen, itemRightParen, itemEOF,
			},
		},
		{
			name:  "brackets",
			input: "[ ]",
			expected: []itemType{
				itemLeftBracket, itemRightBracket, itemEOF,
			},
		},
		{
			name:  "arrow",
			input: "->",
			expected: []itemType{
				itemArrow, itemEOF,
			},
		},
		{
			name:  "various operators",
			input: ": , . @ = ? /",
			expected: []itemType{
				itemColon, itemComma, itemDot, itemAt, itemEquals, itemQuestion, itemSlash, itemEOF,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l := lex("test", tc.input)
			l.run()

			for _, expected := range tc.expected {
				got := l.nextItem()
				assert.Equal(t, expected, got.typ)
			}
		})
	}
}

func TestLexNumbers(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []struct {
			typ itemType
			val string
		}
	}{
		{
			name:  "integer",
			input: "123",
			expected: []struct {
				typ itemType
				val string
			}{
				{itemInt, "123"},
				{itemEOF, ""},
			},
		},
		{
			name:  "float",
			input: "3.14",
			expected: []struct {
				typ itemType
				val string
			}{
				{itemFloat, "3.14"},
				{itemEOF, ""},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l := lex("test", tc.input)
			l.run()

			for _, expected := range tc.expected {
				got := l.nextItem()
				assert.Equal(t, expected.typ, got.typ)
				assert.Equal(t, expected.val, got.val)
			}
		})
	}
}

func TestLexIdentifiers(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []struct {
			typ itemType
			val string
		}
	}{
		{
			name:  "simple identifier",
			input: "foo",
			expected: []struct {
				typ itemType
				val string
			}{
				{itemIdent, "foo"},
				{itemEOF, ""},
			},
		},
		{
			name:  "identifier with underscores",
			input: "foo_bar_baz",
			expected: []struct {
				typ itemType
				val string
			}{
				{itemIdent, "foo_bar_baz"},
				{itemEOF, ""},
			},
		},
		{
			name:  "identifier starting with underscore",
			input: "_internal",
			expected: []struct {
				typ itemType
				val string
			}{
				{itemIdent, "_internal"},
				{itemEOF, ""},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l := lex("test", tc.input)
			l.run()

			for _, expected := range tc.expected {
				got := l.nextItem()
				assert.Equal(t, expected.typ, got.typ)
				assert.Equal(t, expected.val, got.val)
			}
		})
	}
}

func TestLexStrings(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []struct {
			typ itemType
			val string
		}
	}{
		{
			name:  "simple string",
			input: `"hello"`,
			expected: []struct {
				typ itemType
				val string
			}{
				{itemString, `"hello"`},
				{itemEOF, ""},
			},
		},
		{
			name:  "string with spaces",
			input: `"hello world"`,
			expected: []struct {
				typ itemType
				val string
			}{
				{itemString, `"hello world"`},
				{itemEOF, ""},
			},
		},
		{
			name:  "string with escape sequences",
			input: `"hello\nworld\t!"`,
			expected: []struct {
				typ itemType
				val string
			}{
				{itemString, `"hello\nworld\t!"`},
				{itemEOF, ""},
			},
		},
		{
			name:  "string with escaped quote",
			input: `"say \"hi\""`,
			expected: []struct {
				typ itemType
				val string
			}{
				{itemString, `"say \"hi\""`},
				{itemEOF, ""},
			},
		},
		{
			name:  "string with backslash",
			input: `"path\\to\\file"`,
			expected: []struct {
				typ itemType
				val string
			}{
				{itemString, `"path\\to\\file"`},
				{itemEOF, ""},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l := lex("test", tc.input)
			l.run()

			for _, expected := range tc.expected {
				got := l.nextItem()
				assert.Equal(t, expected.typ, got.typ)
				assert.Equal(t, expected.val, got.val)
			}
		})
	}
}
