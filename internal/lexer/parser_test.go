package lexer

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/internal/ast"
)

func TestParseNamespace(t *testing.T) {
	testCases := []struct {
		name                     string
		input                    string
		expectedNamespace        string
		expectedDocumentComments int
		headComment              string
		lineComment              string
		wantErr                  bool
	}{
		{
			name:                     "default namespace",
			input:                    "# file with no namespace\n",
			expectedNamespace:        "default",
			expectedDocumentComments: 1,
			wantErr:                  false,
		},
		{
			name:                     "simple namespace",
			input:                    "namespace acme\n",
			expectedNamespace:        "acme",
			expectedDocumentComments: 0,
			wantErr:                  false,
		},
		{
			name:                     "qualified namespace",
			input:                    "namespace acme.users.v1\n",
			expectedNamespace:        "acme.users.v1",
			expectedDocumentComments: 0,
			wantErr:                  false,
		},
		{
			name:                     "namespace with leading comment",
			input:                    "# my service\nnamespace acme\n",
			expectedNamespace:        "acme",
			expectedDocumentComments: 0,
			headComment:              "# my service",
			wantErr:                  false,
		},
		{
			name:                     "namespace with leading and doc comment",
			input:                    "# top of file\n\n# my service\nnamespace acme\n",
			expectedNamespace:        "acme",
			expectedDocumentComments: 1,
			headComment:              "# my service",
			wantErr:                  false,
		},
		{
			name:                     "namespace with leading, doc, and line comment",
			input:                    "# top of file\n\n# my service\nnamespace acme # line comment\n",
			expectedNamespace:        "acme",
			expectedDocumentComments: 1,
			headComment:              "# my service",
			lineComment:              "# line comment",
			wantErr:                  false,
		},
		{
			name:                     "error on unterminated namespace",
			input:                    "namespace acme.users.\n",
			expectedDocumentComments: 0,
			wantErr:                  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := New()
			assert.NoError(t, err)

			stencil, err := p.Parse(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.NotZero(t, stencil.Namespace, "expected namespace, got nil")
			assert.Equal(t, tc.expectedNamespace, stencil.Namespace.FullName())
			assert.Equal(t, tc.expectedDocumentComments, len(stencil.Comments))
			assert.Equal(t, tc.headComment, stencil.Namespace.HeadComment.String())
			assert.Equal(t, tc.lineComment, stencil.Namespace.LineComment.String())
		})
	}
}

func TestParseImport(t *testing.T) {
	testCases := []struct {
		name                     string
		input                    string
		expected                 []string
		headComment              string
		lineComment              string
		expectedDocumentComments int
		wantErr                  bool
	}{
		{
			name:     "single import",
			input:    "import acme.common.v1 { Page }\n",
			expected: []string{"Page"},
			wantErr:  false,
		},
		{
			name:     "multiple imports",
			input:    "import acme.common.v1 { Page, PaginationInput }\n",
			expected: []string{"Page", "PaginationInput"},
			wantErr:  false,
		},
		{
			name:                     "import with header and line comments",
			input:                    "# header comment\nimport acme.common.v1 { Page } # line comment\n#doc comment",
			expected:                 []string{"Page"},
			wantErr:                  false,
			headComment:              "# header comment",
			lineComment:              "# line comment",
			expectedDocumentComments: 1,
		},
		{
			name:    "error on empty",
			input:   "import acme.common.v1 { }\n",
			wantErr: true,
		},
		{
			name:    "error on unterminated list",
			input:   "import acme.common.v1 { Page, }\n",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := New()
			assert.NoError(t, err)

			stencil, err := p.Parse(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, 1, len(stencil.Imports), "expected 1 import")

			imp := stencil.Imports[0]
			assert.Equal(t, "acme.common.v1", imp.Path.String())
			assert.Equal(t, len(tc.expected), len(imp.Names))

			for i, name := range tc.expected {
				assert.Equal(t, name, imp.Names[i].Value)
			}

			if imp.HeadComment != nil && tc.headComment != "" {
				assert.Equal(t, tc.headComment, imp.HeadComment.Comments[0].Text)
			}

			assert.Equal(t, tc.lineComment, imp.LineComment.String())
			assert.Equal(t, tc.expectedDocumentComments, len(stencil.Comments))
		})
	}
}

func TestParseImport_FQNs(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		expectedFQNs []string
		wantErr      bool
	}{
		{
			name:         "single import to FQN",
			input:        "import acme.common.v1 { Page }\n",
			expectedFQNs: []string{"acme.common.v1.Page"},
			wantErr:      false,
		},
		{
			name:         "multiple imports",
			input:        "import acme.common.v1 { Page, PaginationInput }\n",
			expectedFQNs: []string{"acme.common.v1.Page", "acme.common.v1.PaginationInput"},
			wantErr:      false,
		},
		{
			name:    "error on unterminated list (multiple)",
			input:   "import acme.common.v1 { Page, PaginationInput, }\n",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := New()
			assert.NoError(t, err)

			stencil, err := p.Parse(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, 1, len(stencil.Imports), "expected 1 import")

			actualFQNs := stencil.Imports[0].FQNs()
			assert.Equal(t, len(tc.expectedFQNs), len(actualFQNs))

			for i, expected := range tc.expectedFQNs {
				assert.Equal(t, expected, actualFQNs[i])
			}
		})
	}
}

func TestParseImport_Multiple(t *testing.T) {
	type importExpected struct {
		name              string
		expectedTypeNames []string
		expectedFQNs      []string
	}
	testCases := []struct {
		name     string
		input    string
		expected []importExpected
		wantErr  bool
	}{
		{
			name:  "multiple same base",
			input: "import acme.common.v1 { Page, PaginationInput }\nimport acme.common.v1 { User, UserRole }\n",
			expected: []importExpected{
				{
					name:              "acme.common.v1",
					expectedTypeNames: []string{"Page", "PaginationInput"},
					expectedFQNs:      []string{"acme.common.v1.Page", "acme.common.v1.PaginationInput"},
				},
				{
					name:              "acme.common.v1",
					expectedTypeNames: []string{"User", "UserRole"},
					expectedFQNs:      []string{"acme.common.v1.User", "acme.common.v1.UserRole"},
				},
			},
			wantErr: false,
		},
		{
			name:  "allow duplicates same base", // will later warn but should not error
			input: "import acme.common.v1 { Page, PaginationInput }\nimport acme.common.v1 { Page, UserRole }\n",
			expected: []importExpected{
				{
					name:              "acme.common.v1",
					expectedTypeNames: []string{"Page", "PaginationInput"},
					expectedFQNs:      []string{"acme.common.v1.Page", "acme.common.v1.PaginationInput"},
				},
				{
					name:              "acme.common.v1",
					expectedTypeNames: []string{"Page", "UserRole"},
					expectedFQNs:      []string{"acme.common.v1.Page", "acme.common.v1.UserRole"},
				},
			},
			wantErr: false,
		},
		{
			name:  "multiple different base",
			input: "import acme.common.v1 { Page, PaginationInput }\nimport acme.common.v2 { User, UserRole }\n",
			expected: []importExpected{
				{
					name:              "acme.common.v1",
					expectedTypeNames: []string{"Page", "PaginationInput"},
					expectedFQNs:      []string{"acme.common.v1.Page", "acme.common.v1.PaginationInput"},
				},
				{
					name:              "acme.common.v2",
					expectedTypeNames: []string{"User", "UserRole"},
					expectedFQNs:      []string{"acme.common.v2.User", "acme.common.v2.UserRole"},
				},
			},
			wantErr: false,
		},
		{
			name:    "error on any empty",
			input:   "import acme.common.v1 { Page, PaginationInput }\nimport acme.common.v1 { }\n",
			wantErr: true,
		},
		{
			name:    "error on any unterminated list",
			input:   "import acme.common.v1 { Page, PaginationInput }\nimport acme.common.v1 { }\n",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, err := New()
			assert.NoError(t, err)

			stencil, err := p.Parse(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, len(tc.expected), len(stencil.Imports))

			for idx, expected := range tc.expected {
				imp := stencil.Imports[idx]

				assert.Equal(t, len(expected.expectedTypeNames), len(imp.Names))

				for i, name := range expected.expectedTypeNames {
					assert.Equal(t, name, imp.Names[i].Value)
				}

				actualFQNs := imp.FQNs()
				assert.Equal(t, len(expected.expectedFQNs), len(actualFQNs))

				for i, expectedFQN := range expected.expectedFQNs {
					assert.Equal(t, expectedFQN, actualFQNs[i])
				}
			}
		})
	}
}

func TestParseEnum(t *testing.T) {
	testCases := []struct {
		name                     string
		input                    string
		expectedName             string
		expectedElements         []string
		headComment              string
		expectedDocumentComments int
		wantErr                  bool
	}{
		{
			name:             "simple enum",
			input:            "enum Status { ACTIVE, INACTIVE }\n",
			expectedName:     "Status",
			expectedElements: []string{"ACTIVE", "INACTIVE"},
			wantErr:          false,
		},
		{
			name:             "single element enum",
			input:            "enum Role { USER }\n",
			expectedName:     "Role",
			expectedElements: []string{"USER"},
			wantErr:          false,
		},
		{
			name:             "enum with many elements",
			input:            "enum Color { RED, GREEN, BLUE, YELLOW, CYAN, MAGENTA, BLACK, WHITE }\n",
			expectedName:     "Color",
			expectedElements: []string{"RED", "GREEN", "BLUE", "YELLOW", "CYAN", "MAGENTA", "BLACK", "WHITE"},
			wantErr:          false,
		},
		{
			name:             "enum with leading comment",
			input:            "# status enumeration\nenum Status { ACTIVE, INACTIVE }\n",
			expectedName:     "Status",
			expectedElements: []string{"ACTIVE", "INACTIVE"},
			headComment:      "# status enumeration",
			wantErr:          false,
		},
		{
			name:                     "enum with leading and document comments",
			input:                    "# top level\n\n# status enumeration\nenum Status { ACTIVE, INACTIVE }\n",
			expectedName:             "Status",
			expectedElements:         []string{"ACTIVE", "INACTIVE"},
			headComment:              "# status enumeration",
			expectedDocumentComments: 1,
			wantErr:                  false,
		},
		{
			name:             "enum with multiline definition",
			input:            "enum Status {\n  ACTIVE,\n  INACTIVE\n}\n",
			expectedName:     "Status",
			expectedElements: []string{"ACTIVE", "INACTIVE"},
			wantErr:          false,
		},
		{
			name:    "enum with trailing comma not allowed",
			input:   "enum Status { ACTIVE, INACTIVE, }\n",
			wantErr: true,
		},
		{
			name:    "error on empty enum",
			input:   "enum Status { }\n",
			wantErr: true,
		},
		{
			name:    "error on missing name",
			input:   "enum { ACTIVE, INACTIVE }\n",
			wantErr: true,
		},
		{
			name:    "error on missing opening brace",
			input:   "enum Status ACTIVE, INACTIVE\n",
			wantErr: true,
		},
		{
			name:    "error on missing closing brace",
			input:   "enum Status { ACTIVE, INACTIVE\n",
			wantErr: true,
		},
		{
			name:                     "enum with comment group",
			input:                    "# doc comment\n\n# enum comment 1\n# enum comment 2\nenum Status {\nACTIVE, INACTIVE\n}\n",
			headComment:              "# enum comment 1\n# enum comment 2",
			expectedName:             "Status",
			expectedElements:         []string{"ACTIVE", "INACTIVE"},
			expectedDocumentComments: 1,
			wantErr:                  false,
			// note: the doc comment should be associated with the enum, not the namespace, since it's closer to the enum
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := New()
			assert.NoError(t, err)

			stencil, err := p.Parse(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, 1, len(stencil.Specs), "expected 1 spec")

			enum, ok := stencil.Specs[0].(*ast.Enum)
			assert.True(t, ok, "expected spec to be an Enum")

			assert.Equal(t, tc.expectedName, enum.Name.Value)
			assert.Equal(t, len(tc.expectedElements), len(enum.Elements))

			for i, expectedElement := range tc.expectedElements {
				assert.Equal(t, expectedElement, enum.Elements[i].Value)
			}

			if tc.headComment != "" && enum.HeadComment != nil {
				assert.Equal(t, tc.headComment, enum.HeadComment.String())
			}

			assert.Equal(t, tc.expectedDocumentComments, len(stencil.Comments))
		})
	}
}

func TestParseEnum_MultipleEnums(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []struct {
			name     string
			elements []string
		}
		wantErr bool
	}{
		{
			name:  "multiple enums",
			input: "enum Status { ACTIVE, INACTIVE }\nenum Role { USER, ADMIN }\n",
			expected: []struct {
				name     string
				elements []string
			}{
				{
					name:     "Status",
					elements: []string{"ACTIVE", "INACTIVE"},
				},
				{
					name:     "Role",
					elements: []string{"USER", "ADMIN"},
				},
			},
			wantErr: false,
		},
		{
			name:  "multiple enums with comments",
			input: "# status\nenum Status { ACTIVE, INACTIVE }\n\n# role\nenum Role { USER, ADMIN }\n",
			expected: []struct {
				name     string
				elements []string
			}{
				{
					name:     "Status",
					elements: []string{"ACTIVE", "INACTIVE"},
				},
				{
					name:     "Role",
					elements: []string{"USER", "ADMIN"},
				},
			},
			wantErr: false,
		},
		{
			name:  "three enums",
			input: "enum Status { ACTIVE, INACTIVE }\nenum Role { USER, ADMIN, GUEST }\nenum Color { RED, BLUE }\n",
			expected: []struct {
				name     string
				elements []string
			}{
				{
					name:     "Status",
					elements: []string{"ACTIVE", "INACTIVE"},
				},
				{
					name:     "Role",
					elements: []string{"USER", "ADMIN", "GUEST"},
				},
				{
					name:     "Color",
					elements: []string{"RED", "BLUE"},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := New()
			assert.NoError(t, err)

			stencil, err := p.Parse(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, len(tc.expected), len(stencil.Specs))

			for idx, expected := range tc.expected {
				enum, ok := stencil.Specs[idx].(*ast.Enum)
				assert.True(t, ok, "expected spec at index %d to be an Enum", idx)

				assert.Equal(t, expected.name, enum.Name.Value)
				assert.Equal(t, len(expected.elements), len(enum.Elements))

				for i, elem := range expected.elements {
					assert.Equal(t, elem, enum.Elements[i].Value)
				}
			}
		})
	}
}

func TestParseEnum_WithImportsAndNamespace(t *testing.T) {
	testCases := []struct {
		name              string
		input             string
		expectedNamespace string
		expectedImports   int
		expectedEnums     int
		wantErr           bool
	}{
		{
			name:              "namespace, import, and enum",
			input:             "namespace acme\nimport acme.common { Page }\nenum Status {\nACTIVE\nINACTIVE\n}\n",
			expectedNamespace: "acme",
			expectedImports:   1,
			expectedEnums:     1,
			wantErr:           false,
		},
		{
			name: "multiple imports and enums",
			// note: one enum has no commas (defined in spec), one has commas - should be able to parse both
			input:             "import acme.common { Page }\nimport acme.users { User }\nenum Status {\nACTIVE\nINACTIVE\n}\nenum Role { ADMIN, USER }\n",
			expectedNamespace: "default",
			expectedImports:   2,
			expectedEnums:     2,
			wantErr:           false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := New()
			assert.NoError(t, err)

			stencil, err := p.Parse(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, tc.expectedNamespace, stencil.Namespace.FullName())
			assert.Equal(t, tc.expectedImports, len(stencil.Imports))

			enumCount := 0
			for _, spec := range stencil.Specs {
				if _, ok := spec.(*ast.Enum); ok {
					enumCount++
				}
			}
			assert.Equal(t, tc.expectedEnums, enumCount)
		})
	}
}

func TestParseTypeAlias(t *testing.T) {
	testCases := []struct {
		name                        string
		input                       string
		expectedName                string
		expectedTypeBase            string
		expectedTypeWithConstraints string
		wantErr                     bool
	}{
		{
			name:             "type alias with scalar type",
			input:            "type Email = string\n",
			expectedName:     "Email",
			expectedTypeBase: "string",
			wantErr:          false,
		},
		{
			name:                        "type alias with scalar type optional",
			input:                       "type Email = string?\n",
			expectedName:                "Email",
			expectedTypeBase:            "string",
			expectedTypeWithConstraints: "string?",
			wantErr:                     false,
		},
		{
			name:                        "type alias with scalar type array",
			input:                       "type Email = string[]\n",
			expectedName:                "Email",
			expectedTypeBase:            "string",
			expectedTypeWithConstraints: "string[]",
			wantErr:                     false,
		},
		{
			name:             "type alias with non-scalar type",
			input:            "type UserID = ValueTypeID\n",
			expectedName:     "UserID",
			expectedTypeBase: "ValueTypeID",
			wantErr:          false,
		},
		{
			name:             "type alias with fully qualified non-scalar type",
			input:            "type UserID = acme.common.v1.UUID\n",
			expectedName:     "UserID",
			expectedTypeBase: "acme.common.v1.UUID",
			wantErr:          false,
		},
		{
			name:                        "type alias with generic type",
			input:                       "type MyType = YourType<string, int>\n",
			expectedName:                "MyType",
			expectedTypeBase:            "YourType",
			expectedTypeWithConstraints: "YourType<string, int>",
			wantErr:                     false,
		},
		{
			name:                        "type alias with generic type nested with constraints",
			input:                       "type MyType = YourType<Map<int, string?>>\n",
			expectedName:                "MyType",
			expectedTypeBase:            "YourType",
			expectedTypeWithConstraints: "YourType<Map<int, string?>>",
			wantErr:                     false,
		},
		{
			name:             "type alias with fully qualified type",
			input:            "type UserID = acme.common.v1.UUID\n",
			expectedName:     "UserID",
			expectedTypeBase: "acme.common.v1.UUID",
			wantErr:          false,
		},
		{
			name:    "error on missing equals",
			input:   "type Email string\n",
			wantErr: true,
		},
		{
			name:    "error on missing name",
			input:   "type = string\n",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := New()
			assert.NoError(t, err)

			stencil, err := p.Parse(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, 1, len(stencil.Specs), "expected 1 spec")

			typeAlias, ok := stencil.Specs[0].(*ast.TypeAlias)
			assert.True(t, ok, "expected spec to be a TypeAlias")

			assert.Equal(t, tc.expectedName, typeAlias.Name.Value)
			if tc.expectedTypeBase != "" {
				assert.Equal(t, tc.expectedTypeBase, typeAlias.Type.Base.String())
			}
			if tc.expectedTypeWithConstraints != "" {
				assert.Equal(t, tc.expectedTypeWithConstraints, typeAlias.Type.String())
			}
		})
	}
}

func TestParseInput(t *testing.T) {
	testCases := []struct {
		name               string
		input              string
		expectedFieldCount int
		expectedFieldName  string
		expectedFieldType  string
		expectedDecorators int
		wantErr            bool
	}{
		{
			name:               "input with scalar fields on separate lines",
			input:              "namespace test\ninput CreateUserInput {\n  email: string\n  name: string\n}\n",
			expectedFieldCount: 2,
			wantErr:            false,
		},
		{
			name:               "input with non-scalar fields",
			input:              "namespace test\ninput CreateUserInput {\n  email: string\n  profile: UserProfile\n}\n",
			expectedFieldCount: 2,
			wantErr:            false,
		},
		{
			name:               "input with array field",
			input:              "namespace test\ninput CreateUserInput {\n  email: string\n  tags: string[]\n}\n",
			expectedFieldCount: 2,
			wantErr:            false,
		},
		{
			name:               "input with optional field",
			input:              "namespace test\ninput CreateUserInput {\n  email: string\n  name: string?\n}\n",
			expectedFieldCount: 2,
			wantErr:            false,
		},
		{
			name:               "input with @default decorator",
			input:              "namespace test\ninput CreateUserInput {\n  email: string\n  role: UserRole @default(member)\n}\n",
			expectedFieldName:  "role",
			expectedFieldType:  "UserRole",
			expectedDecorators: 1,
			wantErr:            false,
		},
		{
			name:               "input with @default(now)",
			input:              "namespace test\ninput CreateUserInput {\n  email: string\n  createdAt: timestamp @default(now)\n}\n",
			expectedFieldName:  "createdAt",
			expectedFieldType:  "timestamp",
			expectedDecorators: 1,
			wantErr:            false,
		},
		{
			name:               "input with multiple @default decorators on different fields",
			input:              "namespace test\ninput PaginationInput {\n  limit: int @default(20)\n  cursor: Cursor?\n}\n",
			expectedFieldCount: 2,
			wantErr:            false,
		},
		{
			name:               "input with all optional fields",
			input:              "namespace test\ninput FilterInput {\n  search: string?\n  limit: int?\n  offset: int?\n}\n",
			expectedFieldCount: 3,
			wantErr:            false,
		},
		{
			name:    "error on missing field type",
			input:   "namespace test\ninput CreateUserInput {\n  email\n}\n",
			wantErr: true,
		},
		{
			name:    "error on missing field name",
			input:   "namespace test\ninput CreateUserInput {\n  : string\n}\n",
			wantErr: true,
		},
		{
			name:    "error on missing closing brace",
			input:   "namespace test\ninput CreateUserInput {\n  email: string\n",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := New()
			assert.NoError(t, err)

			stencil, err := p.Parse(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// test allows only one input spec
			var inputSpec *ast.Input
			for _, spec := range stencil.Specs {
				if inp, ok := spec.(*ast.Input); ok {
					inputSpec = inp
					break
				}
			}
			assert.True(t, inputSpec != nil, "expected to find an Input spec")

			if tc.expectedFieldCount > 0 {
				assert.Equal(t, tc.expectedFieldCount, len(inputSpec.Fields))
			}

			if tc.expectedFieldName != "" {
				found := false
				for _, field := range inputSpec.Fields {
					// tc.expectedFieldName allows for testing a single field, but test allows defining multiple
					if field.Name.Value == tc.expectedFieldName {
						found = true
						assert.Equal(t, tc.expectedFieldType, field.Type.String())
						assert.Equal(t, tc.expectedDecorators, len(field.Decorators))
						break
					}
				}
				assert.True(t, found, "expected field %q not found", tc.expectedFieldName)
			}
		})
	}
}

func TestParseModel(t *testing.T) {
	testCases := []struct {
		name               string
		input              string
		expectedFieldCount int
		expectedFieldName  string
		expectedFieldType  string
		expectedDecorators int
		wantErr            bool
	}{
		{
			name:               "model with scalar fields",
			input:              "namespace test\nmodel User {\n  id: uuid\n  email: string\n}\n",
			expectedFieldCount: 2,
			wantErr:            false,
		},
		{
			name:               "model with non-scalar fields",
			input:              "namespace test\nmodel User {\n  id: uuid\n  role: UserRole\n}\n",
			expectedFieldCount: 2,
			wantErr:            false,
		},
		{
			name:               "model with array field",
			input:              "namespace test\nmodel User {\n  id: uuid\n  posts: Post[]\n}\n",
			expectedFieldCount: 2,
			wantErr:            false,
		},
		{
			name:               "model with optional field",
			input:              "namespace test\nmodel User {\n  id: uuid\n  name: string?\n}\n",
			expectedFieldCount: 2,
			wantErr:            false,
		},
		{
			name:               "model with fully qualified field",
			input:              "namespace test\nmodel User {\n  id: uuid\n  role: acme.common.v1.Role\n}\n",
			expectedFieldCount: 2,
			wantErr:            false,
		},
		{
			name:               "model with @primary decorator",
			input:              "namespace test\nmodel User {\n  id: uuid @primary\n  email: string\n}\n",
			expectedFieldName:  "id",
			expectedFieldType:  "uuid",
			expectedDecorators: 1,
			wantErr:            false,
		},
		{
			name:               "model with @unique decorator",
			input:              "namespace test\nmodel User {\n  email: string @unique\n}\n",
			expectedFieldName:  "email",
			expectedFieldType:  "string",
			expectedDecorators: 1,
			wantErr:            false,
		},
		{
			name:               "model with @default decorator",
			input:              "namespace test\nmodel User {\n  role: UserRole @default(member)\n}\n",
			expectedFieldName:  "role",
			expectedFieldType:  "UserRole",
			expectedDecorators: 1,
			wantErr:            false,
		},
		{
			name:               "model with @default(now)",
			input:              "namespace test\nmodel User {\n  createdAt: timestamp @default(now)\n}\n",
			expectedFieldName:  "createdAt",
			expectedFieldType:  "timestamp",
			expectedDecorators: 1,
			wantErr:            false,
		},
		{
			name:               "model with @updatedAt decorator",
			input:              "namespace test\nmodel User {\n  updatedAt: timestamp @updatedAt\n}\n",
			expectedFieldName:  "updatedAt",
			expectedFieldType:  "timestamp",
			expectedDecorators: 1,
			wantErr:            false,
		},
		{
			name:               "model with @relation decorator",
			input:              "namespace test\nmodel Post {\n  authorId: uuid\n  author: User @relation(field: authorId)\n}\n",
			expectedFieldName:  "author",
			expectedFieldType:  "User",
			expectedDecorators: 1,
			wantErr:            false,
		},
		{
			name:               "model with @deprecated decorator",
			input:              "namespace test\nmodel User {\n  oldField: string @deprecated(msg)\n}\n",
			expectedFieldName:  "oldField",
			expectedFieldType:  "string",
			expectedDecorators: 1,
			wantErr:            false,
		},
		{
			name:               "model with multiple decorators on one field",
			input:              "namespace test\nmodel User {\n  id: uuid @primary @unique\n}\n",
			expectedFieldName:  "id",
			expectedFieldType:  "uuid",
			expectedDecorators: 2,
			wantErr:            false,
		},
		{
			name: "model with full example from spec",
			input: `namespace test
model User {
  id: uuid @primary
  email: Email @unique
  role: UserRole @default(member)
  name: string?
  createdAt: timestamp @default(now)
  updatedAt: timestamp @updatedAt
  posts: Post[] @relation
}
`,
			expectedFieldCount: 7,
			wantErr:            false,
		},
		{
			name:               "model with no fields",
			input:              "namespace test\nmodel Empty {\n}\n",
			expectedFieldCount: 0,
			wantErr:            false,
		},
		{
			name:    "error on missing field type",
			input:   "namespace test\nmodel User {\n  id\n}\n",
			wantErr: true,
		},
		{
			name:    "error on missing field name",
			input:   "namespace test\nmodel User {\n  : uuid\n}\n",
			wantErr: true,
		},
		{
			name:    "error on missing opening brace",
			input:   "namespace test\nmodel User id: uuid }\n",
			wantErr: true,
		},
		{
			name:    "error on missing closing brace",
			input:   "namespace test\nmodel User {\n  id: uuid\n",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := New()
			assert.NoError(t, err)

			stencil, err := p.Parse(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Find the Model spec
			var modelSpec *ast.Model
			for _, spec := range stencil.Specs {
				if m, ok := spec.(*ast.Model); ok {
					modelSpec = m
					break
				}
			}
			assert.True(t, modelSpec != nil, "expected to find a Model spec")

			if tc.expectedFieldCount > 0 {
				assert.Equal(t, tc.expectedFieldCount, len(modelSpec.Fields))
			}

			if tc.expectedFieldName != "" {
				found := false
				for _, field := range modelSpec.Fields {
					if field.Name.Value == tc.expectedFieldName {
						found = true
						assert.Equal(t, tc.expectedFieldType, field.Type.String())
						assert.Equal(t, tc.expectedDecorators, len(field.Decorators))
						break
					}
				}
				assert.True(t, found, "expected field %q not found", tc.expectedFieldName)
			}
		})
	}
}

func TestParseModel_WithGenerics(t *testing.T) {
	testCases := []struct {
		name                  string
		input                 string
		expectedName          string
		expectedGenericParams []string
		expectedFieldCount    int
		wantErr               bool
	}{
		{
			name:                  "model with single generic parameter",
			input:                 "namespace test\nmodel Page<T> {\n  data: T[]\n  total: int\n}\n",
			expectedName:          "Page",
			expectedGenericParams: []string{"T"},
			expectedFieldCount:    2,
			wantErr:               false,
		},
		{
			name:                  "model with multiple generic parameters",
			input:                 "namespace test\nmodel Result<T, E> {\n  ok: boolean\n  data: T?\n  error: E?\n}\n",
			expectedName:          "Result",
			expectedGenericParams: []string{"T", "E"},
			expectedFieldCount:    3,
			wantErr:               false,
		},
		{
			name: "model with generic parameters from spec",
			input: `namespace test
model Page<T> {
  data: T[]
  nextCursor: Cursor?
  total: int
}
`,
			expectedName:          "Page",
			expectedGenericParams: []string{"T"},
			expectedFieldCount:    3,
			wantErr:               false,
		},
		{
			name:                  "model with three generic parameters",
			input:                 "namespace test\nmodel Triple<A, B, C> {\n  first: A\n  second: B\n  third: C\n}\n",
			expectedName:          "Triple",
			expectedGenericParams: []string{"A", "B", "C"},
			expectedFieldCount:    3,
			wantErr:               false,
		},
		{
			name:               "model without generic parameters",
			input:              "namespace test\nmodel User {\n  id: uuid\n  name: string\n}\n",
			expectedName:       "User",
			expectedFieldCount: 2,
			wantErr:            false,
		},
		{
			name:    "error on empty generic parameter list",
			input:   "namespace test\nmodel Page<> {\n  data: string\n}\n",
			wantErr: true,
		},
		{
			name:    "error on unterminated generic parameter list",
			input:   "namespace test\nmodel Page<T {\n  data: T\n}\n",
			wantErr: true,
		},
		{
			name:    "error on trailing comma in generic params",
			input:   "namespace test\nmodel Result<T, E,> {\n  data: T\n}\n",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := New()
			assert.NoError(t, err)

			stencil, err := p.Parse(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			var modelSpec *ast.Model
			for _, spec := range stencil.Specs {
				if m, ok := spec.(*ast.Model); ok {
					modelSpec = m
					break
				}
			}
			assert.True(t, modelSpec != nil, "expected to find a Model spec")
			assert.Equal(t, tc.expectedName, modelSpec.Name.Value)

			if tc.expectedGenericParams != nil {
				assert.Equal(t, len(tc.expectedGenericParams), len(modelSpec.GenericParams))
				for i, expectedParam := range tc.expectedGenericParams {
					assert.Equal(t, expectedParam, modelSpec.GenericParams[i].Value)
				}
			} else {
				assert.Equal(t, 0, len(modelSpec.GenericParams))
			}

			if tc.expectedFieldCount > 0 {
				assert.Equal(t, tc.expectedFieldCount, len(modelSpec.Fields))
			}
		})
	}
}
