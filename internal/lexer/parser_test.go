package lexer

import (
	"testing"

	"github.com/alecthomas/assert/v2"
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
