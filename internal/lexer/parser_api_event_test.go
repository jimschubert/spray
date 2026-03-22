package lexer

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/internal/ast"
)

func TestParseApi_RouteEVENTS(t *testing.T) {
	testCases := []struct {
		name             string
		input            string
		expectDirection  ast.EventDirection
		expectName       string
		expectReturnType string
		expectDecorators []string
		wantErr          bool
	}{
		{
			name:             "publish route with return type and no decorators",
			input:            `publish   UserCreated -> UserCreatedEvent`,
			expectDirection:  ast.EventPublish,
			expectName:       "UserCreated",
			expectReturnType: "UserCreatedEvent",
			expectDecorators: []string{},
			wantErr:          false,
		},
		// these decorators don't make a ton of sense here, but that's fine. we're just testing that decorators _can_ be parsed
		{
			name:             "subscribe route with return type and decorators",
			input:            `subscribe UserDeleted -> UserDeletedEvent @query(PaginationInput) @errors(404)`,
			expectDirection:  ast.EventSubscribe,
			expectName:       "UserDeleted",
			expectReturnType: "UserDeletedEvent",
			expectDecorators: []string{"query", "errors"},
			wantErr:          false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			src := `api TestApi @style(events) {
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
			assert.Equal(t, ast.StyleEvents, apiSpec.Style)

			assert.Equal(t, 1, len(apiSpec.Routes), "expected exactly one route")

			route, ok := apiSpec.Routes[0].(*ast.EventRoute)
			assert.True(t, ok, "expected route to be an EventRoute")

			assert.Equal(t, tc.expectName, route.Name.Value)
			assert.Equal(t, tc.expectDirection, route.Direction)
			assert.Equal(t, tc.expectReturnType, route.Event.Base.String())
			assert.Equal(t, len(tc.expectDecorators), len(route.Decorators))
			for i, decorator := range route.Decorators {
				assert.Equal(t, tc.expectDecorators[i], decorator.Name)
			}
		})
	}
}

func TestParseApi_EVENTS(t *testing.T) {
	src := `api TestApi @style(events) {
  publish UserCreated -> UserCreatedEvent
  subscribe UserDeleted -> UserDeletedEvent @query(PaginationInput) @errors(404)
}`
	p, err := New()
	assert.NoError(t, err)

	stencil, err := p.Parse(src)
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
	assert.Equal(t, ast.StyleEvents, apiSpec.Style)
	assert.Equal(t, 2, len(apiSpec.Routes), "expected exactly two routes")

	pubRoute, ok := apiSpec.Routes[0].(*ast.EventRoute)
	assert.True(t, ok, "expected first route to be an EventRoute")
	assert.Equal(t, "UserCreated", pubRoute.Name.Value)
	assert.Equal(t, ast.EventPublish, pubRoute.Direction)
	assert.Equal(t, "UserCreatedEvent", pubRoute.Event.Base.String())
	assert.Equal(t, 0, len(pubRoute.Decorators))

	subRoute, ok := apiSpec.Routes[1].(*ast.EventRoute)
	assert.True(t, ok, "expected second route to be an EventRoute")
	assert.Equal(t, "UserDeleted", subRoute.Name.Value)
	assert.Equal(t, ast.EventSubscribe, subRoute.Direction)
	assert.Equal(t, "UserDeletedEvent", subRoute.Event.Base.String())
	assert.Equal(t, 2, len(subRoute.Decorators))
	assert.Equal(t, "query", subRoute.Decorators[0].Name)
	assert.Equal(t, "errors", subRoute.Decorators[1].Name)
}
