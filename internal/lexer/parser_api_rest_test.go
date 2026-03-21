package lexer

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/internal/ast"
)

func TestParseApi_RouteREST(t *testing.T) {
	testCases := []struct {
		name             string
		input            string
		expectMethod     string
		expectPathSpec   []string
		expectReturnType string
		expectDecorators []string
		wantErr          bool
	}{
		{
			name:             "GET route with return type and no decorators",
			input:            `GET /users -> User[]`,
			expectMethod:     "GET",
			expectPathSpec:   []string{"users"},
			expectReturnType: "User[]",
			expectDecorators: []string{},
			wantErr:          false,
		},
		{
			name:             "POST route with path param and decorator",
			input:            `POST /users/:id -> User @query(PaginationInput) @errors(404)`,
			expectMethod:     "POST",
			expectPathSpec:   []string{"users", "id"},
			expectReturnType: "User",
			expectDecorators: []string{"query", "errors"},
			wantErr:          false,
		},
		{
			name:             "PUT route with path param and decorator",
			input:            `PUT /users/:id -> User @errors(404)`,
			expectMethod:     "PUT",
			expectPathSpec:   []string{"users", "id"},
			expectReturnType: "User",
			expectDecorators: []string{"errors"},
			wantErr:          false,
		},
		{
			name:             "PATCH route with path param and decorator",
			input:            `PATCH /users/:id -> User  @errors(404)`,
			expectMethod:     "PATCH",
			expectPathSpec:   []string{"users", "id"},
			expectReturnType: "User",
			expectDecorators: []string{"errors"},
			wantErr:          false,
		},
		{
			name:             "DELETE route with path param and decorator",
			input:            `DELETE /users/:id -> DeleteResult`,
			expectMethod:     "DELETE",
			expectPathSpec:   []string{"users", "id"},
			expectReturnType: "DeleteResult",
			wantErr:          false,
		},
		{
			name:    "invalid method",
			input:   `FETCH /data -> Data`,
			wantErr: true,
		},
		{name: "missing path",
			input:   `GET -> Data`,
			wantErr: true,
		},
		{
			name:    "missing return type",
			input:   `GET /data ->`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src := `api TestApi @style(rest) {
				` + tc.input + `
			}`
			p, err := New()
			assert.NoError(t, err)

			stencil, err := p.Parse(src)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			var apiSpec *ast.Api

			for _, spec := range stencil.Specs {
				if a, ok := spec.(*ast.Api); ok {
					apiSpec = a
					break
				}
			}
			assert.True(t, apiSpec != nil, "expected to find an Api spec")
			assert.Equal(t, "TestApi", apiSpec.Name.Value)
			assert.Equal(t, ast.REST, apiSpec.Style)

			if len(apiSpec.Routes) != 1 {
				t.Fatalf("expected 1 route, got %d", len(apiSpec.Routes))
			}
			route, ok := apiSpec.Routes[0].(*ast.RestRoute)
			assert.True(t, ok, "expected route to be RestRoute")
			assert.Equal(t, tc.expectMethod, route.Method)
			assert.Equal(t, len(tc.expectPathSpec), len(route.Path), "path segment count mismatch")
			for i, expectedSegment := range tc.expectPathSpec {
				assert.Equal(t, expectedSegment, route.Path[i].Value, "path segment mismatch")
			}
			assert.Equal(t, tc.expectReturnType, route.Return.String())

			assert.Equal(t, len(tc.expectDecorators), len(route.Decorators), "decorator count mismatch")
			for i, expectedDec := range tc.expectDecorators {
				assert.Equal(t, expectedDec, route.Decorators[i].Name, "decorator name mismatch")
			}
		})
	}
}

func TestParseApi_ApiREST(t *testing.T) {
	testCases := []struct {
		name             string
		input            string
		expectName       string
		expectStyle      ast.ApiStyle
		expectDecorators []string
		expectDirectives []string
		endpointCount    int
		wantErr          bool
	}{
		{
			name:          "empty api is allowed",
			input:         "api MyApi { }",
			expectName:    "MyApi",
			endpointCount: 0,
			wantErr:       false,
		},
		{
			name: "api with decorators and directives",
			input: `api MyApi @style(rest) @version(1) {
  @auth(bearer)
  @basePath("/api")
			
  GET /users -> User[]
  
  POST /users -> User
}`,
			expectName:       "MyApi",
			expectStyle:      ast.REST,
			expectDecorators: []string{"style", "version"},
			expectDirectives: []string{"auth", "basePath"},
			endpointCount:    2,
			wantErr:          false,
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

			var apiSpec *ast.Api

			for _, spec := range stencil.Specs {
				if a, ok := spec.(*ast.Api); ok {
					apiSpec = a
					break
				}
			}
			assert.True(t, apiSpec != nil, "expected to find an Api spec")
			assert.Equal(t, tc.expectName, apiSpec.Name.Value)
			assert.Equal(t, tc.expectStyle, apiSpec.Style)

			assert.Equal(t, len(tc.expectDecorators), len(apiSpec.ApiDecorators), "decorator count mismatch")
			for i, expectedDec := range tc.expectDecorators {
				assert.Equal(t, expectedDec, apiSpec.ApiDecorators[i].Name, "decorator name mismatch")
			}

			assert.Equal(t, len(tc.expectDirectives), len(apiSpec.ApiDirectives), "directive count mismatch")
			for i, expectedDir := range tc.expectDirectives {
				assert.Equal(t, expectedDir, apiSpec.ApiDirectives[i].Name, "directive name mismatch")
			}

			assert.Equal(t, tc.endpointCount, len(apiSpec.Routes), "endpoint count mismatch")
		})
	}
}
