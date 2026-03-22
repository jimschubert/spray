package lexer

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/internal/ast"
)

func TestParseRaw(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedTarget string
		expectedPairs  int
		wantErr        bool
	}{
		{
			name: "raw block with string value",
			input: `namespace test
model Webhook {
  id: uuid

  @raw(openapi) {
    "x-webhook-signature": "sha256"
  }
}
`,
			expectedTarget: "openapi",
			expectedPairs:  1,
		},
		{
			name: "raw block with boolean true",
			input: `namespace test
model Webhook {
  id: uuid

  @raw(openapi) {
    "x-internal": true
  }
}
`,
			expectedTarget: "openapi",
			expectedPairs:  1,
		},
		{
			name: "raw block with boolean false",
			input: `namespace test
model Webhook {
  id: uuid

  @raw(openapi) {
    "x-public": false
  }
}
`,
			expectedTarget: "openapi",
			expectedPairs:  1,
		},
		{
			name: "raw block with null value",
			input: `namespace test
model Webhook {
  id: uuid

  @raw(openapi) {
    "x-nullable": null
  }
}
`,
			expectedTarget: "openapi",
			expectedPairs:  1,
		},
		{
			name: "raw block with int value",
			input: `namespace test
model Webhook {
  id: uuid

  @raw(openapi) {
    "x-priority": 42
  }
}
`,
			expectedTarget: "openapi",
			expectedPairs:  1,
		},
		{
			name: "raw block with float value",
			input: `namespace test
model Webhook {
  id: uuid

  @raw(openapi) {
    "x-weight": 0.75
  }
}
`,
			expectedTarget: "openapi",
			expectedPairs:  1,
		},
		{
			name: "raw block with multiple pairs",
			input: `namespace test
model Webhook {
  id: uuid

  @raw(openapi) {
    "x-webhook-signature": "sha256"
    "x-internal": true
  }
}
`,
			expectedTarget: "openapi",
			expectedPairs:  2,
		},
		{
			name: "multiple raw blocks on same model",
			input: `namespace test
model Webhook {
  id: uuid

  @raw(openapi) {
    "x-internal": true
  }

  @raw(mermaid) {
    "style": "fill:#f9f,stroke:#333"
  }
}
`,
			expectedTarget: "openapi",
			expectedPairs:  1,
		},
		{
			name: "raw block with mixed value types",
			input: `namespace test
model Config {
  id: uuid

  @raw(blueprint) {
    "label": "my-service"
    "retries": 3
    "weight": 1.5
    "enabled": true
    "deprecated": false
    "metadata": null
  }
}
`,
			expectedTarget: "blueprint",
			expectedPairs:  6,
		},
		{
			name: "error on missing target",
			input: `namespace test
model Webhook {
  @raw() {
    "key": "value"
  }
}
`,
			wantErr: true,
		},
		{
			name: "error on missing opening brace",
			input: `namespace test
model Webhook {
  @raw(openapi)
    "key": "value"
  }
}
`,
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
			assert.True(t, len(modelSpec.Extensions) > 0, "expected at least one @raw block")
			assert.Equal(t, tc.expectedTarget, modelSpec.Extensions[0].Target.Value)
			assert.Equal(t, tc.expectedPairs, len(modelSpec.Extensions[0].Pairs))
		})
	}
}

func TestParseRaw_MultipleBlocks(t *testing.T) {
	t.Parallel()

	input := `namespace test
model Webhook {
  id: uuid

  @raw(openapi) {
    "x-internal": true
  }

  @raw(mermaid) {
    "style": "fill:#f9f,stroke:#333"
  }

  @raw(blueprint) {
    "some-key": "value"
  }
}
`

	p, err := New()
	assert.NoError(t, err)

	stencil, err := p.Parse(input)
	assert.NoError(t, err)

	var modelSpec *ast.Model
	for _, spec := range stencil.Specs {
		if m, ok := spec.(*ast.Model); ok {
			modelSpec = m
			break
		}
	}
	assert.True(t, modelSpec != nil, "expected to find a Model spec")
	assert.Equal(t, 3, len(modelSpec.Extensions))
	assert.Equal(t, "openapi", modelSpec.Extensions[0].Target.Value)
	assert.Equal(t, "mermaid", modelSpec.Extensions[1].Target.Value)
	assert.Equal(t, "blueprint", modelSpec.Extensions[2].Target.Value)
}

func TestParseRaw_FieldWithDecoratorsBeforeRaw(t *testing.T) {
	t.Parallel()

	input := `namespace test
model User {
  id:    uuid   @primary
  email: string @unique

  @raw(openapi) {
    "x-internal": true
  }
}
`

	p, err := New()
	assert.NoError(t, err)

	stencil, err := p.Parse(input)
	assert.NoError(t, err)

	var modelSpec *ast.Model
	for _, spec := range stencil.Specs {
		if m, ok := spec.(*ast.Model); ok {
			modelSpec = m
			break
		}
	}
	assert.True(t, modelSpec != nil, "expected to find a Model spec")
	assert.Equal(t, 2, len(modelSpec.Fields))
	assert.Equal(t, 1, len(modelSpec.Extensions))
	assert.Equal(t, "openapi", modelSpec.Extensions[0].Target.Value)

	// field decorators must NOT bleed into @raw
	assert.Equal(t, 1, len(modelSpec.Fields[0].Decorators)) // @primary only
	assert.Equal(t, 1, len(modelSpec.Fields[1].Decorators)) // @unique only
}

func TestParseRaw_ApiRestBlock(t *testing.T) {
	t.Parallel()

	input := `namespace test
api SomeService @style(rest) {
  @basePath("/api/v1")

  GET / -> void

  @raw(openapi) {
    "x-service-id": "svc-1"
  }

  @raw(mermaid) {
    "style": "fill:#f9f"
  }
}
`

	p, err := New()
	assert.NoError(t, err)

	stencil, err := p.Parse(input)
	assert.NoError(t, err)

	var apiSpec *ast.Api
	for _, spec := range stencil.Specs {
		if a, ok := spec.(*ast.Api); ok {
			apiSpec = a
			break
		}
	}
	assert.True(t, apiSpec != nil, "expected to find an Api spec")
	assert.Equal(t, 1, len(apiSpec.Routes))
	assert.Equal(t, 2, len(apiSpec.Extensions))
	assert.Equal(t, "openapi", apiSpec.Extensions[0].Target.Value)
	assert.Equal(t, "mermaid", apiSpec.Extensions[1].Target.Value)
}

func TestParseRaw_ApiRpcBlock(t *testing.T) {
	t.Parallel()

	input := `namespace test
api FeedService @style(rpc) {
  rpc GetFeed(GetFeedInput) -> FeedItem @deprecated("use v2")

  @raw(openapi) {
    "x-rpc-version": "1"
  }
}
`

	p, err := New()
	assert.NoError(t, err)

	stencil, err := p.Parse(input)
	assert.NoError(t, err)

	var apiSpec *ast.Api
	for _, spec := range stencil.Specs {
		if a, ok := spec.(*ast.Api); ok {
			apiSpec = a
			break
		}
	}
	assert.True(t, apiSpec != nil, "expected to find an Api spec")
	assert.Equal(t, 1, len(apiSpec.Routes))
	// route decorator must NOT be consumed as @raw
	rpcRoute, ok := apiSpec.Routes[0].(*ast.RpcRoute)
	assert.True(t, ok)
	assert.Equal(t, 1, len(rpcRoute.Decorators))
	assert.Equal(t, "deprecated", rpcRoute.Decorators[0].Name)
	assert.Equal(t, 1, len(apiSpec.Extensions))
	assert.Equal(t, "openapi", apiSpec.Extensions[0].Target.Value)
}

func TestParseRaw_ApiEventBlock(t *testing.T) {
	t.Parallel()

	input := `namespace test
api UserEvents @style(events) {
  publish UserCreated -> UserCreatedEvent @deprecated("use v2")

  @raw(openapi) {
    "x-async": true
  }
}
`

	p, err := New()
	assert.NoError(t, err)

	stencil, err := p.Parse(input)
	assert.NoError(t, err)

	var apiSpec *ast.Api
	for _, spec := range stencil.Specs {
		if a, ok := spec.(*ast.Api); ok {
			apiSpec = a
			break
		}
	}
	assert.True(t, apiSpec != nil, "expected to find an Api spec")
	assert.Equal(t, 1, len(apiSpec.Routes))
	eventRoute, ok := apiSpec.Routes[0].(*ast.EventRoute)
	assert.True(t, ok)
	assert.Equal(t, 1, len(eventRoute.Decorators))
	assert.Equal(t, "deprecated", eventRoute.Decorators[0].Name)
	assert.Equal(t, 1, len(apiSpec.Extensions))
	assert.Equal(t, "openapi", apiSpec.Extensions[0].Target.Value)
}
