// Package parser provides the public API for parsing .stencil source files
// into an AST representation.
//
// Example:
//
//	p, err := parser.New()
//	stencil, err := p.Parse(src)
//
// A Stencil contains a Namespace (explicit or implicit "default"),
// a list of Imports, and Specs (Models, Inputs, Enums, Apis) in source order.
//
// After parsing, pass the Stencil through a resolver.Resolver to resolve
// type references and initialize resolver.ResolvedSchema, which is the input for emitters.
//
//	res := resolver.New(stencil)
//	resolvedSchema, err := res.Resolve()
package parser
