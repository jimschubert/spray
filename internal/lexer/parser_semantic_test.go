package lexer

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestParser_semanticValidation(t *testing.T) {
	tests := []struct {
		name                string
		input               string
		wantErr             bool
		expectErrorIncludes []string
	}{
		{
			name: "duplicate model names",
			input: `namespace test

model User {
  id: uuid
}

model User {
  id: uuid
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`duplicate definition: "User" [line: 7, col: 6]`,
			},
		},
		{
			name: "duplicate input names",
			input: `namespace test

input User {
  id: uuid
}

input User {
  id: uuid
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`duplicate definition: "User" [line: 7, col: 6]`,
			},
		},
		{
			name: "duplicate across model and input names",
			input: `namespace test

model User {
  id: uuid
}

input User {
  id: uuid
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`duplicate definition: "User" [line: 7, col: 6]`,
			},
		},
		{
			name: "duplicate across model and enum names",
			input: `namespace test

model User {
  id: uuid
}

enum User {
  bob
  mary
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`duplicate definition: "User" [line: 7, col: 5]`,
			},
		},
		{
			name: "duplicate enum names",
			input: `namespace test

enum Status {
  active
  inactive
}

enum Status {
  pending
  done
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`duplicate definition: "Status" [line: 8, col: 5]`,
			},
		},
		{
			name: "duplicate type alias names",
			input: `namespace test

type ID = uuid
type ID = string
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`duplicate definition: "ID" [line: 4, col: 5]`,
			},
		},
		{
			name: "duplicate api names",
			input: `namespace test

api Users @style(rest) {
  GET /users -> User[]
}

api Users @style(rest) {
  GET /items -> Item[]
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`duplicate definition: "Users" [line: 7, col: 4]`,
			},
		},
		{
			name: "duplicate across type alias and model",
			input: `namespace test

type Foo = string

model Foo {
  id: uuid
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`duplicate definition: "Foo" [line: 5, col: 6]`,
			},
		},
		{
			name: "no duplicate when names differ",
			input: `namespace test

model User {
  id: uuid
}

model Account {
  id: uuid
}
`,
			wantErr: false,
		},
		{
			name: "multiple duplicate definitions reports all",
			input: `namespace test

model Dup {
  id: uuid
}

model Dup {
  id: uuid
}

model Dup {
  id: uuid
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`duplicate definition: "Dup" [line: 7, col: 6]`,
				`duplicate definition: "Dup" [line: 11, col: 6]`,
			},
		},

		{
			name: "valid model decorators are accepted",
			input: `namespace test

model User {
  id: uuid @primary
  email: string @unique
  name: string @default(anon)
  updated: timestamp @updatedAt
  login: string @deprecated(msg)
}
`,
			wantErr: false,
		},
		{
			name: "invalid decorator on model field",
			input: `namespace test

model User {
  id: uuid @version(1)
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`invalid decorator "version" for model field "id"`,
				`model supports: @primary, @unique, @default, @updatedAt, @relation, @deprecated, @raw`,
			},
		},
		{
			name: "multiple invalid decorators on model fields",
			input: `namespace test

model User {
  id: uuid @style(rest)
  name: string @auth(bearer)
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`invalid decorator "style" for model field "id"`,
				`invalid decorator "auth" for model field "name"`,
			},
		},

		{
			name: "valid input decorators are accepted",
			input: `namespace test

input CreateUser {
  name: string @default(anon)
}
`,
			wantErr: false,
		},
		{
			name: "invalid decorator on input field",
			input: `namespace test

input CreateUser {
  id: uuid @primary
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`invalid decorator "primary" for input field "id"`,
				`input supports: @default, @raw`,
			},
		},
		{
			name: "multiple invalid decorators on input fields",
			input: `namespace test

input CreateUser {
  email: string @unique
  org: Org @relation
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`invalid decorator "unique" for input field "email"`,
				`invalid decorator "relation" for input field "org"`,
			},
		},

		{
			name: "valid api-level decorators are accepted",
			input: `namespace test

api Users @style(rest) @version(1) {
  GET /users -> User[]
}
`,
			wantErr: false,
		},
		{
			name: "invalid api-level decorator",
			input: `namespace test

api Users @style(rest) @deprecated(old) {
  GET /users -> User[]
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`only 'version' and 'style' decorators are allowed on API before block opening brace`,
			},
		},

		{
			name: "duplicate name and invalid decorator reported together",
			input: `namespace test

model User {
  id: uuid @version(1)
}

model User {
  id: uuid
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`duplicate definition: "User" [line: 7, col: 6]`,
				`invalid decorator "version" for model field "id"`,
			},
		},

		{
			name: "position reports correct line for later duplicate",
			input: `namespace test

model First {
  id: uuid
}

model Second {
  id: uuid
}

model First {
  id: uuid
}
`,
			wantErr: true,
			expectErrorIncludes: []string{
				`duplicate definition: "First" [line: 11, col: 6]`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, err := New()
			assert.NoError(t, err)

			_, err = p.Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if err != nil && len(tt.expectErrorIncludes) > 0 {
				errMsg := err.Error()
				t.Logf("%s error (expected):\n%s", tt.name, errMsg)
				for _, substr := range tt.expectErrorIncludes {
					if !strings.Contains(errMsg, substr) {
						t.Errorf("expected error to include %q, got: %s", substr, errMsg)
					}
				}
			}
		})
	}
}
