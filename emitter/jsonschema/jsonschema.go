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
	schema   resolver.ResolvedSchema
	builder  *schema.Builder
	idPrefix string
	draft    string
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

	return out, nil
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
		schema:  *resolved,
		builder: b,
		draft:   defaultDraftURL,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e, nil
}
