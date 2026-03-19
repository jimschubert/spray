package lexer

import (
	"fmt"

	"github.com/jimschubert/spray/internal/ast"
)

// ParsingError represents an error encountered during parsing.
type ParsingError struct {
	Pos     ast.Position
	Message string
}

func (e *ParsingError) Error() string {
	return fmt.Sprintf("parsing error at %d:%d: %s", e.Pos.Line, e.Pos.Col, e.Message)
}

// Parser holds the state of the parserState.
type Parser struct{}

func New() (*Parser, error) {
	return &Parser{}, nil
}

// parserState is bound to the lexer, allowing for a windowed view of the tokens
type parserState struct {
	l       *lexer
	current item  // the most recently consumed token
	peeked  *item // one token of lookahead, if any
}

// next (skips newlines)
func (p *parserState) next() item {
	if p.peeked != nil {
		it := *p.peeked
		p.peeked = nil
		p.current = it
		return it
	}
	for {
		it := p.l.nextItem()
		if it.typ != itemNewline {
			p.current = it
			return it
		}
	}
}

// peek (skips newlines0
func (p *parserState) peek() item {
	if p.peeked != nil {
		return *p.peeked
	}
	for {
		it := p.l.nextItem()
		if it.typ != itemNewline {
			p.peeked = &it
			return it
		}
	}
}

// expect consumes the next token and returns an error if unexpected
func (p *parserState) expect(typ itemType) (item, error) {
	it := p.next()
	if it.typ != typ {
		return it, &ParsingError{
			Pos:     itemPos(it),
			Message: fmt.Sprintf("expected %v but got %v", typ, it.typ),
		}
	}
	return it, nil
}

// collectComments accumulates all comments up until the next token is not a comment
func (p *parserState) collectComments() []*ast.Comment {
	var comments []*ast.Comment
	for {
		it := p.peek()
		if it.typ != itemComment {
			break
		}
		p.next() // consume it
		comments = append(comments, &ast.Comment{
			Pos:  ast.Position{Line: it.line, Col: int(it.pos)},
			Text: it.val,
		})
	}
	return comments
}

// itemPos gets an ast.Position from an item
func itemPos(i item) ast.Position {
	return ast.Position{Line: i.line, Col: int(i.pos)}
}

// parseNamespace parses a namespace declaration, e.g.:
//
//	namespace QualifiedIdent NEWLINE
//
// A namespace may have a header comment (single preceding line is a comment)
// or a line comment, e.g:
//
//	Comment
//	namespace QualifiedIdent Comment NEWLINE
func (p *parserState) parseNamespace(leading *ast.Comment) (*ast.Namespace, error) {
	kw, err := p.expect(itemKeywordNamespace)
	if err != nil {
		return nil, err
	}

	ns := &ast.Namespace{
		Pos:         itemPos(kw),
		HeadComment: leading,
	}

	// can be single identifier like acme, or qualified like acme.users.v1
	first, err := p.expect(itemIdent)
	if err != nil {
		return nil, &ParsingError{Pos: itemPos(first), Message: "expected identifier after 'namespace'"}
	}

	qi := ast.QualifiedIdent{
		Pos:   itemPos(first),
		Parts: []string{first.val},
	}

	for p.peek().typ == itemDot {
		p.next() // consume dot
		part, err := p.expect(itemIdent)
		if err != nil {
			return nil, &ParsingError{Pos: itemPos(part), Message: "expected identifier after '.'"}
		}
		qi.Parts = append(qi.Parts, part.val)
	}

	ns.Name = qi

	if p.peek().typ == itemComment {
		comment := p.next() // consume comment
		ns.LineComment = &ast.Comment{
			Pos:  itemPos(comment),
			Text: comment.val,
		}
	}

	return ns, nil
}

// Parse both lexes and parses the input, returning the root Stencil AST node.
func (p *Parser) Parse(text string) (*ast.Stencil, error) {
	l := lex("input", text)
	l.run()

	pp := &parserState{l: l}
	stencil := &ast.Stencil{}

	for {
		// store comments here so any leading comments can be attached to nodes supporting it
		comments := pp.collectComments()
		// store all comments at the document level as well
		stencil.Comments = append(stencil.Comments, comments...)

		it := pp.peek()

		switch it.typ {
		case itemEOF:
			pp.next()
			return stencil, nil

		case itemError:
			pp.next()
			return nil, &ParsingError{
				Pos:     itemPos(it),
				Message: it.val,
			}

		case itemKeywordNamespace:
			// allows only a single leading comment (for now?)
			var leading *ast.Comment
			if len(comments) > 0 {
				leading = comments[len(comments)-1]
				stencil.Comments = stencil.Comments[:len(stencil.Comments)-1]
			}

			if stencil.Namespace != nil {
				return nil, &ParsingError{
					Pos:     itemPos(it),
					Message: "multiple namespace declarations are not allowed",
				}
			}

			ns, err := pp.parseNamespace(leading)
			if err != nil {
				return nil, err
			}

			stencil.Namespace = ns

		default:
			pp.next()
			return nil, &ParsingError{
				Pos:     itemPos(it),
				Message: fmt.Sprintf("unexpected token %q", it.val),
			}
		}
	}
}
