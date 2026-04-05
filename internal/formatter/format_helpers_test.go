package formatter

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/jimschubert/spray/ast"
)

func typeExpr(name string) ast.TypeExpression {
	return ast.TypeExpression{
		Base: ast.QualifiedIdent{Parts: []string{name}},
	}
}

func optionalTypeExpr(name string) ast.TypeExpression {
	te := typeExpr(name)
	te.IsOptional = true
	return te
}

func arrayTypeExpr(name string) ast.TypeExpression {
	te := typeExpr(name)
	te.IsArray = true
	return te
}

func genericTypeExpr(name string, args ...string) ast.TypeExpression {
	te := typeExpr(name)
	for _, a := range args {
		te.GenericArgs = append(te.GenericArgs, typeExpr(a))
	}
	return te
}

func decorator(name string) ast.Decorator {
	return ast.Decorator{Name: name}
}

func decoratorWithArg(name, key string, value ast.TypeNode) ast.Decorator {
	d := ast.Decorator{Name: name}
	d.Args.Set(key, value, ast.Position{})
	return d
}

func decoratorBareArg(name, key string) ast.Decorator {
	d := ast.Decorator{Name: name}
	d.Args.Set(key, nil, ast.Position{})
	return d
}

func field(name string, typ ast.TypeExpression, decorators ...ast.Decorator) ast.Field {
	return ast.Field{
		Name:       ast.StringLiteral{Value: name},
		Type:       typ,
		Decorators: decorators,
	}
}

func fieldWithComment(name string, typ ast.TypeExpression, comment string, decorators ...ast.Decorator) ast.Field {
	return ast.Field{
		Name:        ast.StringLiteral{Value: name},
		Type:        typ,
		Decorators:  decorators,
		LineComment: &ast.Comment{Text: comment},
	}
}

func restRoute(method string, segments ast.PathSegments, ret ast.TypeExpression, decorators ...ast.Decorator) *ast.RestRoute {
	return &ast.RestRoute{
		Method:     method,
		Path:       segments,
		Return:     ret,
		Decorators: decorators,
	}
}

func pathSegs(parts ...ast.PathSegment) ast.PathSegments {
	return parts
}

func seg(value string) ast.PathSegment {
	return ast.PathSegment{Value: value}
}

func param(value string) ast.PathSegment {
	return ast.PathSegment{Value: value, IsParam: true}
}

func rpcRoute(name string, input, ret ast.TypeExpression, streaming bool, decorators ...ast.Decorator) *ast.RpcRoute {
	return &ast.RpcRoute{
		Name:       ast.StringLiteral{Value: name},
		Input:      input,
		Return:     ret,
		Streaming:  streaming,
		Decorators: decorators,
	}
}

func eventRoute(name string, direction ast.EventDirection, event ast.TypeExpression, decorators ...ast.Decorator) *ast.EventRoute {
	return &ast.EventRoute{
		Direction:  direction,
		Name:       ast.StringLiteral{Value: name},
		Event:      event,
		Decorators: decorators,
	}
}

func commentGroup(lines ...string) *ast.CommentGroup {
	cg := &ast.CommentGroup{}
	for i, line := range lines {
		cg.Comments = append(cg.Comments, &ast.Comment{
			Pos:  ast.Position{Line: i + 1},
			Text: line,
		})
	}
	return cg
}

func formatSpec(t *testing.T, spec ast.SpecNode, opts ...Options) string {
	t.Helper()
	stencil := &ast.Stencil{
		Specs: []ast.SpecNode{spec},
	}
	f, err := New(opts...)
	assert.NoError(t, err)
	got, err := f.Format(stencil)
	assert.NoError(t, err)
	return string(got)
}

