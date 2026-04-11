package resolver

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/jimschubert/spray/ast"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Monomorph represents a concrete instantiation of a generic declaration, flattened into a representative named type for use by emitters.
// Original is *ast.SpecNode to allow for potential future Input generics.
type Monomorph struct {
	Name string
	// Namespace (optional) of original type
	Namespace string
	Original  *ast.SpecNode
	Args      []ast.TypeExpression
	// ArgFQNs holds the fully qualified name (if available) for every index in Args. Scalars or unresolved types will have an empty value.
	ArgFQNs         []string
	key             string
	memoCache       map[string]string
	retainNamespace bool
}

// PrefixedWithNamespace returns a copy of the Monomorph with Name prefixed by Namespace if present.
func (m *Monomorph) PrefixedWithNamespace() *Monomorph {
	cpy := new(Monomorph)
	*cpy = *m
	cpy.retainNamespace = true
	if m.Namespace != "" {
		cpy.Name = m.Namespace + "." + m.Name
	}
	return cpy
}

// EmitAs generates a deterministic emitter-safe name from the monomorph key.
// It strips namespace prefixes from the key and tokenizes the result.
// If no caser is provided, a PascalCase(-ish) default is used.
// Results are memoized per caser to avoid recomputation.
func (m *Monomorph) EmitAs(caser ...cases.Caser) (string, error) {
	if len(caser) > 1 {
		return "", fmt.Errorf("expected at most one caser, got %d", len(caser))
	}

	if m.memoCache == nil {
		m.memoCache = make(map[string]string)
	}

	cacheKey := "default"
	if len(caser) == 1 {
		cacheKey = fmt.Sprintf("%T", caser[0])
	}

	if cached, ok := m.memoCache[cacheKey]; ok {
		return cached, nil
	}

	// see example_test.go of golang.org/x/text for variations of casers.
	selected := cases.Title(language.Und, cases.NoLower)
	if len(caser) == 1 {
		selected = caser[0]
	}

	var targetKey string
	if m.retainNamespace {
		prefix := m.Namespace + "."
		if strings.HasPrefix(m.key, prefix) {
			targetKey = m.key
		} else {
			targetKey = prefix + m.key
		}
	} else {
		targetKey = m.stripNamespaceFromKey(m.key)
	}

	target := strings.Builder{}
	for _, token := range m.splitAlphaNumericTokens(targetKey) {
		target.WriteString(selected.String(token))
	}

	name := target.String()
	if name == "" {
		name = "Monomorph"
	}

	m.memoCache[cacheKey] = name
	return name, nil
}

// stripNamespaceFromKey removes all namespace prefixes from a fully qualified key,
// stripping everything up to and including the last dot at each type name position.
// Example: "acme.v1.Page<acme.v1.User>"  -> "Page<User>"
// Example: "acme.v1.Page<other.v1.User>" -> "Page<User>"
func (m *Monomorph) stripNamespaceFromKey(key string) string {
	result := make([]byte, 0, len(key))
	i := 0
	// Track whether the next non-space character starts a type name
	nextIsTypeStart := true

	for i < len(key) {
		if key[i] == ' ' {
			// spaces are not type-start boundaries
			result = append(result, key[i])
			i++
			continue
		}

		if nextIsTypeStart {
			// collect the full qualified type name
			typeStart := i
			for i < len(key) && key[i] != '<' && key[i] != ',' && key[i] != '>' && key[i] != ' ' {
				i++
			}

			typeExpr := key[typeStart:i]

			// strip *any* namespace prefix
			lastDot := strings.LastIndex(typeExpr, ".")
			if lastDot >= 0 {
				result = append(result, typeExpr[lastDot+1:]...)
			} else {
				result = append(result, typeExpr...)
			}

			nextIsTypeStart = false
		} else {
			switch structuralChar := key[i]; structuralChar {
			case '<', ',':
				nextIsTypeStart = true
			case '>':
				nextIsTypeStart = false
			}

			result = append(result, key[i])
			i++
		}
	}

	return string(result)
}

func (m *Monomorph) splitAlphaNumericTokens(key string) []string {
	tokens := make([]string, 0, 8)
	var sb strings.Builder
	for _, r := range key {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
			continue
		}

		if sb.Len() > 0 {
			tokens = append(tokens, sb.String())
			sb.Reset()
		}
	}

	if sb.Len() > 0 {
		tokens = append(tokens, sb.String())
	}

	return tokens
}

// monomorphizer monomorphizes generic declarations in a resolved and linked ResolvedSchema.
type monomorphizer struct {
	schema ResolvedSchema
	seen   map[string]Monomorph
}

// monomorphize returns a new ResolvedSchema with all generic declarations monomorphized.
func (m *monomorphizer) monomorphize() ResolvedSchema {
	for expr, def := range m.schema.typeLinks {
		if arity(def) == 0 || len(expr.GenericArgs) == 0 {
			continue
		}

		// skip partial instantiations such as Page<T> inside a generic scope
		if !m.isConcrete(expr) {
			continue
		}

		m.collect(def, expr.GenericArgs)
	}
	return m.schema
}

// resolvedKey generates a unique key for a generic declaration based on its fully qualified name and type arguments.
// Example: acme.v1.Page + [acme.v1.User] -> "acme.v1.Page<acme.v1.User>"
func (m *monomorphizer) resolvedKey(fqn string, args []ast.TypeExpression) string {
	if len(args) == 0 {
		return fqn
	}
	var sb strings.Builder
	sb.WriteString(fqn)
	sb.WriteByte('<')
	for i := range args {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(m.resolvedTypeExprKey(&args[i]))
	}
	sb.WriteByte('>')
	return sb.String()
}

func (m *monomorphizer) collect(def ast.SpecNode, args []ast.TypeExpression) {
	for i := range args {
		arg := &args[i]
		if arg.IsScalar() {
			continue
		}

		linked, ok := m.schema.typeLinks[arg]
		if !ok || arity(linked) == 0 || len(arg.GenericArgs) == 0 || !m.isConcrete(arg) {
			continue
		}

		m.collect(linked, arg.GenericArgs)
	}

	baseName := m.typeName(def)
	if baseName == "" {
		return
	}

	ns, _ := m.schema.NamespaceOf(def)

	// build FQN
	fqn := baseName
	if ns != "" {
		fqn = ns + "." + baseName
	}

	key := m.resolvedKey(fqn, args)
	if _, ok := m.seen[key]; ok {
		return
	}

	mono := Monomorph{
		key:       key,
		Namespace: ns,
		Original:  new(def),
		Args:      append([]ast.TypeExpression(nil), args...),
		ArgFQNs:   m.argFQNs(args),
	}

	emitAs, err := mono.EmitAs()
	if err != nil {
		return
	}

	mono.Name = emitAs

	m.seen[key] = mono
	m.schema.monomorphs[key] = mono
}

func (m *monomorphizer) resolvedTypeExprKey(expr *ast.TypeExpression) string {
	name := expr.Base.String()
	if !expr.IsScalar() {
		if def, ok := m.schema.typeLinks[expr]; ok {
			baseName := m.typeName(def)
			ns, _ := m.schema.NamespaceOf(def)
			if ns != "" {
				name = ns + "." + baseName
			} else {
				name = baseName
			}
		}
	}

	if len(expr.GenericArgs) > 0 {
		name = m.resolvedKey(name, expr.GenericArgs)
	}

	if expr.IsArray {
		name += "[]"
	}

	if expr.IsOptional {
		name += "?"
	}

	return name
}

// argFQNs returns a slice of fully qualified names for the given type expressions, if available.
// Scalars or unresolved types will have an empty value. indexes map 1:1 with the original args slice.
func (m *monomorphizer) argFQNs(args []ast.TypeExpression) []string {
	result := make([]string, len(args))
	for i := range args {
		if args[i].IsScalar() {
			continue
		}
		linked, ok := m.schema.typeLinks[&args[i]]
		if !ok {
			continue
		}
		linkedNS, _ := m.schema.NamespaceOf(linked)
		linkedName := ast.NameOf(linked)
		if linkedName == "" {
			continue
		}
		if linkedNS != "" {
			result[i] = linkedNS + "." + linkedName
		} else {
			result[i] = linkedName
		}
	}
	return result
}

func (m *monomorphizer) isConcrete(expr *ast.TypeExpression) bool {
	if expr.IsScalar() {
		return true
	}

	if _, ok := m.schema.typeLinks[expr]; !ok {
		return false
	}

	for i := range expr.GenericArgs {
		if !m.isConcrete(&expr.GenericArgs[i]) {
			return false
		}
	}

	return true
}

func (m *monomorphizer) typeName(def ast.SpecNode) string {
	switch d := def.(type) {
	case *ast.Model:
		return d.Name.Value
	case *ast.Input:
		return d.Name.Value
	default:
		return ""
	}
}
