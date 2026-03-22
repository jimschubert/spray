package parser

import (
	"github.com/jimschubert/spray/ast"
	"github.com/jimschubert/spray/internal/lexer"
)

type Parser struct {
	internal *lexer.Parser
}

func New() (*Parser, error) {
	p, err := lexer.New()
	if err != nil {
		return nil, err
	}
	return &Parser{internal: p}, nil
}

func (p *Parser) Parse(text string) (*ast.Stencil, error) {
	return p.internal.Parse(text)
}
