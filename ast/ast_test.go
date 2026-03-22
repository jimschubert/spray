package ast

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestDecorator_String(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		build    func() *Decorator
		expected string
	}{
		// --- No-arg decorators (model fields) ---
		{
			name: "primary",
			build: func() *Decorator {
				return &Decorator{Name: "primary"}
			},
			expected: "@primary",
		},
		{
			name: "unique",
			build: func() *Decorator {
				return &Decorator{Name: "unique"}
			},
			expected: "@unique",
		},
		{
			name: "updatedAt",
			build: func() *Decorator {
				return &Decorator{Name: "updatedAt"}
			},
			expected: "@updatedAt",
		},
		{
			name: "relation without args",
			build: func() *Decorator {
				return &Decorator{Name: "relation"}
			},
			expected: "@relation",
		},
		{
			name: "paginated",
			build: func() *Decorator {
				return &Decorator{Name: "paginated"}
			},
			expected: "@paginated",
		},

		// --- Single positional arg ---
		{
			name: "default with identifier",
			build: func() *Decorator {
				d := &Decorator{Name: "default"}
				d.Args.Set("member", nil, Position{Line: 1, Col: 10})
				return d
			},
			expected: "@default(member)",
		},
		{
			name: "default with now",
			build: func() *Decorator {
				d := &Decorator{Name: "default"}
				d.Args.Set("now", nil, Position{Line: 1, Col: 10})
				return d
			},
			expected: "@default(now)",
		},
		{
			name: "version with integer",
			build: func() *Decorator {
				d := &Decorator{Name: "version"}
				d.Args.Set("2", nil, Position{Line: 1, Col: 10})
				return d
			},
			expected: "@version(2)",
		},
		{
			name: "style with rest",
			build: func() *Decorator {
				d := &Decorator{Name: "style"}
				d.Args.Set("rest", nil, Position{Line: 1, Col: 10})
				return d
			},
			expected: "@style(rest)",
		},
		{
			name: "style with rpc",
			build: func() *Decorator {
				d := &Decorator{Name: "style"}
				d.Args.Set("rpc", nil, Position{Line: 1, Col: 10})
				return d
			},
			expected: "@style(rpc)",
		},
		{
			name: "style with events",
			build: func() *Decorator {
				d := &Decorator{Name: "style"}
				d.Args.Set("events", nil, Position{Line: 1, Col: 10})
				return d
			},
			expected: "@style(events)",
		},
		{
			name: "auth with bearer",
			build: func() *Decorator {
				d := &Decorator{Name: "auth"}
				d.Args.Set("bearer", nil, Position{Line: 1, Col: 10})
				return d
			},
			expected: "@auth(bearer)",
		},
		{
			name: "auth with apiKey",
			build: func() *Decorator {
				d := &Decorator{Name: "auth"}
				d.Args.Set("apiKey", nil, Position{Line: 1, Col: 10})
				return d
			},
			expected: "@auth(apiKey)",
		},
		{
			name: "auth with basic",
			build: func() *Decorator {
				d := &Decorator{Name: "auth"}
				d.Args.Set("basic", nil, Position{Line: 1, Col: 10})
				return d
			},
			expected: "@auth(basic)",
		},
		{
			name: "auth with none",
			build: func() *Decorator {
				d := &Decorator{Name: "auth"}
				d.Args.Set("none", nil, Position{Line: 1, Col: 10})
				return d
			},
			expected: "@auth(none)",
		},
		{
			name: "basePath with quoted string",
			build: func() *Decorator {
				d := &Decorator{Name: "basePath"}
				d.Args.Set(`"/api/v2/users"`, nil, Position{Line: 1, Col: 12})
				return d
			},
			expected: `@basePath("/api/v2/users")`,
		},
		{
			name: "body with input type",
			build: func() *Decorator {
				d := &Decorator{Name: "body"}
				d.Args.Set("CreateUserInput", nil, Position{Line: 1, Col: 10})
				return d
			},
			expected: "@body(CreateUserInput)",
		},
		{
			name: "query with input type",
			build: func() *Decorator {
				d := &Decorator{Name: "query"}
				d.Args.Set("PaginationInput", nil, Position{Line: 1, Col: 10})
				return d
			},
			expected: "@query(PaginationInput)",
		},
		{
			name: "summary with quoted string",
			build: func() *Decorator {
				d := &Decorator{Name: "summary"}
				d.Args.Set(`"Get user by ID"`, nil, Position{Line: 1, Col: 12})
				return d
			},
			expected: `@summary("Get user by ID")`,
		},
		{
			name: "tag with quoted string",
			build: func() *Decorator {
				d := &Decorator{Name: "tag"}
				d.Args.Set(`"users"`, nil, Position{Line: 1, Col: 8})
				return d
			},
			expected: `@tag("users")`,
		},
		{
			name: "deprecated with message",
			build: func() *Decorator {
				d := &Decorator{Name: "deprecated"}
				d.Args.Set(`"Use GET / with query params"`, nil, Position{Line: 1, Col: 14})
				return d
			},
			expected: `@deprecated("Use GET / with query params")`,
		},
		{
			name: "raw with target",
			build: func() *Decorator {
				d := &Decorator{Name: "raw"}
				d.Args.Set("openapi", nil, Position{Line: 1, Col: 8})
				return d
			},
			expected: "@raw(openapi)",
		},

		// --- Multiple positional args ---
		{
			name: "errors with two codes",
			build: func() *Decorator {
				d := &Decorator{Name: "errors"}
				d.Args.Set("401", nil, Position{Line: 1, Col: 10})
				d.Args.Set("404", nil, Position{Line: 1, Col: 15})
				return d
			},
			expected: "@errors(401, 404)",
		},
		{
			name: "errors with three codes",
			build: func() *Decorator {
				d := &Decorator{Name: "errors"}
				d.Args.Set("400", nil, Position{Line: 1, Col: 10})
				d.Args.Set("401", nil, Position{Line: 1, Col: 15})
				d.Args.Set("404", nil, Position{Line: 1, Col: 20})
				return d
			},
			expected: "@errors(400, 401, 404)",
		},
		{
			name: "relation with field reference",
			build: func() *Decorator {
				d := &Decorator{Name: "relation"}
				d.Args.Set("field", &TypeExpression{
					Base: QualifiedIdent{Parts: []string{"authorId"}},
				}, Position{Line: 1, Col: 14})
				return d
			},
			expected: "@relation(field: authorId)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decorator := tc.build()
			assert.Equal(t, tc.expected, decorator.String())
		})
	}
}
