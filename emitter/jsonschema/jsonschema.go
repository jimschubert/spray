package jsonschema

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/jimschubert/spray/ast"
	"github.com/jimschubert/spray/emitter"
	"github.com/jimschubert/spray/emitter/schema"
	"github.com/jimschubert/spray/resolver"
)

const defaultDraftURL = "https://json-schema.org/draft/2020-12/schema"

type schemaFile struct {
	filename string
	schema   *Schema
}

func (s *schemaFile) Filename() string {
	return s.filename
}

func (s *schemaFile) Contents() []byte {
	b, err := json.MarshalIndent(s.schema, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("failed to marshal schema for file %s: %v", s.filename, err))
	}
	return b
}

func (s *schemaFile) ContentType() emitter.ContentType {
	return emitter.ContentText
}

// Schema is a representation of a JSON Schema document (draft 2020-12).
// see: https://json-schema.org/draft/2020-12/json-schema-core
type Schema struct {
	Schema      string             `json:"$schema,omitempty"`
	ID          string             `json:"$id,omitempty"`
	Title       string             `json:"title,omitempty"`
	Type        string             `json:"type,omitempty"`
	Enum        []string           `json:"enum,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Ref         string             `json:"$ref,omitempty"`
	Defs        map[string]*Schema `json:"$defs,omitempty"`
	AnyOf       []*Schema          `json:"anyOf,omitempty"`
	Description string             `json:"description,omitempty"`
	Format      string             `json:"format,omitempty"`
	Default     any                `json:"default,omitempty"`
	Extensions  map[string]any     `json:"-"`
}

func (s Schema) MarshalJSON() ([]byte, error) {
	type Alias Schema
	b, err := json.Marshal(Alias(s))
	if err != nil || len(s.Extensions) == 0 {
		return b, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return b, err
	}
	// merge extensions into the root object; caution: overwrites any existing keys
	for k, v := range s.Extensions {
		m[k] = v
	}
	return json.Marshal(m)
}

type emitJsonSchema struct {
	schema        resolver.ResolvedSchema
	builder       *schema.Builder
	idPrefix      string
	draft         string
	refProcessing string
}

func (e *emitJsonSchema) EmitAll() ([]emitter.Output, error) {
	out := make([]emitter.Output, 0)

	for _, stencil := range e.schema.Stencils {
		for _, node := range stencil.Specs {
			s := e.builder.Spec(node)
			if s != nil {
				out = append(out, &schemaFile{
					filename: fmt.Sprintf("%s.json", strings.ToLower(ast.NameOf(node))),
					schema:   e.asJsonSchema(ast.NameOf(node), s),
				})
			}
		}
	}

	monomorphs := e.schema.Monomorphs()
	monoKeys := make([]string, 0, len(monomorphs))
	for k := range monomorphs {
		monoKeys = append(monoKeys, k)
	}

	// we want stable sorted output
	slices.Sort(monoKeys)

	for _, k := range monoKeys {
		monomorph := monomorphs[k]
		ms := e.builder.MonomorphSchema(monomorph)
		if ms == nil {
			continue
		}
		out = append(out, &schemaFile{
			filename: fmt.Sprintf("%s.json", strings.ToLower(monomorph.Name)),
			schema:   e.asJsonSchema(monomorph.Name, ms),
		})
	}

	slices.SortFunc(out, func(a, b emitter.Output) int {
		return strings.Compare(a.Filename(), b.Filename())
	})

	fileBase := make(map[string]*schemaFile, len(out))
	for _, o := range out {
		f := o.(*schemaFile)
		fileBase[strings.TrimSuffix(f.filename, ".json")] = f
	}

	switch e.refProcessing {
	case "file":
		// e.g. rewrite #/$defs/User → ./user.json
		for _, o := range out {
			visitRefs(o.(*schemaFile).schema, func(ref string) string {
				if !strings.HasPrefix(ref, "#/$defs/") {
					return ref
				}
				return "./" + strings.ToLower(strings.TrimPrefix(ref, "#/$defs/")) + ".json"
			})
		}
	case "inline":
		// e.g. embed #/$defs/User schema into the root schema's $defs
		for _, o := range out {
			root := o.(*schemaFile).schema
			visitRefs(root, func(ref string) string {
				if !strings.HasPrefix(ref, "#/$defs/") {
					return ref
				}
				refName := strings.TrimPrefix(ref, "#/$defs/")
				if _, exists := root.Defs[refName]; !exists {
					if target, ok := fileBase[strings.ToLower(refName)]; ok {
						// shallow copy to prevent circular references (e.g. User -> []Post -> User)
						def := *target.schema
						def.Defs = make(map[string]*Schema)
						root.Defs[refName] = &def
					}
				}
				return ref
			})
		}
	case "id":
		// change #/$defs/User to the $id of the User schema (e.g. "https://example.com/User")
		for _, o := range out {
			visitRefs(o.(*schemaFile).schema, func(ref string) string {
				if !strings.HasPrefix(ref, "#/$defs/") {
					return ref
				}
				refName := strings.TrimPrefix(ref, "#/$defs/")
				if target, ok := fileBase[strings.ToLower(refName)]; ok && target.schema.ID != "" {
					return target.schema.ID
				}
				return ref
			})
		}
	}

	return out, nil
}

// visitRefs traverses the schema and applies the given function to each $ref
func visitRefs(s *Schema, fn func(ref string) string) {
	// visited - avoids infinite recursion on circular references
	visited := make(map[*Schema]bool)
	var visit func(*Schema)
	visit = func(s *Schema) {
		if s == nil || visited[s] {
			return
		}
		visited[s] = true
		if s.Ref != "" {
			s.Ref = fn(s.Ref)
		}
		visit(s.Items)
		for _, prop := range s.Properties {
			visit(prop)
		}
		for _, def := range s.Defs {
			visit(def)
		}
		for _, anyOf := range s.AnyOf {
			visit(anyOf)
		}
	}
	visit(s)
}

// EmitOne emits a single spec of the given type and name.
func (e *emitJsonSchema) EmitOne(typ emitter.SpecType, name string) (emitter.Output, error) {
	// TODO implement me
	panic("implement me")
}

func (e *emitJsonSchema) asJsonSchema(fqn string, generalSchema *schema.Schema) *Schema {
	if generalSchema == nil {
		return nil
	}

	result := Schema{
		Type:        generalSchema.Type,
		Format:      generalSchema.Format,
		Enum:        generalSchema.Enum,
		Ref:         generalSchema.Ref,
		Description: generalSchema.Description,
		Default:     generalSchema.Default,
		Properties:  make(map[string]*Schema),
		Defs:        make(map[string]*Schema),
		Required:    generalSchema.Required,
		Items:       e.asJsonSchema("", generalSchema.Items),
		AnyOf:       make([]*Schema, len(generalSchema.AnyOf)),
		Extensions:  make(map[string]any),
	}

	if fqn != "" {
		result.Schema = e.draft
		result.ID = e.idPrefix + fqn
		result.Title = fqn
	}

	for key, prop := range generalSchema.Properties {
		result.Properties[key] = e.asJsonSchema("", prop)
	}

	for key, def := range generalSchema.Defs {
		result.Defs[key] = e.asJsonSchema("", def)
	}

	for i, anyOf := range generalSchema.AnyOf {
		result.AnyOf[i] = e.asJsonSchema("", anyOf)
	}

	for key, value := range generalSchema.Extensions {
		result.Extensions[key] = value
	}

	return new(result)
}

// New creates a new JSON Schema emitter with the given resolved schema.
func New(resolved *resolver.ResolvedSchema, opts ...Options) (emitter.Emitter, error) {
	if resolved == nil {
		return nil, fmt.Errorf("schema cannot be nil")
	}

	b := schema.NewBuilder(*resolved).WithNullableStrategy(schema.NullableAnyOf).WithRefStrategy(schema.RefDefs)
	e := &emitJsonSchema{
		schema:        *resolved,
		builder:       b,
		draft:         defaultDraftURL,
		refProcessing: "file",
	}

	for _, opt := range opts {
		opt(e)
	}

	return e, nil
}
