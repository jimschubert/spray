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
				assert.Equal(t, tc.headComment, enum.HeadComment.Text)
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
