package lexer

import (
	"fmt"

	"github.com/jimschubert/spray/internal/ast"
)

type ParsingError struct {
	Pos     ast.Position
	Message string
}

func (e *ParsingError) Error() string {
	return fmt.Sprintf("parsing error at %d:%d: %s", e.Pos.Line, e.Pos.Col, e.Message)
}

type Parser struct{}

func New() (*Parser, error) {
	return &Parser{}, nil
}

func (p *Parser) Parse(text string) (*ast.Stencil, error) {
	l := lex("input", text)
	l.run()

	for {
		it := l.nextItem()
		fmt.Printf("Token: %s (pos: %d, line: %d)\n", it, it.pos, it.line)

		if it.typ == itemEOF {
			break
		}
		if it.typ == itemError {
			return nil, &ParsingError{
				Pos:     ast.Position{Line: it.line, Col: int(it.pos)},
				Message: it.val,
			}
		}
	}
	return nil, nil
}
