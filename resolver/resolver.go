package resolver

import (
	"errors"
	"fmt"

	"github.com/jimschubert/spray/ast"
)

type FQN = string

// ResolvedSchema is the final, read-only output of the resolver.
type ResolvedSchema struct {
	Stencils    []*ast.Stencil
	definitions map[FQN]ast.SpecNode
	typeLinks   map[*ast.TypeExpression]ast.SpecNode
}

// Definition looks up a type by its Fully Qualified Name (e.g., "acme.v1.User").
func (s *ResolvedSchema) Definition(fqn string) (ast.SpecNode, bool) {
	node, exists := s.definitions[fqn]
	return node, exists
}

// ResolveType takes a TypeExpression from the AST and returns its linked definition.
func (s *ResolvedSchema) ResolveType(expr *ast.TypeExpression) (ast.SpecNode, bool) {
	if expr.IsScalar() {
		// scalars don't have user-defined specification nodes
		return nil, false
	}
	node, exists := s.typeLinks[expr]
	return node, exists
}

// ResolveModel ensures the TypeExpression resolves specifically to an *ast.Model.
func (s *ResolvedSchema) ResolveModel(expr *ast.TypeExpression) (*ast.Model, error) {
	node, exists := s.ResolveType(expr)
	if !exists {
		return nil, fmt.Errorf("type reference '%s' is unresolved", expr.Base.String())
	}

	model, ok := node.(*ast.Model)
	if !ok {
		return nil, fmt.Errorf("expected '%s' to be a model, but got %T", expr.Base.String(), node)
	}
	return model, nil
}

// ResolveEnum ensures the TypeExpression resolves specifically to an *ast.Enum.
func (s *ResolvedSchema) ResolveEnum(expr *ast.TypeExpression) (*ast.Enum, error) {
	node, exists := s.ResolveType(expr)
	if !exists {
		return nil, fmt.Errorf("type reference '%s' is unresolved", expr.Base.String())
	}

	enum, ok := node.(*ast.Enum)
	if !ok {
		return nil, fmt.Errorf("expected '%s' to be an enum, but got %T", expr.Base.String(), node)
	}
	return enum, nil
}

// ResolveApi ensures the TypeExpression resolves specifically to an *ast.Api.
func (s *ResolvedSchema) ResolveApi(expr *ast.TypeExpression) (*ast.Api, error) {
	node, exists := s.ResolveType(expr)
	if !exists {
		return nil, fmt.Errorf("type reference '%s' is unresolved", expr.Base.String())
	}

	api, ok := node.(*ast.Api)
	if !ok {
		return nil, fmt.Errorf("expected '%s' to be an api, but got %T", expr.Base.String(), node)
	}
	return api, nil
}

// ResolveInput ensures the TypeExpression resolves specifically to an *ast.Input.
func (s *ResolvedSchema) ResolveInput(expr *ast.TypeExpression) (*ast.Input, error) {
	node, exists := s.ResolveType(expr)
	if !exists {
		return nil, fmt.Errorf("type reference '%s' is unresolved", expr.Base.String())
	}

	input, ok := node.(*ast.Input)
	if !ok {
		return nil, fmt.Errorf("expected '%s' to be an input, but got %T", expr.Base.String(), node)
	}
	return input, nil
}

// Resolver performs the two-pass resolution process on a set of parsed stencils.
type Resolver struct {
	stencils []*ast.Stencil
	schema   *ResolvedSchema
	errors   []error
}

// Error returns a combined error if any resolution errors were encountered.
func (r *Resolver) Error() error {
	var err error
	if len(r.errors) > 0 {
		for i := range r.errors {
			err = errors.Join(err, r.errors[i])
		}
	}
	return err
}

// New initializes a Resolver with the given stencils, which are expected to be the output of the parser.
func New(stencils ...*ast.Stencil) *Resolver {
	return &Resolver{
		stencils: stencils,
		schema: &ResolvedSchema{
			Stencils:    stencils,
			definitions: make(map[FQN]ast.SpecNode),
			typeLinks:   make(map[*ast.TypeExpression]ast.SpecNode),
		},
	}
}

// Resolve executes the two-pass resolution process.
func (r *Resolver) Resolve() (*ResolvedSchema, error) {
	// first pass: collects all top-level definitions
	r.registerDefinitions()

	if len(r.errors) > 0 {
		return nil, fmt.Errorf("resolution failed with %d errors", len(r.errors))
	}

	// second pass: type linking
	// walk every stencil again, look at every field/route, and link its TypeExpression to first pass's definition
	r.linkTypes()

	if len(r.errors) > 0 {
		return nil, fmt.Errorf("type linking failed with %d errors", len(r.errors))
	}

	return r.schema, nil
}

// registerDefinitions performs first pass of the resolution phase to register all discovered type definitions.
// Walks all parsed files and builds a flat, O(1) lookup map of every Fully Qualified Name (FQN) to its AST node.
func (r *Resolver) registerDefinitions() {
	for _, stencil := range r.stencils {
		namespace := ""
		if stencil.Namespace != nil && !stencil.Namespace.Implicit {
			namespace = stencil.Namespace.FullName()
		}

		for _, spec := range stencil.Specs {
			var name FQN

			switch node := spec.(type) {
			case *ast.Model:
				name = node.Name.Value
			case *ast.Input:
				name = node.Name.Value
			case *ast.Enum:
				name = node.Name.Value
			case *ast.TypeAlias:
				name = node.Name.Value
			case *ast.Api:
				name = node.Name.Value
			default:
				// should never happen
				continue
			}

			fqn := name
			// note: namespace should never be empty (we set to `default` if empty in parser)
			if namespace != "" {
				fqn = namespace + "." + name
			}

			// semantic validation prevents duplicate within a file; this prevents duplicates across files
			if existing, exists := r.schema.definitions[fqn]; exists {
				r.errors = append(r.errors, fmt.Errorf(
					"duplicate type definition: '%s' defined at line %d, previously defined at line %d",
					fqn,
					spec.Position().Line,
					existing.Position().Line,
				))
				continue
			}

			r.schema.definitions[fqn] = spec
		}
	}

	// validate imports after all definitions are registered
	r.validateImports()
}

// linkTypes performs the second pass, where each TypeExpression pointer is associated to its SpecNode in the typeLinks map.
func (r *Resolver) linkTypes() {
	for _, stencil := range r.stencils {
		for _, spec := range stencil.Specs {
			switch node := spec.(type) {
			case *ast.Model:
				for i := range node.Fields {
					r.linkTypeExprWithGenericScope(stencil, &node.Fields[i].Type, node.GenericParams)
				}
			case *ast.Input:
				for i := range node.Fields {
					r.linkTypeExpr(stencil, &node.Fields[i].Type)
				}
			case *ast.TypeAlias:
				r.linkTypeExpr(stencil, &node.Type)
			case *ast.Api:
				for _, route := range node.Routes {
					switch rt := route.(type) {
					case *ast.RestRoute:
						r.linkTypeExpr(stencil, &rt.Return)
					case *ast.RpcRoute:
						r.linkTypeExpr(stencil, &rt.Input)
						r.linkTypeExpr(stencil, &rt.Return)
					case *ast.EventRoute:
						r.linkTypeExpr(stencil, &rt.Event)
					}
				}
			}
		}
	}
}

// linkTypeExpr attempts to resolve a single TypeExpression
func (r *Resolver) linkTypeExpr(stencil *ast.Stencil, expr *ast.TypeExpression) {
	if expr.IsScalar() {
		// no need to link built-ins
		return
	}

	fqn, found := r.resolveFQN(stencil, expr)
	if !found {
		r.errors = append(r.errors, fmt.Errorf(
			"unresolved type reference: '%s' at line %d",
			expr.Base.String(),
			expr.Position().Line,
		))
		return
	}

	r.schema.typeLinks[expr] = r.schema.definitions[fqn]
	for i := range expr.GenericArgs {
		r.linkTypeExpr(stencil, &expr.GenericArgs[i])
	}
}

// linkTypeExprWithGenericScope attempts to resolve a single TypeExpression and its generic arguments, based on any generic parameters in scope.
func (r *Resolver) linkTypeExprWithGenericScope(stencil *ast.Stencil, expr *ast.TypeExpression, genericsInScope []ast.StringLiteral) {
	if expr.IsScalar() {
		// no need to link built-ins
		return
	}

	// first check if this is a reference to a generic parameter in the current scope.
	// if so, we don't link. e.g. model Page<T> { detail T } would not link T to anything concrete when evaluating the "detail" expression.
	if isGenericParameterInContext(expr, genericsInScope) {
		return
	}

	fqn, found := r.resolveFQN(stencil, expr)
	if !found {
		r.errors = append(r.errors, fmt.Errorf(
			"unknown type: '%s' at line %d",
			expr.Base.String(),
			expr.Position().Line,
		))
		return
	}

	node := r.schema.definitions[fqn]
	r.schema.typeLinks[expr] = node

	// num of parameters (arity) of the type expresion has to match the num of generic parameters expected by the definition.
	// e.g. if Page<T> is defined with 1 generic parameter, then Page<string, int> would be an error.
	expectedArity := r.getGenericArity(node)
	actualArity := len(expr.GenericArgs)
	if expectedArity > 0 && actualArity != expectedArity {
		r.errors = append(r.errors, fmt.Errorf(
			"type '%s' requires %d type argument(s), got %d at line %d",
			expr.Base.String(),
			expectedArity,
			actualArity,
			expr.Position().Line,
		))
		return
	}

	for i := range expr.GenericArgs {
		r.linkTypeExprWithGenericScope(stencil, &expr.GenericArgs[i], genericsInScope)
	}
}

// isGenericParameterInContext checks if a TypeExpression is a generic parameter in the given context.
func isGenericParameterInContext(expr *ast.TypeExpression, genericParams []ast.StringLiteral) bool {
	// generic parameters must be a single identifier with no generic arguments (e.g. T, not Page<T> or Map<K,V>)
	// len(expr.Base.Parts) == 1, e.g. "T" because "common.v1.T" is not a valid generic placeholder
	// len(expr.GenericArgs) == 0 because generic parameters can't have their own generic arguments (e.g. T<string> is not valid)
	// the conditionals below are just negations of the above
	if len(expr.Base.Parts) != 1 || len(expr.GenericArgs) != 0 {
		return false
	}

	name := expr.Base.Parts[0]
	for _, param := range genericParams {
		if param.Value == name {
			return true
		}
	}
	return false
}

// getGenericArity returns the number of generic parameters expected by a type definition.
func (r *Resolver) getGenericArity(node ast.SpecNode) int {
	switch n := node.(type) {
	case *ast.Model:
		return len(n.GenericParams)
	default:
		return 0
	}
}

// resolveFQN determines the FQN for an expression within the file's (*ast.Stencil) scoping.
func (r *Resolver) resolveFQN(stencil *ast.Stencil, expr *ast.TypeExpression) (FQN, bool) {
	targetName := expr.Base.String()

	// when type is already fully qualified, nothing to do
	if _, exists := r.schema.definitions[targetName]; exists {
		return targetName, true
	}

	// when it is imported, find FQN based on import
	for _, imp := range stencil.Imports {
		// this may look backward. for `import acme.common.v1 { User }`, we're looking at a TypeExpression name=User.
		// so iterate Names first, find a match, then combine with the import's path for FQN.
		for _, importedName := range imp.Names {
			if importedName.Value == targetName {
				fqn := imp.Path.String() + "." + targetName
				if _, exists := r.schema.definitions[fqn]; exists {
					return fqn, true
				}
			}
		}
	}

	// not already FQN, and not imported, so check locally in the file represented by ast.Stencil
	if stencil.Namespace != nil {
		localFQN := stencil.Namespace.FullName() + "." + targetName
		if _, exists := r.schema.definitions[localFQN]; exists {
			return localFQN, true
		}
	}

	return "", false
}

// validateImports checks that all imported symbols exist and don't conflict with local definitions.
func (r *Resolver) validateImports() {
	for _, stencil := range r.stencils {
		localNames := make(map[string]bool)
		for _, spec := range stencil.Specs {
			var name string
			switch s := spec.(type) {
			case *ast.Model:
				name = s.Name.Value
			case *ast.Input:
				name = s.Name.Value
			case *ast.Enum:
				name = s.Name.Value
			case *ast.TypeAlias:
				name = s.Name.Value
			case *ast.Api:
				name = s.Name.Value
			default:
				continue
			}
			localNames[name] = true
		}

		// validate each import
		for _, imp := range stencil.Imports {
			for _, importedName := range imp.Names {
				name := importedName.Value

				// does it exist?
				fqn := imp.Path.String() + "." + name
				if _, exists := r.schema.definitions[fqn]; !exists {
					r.errors = append(r.errors, fmt.Errorf(
						"cannot resolve import: '%s' from '%s' at line %d",
						name,
						imp.Path.String(),
						imp.Position().Line,
					))
					continue
				}

				// does it conflict with local definition?
				if localNames[name] {
					r.errors = append(r.errors, fmt.Errorf(
						"import '%s' conflicts with locally defined type '%s' at line %d",
						name,
						name,
						imp.Position().Line,
					))
				}
			}
		}
	}
}
