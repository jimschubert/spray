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

// peek (skips newlines)
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

// peekRaw returns the next token without skipping newlines
func (p *parserState) peekRaw() item {
	if p.peeked != nil {
		return *p.peeked
	}
	it := p.l.nextItem()
	p.peeked = &it
	return it
}

// nextRaw returns the next token without skipping newlines
func (p *parserState) nextRaw() item {
	if p.peeked != nil {
		it := *p.peeked
		p.peeked = nil
		p.current = it
		return it
	}
	it := p.l.nextItem()
	p.current = it
	return it
}

// expect consumes the next token and returns an error if unexpected
func (p *parserState) expect(typ itemType) (item, error) {
	it := p.next()
	if it.typ != typ {
		return it, &ParsingError{
			Pos:     itemPos(it),
			Message: fmt.Sprintf("expected a %q but got a %q", symbolsDescriptions[typ], symbolsDescriptions[it.typ]),
		}
	}
	return it, nil
}

// collectComments accumulates all comments up until the next token is not a comment.
// Comments are grouped such that \n\n terminates a group.
func (p *parserState) collectComments() []*ast.Comment {
	var comments []*ast.Comment
	newlineCount := 0

	for {
		it := p.peekRaw()

		if it.typ == itemNewline {
			newlineCount++
			if newlineCount >= 2 {
				// \n\n ends _this_ group of comments, allowing parser to associate CommentGroup with certain defs.
				break
			}
			p.nextRaw() // consume \n
			continue
		}

		if it.typ != itemComment {
			break
		}

		p.nextRaw() // consume comment
		comments = append(comments, &ast.Comment{
			Pos:  ast.Position{Line: it.line, Col: int(it.pos)},
			Text: it.val,
		})

		newlineCount = 0
	}

	return comments
}

// itemPos gets an ast.Position from an item
func itemPos(i item) ast.Position {
	return ast.Position{Line: i.line, Col: int(i.pos)}
}

// itemStringLiteral converts an item to an ast.StringLiteral
func itemStringLiteral(i item) ast.StringLiteral {
	return ast.StringLiteral{
		Pos:   itemPos(i),
		Value: i.val,
	}
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

	qi, err := p.parseQualifiedIdent("namespace")
	if err != nil {
		return nil, err
	}
	ns.Name = *qi

	if p.peek().typ == itemComment {
		comment := p.next() // consume comment
		ns.LineComment = &ast.Comment{
			Pos:  itemPos(comment),
			Text: comment.val,
		}
	}

	return ns, nil
}

func (p *parserState) parseImport(commentGroup *ast.CommentGroup) (*ast.Import, error) {
	kw, err := p.expect(itemKeywordImport)
	if err != nil {
		return nil, err
	}

	imp := &ast.Import{
		Pos:         itemPos(kw),
		HeadComment: commentGroup,
	}

	// must start with a string, not dot or other
	if p.peek().typ != itemIdent {
		return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "expected identifier after 'import'"}
	}

	qi, err := p.parseQualifiedIdent("import")
	if err != nil {
		return nil, err
	}
	imp.Path = *qi

	// { Name [, Name] }
	_, err = p.expect(itemLeftBrace)
	if err != nil {
		return nil, err
	}
	for {
		if p.peek().typ != itemIdent {
			// don't allow empty grouping or undefined after comma; the trailing brace will be handled via peek later on identifier
			return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "expected identifier in import name list"}
		}

		imp.Names = append(imp.Names, itemStringLiteral(p.next())) // consume name

		if p.peek().typ == itemComma {
			p.next() // consume comma but next needs to be an identifier
			continue
		} else if p.peek().typ == itemRightBrace {
			// don't consume yet, just exit loop
			break
		} else {
			return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "expected ',' or '}' in import name list"}
		}
	}

	_, err = p.expect(itemRightBrace)
	if err != nil {
		return nil, err
	}

	if p.peek().typ == itemComment {
		comment := p.next() // consume comment
		imp.LineComment = &ast.Comment{
			Pos:  itemPos(comment),
			Text: comment.val,
		}
	}

	return imp, nil
}

func (p *parserState) parseQualifiedIdent(context string) (*ast.QualifiedIdent, error) {
	if p.peek().typ != itemIdent {
		return nil, &ParsingError{Pos: itemPos(p.peek()), Message: fmt.Sprintf("expected identifier after %q", context)}
	}
	qi := &ast.QualifiedIdent{
		Pos:   itemPos(p.peek()),
		Parts: []string{p.next().val},
	}

	for p.peek().typ == itemDot {
		p.next() // consume dot
		if p.peek().typ != itemIdent {
			return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "expected identifier after '.'"}
		}
		qi.Parts = append(qi.Parts, p.next().val)
	}

	return qi, nil
}

func (p *parserState) parseEnum(group *ast.CommentGroup) (*ast.Enum, error) {
	kw, err := p.expect(itemKeywordEnum)
	if err != nil {
		return nil, err
	}

	name, err := p.parseIdent("enum")
	if err != nil {
		return nil, err
	}

	enum := &ast.Enum{
		Pos:         itemPos(kw),
		HeadComment: group,
		Name:        *name,
	}

	_, err = p.expect(itemLeftBrace)
	if err != nil {
		return nil, err
	}

	for {
		element, err := p.parseIdent("enum element")
		if err != nil {
			return nil, err
		}

		enum.Elements = append(enum.Elements, *element)

		if p.peek().typ == itemIdent {
			continue
		}
		if p.peek().typ == itemComma {
			p.next() // consume comma, continue to next element
			// e.g ,}
			if p.peek().typ == itemRightBrace {
				return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "trailing comma not allowed in enum element list"}
			}
			continue
		} else if p.peek().typ == itemRightBrace {
			// don't consume yet, will be consumed by expect() below
			break
		} else {
			return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "expected ',' or '}' in enum element list"}
		}
	}

	_, err = p.expect(itemRightBrace)
	if err != nil {
		return nil, err
	}

	return enum, nil
}

func (p *parserState) parseIdent(context string) (*ast.StringLiteral, error) {
	if p.peek().typ != itemIdent {
		// don't allow empty grouping or undefined after comma; the trailing brace will be handled via peek later on identifier
		return nil, &ParsingError{Pos: itemPos(p.peek()), Message: fmt.Sprintf("expected identifier after %q", context)}
	}
	v := p.next() // consume ident
	return &ast.StringLiteral{
		Pos:   itemPos(v),
		Value: v.val,
	}, nil
}

func (p *parserState) parseStringLiteral() (*ast.StringLiteral, error) {
	it, err := p.expect(itemString)
	if err != nil {
		return nil, err
	}
	return &ast.StringLiteral{
		Pos:   itemPos(it),
		Value: it.val,
	}, nil
}

// semanticValidation validates the semantics of the AST, returning a composite error if any issues are found (e.g.
// duplicate declarations, imports defined after models, etc.)
func (p *Parser) semanticValidation(stencil *ast.Stencil) error {
	// TODO: imports must come before any models, apis, or other declarations, but after namespace
	// TODO: warnings on duplicate imported types
	return nil
}

// Parse both lexes and parses the input, returning the root Stencil AST node.
// The AST semantics are validated prior to return.
// NOTE: the AST is non-nil in the event of an error, allowing the caller to inspect the partially constructed AST.
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

		// collectComments will leave the last \n from the \n\n group termination
		for pp.peekRaw().typ == itemNewline {
			// consume that last \n
			pp.nextRaw()
		}

		// new comment or comment group?
		if pp.peekRaw().typ == itemComment {
			continue
		}

		it := pp.peek()

		switch it.typ {
		case itemEOF:
			pp.next()

			if stencil.Namespace == nil {
				stencil.Namespace = &ast.Namespace{
					Name: ast.QualifiedIdent{
						Parts: []string{"default"}},
					Implicit: true,
				}
			}

			err := p.semanticValidation(stencil)
			return stencil, err

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
				// syntax error, not semantic, since multiple namespaces are not allowed in a single file
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

		case itemKeywordImport:
			var commentGroup *ast.CommentGroup

			// any grouping of comments (those immediately preceding import) are a comment "group" heading
			if len(comments) > 0 {
				commentGroup = &ast.CommentGroup{Comments: comments}
				// trim from file comments
				stencil.Comments = stencil.Comments[:len(stencil.Comments)-len(comments)]
			}

			i, err := pp.parseImport(commentGroup)
			if err != nil {
				return nil, err
			}

			if i != nil {
				stencil.Imports = append(stencil.Imports, *i)
			}

		case itemKeywordEnum:
			var group *ast.CommentGroup
			if len(comments) > 0 {
				group = &ast.CommentGroup{Comments: comments}
				// trim from file comments
				stencil.Comments = stencil.Comments[:len(stencil.Comments)-len(comments)]
			}

			e, err := pp.parseEnum(group)
			if err != nil {
				return nil, err
			}

			stencil.Specs = append(stencil.Specs, e)

		default:
			pp.next()
			return nil, &ParsingError{
				Pos:     itemPos(it),
				Message: fmt.Sprintf("unexpected token %q", it.val),
			}
		}
	}
}
