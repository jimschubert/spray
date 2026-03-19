package lexer

import (
	"testing"
)

func TestParseNamespace(t *testing.T) {
	t.Parallel()

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
	}

	for _, tc := range testCases {
		tc := tc
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
