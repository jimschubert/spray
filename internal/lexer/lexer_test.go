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

func TestLexPositions(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []struct {
			typ  itemType
			val  string
			line int
			col  int
		}
	}{
		{
			name:  "single line positions",
			input: "namespace acme\n",
			expected: []struct {
				typ  itemType
				val  string
				line int
				col  int
			}{
				{
					typ:  itemKeywordNamespace,
					val:  "namespace",
					line: 1,
					col:  0,
				},
				{
					typ:  itemIdent,
					val:  "acme",
					line: 1,
					col:  10,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 1,
					col:  14,
				},
			},
		},
		{
			name:  "multiline positions",
			input: "model User {\n  id: Foo\n}\n",
			expected: []struct {
				typ  itemType
				val  string
				line int
				col  int
			}{
				{
					typ:  itemKeywordModel,
					val:  "model",
					line: 1,
					col:  0,
				},
				{
					typ:  itemIdent,
					val:  "User",
					line: 1,
					col:  6,
				},
				{
					typ:  itemLeftBrace,
					val:  "{",
					line: 1,
					col:  11,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 1,
					col:  12,
				},
				{
					typ:  itemIdent,
					val:  "id",
					line: 2,
					col:  2,
				},
				{
					typ:  itemColon,
					val:  ":",
					line: 2,
					col:  4,
				},
				{
					typ:  itemIdent,
					val:  "Foo",
					line: 2,
					col:  6,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 2,
					col:  9,
				},
				{
					typ:  itemRightBrace,
					val:  "}",
					line: 3,
					col:  0,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 3,
					col:  1,
				},
			},
		},
		{
			name:  "identifier at end of line has correct column",
			input: "  foo\nbar\n",
			expected: []struct {
				typ  itemType
				val  string
				line int
				col  int
			}{
				{
					typ:  itemIdent,
					val:  "foo",
					line: 1,
					col:  2,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 1,
					col:  5,
				},
				{
					typ:  itemIdent,
					val:  "bar",
					line: 2,
					col:  0,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 2,
					col:  3,
				},
			},
		},
		{
			name:  "number at end of line has correct column",
			input: "  42\n7\n",
			expected: []struct {
				typ  itemType
				val  string
				line int
				col  int
			}{
				{
					typ:  itemInt,
					val:  "42",
					line: 1,
					col:  2,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 1,
					col:  4,
				},
				{
					typ:  itemInt,
					val:  "7",
					line: 2,
					col:  0,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 2,
					col:  1,
				},
			},
		},
		{
			name:  "comment positions",
			input: "namespace acme # comment\nmodel X {\n}\n",
			expected: []struct {
				typ  itemType
				val  string
				line int
				col  int
			}{
				{
					typ:  itemKeywordNamespace,
					val:  "namespace",
					line: 1,
					col:  0,
				},
				{
					typ:  itemIdent,
					val:  "acme",
					line: 1,
					col:  10,
				},
				{
					typ:  itemComment,
					val:  "# comment",
					line: 1,
					col:  15,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 1,
					col:  24,
				},
				{
					typ:  itemKeywordModel,
					val:  "model",
					line: 2,
					col:  0,
				},
				{
					typ:  itemIdent,
					val:  "X",
					line: 2,
					col:  6,
				},
				{
					typ:  itemLeftBrace,
					val:  "{",
					line: 2,
					col:  8,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 2,
					col:  9,
				},
				{
					typ:  itemRightBrace,
					val:  "}",
					line: 3,
					col:  0,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 3,
					col:  1,
				},
			},
		},
		{
			name:  "decorator positions",
			input: "  @default(\"x\")\n",
			expected: []struct {
				typ  itemType
				val  string
				line int
				col  int
			}{
				{
					typ:  itemAt,
					val:  "@",
					line: 1,
					col:  2,
				},
				{
					typ:  itemIdent,
					val:  "default",
					line: 1,
					col:  3,
				},
				{
					typ:  itemLeftParen,
					val:  "(",
					line: 1,
					col:  10,
				},
				{
					typ:  itemString,
					val:  `"x"`,
					line: 1,
					col:  11,
				},
				{
					typ:  itemRightParen,
					val:  ")",
					line: 1,
					col:  14,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 1,
					col:  15,
				},
			},
		},
		{
			name:  "blank lines reset column correctly",
			input: "namespace test\n\n  model User {\n  }\n",
			expected: []struct {
				typ  itemType
				val  string
				line int
				col  int
			}{
				{
					typ:  itemKeywordNamespace,
					val:  "namespace",
					line: 1,
					col:  0,
				},
				{
					typ:  itemIdent,
					val:  "test",
					line: 1,
					col:  10,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 1,
					col:  14,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 2,
					col:  0,
				},
				{
					typ:  itemKeywordModel,
					val:  "model",
					line: 3,
					col:  2,
				},
				{
					typ:  itemIdent,
					val:  "User",
					line: 3,
					col:  8,
				},
				{
					typ:  itemLeftBrace,
					val:  "{",
					line: 3,
					col:  13,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 3,
					col:  14,
				},
				{
					typ:  itemRightBrace,
					val:  "}",
					line: 4,
					col:  2,
				},
				{
					typ:  itemNewline,
					val:  "\n",
					line: 4,
					col:  3,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l := lex("test", tc.input)
			l.run()

			for i, expected := range tc.expected {
				got := l.nextItem()
				assert.Equal(t, expected.typ, got.typ, "token %d type", i)
				assert.Equal(t, expected.val, got.val, "token %d val", i)
				assert.Equal(t, expected.line, got.line, "token %d line (val=%q)", i, got.val)
				assert.Equal(t, expected.col, l.columnOf(got.pos), "token %d col (val=%q)", i, got.val)
			}
		})
	}
}

