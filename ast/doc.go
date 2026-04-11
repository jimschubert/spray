// Package ast defines the AST node types that represent a parsed .stencil file.
//
// The root type is Stencil, which holds a Namespace, Imports, and an ordered
// list of Specs (Model, Input, Enum, Api, TypeAlias). Consumers iterate Specs
// with type assertions to process specific node kinds.
//
// Example:
//
//	p := parser.NewParser()
//	stencil, err := p.Parse(src)
//	for _, spec := range stencil.Specs {
//		switch s := spec.(type) {
//		case *ast.Model:  // data model with fields
//		case *ast.Input:  // API input type
//		case *ast.Enum:   // enumeration of string values
//		case *ast.Api:    // API with routes
//		}
//	}
//
// Marker interfaces (SpecNode, TypeNode) restrict which nodes are valid in
// certain contexts — a strategy taken from Go's own go/ast package.
// Use ast.NameOf(node) to get the name of any SpecNode, and
// ast.IsBuiltinScalar(name) to check for built-in scalar types.
package ast
