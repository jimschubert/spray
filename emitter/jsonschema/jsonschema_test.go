package jsonschema

import (
	"encoding/json"
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

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		opts       []Options
		wantDraft  string
		wantID     string
		wantSchema bool
	}{
		{
			name:      "without options",
			src:       `
namespace test

model M {
  n: string
}
`,
			wantDraft: defaultDraftURL,
			wantSchema: true,
		},
		{
			name: "with IDPrefix option",
			src:  `namespace test`,
			opts: []Options{WithIDPrefix("https://example.com/schemas/")},
		},
		{
			name: "with Draft option",
			src:  `namespace test`,
			opts: []Options{WithDraft("2019-09")},
		},
		{
			name: "with multiple options",
			src:  `namespace test`,
			opts: []Options{WithIDPrefix("https://example.com/"), WithDraft("2020-12")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := parseAndResolve(t, tt.src)

			emitterImpl, err := New(resolved, tt.opts...)
			assert.NoError(t, err)
			assert.True(t, emitterImpl != nil)

			if !tt.wantSchema {
				return
			}

			outputs, err := emitterImpl.EmitAll()
			assert.NoError(t, err)
			assert.True(t, len(outputs) >= 1)

			var doc map[string]any
			assert.NoError(t, json.Unmarshal(outputs[0].Contents(), &doc))
			schemaURL, ok := doc["$schema"].(string)
			assert.True(t, ok)
			assert.Equal(t, tt.wantDraft, schemaURL)
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

func TestVisitRefs(t *testing.T) {
	replace := func(ref string) string { return "replaced:" + ref }

	tests := []struct {
		name      string
		setup     func() *Schema
		input     *Schema
		wantRoot  string
		wantProps map[string]string
		wantDefs  map[string]string
		wantAnyOf []string
		wantItems string
		wantPanic bool
	}{
		{
			name:     "root ref is visited",
			input:    &Schema{Ref: "#/$defs/Foo"},
			wantRoot: "replaced:#/$defs/Foo",
		},
		{
			name: "property refs are visited",
			input: &Schema{
				Properties: map[string]*Schema{
					"a": {Ref: "#/$defs/A"},
					"b": {Type: "string"},
				},
			},
			wantProps: map[string]string{
				"a": "replaced:#/$defs/A",
				"b": "",
			},
		},
		{
			name: "items ref is visited",
			input: &Schema{
				Type:  "array",
				Items: &Schema{Ref: "#/$defs/Item"},
			},
			wantItems: "replaced:#/$defs/Item",
		},
		{
			name: "property items ref is visited",
			input: &Schema{
				Properties: map[string]*Schema{
					"tags": {Type: "array", Items: &Schema{Ref: "#/$defs/Tag"}},
				},
			},
			wantProps: map[string]string{
				"tags": "",
			},
		},
		{
			name: "def refs are visited",
			input: &Schema{
				Defs: map[string]*Schema{
					"X": {Ref: "#/$defs/X"},
				},
			},
			wantDefs: map[string]string{
				"X": "replaced:#/$defs/X",
			},
		},
		{
			name: "anyOf refs are visited",
			input: &Schema{
				AnyOf: []*Schema{
					{Ref: "#/$defs/A"},
					{Type: "null"},
				},
			},
			wantAnyOf: []string{"replaced:#/$defs/A", ""},
		},
		{
			name:     "empty refs are skipped",
			input:    &Schema{Properties: map[string]*Schema{"a": {Type: "string"}}},
			wantRoot: "",
		},
		{
			name: "cycles do not panic",
			setup: func() *Schema {
				root := &Schema{Type: "object"}
				other := &Schema{Type: "object", Defs: map[string]*Schema{"Root": root}}
				root.Defs = map[string]*Schema{"Other": other}
				return root
			},
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			if tt.setup != nil {
				input = tt.setup()
			}

			assert.NotPanics(t, func() {
				visitRefs(input, replace)
			})

			if tt.wantPanic {
				return
			}

			assert.Equal(t, tt.wantRoot, input.Ref)
			for key, want := range tt.wantProps {
				assert.Equal(t, want, input.Properties[key].Ref)
			}
			for key, want := range tt.wantDefs {
				assert.Equal(t, want, input.Defs[key].Ref)
			}
			for i, want := range tt.wantAnyOf {
				assert.Equal(t, want, input.AnyOf[i].Ref)
			}
			if tt.wantItems != "" {
				assert.True(t, input.Items != nil)
				assert.Equal(t, tt.wantItems, input.Items.Ref)
			}
		})
	}
}

func TestEmitOne(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		typ          emitter.SpecType
		specName     string
		wantFilename string
		wantTitle    string
		wantErr      string
	}{
		{
			name: "emit single model",
			src: `
namespace test

model User {
  id:   uuid
  name: string
}

model Post {
  id:    uuid
  title: string
}
`,
			typ:          emitter.SpecModel,
			specName:     "User",
			wantFilename: "user.json",
			wantTitle:    "User",
		},
		{
			name: "emit single enum",
			src: `
namespace test

enum Role {
  admin
  member
}

enum Status {
  active
  inactive
}
`,
			typ:          emitter.SpecEnum,
			specName:     "Role",
			wantFilename: "role.json",
			wantTitle:    "Role",
		},
		{
			name: "emit single input",
			src: `
namespace test

input CreateUserInput {
  name:  string
  email: string
}

input UpdateUserInput {
  id:   uuid
  name: string
}
`,
			typ:          emitter.SpecInput,
			specName:     "CreateUserInput",
			wantFilename: "createuserinput.json",
			wantTitle:    "CreateUserInput",
		},
		{
			name: "model not found",
			src: `
namespace test

model User {
  id: uuid
}
`,
			typ:      emitter.SpecModel,
			specName: "NotFound",
			wantErr:  `"NotFound" not found`,
		},
		{
			name: "spec type not found",
			src: `
namespace test
`,
			typ:      emitter.SpecApi,
			specName: "SomeApi",
			wantErr:  `"SomeApi" not found`,
		},
		{
			name: "emit model with references",
			src: `
namespace test

model Address {
  street: string
  city:   string
}

model User {
  id:      uuid
  name:    string
  address: Address
}
`,
			typ:          emitter.SpecModel,
			specName:     "User",
			wantFilename: "user.json",
			wantTitle:    "User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := parseAndResolve(t, tt.src)
			emitterImpl, err := New(resolved)
			assert.NoError(t, err)

			output, err := emitterImpl.EmitOne(tt.typ, tt.specName)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			assert.NoError(t, err)
			assert.True(t, output != nil)
			assert.Equal(t, tt.wantFilename, output.Filename())
			assert.True(t, json.Valid(output.Contents()), "output should be valid JSON")

			var doc map[string]any
			assert.NoError(t, json.Unmarshal(output.Contents(), &doc))

			if tt.wantTitle != "" {
				title, ok := doc["title"].(string)
				assert.True(t, ok)
				assert.Equal(t, tt.wantTitle, title)
			}
		})
	}
}

func TestEmitOne_refProcessing(t *testing.T) {
	src := `
namespace test

model Address {
  street: string
  city:   string
}

model User {
  id:      uuid
  address: Address
}
`

	tests := []struct {
		name       string
		opts       []Options
		wantRef    string
		wantHasDef bool
	}{
		{
			name:       "file mode rewrites ref to relative path",
			opts:       []Options{WithRefProcessing("file")},
			wantRef:    "./address.json",
			wantHasDef: false,
		},
		{
			name:       "inline mode keeps ref and populates $defs",
			opts:       []Options{WithRefProcessing("inline")},
			wantRef:    "#/$defs/Address",
			wantHasDef: true,
		},
		{
			name:       "id mode rewrites ref to $id URI",
			opts:       []Options{WithRefProcessing("id"), WithIDPrefix("https://example.com/")},
			wantRef:    "https://example.com/Address",
			wantHasDef: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := parseAndResolve(t, src)
			emitterImpl, err := New(resolved, tt.opts...)
			assert.NoError(t, err)

			output, err := emitterImpl.EmitOne(emitter.SpecModel, "User")
			assert.NoError(t, err)
			assert.True(t, output != nil)

			sf := output.(*schemaFile)
			addrProp := sf.schema.Properties["address"]
			assert.True(t, addrProp != nil)
			assert.Equal(t, tt.wantRef, addrProp.Ref)

			_, hasDef := sf.schema.Defs["Address"]
			assert.Equal(t, tt.wantHasDef, hasDef)
		})
	}
}

func TestEmitAll_refProcessing(t *testing.T) {
	const refSrc = `
namespace test

model Address {
  street: string
  city:   string
}

model User {
  id:      uuid
  address: Address
}
`
	const arrayRefSrc = `
namespace test

model Tag {
  name: string
}

model Post {
  id:   uuid
  tags: Tag[]
}
`

	tests := []struct {
		name          string
		src           string
		opts          []Options
		wantFilename  string
		wantRef       string
		wantHasDef    bool
		wantDefName   string
		wantItemsRef  string
		wantDefFields []string
	}{
		{
			name:         "default strategy is file",
			src:          refSrc,
			wantFilename: "user.json",
			wantRef:      "./address.json",
			wantHasDef:   false,
		},
		{
			name:         "file rewrites ref to relative path",
			src:          refSrc,
			opts:         []Options{WithRefProcessing("file")},
			wantFilename: "user.json",
			wantRef:      "./address.json",
			wantHasDef:   false,
		},
		{
			name:         "inline keeps ref and populates root $defs",
			src:          refSrc,
			opts:         []Options{WithRefProcessing("inline")},
			wantFilename: "user.json",
			wantRef:      "#/$defs/Address",
			wantHasDef:   true,
			wantDefName:  "Address",
			wantDefFields: []string{"street", "city"},
		},
		{
			name:         "id rewrites ref to the $id URI of the referenced schema",
			src:          refSrc,
			opts:         []Options{WithRefProcessing("id"), WithIDPrefix("https://example.com/")},
			wantFilename: "user.json",
			wantRef:      "https://example.com/Address",
			wantHasDef:   false,
		},
		{
			name:         "inline preserves array item refs",
			src:          arrayRefSrc,
			opts:         []Options{WithRefProcessing("inline")},
			wantFilename: "post.json",
			wantRef:      "",
			wantHasDef:   true,
			wantDefName:  "Tag",
			wantItemsRef: "#/$defs/Tag",
			wantDefFields: []string{"name"},
		},
		{
			name: "inline defs strip root-level metadata",
			src: `
namespace test

model Address {
  street: string
}

model User {
  id:      uuid
  address: Address
}
`,
			opts:         []Options{WithRefProcessing("inline"), WithIDPrefix("https://example.com/")},
			wantFilename: "user.json",
			wantRef:      "#/$defs/Address",
			wantHasDef:   true,
			wantDefName:  "Address",
			wantDefFields: []string{"street"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := tt.src
			resolved := parseAndResolve(t, src)
			emitterImpl, err := New(resolved, tt.opts...)
			assert.NoError(t, err)

			outputs, err := emitterImpl.EmitAll()
			assert.NoError(t, err)

			var userFile *schemaFile
			for _, o := range outputs {
				if o.Filename() == tt.wantFilename {
					userFile = o.(*schemaFile)
					break
				}
			}
			assert.True(t, userFile != nil, "expected %s in outputs", tt.wantFilename)

			if tt.wantFilename == "user.json" {
				addrProp := userFile.schema.Properties["address"]
				assert.True(t, addrProp != nil, "expected address property in user schema")
				assert.Equal(t, tt.wantRef, addrProp.Ref)
			}

			if tt.wantFilename == "post.json" {
				tagsProp := userFile.schema.Properties["tags"]
				assert.True(t, tagsProp != nil, "expected tags property in post schema")
				assert.True(t, tagsProp.Items != nil, "expected tags.items")
				assert.Equal(t, tt.wantItemsRef, tagsProp.Items.Ref)
			}

			def, hasDef := userFile.schema.Defs[tt.wantDefName]
			assert.Equal(t, tt.wantHasDef, hasDef)
			if tt.wantHasDef {
				assert.Equal(t, "object", def.Type)
				for _, field := range tt.wantDefFields {
					assert.True(t, def.Properties[field] != nil)
				}
				// inlined defs should not have root-level metadata
				assert.Equal(t, "", def.Schema, "inlined def should not have $schema")
				assert.Equal(t, "", def.ID, "inlined def should not have $id")
				assert.Equal(t, "", def.Title, "inlined def should not have title")
				assert.Equal(t, 0, len(def.Defs), "inlined def should not have nested $defs")
			}
		})
	}
}
