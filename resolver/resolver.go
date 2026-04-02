package resolver

import (
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/jimschubert/spray/ast"
)

type FQN = string

// ResolvedSchema is the final, read-only output of the resolver.
type ResolvedSchema struct {
	Stencils    []*ast.Stencil
	definitions map[FQN]ast.SpecNode
	typeLinks   map[*ast.TypeExpression]ast.SpecNode
	nodeNS      map[ast.SpecNode]string
	monomorphs  map[string]Monomorph
}

// Monomorphs returns a copy of discovered concrete generic instantiations keyed by canonical resolved type key.
func (s *ResolvedSchema) Monomorphs() map[string]Monomorph {
	copyMap := make(map[string]Monomorph, len(s.monomorphs))
	maps.Copy(copyMap, s.monomorphs)
	return copyMap
}

// MonomorphFor returns the Monomorph for an ast.TypeExpression, if one exists.
// expr must be an original AST pointer (i.e. from a field type, route return, etc.) — its
// GenericArgs are also expected to be in typeLinks.
func (s *ResolvedSchema) MonomorphFor(expr *ast.TypeExpression) (Monomorph, bool) {
	if len(expr.GenericArgs) == 0 {
		return Monomorph{}, false
	}
	node, ok := s.typeLinks[expr]
	if !ok {
		return Monomorph{}, false
	}
	baseName := ast.NameOf(node)
	if baseName == "" {
		return Monomorph{}, false
	}
	ns, _ := s.NamespaceOf(node)
	fqn := baseName
	if ns != "" {
		fqn = ns + "." + baseName
	}
	key := s.monomorphKey(fqn, expr.GenericArgs)
	mono, ok := s.monomorphs[key]
	return mono, ok
}

// Definition looks up a type by its Fully Qualified Name (e.g., "acme.v1.User").
func (s *ResolvedSchema) Definition(fqn string) (ast.SpecNode, bool) {
	node, exists := s.definitions[fqn]
	return node, exists
}

// ResolveType returns the definition linked to expr, or false if expr is a scalar.
func (s *ResolvedSchema) ResolveType(expr *ast.TypeExpression) (ast.SpecNode, bool) {
	if expr.IsScalar() {
		// scalars don't have user-defined specification nodes
		return nil, false
	}
	node, exists := s.typeLinks[expr]
	return node, exists
}

// NamespaceOf returns the namespace of a registered spec node.
func (s *ResolvedSchema) NamespaceOf(node ast.SpecNode) (string, bool) {
	ns, ok := s.nodeNS[node]
	return ns, ok
}

// ResolveModel resolves expr as *ast.Model, returning an error if unresolved or not a model.
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

// ResolveEnum resolves expr as *ast.Enum, returning an error if unresolved or not an enum.
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

// ResolveApi resolves expr as *ast.Api, returning an error if unresolved or not an api.
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

// ResolveInput resolves expr as *ast.Input, returning an error if unresolved or not an input.
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

// New initializes a Resolver for the given parsed stencils.
func New(stencils ...*ast.Stencil) *Resolver {
	return &Resolver{
		stencils: stencils,
		schema: &ResolvedSchema{
			Stencils:    stencils,
			definitions: make(map[FQN]ast.SpecNode),
			typeLinks:   make(map[*ast.TypeExpression]ast.SpecNode),
			nodeNS:      make(map[ast.SpecNode]string),
			monomorphs:  make(map[string]Monomorph),
		},
	}
}

// Resolve executes the two-pass resolution process.
func (r *Resolver) Resolve() (*ResolvedSchema, error) {
	r.registerDefinitions()

	if len(r.errors) > 0 {
		return nil, fmt.Errorf("resolution failed with %d errors", len(r.errors))
	}

	// second pass: type linking
	r.linkTypes()

	// build monomorphs from linked types; must run after linkTypes
	r.monomorphize()

	if len(r.errors) > 0 {
		return nil, fmt.Errorf("type linking failed with %d errors", len(r.errors))
	}

	return r.schema, nil
}

// registerDefinitions performs the first resolution pass, building a flat O(1) FQN→AST-node lookup map.
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
			r.schema.nodeNS[spec] = namespace
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

// linkTypeExprWithGenericScope resolves expr within the scope of the given generic parameters.
func (r *Resolver) linkTypeExprWithGenericScope(stencil *ast.Stencil, expr *ast.TypeExpression, genericsInScope []ast.StringLiteral) {
	if expr.IsScalar() {
		// no need to link built-ins
		return
	}

	// skip generic parameter references — e.g. T in `model Page<T> { detail T }` is not a concrete type.
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

	// arity of the type expression must match the arity of its definition.
	// e.g. Page<T> expects 1 argument, so Page<string, int> is an error.
	expectedArity := arity(node)
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

// isGenericParameterInContext reports whether expr refers to one of the given generic parameters.
// A generic parameter is a single unqualified identifier with no generic arguments (e.g. T, not Page<T> or common.v1.T).
func isGenericParameterInContext(expr *ast.TypeExpression, genericParams []ast.StringLiteral) bool {
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

// resolveFQN resolves the FQN for expr within the stencil's scope.
func (r *Resolver) resolveFQN(stencil *ast.Stencil, expr *ast.TypeExpression) (FQN, bool) {
	targetName := expr.Base.String()

	// already fully qualified
	if _, exists := r.schema.definitions[targetName]; exists {
		return targetName, true
	}

	// check imports: for `import acme.common.v1 { User }`, match on the short name then build the FQN.
	for _, imp := range stencil.Imports {
		for _, importedName := range imp.Names {
			if importedName.Value == targetName {
				fqn := imp.Path.String() + "." + targetName
				if _, exists := r.schema.definitions[fqn]; exists {
					return fqn, true
				}
			}
		}
	}

	// fall back to the local namespace
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

		for _, imp := range stencil.Imports {
			for _, importedName := range imp.Names {
				name := importedName.Value

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

// monomorphize generates concrete type definitions from generic declarations based on their usage.
// For example, if `Page<T>` is used as `Page<User>` and `Page<Product>`, two new models `PageUser`
// and `PageProduct` are generated with T substituted, so emitters only deal with concrete types.
func (r *Resolver) monomorphize() {
	m := monomorphizer{
		schema: *r.schema,
		seen:   make(map[string]Monomorph),
	}

	r.schema = new(m.monomorphize())
}

// monomorphKey builds the key for a generic concrete type, mirroring monomorphizer.
// Example: fqn="acme.v1.Page", args=[User] → "acme.v1.Page<acme.v1.User>"
func (s *ResolvedSchema) monomorphKey(fqn string, args []ast.TypeExpression) string {
	var sb strings.Builder
	sb.WriteString(fqn)
	sb.WriteByte('<')
	for i := range args {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(s.typeExprKey(&args[i]))
	}
	sb.WriteByte('>')
	return sb.String()
}

// typeExprKey builds the FQN key part for an ast.TypeExpression
func (s *ResolvedSchema) typeExprKey(expr *ast.TypeExpression) string {
	if expr.IsScalar() {
		return expr.Base.String()
	}
	name := expr.Base.String()
	if def, ok := s.typeLinks[expr]; ok {
		baseName := ast.NameOf(def)
		ns, _ := s.NamespaceOf(def)
		if ns != "" {
			name = ns + "." + baseName
		} else {
			name = baseName
		}
	}
	if len(expr.GenericArgs) > 0 {
		name = s.monomorphKey(name, expr.GenericArgs)
	}
	return name
}
