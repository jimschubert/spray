package jsonschema

import (
	"encoding/json"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/emitter/schema"
	"github.com/jimschubert/spray/parser"
	"github.com/jimschubert/spray/resolver"
)

func parseAndResolve(t *testing.T, src string) *resolver.ResolvedSchema {
	t.Helper()
	p, err := parser.New()
	assert.NoError(t, err)

	stencil, err := p.Parse(src)
	assert.NoError(t, err)

	r := resolver.New(stencil)
	resolved, err := r.Resolve()
	assert.NoError(t, err)

	return resolved
}

func TestSchemaFileContents(t *testing.T) {
	tests := []struct {
		name         string
		schema       *Schema
		wantContains []string
	}{
		{
			name: "simple schema with title and type",
			schema: &Schema{
				Title: "User",
				Type:  "object",
			},
			wantContains: []string{"User", "object"},
		},
		{
			name: "schema with properties",
			schema: &Schema{
				Title: "Post",
				Type:  "object",
				Properties: map[string]*Schema{
					"id": {
						Type: "string",
					},
					"title": {
						Type: "string",
					},
				},
			},
			wantContains: []string{"Post", "properties", "id", "title"},
		},
		{
			name: "schema with required fields",
			schema: &Schema{
				Title:    "Comment",
				Type:     "object",
				Required: []string{"id", "content"},
			},
			wantContains: []string{"required", "id", "content"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sf := &schemaFile{
				filename: "test.json",
				schema:   tt.schema,
			}
			contents := sf.Contents()
			assert.True(t, len(contents) > 0)

			for _, want := range tt.wantContains {
				assert.True(t, json.Valid(contents), "schema contents should be valid JSON")
				assert.Contains(t, string(contents), want)
			}
		})
	}
}

func TestSchemaMarshalJSON(t *testing.T) {
	tests := []struct {
		name       string
		schema     *Schema
		wantFields map[string]bool
	}{
		{
			name: "simple schema marshals basic fields",
			schema: &Schema{
				Title: "User",
				Type:  "object",
			},
			wantFields: map[string]bool{
				"title": true,
				"type":  true,
			},
		},
		{
			name: "schema with extensions merges into JSON",
			schema: &Schema{
				Title: "User",
				Type:  "object",
				Extensions: map[string]any{
					"x-custom":  "value",
					"x-another": 42,
				},
			},
			wantFields: map[string]bool{
				"title":     true,
				"type":      true,
				"x-custom":  true,
				"x-another": true,
			},
		},
		{
			name: "schema with $schema, $id and title",
			schema: &Schema{
				Schema: "https://json-schema.org/draft/2020-12/schema",
				ID:     "https://example.com/user",
				Title:  "User",
				Type:   "object",
			},
			wantFields: map[string]bool{
				"$schema": true,
				"$id":     true,
				"title":   true,
				"type":    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.schema)
			assert.NoError(t, err)

			var m map[string]any
			err = json.Unmarshal(b, &m)
			assert.NoError(t, err)

			for field, shouldExist := range tt.wantFields {
				_, exists := m[field]
				assert.Equal(t, shouldExist, exists, "field %q should exist: %v", field, shouldExist)
			}
		})
	}
}

func TestAsJsonSchema(t *testing.T) {
	resolved := parseAndResolve(t, `namespace test`)
	b := schema.NewBuilder(*resolved).WithNullableStrategy(schema.NullableAnyOf).WithRefStrategy(schema.RefDefs)
	e := &emitJsonSchema{
		schema:  *resolved,
		builder: b,
		draft:   "https://json-schema.org/draft/2020-12/schema",
	}

	tests := []struct {
		name           string
		fqn            string
		input          *schema.Schema
		wantNil        bool
		wantDraft      string
		wantID         string
		wantTitle      string
		wantPropsLen   int
		wantDefsLen    int
		wantAnyOfTypes []string
		wantExts       map[string]any
		wantRequired   []string
	}{
		{
			name:    "nil input returns nil",
			fqn:     "test.User",
			input:   nil,
			wantNil: true,
		},
		{
			name:  "empty fqn omits schema, id, and title",
			fqn:   "",
			input: &schema.Schema{Type: "string"},
		},
		{
			name:      "fqn sets draft, id, and title",
			fqn:       "test.User",
			input:     &schema.Schema{Type: "object"},
			wantDraft: "https://json-schema.org/draft/2020-12/schema",
			wantID:    "test.User",
			wantTitle: "test.User",
		},
		{
			name: "properties are converted",
			fqn:  "test.User",
			input: &schema.Schema{
				Type: "object",
				Properties: map[string]*schema.Schema{
					"id":   {Type: "string"},
					"name": {Type: "string"},
				},
				Required: []string{"id"},
			},
			wantPropsLen: 2,
			wantRequired: []string{"id"},
		},
		{
			name: "defs are converted",
			fqn:  "test.User",
			input: &schema.Schema{
				Type: "object",
				Defs: map[string]*schema.Schema{
					"Address": {Type: "object"},
					"Contact": {Type: "object"},
				},
			},
			wantDefsLen: 2,
		},
		{
			name: "anyOf entries are converted",
			fqn:  "test.Optional",
			input: &schema.Schema{
				AnyOf: []*schema.Schema{
					{Type: "string"},
					{Type: "null"},
				},
			},
			wantAnyOfTypes: []string{"string", "null"},
		},
		{
			name: "extensions are copied",
			fqn:  "test.User",
			input: &schema.Schema{
				Type: "object",
				Extensions: map[string]any{
					"x-custom":     "value",
					"x-deprecated": true,
				},
			},
			wantExts: map[string]any{
				"x-custom":     "value",
				"x-deprecated": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := e.asJsonSchema(tt.fqn, tt.input)

			if tt.wantNil {
				assert.True(t, got == nil)
				return
			}

			assert.True(t, got != nil)
			if tt.wantDraft != "" {
				assert.Equal(t, tt.wantDraft, got.Schema)
			}
			if tt.wantID != "" {
				assert.Contains(t, got.ID, tt.wantID)
			}
			if tt.wantTitle != "" {
				assert.Equal(t, tt.wantTitle, got.Title)
			}
			assert.Equal(t, tt.wantPropsLen, len(got.Properties))
			assert.Equal(t, tt.wantDefsLen, len(got.Defs))
			assert.Equal(t, len(tt.wantAnyOfTypes), len(got.AnyOf))
			for i, wantType := range tt.wantAnyOfTypes {
				assert.Equal(t, wantType, got.AnyOf[i].Type)
			}
			assert.Equal(t, len(tt.wantExts), len(got.Extensions))
			for k, v := range tt.wantExts {
				assert.Equal(t, v, got.Extensions[k])
			}
			assert.Equal(t, tt.wantRequired, got.Required)
		})
	}
}

func TestNewSuccess(t *testing.T) {
	tests := []struct {
		name string
		opts []Options
	}{
		{
			name: "without options",
			opts: nil,
		},
		{
			name: "with IDPrefix option",
			opts: []Options{WithIDPrefix("https://example.com/schemas/")},
		},
		{
			name: "with Draft option",
			opts: []Options{WithDraft("2019-09")},
		},
		{
			name: "with multiple options",
			opts: []Options{
				WithIDPrefix("https://example.com/"),
				WithDraft("2020-12"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := parseAndResolve(t, `namespace test`)

			emitterImpl, err := New(resolved, tt.opts...)
			assert.NoError(t, err)
			assert.True(t, emitterImpl != nil)
		})
	}
}

func TestEmitAll(t *testing.T) {
	tests := []struct {
		name          string
		src           string
		opts          []Options
		wantFilenames []string
	}{
		{
			name: "enum and model emit separate files",
			src: `
namespace acme.users

enum Role {
  admin
  member
}

model User {
  id:   uuid
  role: Role
}
`,
			opts:          []Options{WithIDPrefix("https://acme.com/schemas/")},
			wantFilenames: []string{"user.json", "role.json"},
		},
		{
			name: "single model emits single JSON",
			src: `
namespace test.v1

model Product {
  name:  string
  price: float
}
`,
			wantFilenames: []string{"product.json"},
		},
		{
			name: "multiple models each emit a file",
			src: `
namespace blog

model Author {
  id:   uuid
  name: string
}

model Post {
  id:       uuid
  title:    string
  authorId: uuid
}

model Comment {
  id:     uuid
  postId: uuid
  text:   string
}
`,
			wantFilenames: []string{"author.json", "post.json", "comment.json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := parseAndResolve(t, tt.src)
			emitterImpl, err := New(resolved, tt.opts...)
			assert.NoError(t, err)

			outputs, err := emitterImpl.EmitAll()
			assert.NoError(t, err)
			assert.True(t, len(outputs) > 0)

			filenames := make(map[string]bool, len(outputs))
			for _, out := range outputs {
				assert.True(t, json.Valid(out.Contents()), "output should be valid JSON for %s", out.Filename())
				filenames[out.Filename()] = true
			}

			for _, want := range tt.wantFilenames {
				assert.True(t, filenames[want], "expected output file %q", want)
			}

			for i := 1; i < len(outputs); i++ {
				prev, cur := outputs[i-1].Filename(), outputs[i].Filename()
				assert.True(t, prev <= cur, "EmitAll outputs should be sorted by filename: %q > %q", prev, cur)
			}
		})
	}
}

func TestEmitAll_monomorphs(t *testing.T) {
	src := `
namespace test

model User {
  id: uuid
}

model Page<T> {
  items: T[]
  total: int
}

api TestApi @style(rest) {
  GET / -> Page<User>
}
`
	resolved := parseAndResolve(t, src)
	prefix := "https://example.com/schemas/"
	em, err := New(resolved, WithIDPrefix(prefix))
	assert.NoError(t, err)

	outputs, err := em.EmitAll()
	assert.NoError(t, err)

	var pageUserDoc map[string]any
	found := false
	for _, o := range outputs {
		if o.Filename() == "pageuser.json" {
			found = true
			assert.NoError(t, json.Unmarshal(o.Contents(), &pageUserDoc))
			break
		}
	}
	assert.True(t, found, "expected pageuser.json to be emitted")

	title, _ := pageUserDoc["title"].(string)
	id, _ := pageUserDoc["$id"].(string)
	schemaURL, _ := pageUserDoc["$schema"].(string)
	assert.Equal(t, "PageUser", title)
	assert.Equal(t, prefix+"PageUser", id)
	assert.Equal(t, defaultDraftURL, schemaURL)
}

func TestNew_setsDraft(t *testing.T) {
	resolved := parseAndResolve(t, `
namespace x

model M {
  n: string
}
`)
	em, err := New(resolved)
	assert.NoError(t, err)
	outputs, err := em.EmitAll()
	assert.NoError(t, err)
	assert.True(t, len(outputs) >= 1)

	var doc map[string]any
	assert.NoError(t, json.Unmarshal(outputs[0].Contents(), &doc))
	schemaURL, ok := doc["$schema"].(string)
	assert.True(t, ok)
	assert.Equal(t, defaultDraftURL, schemaURL)
}
