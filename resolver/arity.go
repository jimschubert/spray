package resolver

import "github.com/jimschubert/spray/ast"

// arity returns the number of generic parameters expected by a type definition.
func arity(node ast.SpecNode) int {
	switch n := node.(type) {
	case *ast.Model:
		return len(n.GenericParams)
	default:
		return 0
	}
}
