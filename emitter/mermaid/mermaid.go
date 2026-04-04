package mermaid

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jimschubert/spray/ast"
	"github.com/jimschubert/spray/emitter"
	"github.com/jimschubert/spray/emitter/schema"
	"github.com/jimschubert/spray/resolver"
)

type erdFile struct {
	filename  string
	entities  map[string]*schema.Schema
	relations []relation
}

type relation struct {
	from        string
	to          string
	cardinality string
	label       string
}

func (e *erdFile) Filename() string {
	return e.filename
}

func (e *erdFile) ContentType() emitter.ContentType {
	return emitter.ContentText
}

func (e *erdFile) indent(level int) string {
	return strings.Repeat(" ", level)
}

func (e *erdFile) Contents() []byte {
	var sb strings.Builder
	sb.WriteString("erDiagram\n")

	names := make([]string, 0, len(e.entities))
	for name := range e.entities {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		ent := e.entities[name]
		sb.WriteString(e.indent(4))
		sb.WriteString(name)
		sb.WriteString(" {\n")

		// collect and sort property names for stable output
		propNames := make([]string, 0, len(ent.Properties))
		for pname := range ent.Properties {
			propNames = append(propNames, pname)
		}
		slices.Sort(propNames)

		for _, propertyName := range propNames {
			prop := ent.Properties[propertyName]
			sb.WriteString(e.indent(8))
			sb.WriteString(e.attributeType(prop))
			sb.WriteString(" ")
			sb.WriteString(e.attributeName(propertyName, prop))
			if comment := e.attributeComment(prop); comment != "" {
				sb.WriteString(" \"")
				sb.WriteString(comment)
				sb.WriteString("\"")
			}
			sb.WriteString("\n")
		}

		sb.WriteString(e.indent(4))
		sb.WriteString("}\n")
	}

	for _, r := range e.relations {
		sb.WriteString(e.indent(4))
		sb.WriteString(r.from)
		sb.WriteString(" ")
		sb.WriteString(r.cardinality)
		sb.WriteString(" ")
		sb.WriteString(r.to)
		if r.label != "" {
			sb.WriteString(" : \"")
			sb.WriteString(r.label)
			sb.WriteString("\"")
		}
		sb.WriteString("\n")
	}

	return []byte(sb.String())
}

func (e *erdFile) attributeType(prop *schema.Schema) string {
	if prop.Type == "array" && prop.Items != nil {
		return e.attributeType(prop.Items)
	}
	if len(prop.AnyOf) > 0 {
		for _, a := range prop.AnyOf {
			if a.Type != "null" {
				return e.attributeType(a)
			}
		}
	}

	if prop.Ref != "" {
		return "ref"
	}

	// type with optional format, e.g. "string(uuid)"
	typ := prop.Type
	if prop.Format != "" {
		typ += "(" + prop.Format + ")"
	}

	// no type defaults to "any"
	if typ == "" {
		return "any"
	}

	return typ
}

// attributeName returns the attribute name with key modifiers (PK, FK, UK).
func (e *erdFile) attributeName(name string, prop *schema.Schema) string {
	sb := strings.Builder{}
	if _, ok := prop.Extensions["x-primary"]; ok {
		sb.WriteString("*")
	}
	sb.WriteString(name)

	keys := make([]string, 0, 2)
	if _, ok := prop.Extensions["x-primary"]; ok {
		keys = append(keys, "PK")
	}
	if e.hasRef(prop) {
		keys = append(keys, "FK")
	}
	if _, ok := prop.Extensions["x-unique"]; ok {
		keys = append(keys, "UK")
	}

	if len(keys) > 0 {
		sb.WriteString(" ")
		sb.WriteString(strings.Join(keys, ","))
	}

	return sb.String()
}

func (e *erdFile) hasRef(prop *schema.Schema) bool {
	if prop.Ref != "" {
		return true
	}
	if prop.Type == "array" && prop.Items != nil && prop.Items.Ref != "" {
		return true
	}
	return slices.ContainsFunc(prop.AnyOf, e.hasRef)
}

func (e *erdFile) attributeComment(prop *schema.Schema) string {
	if prop.Description != "" {
		return prop.Description
	}
	if prop.Default != nil {
		return fmt.Sprintf("default: %v", prop.Default)
	}
	return ""
}

type emitMermaid struct {
	schema  resolver.ResolvedSchema
	builder *schema.Builder
}

func (e *emitMermaid) EmitAll() ([]emitter.Output, error) {
	entities := make(map[string]*schema.Schema)
	relations := make([]relation, 0)

	for _, stencil := range e.schema.Stencils {
		for _, node := range stencil.Specs {
			s := e.builder.Spec(node)
			if s == nil {
				continue
			}
			name := ast.NameOf(node)
			entities[name] = s

			// extract relations from properties
			for propName, prop := range s.Properties {
				refName, propOrInner := e.resolveRef(prop)
				if refName == "" {
					continue
				}

				relations = append(relations, e.buildRelation(name, refName, propName, prop, propOrInner))
			}
		}
	}

	// also handle monomorphs
	monomorphs := e.schema.Monomorphs()
	monoKeys := make([]string, 0, len(monomorphs))
	for k := range monomorphs {
		monoKeys = append(monoKeys, k)
	}
	slices.Sort(monoKeys)

	for _, k := range monoKeys {
		mono := monomorphs[k]
		ms := e.builder.MonomorphSchema(mono)
		if ms == nil {
			continue
		}
		entities[mono.Name] = ms

		for name, property := range ms.Properties {
			refName, propOrInner := e.resolveRef(property)
			if refName == "" {
				continue
			}
			relations = append(relations, e.buildRelation(mono.Name, refName, name, property, propOrInner))
		}
	}

	return []emitter.Output{&erdFile{
		filename:  "diagram.mmd",
		entities:  entities,
		relations: relations,
	}}, nil
}

func (e *emitMermaid) EmitOne(typ emitter.SpecType, name string) (emitter.Output, error) {
	collected := emitter.CollectAll(e.schema.Stencils...)
	specs, ok := collected[typ]
	if !ok {
		return nil, fmt.Errorf("no specs of type %d found", typ)
	}

	entities := make(map[string]*schema.Schema)
	relations := make([]relation, 0)

	for _, spec := range specs {
		specName := ast.NameOf(spec)
		if specName != name {
			continue
		}
		s := e.builder.Spec(spec)
		if s == nil {
			return nil, fmt.Errorf("%q not found", name)
		}
		entities[specName] = s

		for propName, prop := range s.Properties {
			refName, propOrInner := e.resolveRef(prop)
			if refName == "" {
				continue
			}
			relations = append(relations, e.buildRelation(specName, refName, propName, prop, propOrInner))
		}
		break
	}

	if len(entities) == 0 {
		return nil, fmt.Errorf("%q not found", name)
	}

	return &erdFile{
		filename:  strings.ToLower(name) + ".mmd",
		entities:  entities,
		relations: relations,
	}, nil
}

// buildRelation constructs a relation from a property schema.
// origProp is the original property (for cardinality), relProp is the metadata-bearing property (for decorators).
func (e *emitMermaid) buildRelation(fromName string, toName string, propName string, origProp *schema.Schema, relationProperty *schema.Schema) relation {
	cardinality := cardinality(origProp)
	label := propName

	// check for @relation decorator to get the label
	if extension, ok := relationProperty.Extensions["x-relation"]; ok {
		var args *ast.OrderedTypeMap
		switch v := extension.(type) {
		case *ast.OrderedTypeMap:
			args = v
		case ast.OrderedTypeMap:
			args = &v
		}
		if args != nil {
			if fieldNode, found := args.Get("field"); found {
				switch v := fieldNode.(type) {
				case *ast.StringLiteral:
					label = v.Value
				case *ast.TypeExpression:
					label = v.String()
				}
			}
		}
	}

	return relation{
		from:        fromName,
		to:          toName,
		cardinality: cardinality,
		label:       label,
	}
}

// cardinality determines the mermaid tokens for a given relation property.
//
// Mermaid ERD cardinality syntax:
//
//	   T (required): ||  (exactly one)
//	  T? (optional): o|  (zero or one)
//	 T[] (required): |{  (one or more)
//	T[]? (optional): o{  (zero or more)
//
// see: https://mermaid.ai/open-source/syntax/entityRelationshipDiagram.html#relationship-syntax
func cardinality(prop *schema.Schema) string {
	actual := prop
	if len(prop.AnyOf) > 0 {
		for _, a := range prop.AnyOf {
			if a.Type != "null" {
				actual = a
				break
			}
		}
	}

	isOptional := len(prop.AnyOf) > 0
	isArray := actual.Type == "array"

	// target side: rightMin (min) and rightMax (max)
	rightMin := "|"
	rightMax := "|"
	if isOptional {
		rightMin = "o"
	}
	if isArray {
		rightMax = "{"
	}

	return "||--" + rightMin + rightMax
}

// refToName extracts the entity name from a $ref string like "#/$defs/User".
func (e *emitMermaid) refToName(ref string) string {
	if !strings.HasPrefix(ref, "#/$defs/") {
		return ""
	}
	return strings.TrimPrefix(ref, "#/$defs/")
}

// resolveRef unwraps a property schema to find a referenced entity.
func (e *emitMermaid) resolveRef(prop *schema.Schema) (string, *schema.Schema) {
	if prop.Ref != "" {
		if name := e.refToName(prop.Ref); name != "" {
			// direct ref
			return name, prop
		}
	}

	if prop.Type == "array" && prop.Items != nil {
		if prop.Items.Ref != "" {
			if name := e.refToName(prop.Items.Ref); name != "" {
				// ref item in array
				return name, prop
			}
		}
	}

	if len(prop.AnyOf) > 0 {
		for _, a := range prop.AnyOf {
			if a.Type == "null" {
				continue
			}
			if name, _ := e.resolveRef(a); name != "" {
				// keep the outer schema so decorators on the nullable wrapper are preserved
				return name, prop
			}
		}
	}

	return "", nil
}

// New creates a new Mermaid ERD emitter with the given resolved schema.
func New(resolved *resolver.ResolvedSchema) (emitter.Emitter, error) {
	if resolved == nil {
		return nil, fmt.Errorf("schema cannot be nil")
	}

	b := schema.NewBuilder(*resolved).WithNullableStrategy(schema.NullableAnyOf).WithRefStrategy(schema.RefDefs)
	return &emitMermaid{
		schema:  *resolved,
		builder: b,
	}, nil
}
