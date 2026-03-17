package lexer

import "testing"

func TestLexTextNamespace(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected []itemType
	}{
		{
			name:  "standalone namespace keyword",
			input: "namespace acme.users.v1",
			expected: []itemType{
				itemNamespace,
				itemText,
				itemEOF,
			},
		},
		{
			name:  "keyword prefix is not namespace token",
			input: "namespacex acme.users.v1",
			expected: []itemType{
				itemText,
				itemEOF,
			},
		},
		{
			name:  "text before namespace",
			input: "hello namespace acme.users.v1",
			expected: []itemType{
				itemError,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l := &lexer{
				name:      "test",
				input:     tc.input,
				start:     0,
				pos:       0,
				line:      1,
				startLine: 1,
				width:     0,
				items:     make(chan item, 8),
				state:     lexText,
			}

			var got []itemType
			for {
				it := l.nextItem()
				got = append(got, it.typ)
				if it.typ == itemEOF || it.typ == itemError {
					break
				}
			}

			if len(got) != len(tc.expected) {
				t.Fatalf("token count mismatch: got=%v expected=%v", got, tc.expected)
			}

			for i := range tc.expected {
				if got[i] != tc.expected[i] {
					t.Fatalf("token mismatch at index %d: got=%v expected=%v", i, got[i], tc.expected[i])
				}
			}
		})
	}
}

func TestLexTextNamespaceTokenValue(t *testing.T) {
	t.Parallel()

	l := &lexer{
		name:      "test",
		input:     "namespace acme.users.v1",
		start:     0,
		pos:       0,
		line:      1,
		startLine: 1,
		width:     0,
		items:     make(chan item, 8),
		state:     lexText,
	}

	first := l.nextItem()
	if first.typ != itemNamespace {
		t.Fatalf("first token type: got=%v expected=%v", first.typ, itemNamespace)
	}
	if first.val != "namespace" {
		t.Fatalf("namespace token value: got=%q expected=%q", first.val, "namespace")
	}
}
