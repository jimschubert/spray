package parser

import (
	"github.com/jimschubert/spray/ast"
	"github.com/jimschubert/spray/internal/lexer"
)

// Parser provides a high-level API for parsing Spray sources into an AST.
type Parser struct {
	internal *lexer.Parser
}

// New creates a new Parser instance.
func New() (*Parser, error) {
	p, err := lexer.New()
	if err != nil {
		return nil, err
	}
	return &Parser{internal: p}, nil
}

// Parse parses the provided text into an ast.Stencil instance.
func (p *Parser) Parse(text string) (*ast.Stencil, error) {
	return p.internal.Parse(text)
}
