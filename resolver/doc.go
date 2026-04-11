// Package resolver performs two-pass type resolution across one or more
// parsed Stencils, producing a ResolvedSchema that links every TypeExpression
// in the AST to its definition.
//
// Pass 1 registers all spec names as Fully Qualified Names (FQNs) like
// "acme.v1.User" in a flat lookup map. Pass 2 links every *ast.TypeExpression
// pointer to its SpecNode definition and collects Monomorph instantiations
// for generic types.
//
//	res := resolver.New(stencils...)
//	schema, err := res.Resolve()
//
//	node, ok := schema.Definition("acme.v1.User")  // look up by FQN
//	def, ok  := schema.ResolveType(expr)           // look up by TypeExpression
//
// For generic types like Page<T> used as Page<User> and Page<Product>,
// the resolver generates concrete Monomorph instances that emitters can
// render without dealing with type parameters:
//
//	mono, ok := schema.MonomorphFor(expr)
//	name := mono.EmitAs(caser.Pascal)  // e.g. Page<User> -> PageUser
package resolver
