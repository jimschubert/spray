package schema

import (
	"fmt"
	"strings"

	"github.com/jimschubert/spray/ast"
	"github.com/jimschubert/spray/internal/sequence"
	"github.com/jimschubert/spray/resolver"
)

// NullableStrategy controls how nullable (optional) fields are defined.
type NullableStrategy int

const (
	// NullableAnyOf uses a JSON Schema "anyOf" with one element having type "null".
	NullableAnyOf NullableStrategy = iota

	// NullableKeyword is used by OpenAPI 3.0.x and emits nullable=true on the field.
	NullableKeyword
)

// RefStrategy controls how `$ref` paths are defined.
type RefStrategy int

const (
	// RefDefs uses JSON Schema $defs such as "#/$defs/User"
	RefDefs RefStrategy = iota

	// RefComponents uses OpenAPI component paths such as "#/components/schemas/User"
	RefComponents
)

// Schema is a common schema for OpenAPI and JSON Schema outputs.
type Schema struct {
	// Common fields for both outputs
	Type        string
	Format      string
	Enum        []string
	Properties  map[string]*Schema
	Required    []string
	Items       *Schema
	Ref         string
	Description string
	Default     any
	ReadOnly    bool
	WriteOnly   bool

	// Nullable as determined by NullableStrategy (OpenAPI 3.0)
	Nullable bool

	// AnyOf used for nullability as determined by NullableStrategy (JSON Schema / OpenAPI 3.1)
	AnyOf []*Schema

	// Defs used for $defs or component
	Defs map[string]*Schema

	// Extensions from @raw blocks
	Extensions map[string]any
}

// Builder constructs Schema objects from AST nodes using specified strategies for nullability and references.
type Builder struct {
	nullableStrategy NullableStrategy
	refStrategy      RefStrategy
	schema           resolver.ResolvedSchema
}

// NewBuilder creates a Builder for the given resolved schema with default strategies (NullableAnyOf, RefDefs).
func NewBuilder(schema resolver.ResolvedSchema) *Builder {
	return &Builder{
		nullableStrategy: NullableAnyOf,
		refStrategy:      RefDefs,
		schema:           schema,
	}
}

// WithNullableStrategy returns a new Builder with the specified NullableStrategy.
func (b *Builder) WithNullableStrategy(strategy NullableStrategy) *Builder {
	return &Builder{
		nullableStrategy: strategy,
		refStrategy:      b.refStrategy,
		schema:           b.schema,
	}
}

// WithRefStrategy returns a new Builder with the specified RefStrategy.
func (b *Builder) WithRefStrategy(strategy RefStrategy) *Builder {
	return &Builder{
		nullableStrategy: b.nullableStrategy,
		refStrategy:      strategy,
		schema:           b.schema,
	}
}

// Scalar returns a Schema for a primitive scalar type name.
func (b *Builder) Scalar(name string) *Schema {
	var value, format string
	switch name {
	case "string":
		value = "string"
	case "int":
		value = "integer"
	case "float":
		value = "number"
		format = "float"
	case "boolean":
		value = "boolean"
	case "uuid":
		value = "string"
		format = "uuid"
	case "timestamp":
		value = "string"
		format = "date-time"
	case "date":
		value = "string"
		format = "date"
	case "any":
	default:
		value = "string"
	}

	return &Schema{Type: value, Format: format}
}

// Spec dispatches an AST SpecNode to the appropriate schema builder method based on its concrete type.
func (b *Builder) Spec(node ast.SpecNode) *Schema {
	switch def := node.(type) {
	case *ast.Enum:
		return b.Enum(def)
	case *ast.Model:
		return b.Model(def)
	case *ast.Input:
		return b.Input(def)
	case *ast.Api:
		return b.Api(def)
	default:
		return nil
	}
}

// TypeNode dispatches an AST TypeNode to the appropriate schema builder method based on its concrete type.
func (b *Builder) TypeNode(node ast.TypeNode) *Schema {
	switch n := node.(type) {
	case *ast.TypeExpression:
		return b.TypeExpression(n)
	case *ast.StringLiteral:
		return &Schema{Type: "string", Enum: []string{n.Value}}
	case *ast.IntLiteral:
		return &Schema{Type: "integer", Enum: []string{fmt.Sprintf("%d", n.Value)}}
	case *ast.FloatLiteral:
		return &Schema{Type: "number", Enum: []string{fmt.Sprintf("%g", n.Value)}}
	default:
		return nil
	}
}

// Enum returns a Schema of type string with enum's values.
func (b *Builder) Enum(enum *ast.Enum) *Schema {
	values := make([]string, 0, len(enum.Elements))
	for i := range enum.Elements {
		values = append(values, enum.Elements[i].Value)
	}
	return &Schema{Type: "string", Enum: values}
}

// Model returns a Schema of type "object".
func (b *Builder) Model(model *ast.Model) *Schema {
	s := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
		Extensions: make(map[string]any),
	}

	if model.HeadComment != nil && !model.HeadComment.IsEmpty() {
		s.Description = model.HeadComment.String()
	}

	for i := range model.Fields {
		f := &model.Fields[i]
		fs := b.field(f, model.GenericParams)
		if fs != nil {
			s.Properties[f.Name.Value] = fs
			if !f.Type.IsOptional {
				s.Required = append(s.Required, f.Name.Value)
			}
		}
	}

	for i := range model.Extensions {
		b.applyRawSchema(&model.Extensions[i], s)
	}

	return s
}

// Input returns a Schema of type "object".
func (b *Builder) Input(input *ast.Input) *Schema {
	s := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
		Extensions: make(map[string]any),
	}

	if input.HeadComment != nil && !input.HeadComment.IsEmpty() {
		s.Description = input.HeadComment.String()
	}

	for i := range input.Fields {
		f := &input.Fields[i]
		fs := b.field(f, nil)
		if fs != nil {
			s.Properties[f.Name.Value] = fs
			if !f.Type.IsOptional {
				s.Required = append(s.Required, f.Name.Value)
			}
		}
	}

	return s
}

// Api returns nil — APIs are not schema types; openapi emitter will need to create separately.
func (b *Builder) Api(_ *ast.Api) *Schema {
	return nil
}

// MonomorphSchema returns a schema for the resolved concrete type of a generic, as defined by the provided monomorph.
func (b *Builder) MonomorphSchema(mono resolver.Monomorph) *Schema {
	original := *mono.Original
	model, ok := original.(*ast.Model)
	if !ok {
		return nil
	}

	// build substitution map: generic param name -> concrete type expression
	subs := make(map[string]*ast.TypeExpression, len(model.GenericParams))
	for i, param := range model.GenericParams {
		if i < len(mono.Args) {
			subs[param.Value] = new(mono.Args[i])
		}
	}

	s := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
		Extensions: make(map[string]any),
	}

	if model.HeadComment != nil && !model.HeadComment.IsEmpty() {
		s.Description = model.HeadComment.String()
	}

	for i := range model.Fields {
		f := &model.Fields[i]
		fs := b.fieldWithSubs(f, subs)
		if fs != nil {
			s.Properties[f.Name.Value] = fs
			if !f.Type.IsOptional {
				s.Required = append(s.Required, f.Name.Value)
			}
		}
	}

	for i := range model.Extensions {
		b.applyRawSchema(&model.Extensions[i], s)
	}

	return s
}

// TypeExpression dispatches a type expression to the appropriate schema representation.
func (b *Builder) TypeExpression(expr *ast.TypeExpression) *Schema {
	return b.typeExpression(expr, nil)
}

func (b *Builder) typeExpression(expr *ast.TypeExpression, genericsInScope []ast.StringLiteral) *Schema {
	if expr == nil {
		return nil
	}

	// check if this references a generic parameter in scope (e.g. T inside model Page<T>)
	if b.isGenericParam(expr, genericsInScope) {
		return b.wrapType(&Schema{}, expr.IsArray, expr.IsOptional)
	}

	base := b.resolveLinked(expr, genericsInScope, nil)
	return b.wrapType(base, expr.IsArray, expr.IsOptional)
}

// typeExpressionWithSubs resolves a type expression using concrete substitutions for generic parameters.
func (b *Builder) typeExpressionWithSubs(expr *ast.TypeExpression, subs map[string]*ast.TypeExpression) *Schema {
	if expr == nil {
		return nil
	}

	if len(expr.Base.Parts) == 1 {
		if sub, ok := subs[expr.Base.Parts[0]]; ok {
			base := b.typeExpression(sub, nil)
			return b.wrapType(base, expr.IsArray, expr.IsOptional)
		}
	}

	base := b.resolveLinked(expr, nil, subs)
	return b.wrapType(base, expr.IsArray, expr.IsOptional)
}

// wrapType applies array and optional wrappers around a base schema.
func (b *Builder) wrapType(base *Schema, isArray bool, isOptional bool) *Schema {
	if isArray {
		s := &Schema{Type: "array", Items: base}
		if isOptional {
			return b.nullable(s)
		}
		return s
	}
	if isOptional {
		return b.nullable(base)
	}
	return base
}

// resolveLinked resolves a linked type, handling both generic substitution (subs)
// and generic parameters in scope. Unresolved types emit as $ref.
func (b *Builder) resolveLinked(expr *ast.TypeExpression, genericsInScope []ast.StringLiteral, subs map[string]*ast.TypeExpression) *Schema {
	if expr.IsScalar() {
		return b.Scalar(expr.Base.Parts[0])
	}

	node, ok := b.schema.ResolveType(expr)
	if !ok {
		// unresolved — emit as $ref using the short name
		return &Schema{Ref: b.ref(expr.Base.String())}
	}

	switch def := node.(type) {
	case *ast.Enum:
		return b.Enum(def)
	case *ast.Model:
		if subs == nil && len(expr.GenericArgs) > 0 {
			if mono, ok := b.schema.MonomorphFor(expr); ok {
				return &Schema{Ref: b.ref(mono.Name)}
			}
		}
		return &Schema{Ref: b.ref(def.Name.Value)}
	case *ast.Input:
		return &Schema{Ref: b.ref(def.Name.Value)}
	case *ast.TypeAlias:
		if subs != nil {
			return b.typeExpressionWithSubs(&def.Type, subs)
		}
		return b.typeExpression(&def.Type, genericsInScope)
	default:
		return &Schema{Ref: b.ref(expr.Base.String())}
	}
}

func (b *Builder) field(field *ast.Field, genericsInScope []ast.StringLiteral) *Schema {
	s := b.typeExpression(&field.Type, genericsInScope)
	return b.applyFieldMeta(field, s)
}

// fieldWithSubs resolves a field using concrete substitutions for generic parameters.
func (b *Builder) fieldWithSubs(field *ast.Field, subs map[string]*ast.TypeExpression) *Schema {
	s := b.typeExpressionWithSubs(&field.Type, subs)
	return b.applyFieldMeta(field, s)
}

// applyFieldMeta applies comments and decorators to a field schema.
func (b *Builder) applyFieldMeta(field *ast.Field, s *Schema) *Schema {
	if s == nil {
		return nil
	}

	if field.HeadComment != nil && !field.HeadComment.IsEmpty() {
		s.Description = field.HeadComment.String()
	} else if field.LineComment != nil {
		s.Description = field.LineComment.String()
	}

	for i := range field.Decorators {
		s = b.applyDecorator(&field.Decorators[i], s)
	}

	return s
}

// applyDecorator applies a single decorator to a field schema, modifying it according to the decorator's name and arguments.
func (b *Builder) applyDecorator(decorator *ast.Decorator, s *Schema) *Schema {
	if decorator != nil {
		switch decorator.Name {
		case "default":
			s.Default, _, _ = sequence.GetFirstTuple(decorator.Args.All())
		case "deprecated":
			if s.Extensions == nil {
				s.Extensions = make(map[string]any)
			}
			msg, _, _ := sequence.GetFirstTuple(decorator.Args.All())
			s.Extensions["x-deprecated"] = msg
		case "updatedAt", "createdAt":
			// TODO: some of these decorators not currently supported
			s.ReadOnly = true
		default:
			if s.Extensions == nil {
				s.Extensions = make(map[string]any)
			}
			s.Extensions[decorator.Name] = decorator.Args
		}
	}
	return s
}

// applyRawSchema applies the key-value pairs from a @raw decorator to the schema as either normal fields or extensions.
func (b *Builder) applyRawSchema(raw *ast.RawBlock, s *Schema) *Schema {
	if raw != nil {
		target := b.rawName()
		if raw.Target.Value == target {
			for _, pair := range raw.Pairs {
				switch pair.Key.Value {
				case "description":
					if v, ok := pair.Value.(*ast.StringLiteral); ok {
						s.Description = v.Value
					}
				case "writeOnly":
					s.WriteOnly = true
				case "readOnly":
					s.ReadOnly = true
				case "type":
					if v, ok := pair.Value.(*ast.StringLiteral); ok {
						s.Type = v.Value
					}
				case "format":
					if v, ok := pair.Value.(*ast.StringLiteral); ok {
						s.Format = v.Value
					}
				case "enum":
					if v, ok := pair.Value.(*ast.StringLiteral); ok {
						s.Enum = strings.Split(v.Value, ",")
					}
				default:
					if s.Extensions == nil {
						s.Extensions = make(map[string]any)
					}
					var key string
					if strings.HasPrefix(strings.ToLower(pair.Key.Value), "x-") {
						key = pair.Key.Value
					} else {
						key = "x-" + pair.Key.Value
					}

					s.Extensions[key] = b.typeNodeString(pair.Value)
				}
			}
		}
	}
	return s
}

func (b *Builder) typeNodeString(node ast.TypeNode) string {
	switch v := node.(type) {
	case *ast.StringLiteral:
		return v.Value
	case *ast.IntLiteral:
		return fmt.Sprintf("%d", v.Value)
	case *ast.FloatLiteral:
		return fmt.Sprintf("%g", v.Value)
	case *ast.TypeExpression:
		return v.String()
	case fmt.Stringer:
		return v.String()
	}
	return ""
}

// nullable applies nullable strategy to a schema.
func (b *Builder) nullable(s *Schema) *Schema {
	switch b.nullableStrategy {
	case NullableAnyOf:
		return &Schema{
			AnyOf: []*Schema{
				s,
				{Type: "null"},
			},
		}
	case NullableKeyword:
		s.Nullable = true
	}
	return s
}

// ref returns a JSON reference to a named schema. b.refStrategy must be RefDefs or RefComponents; any other value yields an empty string.q
func (b *Builder) ref(name string) string {
	switch b.refStrategy {
	case RefDefs:
		return "#/$defs/" + name
	case RefComponents:
		return "#/components/schemas/" + name
	}
	return ""
}

func (b *Builder) rawName() string {
	if b.refStrategy == RefComponents {
		return "openapi" // only openapi uses components/*
	}
	return "jsonschema"
}

// isGenericParam returns true if the type expression refers to a generic parameter in scope.
// This would be an unbounded generic parameter, not a type alias or model instantiation with concrete type arguments.
// E.g.: model Box<T> { value: T } - here T is a generic parameter, and would return true.
func (b *Builder) isGenericParam(expr *ast.TypeExpression, genericsInScope []ast.StringLiteral) bool {
	if expr == nil || len(expr.Base.Parts) == 0 {
		return false
	}
	for _, g := range genericsInScope {
		if expr.Base.Parts[0] == g.Value {
			return true
		}
	}
	return false
}
