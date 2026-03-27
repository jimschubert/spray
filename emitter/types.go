package emitter

import "github.com/jimschubert/spray/ast"

type SpecType int

const (
	SpecModel SpecType = iota
	SpecInput
	SpecEnum
	SpecApi
)

type ContentType int

const (
	ContentText ContentType = iota
	ContentBinary
)

// Output defines the contract for an emitter's output artifact.
// The result of an emit may be text, binary, or something else, but it must be representable as a byte slice.
type Output interface {
	Filename() string
	Contents() []byte
	ContentType() ContentType
}

type Emitter interface {
	EmitAll() ([]Output, error)
	EmitOne(typ SpecType, name string) (Output, error)
}

func CollectAll(stencils ...*ast.Stencil) map[SpecType][]ast.SpecNode {
	apis := make([]ast.SpecNode, 0)
	models := make([]ast.SpecNode, 0)
	inputs := make([]ast.SpecNode, 0)
	enums := make([]ast.SpecNode, 0)

	for i := range stencils {
		current := stencils[i]
		for j := range current.Specs {
			spec := current.Specs[j]
			switch s := spec.(type) {
			case *ast.Api:
				apis = append(apis, s)
			case *ast.Model:
				models = append(models, s)
			case *ast.Input:
				inputs = append(inputs, s)
			case *ast.Enum:
				enums = append(enums, s)
			}
		}
	}

	return map[SpecType][]ast.SpecNode{
		SpecApi:   apis,
		SpecModel: models,
		SpecInput: inputs,
		SpecEnum:  enums,
	}
}
