package plug

import (
	"fmt"

	"github.com/jimschubert/spray/ast"
	"github.com/jimschubert/spray/resolver"
)

// Mapper transforms a resolver.ResolvedSchema into a PluginSchema.
type Mapper struct {
	schema *resolver.ResolvedSchema
}

// NewMapper creates a new Mapper for the given resolved schema.
func NewMapper(schema *resolver.ResolvedSchema) *Mapper {
	return &Mapper{schema: schema}
}

// Map converts the resolved schema into a PluginSchema suitable for plugin consumption.
func (m *Mapper) Map() PluginSchema {
	result := PluginSchema{
		Aliases:    make(map[string][]PluginTypeAlias),
		Extensions: make(map[string][]PluginExtension),
	}

	if m.schema == nil {
		return result
	}

	for _, stencil := range m.schema.Stencils {
		ns := m.namespaceName(stencil)

		if result.Namespace == "" && ns != "" {
			result.Namespace = ns
		}

		for _, spec := range stencil.Specs {
			switch node := spec.(type) {
			case *ast.Model:
				result.Models = append(result.Models, m.mapModel(node, ns))
			case *ast.Input:
				result.Inputs = append(result.Inputs, m.mapInput(node, ns))
			case *ast.Enum:
				result.Enums = append(result.Enums, m.mapEnum(node, ns))
			case *ast.Api:
				result.Apis = append(result.Apis, m.mapApi(node, ns)...)
			case *ast.TypeAlias:
				result.Aliases[ns] = append(result.Aliases[ns], m.mapTypeAlias(node))
			}
		}
	}

	result.Monomorphs = m.mapMonomorphs()
	m.mapExtensions(result.Extensions)

	return result
}

func (m *Mapper) namespaceName(stencil *ast.Stencil) string {
	if stencil.Namespace != nil && !stencil.Namespace.Implicit {
		return stencil.Namespace.FullName()
	}
	return ""
}

func (m *Mapper) mapModel(model *ast.Model, ns string) PluginModel {
	return PluginModel{
		Name:        model.Name.Value,
		Namespace:   ns,
		HeadComment: model.HeadComment.String(),
		Fields:      m.mapFields(model.Fields),
	}
}

func (m *Mapper) mapInput(input *ast.Input, ns string) PluginInput {
	return PluginInput{
		Name:        input.Name.Value,
		Namespace:   ns,
		HeadComment: input.HeadComment.String(),
		Fields:      m.mapFields(input.Fields),
	}
}

func (m *Mapper) mapEnum(enum *ast.Enum, ns string) PluginEnum {
	values := make([]string, len(enum.Elements))
	for i, elem := range enum.Elements {
		values[i] = elem.Value
	}

	return PluginEnum{
		Name:        enum.Name.Value,
		Namespace:   ns,
		HeadComment: enum.HeadComment.String(),
		Values:      values,
	}
}

func (m *Mapper) mapApi(api *ast.Api, ns string) []PluginApi {
	var result []PluginApi

	for _, route := range api.Routes {
		switch rt := route.(type) {
		case *ast.RestRoute:
			result = append(result, m.mapRestRoute(rt, ns))
		case *ast.RpcRoute:
			result = append(result, m.mapRpcRoute(rt, ns))
		case *ast.EventRoute:
			result = append(result, m.mapEventRoute(rt, ns))
		}
	}

	return result
}

func (m *Mapper) mapRestRoute(route *ast.RestRoute, ns string) PluginApi {
	var input *PluginTypeRef
	if ref := m.extractRouteInput(route.Decorators); ref.FQN != "" {
		input = &ref
	}

	return PluginApi{
		Style:       RouteStyleRest,
		Name:        route.Method + " " + route.Path.String(),
		Namespace:   ns,
		HeadComment: route.HeadComment.String(),
		Method:      Method(route.Method),
		Path:        route.Path.String(),
		Input:       input,
		Return:      m.mapTypeExpr(&route.Return),
		Decorators:  m.mapDecorators(route.Decorators),
	}
}

func (m *Mapper) mapRpcRoute(route *ast.RpcRoute, ns string) PluginApi {
	var input *PluginTypeRef
	if !route.Input.IsAbsent() {
		ref := m.mapTypeExpr(&route.Input)
		input = &ref
	}

	return PluginApi{
		Style:       RouteStyleRpc,
		Name:        route.Name.Value,
		Namespace:   ns,
		HeadComment: route.HeadComment.String(),
		Input:       input,
		Return:      m.mapTypeExpr(&route.Return),
		Streaming:   route.Streaming,
		Decorators:  m.mapDecorators(route.Decorators),
	}
}

func (m *Mapper) mapEventRoute(route *ast.EventRoute, ns string) PluginApi {
	direction := EventPublish
	if route.Direction == ast.EventSubscribe {
		direction = EventSubscribe
	}

	return PluginApi{
		Style:       RouteStyleEvents,
		Name:        route.Name.Value,
		Namespace:   ns,
		HeadComment: route.HeadComment.String(),
		Return:      m.mapTypeExpr(&route.Event),
		Direction:   direction,
		Decorators:  m.mapDecorators(route.Decorators),
	}
}

// extractRouteInput looks for @body or @query decorators and extracts the input type.
func (m *Mapper) extractRouteInput(decorators []ast.Decorator) PluginTypeRef {
	for _, dec := range decorators {
		if dec.Name == "body" || dec.Name == "query" {
			// the decorator's first positional arg is the input TypeExpression
			var found PluginTypeRef
			dec.Args.All()(func(key string, value ast.TypeNode) bool {
				if expr, ok := value.(*ast.TypeExpression); ok {
					found = m.mapTypeExpr(expr)
					return false
				}
				return true
			})
			if found.FQN != "" {
				return found
			}
		}
	}
	return PluginTypeRef{}
}

func (m *Mapper) mapTypeAlias(alias *ast.TypeAlias) PluginTypeAlias {
	return PluginTypeAlias{
		Name:        alias.Name.Value,
		Type:        m.mapTypeExpr(&alias.Type),
		HeadComment: alias.HeadComment.String(),
		LineComment: alias.LineComment.String(),
	}
}

func (m *Mapper) mapMonomorphs() []PluginMonomorph {
	monomorphs := m.schema.Monomorphs()
	result := make([]PluginMonomorph, 0, len(monomorphs))

	for _, mono := range monomorphs {
		result = append(result, PluginMonomorph{
			Name:        mono.Name,
			Namespace:   mono.Namespace,
			HeadComment: m.monomorphHeadComment(&mono),
			Original:    m.monomorphOriginalFQN(&mono),
			Args:        m.mapMonomorphArgs(mono.Args, mono.ArgFQNs),
		})
	}

	return result
}

// mapMonomorphArgs maps monomorph type arguments using pre-resolved FQNs from ArgFQNs.
// mono.Args are copies whose pointers don't exist in typeLinks, so ResolveType can't resolve them;
// ArgFQNs holds the FQNs captured at collection time.
func (m *Mapper) mapMonomorphArgs(args []ast.TypeExpression, fqns []string) []PluginTypeRef {
	result := make([]PluginTypeRef, len(args))
	for i := range args {
		ref := PluginTypeRef{
			FQN:        args[i].Base.String(),
			IsArray:    args[i].IsArray,
			IsOptional: args[i].IsOptional,
			IsScalar:   args[i].IsScalar(),
		}
		if i < len(fqns) && fqns[i] != "" {
			ref.FQN = fqns[i]
		}
		if len(args[i].GenericArgs) > 0 {
			ref.Args = m.mapTypeArgs(args[i].GenericArgs)
		}
		result[i] = ref
	}
	return result
}

func (m *Mapper) mapExtensions(ext map[string][]PluginExtension) {
	for _, stencil := range m.schema.Stencils {
		ns := m.namespaceName(stencil)

		for _, spec := range stencil.Specs {
			fqn := m.specFQN(spec, ns)

			var rawBlocks []ast.RawBlock
			switch node := spec.(type) {
			case *ast.Model:
				rawBlocks = node.Extensions
			case *ast.Api:
				rawBlocks = node.Extensions
			default:
				continue
			}

			if len(rawBlocks) > 0 {
				ext[fqn] = m.mapRawBlocks(rawBlocks)
			}
		}
	}
}

func (m *Mapper) mapFields(fields []ast.Field) []PluginField {
	result := make([]PluginField, len(fields))
	for i := range fields {
		result[i] = PluginField{
			Name:        fields[i].Name.Value,
			Type:        m.mapTypeExpr(&fields[i].Type),
			HeadComment: fields[i].HeadComment.String(),
			LineComment: fields[i].LineComment.String(),
			Decorators:  m.mapDecorators(fields[i].Decorators),
		}
	}
	return result
}

func (m *Mapper) mapTypeExpr(expr *ast.TypeExpression) PluginTypeRef {
	ref := PluginTypeRef{
		FQN:        expr.Base.String(),
		IsArray:    expr.IsArray,
		IsOptional: expr.IsOptional,
		IsScalar:   expr.IsScalar(),
	}

	if !expr.IsScalar() {
		if def, ok := m.schema.ResolveType(expr); ok {
			ns, _ := m.schema.NamespaceOf(def)
			baseName := ast.NameOf(def)
			if ns != "" {
				ref.FQN = ns + "." + baseName
			} else {
				ref.FQN = baseName
			}
		}
	}

	if len(expr.GenericArgs) > 0 {
		ref.Args = m.mapTypeArgs(expr.GenericArgs)
	}

	return ref
}

func (m *Mapper) mapTypeArgs(args []ast.TypeExpression) []PluginTypeRef {
	result := make([]PluginTypeRef, len(args))
	for i := range args {
		result[i] = m.mapTypeExpr(&args[i])
	}
	return result
}

func (m *Mapper) mapDecorators(decorators []ast.Decorator) []PluginDecorator {
	if len(decorators) == 0 {
		return nil
	}

	result := make([]PluginDecorator, len(decorators))
	for i, dec := range decorators {
		result[i] = PluginDecorator{
			Name: dec.Name,
			Args: m.mapDecoratorArgs(dec.Args),
		}
	}
	return result
}

func (m *Mapper) mapDecoratorArgs(args ast.OrderedTypeMap) []PluginDecoratorArg {
	var result []PluginDecoratorArg

	args.All()(func(key string, value ast.TypeNode) bool {
		arg := PluginDecoratorArg{}

		// parser stores decorator args with key always set;
		// if value is nil, it's a positional arg, otherwise key=name and value=TypeNode.
		if value == nil {
			v := key
			arg.Value = new(v)
		} else {
			k := key
			arg.Name = new(k)
			v := m.decoratorArgValue(value)
			arg.Value = new(v)
		}

		result = append(result, arg)
		return true
	})

	return result
}

func (m *Mapper) decoratorArgValue(node ast.TypeNode) string {
	switch v := node.(type) {
	case *ast.StringLiteral:
		return v.Value
	case *ast.IntLiteral:
		return fmt.Sprintf("%d", v.Value)
	case *ast.FloatLiteral:
		return fmt.Sprintf("%g", v.Value)
	case *ast.TypeExpression:
		return v.String()
	default:
		return ""
	}
}

func (m *Mapper) mapRawBlocks(blocks []ast.RawBlock) []PluginExtension {
	result := make([]PluginExtension, len(blocks))
	for i, block := range blocks {
		pairs := make([]PluginExtensionPair, len(block.Pairs))
		for j, pair := range block.Pairs {
			pairs[j] = PluginExtensionPair{
				Key:   pair.Key.Value,
				Value: m.rawPairValue(pair.Value),
			}
		}

		result[i] = PluginExtension{
			Target: block.Target.Value,
			Pairs:  pairs,
		}
	}
	return result
}

func (m *Mapper) rawPairValue(node ast.TypeNode) any {
	switch v := node.(type) {
	case *ast.StringLiteral:
		return v.Value
	case *ast.IntLiteral:
		return v.Value
	case *ast.FloatLiteral:
		return v.Value
	default:
		return nil
	}
}

func (m *Mapper) specFQN(spec ast.SpecNode, ns string) string {
	name := ast.NameOf(spec)
	if ns != "" {
		return ns + "." + name
	}
	return name
}

func (m *Mapper) monomorphHeadComment(mono *resolver.Monomorph) string {
	if mono.Original == nil {
		return ""
	}

	switch node := (*mono.Original).(type) {
	case *ast.Model:
		return node.HeadComment.String()
	case *ast.Input:
		return node.HeadComment.String()
	default:
		return ""
	}
}

func (m *Mapper) monomorphOriginalFQN(mono *resolver.Monomorph) string {
	if mono.Original == nil {
		return ""
	}

	name := ""
	switch node := (*mono.Original).(type) {
	case *ast.Model:
		name = node.Name.Value
	case *ast.Input:
		name = node.Name.Value
	}

	if mono.Namespace != "" && name != "" {
		return mono.Namespace + "." + name
	}
	return name
}
