package mermaid

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/emitter"
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

func TestEmitAll(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		wantContains []string
	}{
		{
			name: "basic model",
			src: `
namespace test

model User {
  id:   uuid   @primary
  name: string
}
`,
			wantContains: []string{
				"erDiagram",
				"User {",
				"string(uuid) *id PK",
				"string name",
			},
		},
		{
			name: "enum type with default",
			src: `
namespace test

enum Role {
  admin
  member
  guest
}

model User {
  id:   uuid @primary
  role: Role @default(member)
}
`,
			wantContains: []string{
				"Role {",
				"User {",
				"string(uuid) *id PK",
				"string role \"default: member\"",
			},
		},
		{
			name: "generic monomorph",
			src: `
namespace test

model User {
  id: uuid @primary
}

model Page<T> {
  items: T[]
  total: int
}

api TestApi @style(rest) {
  GET / -> Page<User>
}
`,
			wantContains: []string{
				"PageUser {",
				"ref items FK",
				"integer total",
				"PageUser ||--|{ User",
			},
		},
		{
			name: "required relation preserves label",
			src: `
namespace test

model User {
  id:   uuid   @primary
  name: string
}

model Post {
  id:       uuid @primary
  title:    string
  authorId: uuid
  author:   User @relation(field: authorId)
}
`,
			wantContains: []string{
				"Post {",
				"User {",
				"Post ||--|| User : \"authorId\"",
			},
		},
		{
			name: "optional relation preserves label",
			src: `
namespace test

model User {
  id:   uuid   @primary
  name: string
}

model Post {
  id:       uuid   @primary
  title:    string
  authorId: uuid
  author:   User?  @relation(field: authorId)
}
`,
			wantContains: []string{
				"Post {",
				"User {",
				"Post ||--o| User : \"authorId\"",
			},
		},
		{
			name: "array relation",
			src: `
namespace test

model User {
  id:    uuid   @primary
  name:  string
  posts: Post[] @relation
}

model Post {
  id:    uuid @primary
  title: string
}
`,
			wantContains: []string{
				"User ||--|{ Post",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := parseAndResolve(t, tt.src)
			em, err := New(resolved)
			assert.NoError(t, err)

			outputs, err := em.EmitAll()
			assert.NoError(t, err)

			content := string(outputs[0].Contents())
			for _, want := range tt.wantContains {
				assert.Contains(t, content, want)
			}
		})
	}
}

func TestEmitAll_sorted_output(t *testing.T) {
	src := `
namespace test

model Zeta {
  id: uuid @primary
}

model Alpha {
  id: uuid @primary
}

model Gamma {
  id: uuid @primary
}
`
	resolved := parseAndResolve(t, src)
	em, err := New(resolved)
	assert.NoError(t, err)

	outputs, err := em.EmitAll()
	assert.NoError(t, err)

	content := string(outputs[0].Contents())
	alphaIdx := strings.Index(content, "Alpha {")
	gammaIdx := strings.Index(content, "Gamma {")
	zetaIdx := strings.Index(content, "Zeta {")

	assert.True(t, alphaIdx < gammaIdx, "Alpha should come before Gamma")
	assert.True(t, gammaIdx < zetaIdx, "Gamma should come before Zeta")
}

func TestEmitOne(t *testing.T) {
	const twoModelSrc = `
namespace test

model User {
  id:   uuid   @primary
  name: string
}

model Post {
  id:    uuid @primary
  title: string
}
`

	tests := []struct {
		name            string
		src             string
		specType        emitter.SpecType
		modelName       string
		wantFilename    string
		wantContains    []string
		wantNotContains []string
		wantErr         bool
	}{
		{
			name:            "existing model emits correct file",
			src:             twoModelSrc,
			specType:        emitter.SpecModel,
			modelName:       "User",
			wantFilename:    "user.mmd",
			wantContains:    []string{"User {"},
			wantNotContains: []string{"Post {"},
		},
		{
			name:      "non-existent model returns error",
			src:       "namespace test",
			specType:  emitter.SpecModel,
			modelName: "NonExistent",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := parseAndResolve(t, tt.src)
			em, err := New(resolved)
			assert.NoError(t, err)

			out, err := em.EmitOne(tt.specType, tt.modelName)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantFilename, out.Filename())

			content := string(out.Contents())
			for _, want := range tt.wantContains {
				assert.Contains(t, content, want)
			}
			for _, want := range tt.wantNotContains {
				assert.NotContains(t, content, want)
			}
		})
	}
}

func TestErdFileContents(t *testing.T) {
	f := &erdFile{
		filename: "test.mmd",
		entities: map[string]*schema.Schema{
			"User": {
				Type: "object",
				Properties: map[string]*schema.Schema{
					"id": {
						Type:       "string",
						Format:     "uuid",
						Extensions: map[string]any{"x-primary": struct{}{}},
					},
					"name": {
						Type:        "string",
						Description: "the user's name",
					},
				},
				Required: []string{"id", "name"},
			},
		},
		relations: []relation{
			{from: "User", to: "Post", cardinality: "||--|{", label: "owns"},
		},
	}

	content := string(f.Contents())
	assert.Contains(t, content, "erDiagram")
	assert.Contains(t, content, "User {")
	assert.Contains(t, content, "string(uuid) *id PK")
	assert.Contains(t, content, "string name \"the user's name\"")
	assert.Contains(t, content, "User ||--|{ Post : \"owns\"")
}

func TestCardinality(t *testing.T) {
	tests := []struct {
		name string
		prop *schema.Schema
		want string
	}{
		{
			name: "required single",
			prop: &schema.Schema{Type: "string"},
			want: "||--||",
		},
		{
			name: "optional single",
			prop: &schema.Schema{AnyOf: []*schema.Schema{{Type: "string"}, {Type: "null"}}},
			want: "||--o|",
		},
		{
			name: "required array",
			prop: &schema.Schema{Type: "array", Items: &schema.Schema{Type: "string"}},
			want: "||--|{",
		},
		{
			name: "optional array",
			prop: &schema.Schema{
				AnyOf: []*schema.Schema{
					{Type: "array", Items: &schema.Schema{Type: "string"}},
					{Type: "null"},
				},
			},
			want: "||--o{",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cardinality(tt.prop)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAttributeType(t *testing.T) {
	tests := []struct {
		name string
		prop *schema.Schema
		want string
	}{
		{
			name: "string with format",
			prop: &schema.Schema{Type: "string", Format: "uuid"},
			want: "string(uuid)",
		},
		{
			name: "ref type",
			prop: &schema.Schema{Ref: "#/$defs/User"},
			want: "ref",
		},
		{
			name: "array of refs",
			prop: &schema.Schema{Type: "array", Items: &schema.Schema{Ref: "#/$defs/Post"}},
			want: "ref",
		},
		{
			name: "optional string",
			prop: &schema.Schema{AnyOf: []*schema.Schema{{Type: "string"}, {Type: "null"}}},
			want: "string",
		},
		{
			name: "empty type defaults to any",
			prop: &schema.Schema{},
			want: "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &erdFile{}
			got := e.attributeType(tt.prop)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAttributeName(t *testing.T) {
	tests := []struct {
		name     string
		propName string
		prop     *schema.Schema
		want     string
	}{
		{
			name:     "primary key",
			propName: "id",
			prop:     &schema.Schema{Extensions: map[string]any{"x-primary": struct{}{}}},
			want:     "*id PK",
		},
		{
			name:     "foreign key",
			propName: "author",
			prop:     &schema.Schema{Ref: "#/$defs/User"},
			want:     "author FK",
		},
		{
			name:     "unique",
			propName: "email",
			prop:     &schema.Schema{Extensions: map[string]any{"x-unique": struct{}{}}},
			want:     "email UK",
		},
		{
			name:     "primary and foreign key",
			propName: "id",
			prop: &schema.Schema{
				Ref:        "#/$defs/User",
				Extensions: map[string]any{"x-primary": struct{}{}},
			},
			want: "*id PK,FK",
		},
		{
			name:     "plain field",
			propName: "name",
			prop:     &schema.Schema{Type: "string"},
			want:     "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &erdFile{}
			got := e.attributeName(tt.propName, tt.prop)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRefToName(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want string
	}{
		{"valid ref", "#/$defs/User", "User"},
		{"file ref returns empty", "./user.json", ""},
		{"id ref returns empty", "https://example.com/User", ""},
		{"empty returns empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &emitMermaid{}
			got := e.refToName(tt.ref)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveRef(t *testing.T) {
	direct := &schema.Schema{Ref: "#/$defs/User"}
	array := &schema.Schema{Type: "array", Items: &schema.Schema{Ref: "#/$defs/Post"}}
	optional := &schema.Schema{
		AnyOf: []*schema.Schema{
			{Ref: "#/$defs/User"},
			{Type: "null"},
		},
	}

	tests := []struct {
		name     string
		prop     *schema.Schema
		wantName string
		wantProp *schema.Schema
	}{
		{
			name:     "direct ref",
			prop:     direct,
			wantName: "User",
			wantProp: direct,
		},
		{
			name:     "array ref",
			prop:     array,
			wantName: "Post",
			wantProp: array,
		},
		{
			name:     "nullable ref keeps outer schema",
			prop:     optional,
			wantName: "User",
			wantProp: optional,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &emitMermaid{}
			gotName, gotProp := e.resolveRef(tt.prop)

			assert.Equal(t, tt.wantName, gotName)
			assert.True(t, gotProp == tt.wantProp)
		})
	}
}

func TestNew_nil_returns_error(t *testing.T) {
	_, err := New(nil)
	assert.Error(t, err)
}
