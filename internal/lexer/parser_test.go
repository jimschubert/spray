package lexer

import (
	"testing"
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
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}

			stencil, err := p.Parse(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if stencil.Namespace == nil {
				t.Fatal("expected namespace, got nil")
			}

			if stencil.Namespace.FullName() != tc.expectedNamespace {
				t.Errorf("namespace mismatch: got=%q expected=%q", stencil.Namespace.FullName(), tc.expectedNamespace)
			}

			if len(stencil.Comments) != tc.expectedDocumentComments {
				t.Errorf("comment count mismatch: got=%d expected=%d", len(stencil.Comments), tc.expectedDocumentComments)
			}

			if stencil.Namespace.HeadComment.String() != tc.headComment {
				t.Errorf("head comment mismatch: got=%q expected=%q", stencil.Namespace.HeadComment.String(), tc.headComment)
			}

			if stencil.Namespace.LineComment.String() != tc.lineComment {
				t.Errorf("line comment mismatch: got=%q expected=%q", stencil.Namespace.LineComment.String(), tc.lineComment)
			}
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
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}

			stencil, err := p.Parse(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if len(stencil.Imports) != 1 {
				t.Fatalf("expected 1 import, got %d", len(stencil.Imports))
			}

			imp := stencil.Imports[0]
			if imp.Path.String() != "acme.common.v1" {
				t.Errorf("import path mismatch: got=%q expected=%q", imp.Path.String(), "acme.common.v1")
			}

			if len(imp.Names) != len(tc.expected) {
				t.Fatalf("import names count mismatch: got=%d expected=%d", len(imp.Names), len(tc.expected))
			}

			for i, name := range tc.expected {
				if imp.Names[i].Value != name {
					t.Errorf("import name %d mismatch: got=%q expected=%q", i, imp.Names[i].Value, name)
				}
			}

			if imp.HeadComment != nil && imp.HeadComment.Comments[0].Text != tc.headComment {
				t.Errorf("head comment mismatch: got=%q expected=%q", imp.HeadComment.Comments[0].Text, tc.headComment)
			}

			if imp.LineComment.String() != tc.lineComment {
				t.Errorf("line comment mismatch: got=%q expected=%q", imp.LineComment.String(), tc.lineComment)
			}

			if len(stencil.Comments) != tc.expectedDocumentComments {
				t.Errorf("document comment count mismatch: got=%d expected=%d", len(stencil.Comments), tc.expectedDocumentComments)
			}
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
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}

			stencil, err := p.Parse(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErr {
				return
			}

			if len(stencil.Imports) != 1 {
				t.Fatalf("expected 1 import, got %d", len(stencil.Imports))
			}

			actualFQNs := stencil.Imports[0].FQNs()
			if len(actualFQNs) != len(tc.expectedFQNs) {
				t.Fatalf("FQN count mismatch: got=%d expected=%d", len(actualFQNs), len(tc.expectedFQNs))
			}

			for i, expected := range tc.expectedFQNs {
				if actualFQNs[i] != expected {
					t.Errorf("FQN %d mismatch: got=%q expected=%q", i, actualFQNs[i], expected)
				}
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
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}

			stencil, err := p.Parse(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErr {
				return
			}

			if len(stencil.Imports) != len(tc.expected) {
				t.Fatalf("expected %d import, got %d", len(tc.expected), len(stencil.Imports))
			}

			for idx, expected := range tc.expected {
				imp := stencil.Imports[idx]

				if len(imp.Names) != len(expected.expectedTypeNames) {
					t.Fatalf("import names count mismatch: got=%d expected=%d", len(imp.Names), len(expected.expectedTypeNames))
				}

				for i, name := range expected.expectedTypeNames {
					if imp.Names[i].Value != name {
						t.Errorf("import name %d mismatch: got=%q expected=%q", i, imp.Names[i].Value, name)
					}
				}

				actualFQNs := imp.FQNs()
				if len(actualFQNs) != len(expected.expectedFQNs) {
					t.Fatalf("FQN count mismatch: got=%d expected=%d", len(actualFQNs), len(expected.expectedFQNs))
				}

				for i, expectedFQN := range expected.expectedFQNs {
					if actualFQNs[i] != expectedFQN {
						t.Errorf("FQN %d mismatch: got=%q expected=%q", i, actualFQNs[i], expectedFQN)
					}
				}
			}
		})
	}
}
