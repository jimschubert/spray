package lexer

import (
	"fmt"
	"strings"

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

// peekAny checks if the next token is any of the provided types, returning the token and true if it is.
func (p *parserState) peekAny(types ...itemType) (item, bool) {
	it := p.peek()
	for _, typ := range types {
		if it.typ == typ {
			return it, true
		}
	}
	return it, false
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

// isKeyword checks if an itemType is a keyword token
func isKeyword(typ itemType) bool {
	return typ >= itemKeywordNamespace && typ <= itemKeywordAny
}

// itemStringLiteral converts an item to an ast.StringLiteral
func itemStringLiteral(i item) ast.StringLiteral {
	return ast.StringLiteral{
		Pos:   itemPos(i),
		Value: i.val,
	}
}

// parseNamespace parses a namespace declaration.
//
// A namespace may have a header comment (single preceding line is a comment) or a line comment.
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

	if _, err = p.expect(itemLeftBrace); err != nil {
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

func (p *parserState) parseTypeAlias(group *ast.CommentGroup) (*ast.TypeAlias, error) {
	it, err := p.expect(itemKeywordType)
	if err != nil {
		return nil, err
	}

	name, err := p.parseIdent("type alias")
	if err != nil {
		return nil, err
	}

	typeAlias := &ast.TypeAlias{
		Pos:         itemPos(it),
		HeadComment: group,
		Name:        *name,
	}

	if _, err = p.expect(itemEquals); err != nil {
		return nil, err
	}

	expr, err := p.parseTypeExpression()
	if err != nil {
		return nil, err
	}

	typeAlias.Type = *expr

	return typeAlias, err
}

func (p *parserState) parseTypeExpression() (*ast.TypeExpression, error) {
	var err error
	var aliased *ast.QualifiedIdent
	if it, ok := p.peekAny(
		itemKeywordString,
		itemKeywordInt,
		itemKeywordFloat,
		itemKeywordBoolean,
		itemKeywordUUID,
		itemKeywordTimestamp,
		itemKeywordDate,
		itemKeywordAny,
	); ok {
		p.next() // consume scalar type
		aliased = &ast.QualifiedIdent{
			Pos:   itemPos(it),
			Parts: []string{it.String()},
		}
	} else {
		aliased, err = p.parseQualifiedIdent("type alias type")
		if err != nil {
			return nil, err
		}
	}

	var args []ast.TypeExpression
	var isOptional bool
	var isArray bool

	if p.peek().typ == itemQuestion {
		isOptional = true
		p.next() // consume '?'
	} else if p.peek().typ == itemLeftBracket {
		isArray = true
		p.next() // consume '['
		if _, err = p.expect(itemRightBracket); err != nil {
			return nil, err
		}
	} else if p.peek().typ == itemLeftAngle {
		p.next() // consume '<'
		for {
			arg, err := p.parseTypeExpression()
			if err != nil {
				return nil, err
			}
			args = append(args, *arg)

			if p.peek().typ == itemComma {
				p.next() // consume comma, continue to next arg
				continue
			} else if p.peek().typ == itemRightAngle {
				p.next() // consume '>'
				if len(args) == 0 {
					return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "generic type argument list cannot be empty"}
				}
				break
			} else {
				return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "expected ',' or '>' in generic type argument list"}
			}
		}
	}

	return &ast.TypeExpression{
		Pos:         aliased.Position(),
		Base:        *aliased,
		GenericArgs: args,
		IsOptional:  isOptional,
		IsArray:     isArray,
	}, nil
}

func (p *parserState) parseDecorator() (*ast.Decorator, error) {
	kw, err := p.expect(itemAt)
	if err != nil {
		return nil, err
	}

	decorator := &ast.Decorator{
		Pos: itemPos(kw),
	}

	it := p.peek()
	if it.typ == itemIdent {
		p.next() // consume identifier
		decorator.Name = it.val
	} else {
		return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "expected decorator name after '@'"}
	}

	if p.peek().typ == itemLeftParen {
		p.next() // consume '('

		// check for special literal values first
		if special, ok := p.peekAny(itemKeywordNow, itemString, itemInt, itemFloat); ok {
			p.next() // consume keyword or literal
			decorator.Args.Set(special.val, nil, itemPos(special))
		} else if p.peek().typ == itemIdent || isKeyword(p.peek().typ) {
			// Accept identifiers or any keyword as decorator argument values
			// Keywords like 'rest', 'bearer', 'events' can be used as values in decorators
			keyOrValue := p.next() // consume identifier or keyword

			// Check if this is `key: value` syntax
			if p.peek().typ == itemColon {
				p.next() // consume ':'

				// Value after colon can be identifier or keyword
				if p.peek().typ == itemIdent || isKeyword(p.peek().typ) {
					valueToken := p.next() // consume value
					typeExpr := &ast.TypeExpression{
						Pos: itemPos(valueToken),
						Base: ast.QualifiedIdent{
							Pos:   itemPos(valueToken),
							Parts: []string{valueToken.val},
						},
					}
					decorator.Args.Set(keyOrValue.val, typeExpr, itemPos(keyOrValue))
				} else {
					return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "expected identifier or keyword after ':' in decorator argument"}
				}
			} else {
				// Simple value like @style(rest) or @default(member)
				typeExpr := &ast.TypeExpression{
					Pos: itemPos(keyOrValue),
					Base: ast.QualifiedIdent{
						Pos:   itemPos(keyOrValue),
						Parts: []string{keyOrValue.val},
					},
				}
				decorator.Args.Set(keyOrValue.val, typeExpr, itemPos(keyOrValue))
			}
		} else {
			return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "expected identifier, keyword, string literal, or special value as decorator argument"}
		}

		if _, err = p.expect(itemRightParen); err != nil {
			return nil, err
		}
	}

	return decorator, nil
}

func (p *parserState) parseGenericParams() ([]ast.StringLiteral, error) {
	if p.peek().typ != itemLeftAngle {
		return nil, nil
	}

	p.next() // consume '<'

	var params []ast.StringLiteral
	for {
		if p.peek().typ != itemIdent {
			return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "expected identifier in generic parameter list"}
		}

		param := p.next() // consume parameter name
		params = append(params, itemStringLiteral(param))

		if p.peek().typ == itemComma {
			p.next() // consume comma, continue to next parameter
			continue
		} else if p.peek().typ == itemRightAngle {
			p.next() // consume '>'
			if len(params) == 0 {
				return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "generic parameter list cannot be empty"}
			}
			break
		} else {
			return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "expected ',' or '>' in generic parameter list"}
		}
	}

	return params, nil
}

func (p *parserState) parseField() (*ast.Field, error) {
	// collect comments to head comments. multiple leading comment groups will be a parser error
	comments := p.collectComments()
	if p.peek().typ != itemIdent {
		return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "expected identifier at start of field declaration"}
	}
	name := p.next() // consume field name

	_, err := p.expect(itemColon)
	if err != nil {
		return nil, err
	}

	typeExpr, err := p.parseTypeExpression()
	if err != nil {
		return nil, err
	}

	var leading *ast.CommentGroup
	if len(comments) > 0 {
		leading = &ast.CommentGroup{Comments: comments}
	}

	field := &ast.Field{
		Pos:         itemPos(name),
		Name:        itemStringLiteral(name),
		Type:        *typeExpr,
		HeadComment: leading,
	}

	for {
		next := p.peek()
		switch {
		case next.typ == itemAt:
			decorator, err := p.parseDecorator()
			if err != nil {
				return nil, err
			}
			field.Decorators = append(field.Decorators, *decorator)
			continue
		case next.typ == itemComment:
			comment := p.next()
			field.LineComment = &ast.Comment{
				Pos:  itemPos(comment),
				Text: comment.val,
			}
			continue
		default:
			return field, nil
		}
	}
}

func (p *parserState) parseInput(group *ast.CommentGroup) (*ast.Input, error) {
	kw, err := p.expect(itemKeywordInput)
	if err != nil {
		return nil, err
	}

	name, err := p.parseIdent("input")
	if err != nil {
		return nil, err
	}

	input := &ast.Input{
		Pos:         itemPos(kw),
		HeadComment: group,
		Name:        *name,
	}

	_, err = p.expect(itemLeftBrace)
	if err != nil {
		return nil, err
	}

	for {
		next := p.peek()
		if next.typ == itemRightBrace {
			p.next() // consume right brace
			break
		}
		field, err := p.parseField()
		if err != nil {
			return nil, err
		}
		input.Fields = append(input.Fields, *field)
	}

	return input, nil
}

func (p *parserState) parseModel(group *ast.CommentGroup) (*ast.Model, error) {
	kw, err := p.expect(itemKeywordModel)
	if err != nil {
		return nil, err
	}

	name, err := p.parseIdent("model")
	if err != nil {
		return nil, err
	}

	model := &ast.Model{
		Pos:         itemPos(kw),
		HeadComment: group,
		Name:        *name,
	}

	genericParams, err := p.parseGenericParams()
	if err != nil {
		return nil, err
	}
	model.GenericParams = genericParams

	_, err = p.expect(itemLeftBrace)
	if err != nil {
		return nil, err
	}

	for {
		next := p.peek()
		if next.typ == itemRightBrace {
			p.next() // consume right brace
			break
		}
		field, err := p.parseField()
		if err != nil {
			return nil, err
		}
		model.Fields = append(model.Fields, *field)
	}

	return model, nil
}

func (p *parserState) parseRestRoute() (*ast.RestRoute, error) {
	comments := p.collectComments()
	var kw item
	if i, ok := p.peekAny(itemKeywordGET, itemKeywordPOST, itemKeywordPUT, itemKeywordPATCH, itemKeywordDELETE, itemKeywordOPTIONS, itemKeywordHEAD); ok {
		kw = i
		p.next() // consume HTTP method keyword
	} else {
		return nil, &ParsingError{
			Pos:     itemPos(p.peek()),
			Message: fmt.Sprintf("invalid HTTP method '%s' for REST route", p.peek().val),
		}
	}

	route := &ast.RestRoute{
		Pos:    itemPos(kw),
		Method: kw.val,
	}

	if len(comments) > 0 {
		route.HeadComment = &ast.CommentGroup{Comments: comments}
	}

	_, err := p.expect(itemSlash)
	if err != nil {
		return nil, err
	}

	for {
		it := p.peek()
		if it.typ == itemArrow {
			p.next() // consume '->'
			break
		}
		if it.typ == itemSlash {
			p.next() // consume '/'
			continue
		}
		if it.typ == itemIdent || it.typ == itemColon {
			pathSegment := ast.PathSegment{
				Pos:     itemPos(it),
				Value:   it.val,
				IsParam: strings.HasPrefix(it.val, ":"),
			}
			if it.typ == itemColon {
				pathSegment.IsParam = true
				p.next() // consume ':'
				if p.peek().typ != itemIdent {
					return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "expected identifier after ':' in path parameter"}
				}
				pathSegment.Value = p.next().val // consume parameter name
			} else {
				p.next() // consume path segment
			}

			route.Path = append(route.Path, pathSegment)
		} else {
			return nil, &ParsingError{Pos: itemPos(it), Message: "unexpected token in route path; expected identifier, ':', or '->'"}
		}
	}

	if p.peek().typ != itemIdent {
		return nil, &ParsingError{Pos: itemPos(p.peek()), Message: "expected return type after '->' in route declaration"}
	}

	returnType, err := p.parseTypeExpression()
	if err != nil {
		return nil, err
	}
	route.Return = *returnType

	// parse optional decorators on route
	for p.peek().typ == itemAt {
		decorator, err := p.parseDecorator()
		if err != nil {
			return nil, err
		}
		route.Decorators = append(route.Decorators, *decorator)
	}

	return route, nil
}

func (p *parserState) parseApi(group *ast.CommentGroup) (*ast.Api, error) {
	kw, err := p.expect(itemKeywordAPI)
	if err != nil {
		return nil, err
	}

	name, err := p.parseIdent("api")
	if err != nil {
		return nil, err
	}

	api := &ast.Api{
		Pos:         itemPos(kw),
		HeadComment: group,
		Name:        *name,
	}

	for p.peek().typ == itemAt {
		decorator, err := p.parseDecorator()
		if err != nil {
			return nil, err
		}

		if decorator.Name != "version" && decorator.Name != "style" {
			return nil, &ParsingError{
				Pos:     decorator.Position(),
				Message: "only 'version' and 'style' decorators are allowed on API before block opening brace",
			}
		}

		// if ApiDecorators already contains the decorator, this is an error (no duplicates)
		for _, d := range api.ApiDecorators {
			if d.Name == decorator.Name {
				return nil, &ParsingError{
					Pos:     decorator.Position(),
					Message: fmt.Sprintf("duplicate decorator '%s' on API", decorator.Name),
				}
			}
		}

		api.ApiDecorators = append(api.ApiDecorators, *decorator)

		if decorator.Name == "style" {
			decorator.Args.All()(func(s string, node ast.TypeNode) bool {
				switch s {
				case "rest":
					api.Style = ast.REST
				case "rpc":
					api.Style = ast.RPC
				case "events":
					api.Style = ast.EVENTS
				default:
					err = &ParsingError{
						Pos:     decorator.Position(),
						Message: fmt.Sprintf("invalid argument '%s' for @style decorator; expected 'rest', 'rpc', or 'events'", s),
					}
				}
				return true
			})
		}
	}

	_, err = p.expect(itemLeftBrace) // consume '{'
	if err != nil {
		return nil, err
	}

	if len(api.ApiDecorators) == 0 {
		api.Style = ast.REST
	}

	for p.peek().typ == itemAt {
		directive, err := p.parseDecorator()
		if err != nil {
			return nil, err
		}
		api.ApiDirectives = append(api.ApiDirectives, *directive)
	}

	// TODO: currently only support REST…implement parse of other routes
	switch api.Style {
	case ast.REST:
		for {
			next := p.peek()
			if next.typ == itemRightBrace {
				break
			}
			route, err := p.parseRestRoute()
			if err != nil {
				return nil, err
			}
			api.Routes = append(api.Routes, route)
		}
	default:
		return nil, &ParsingError{
			Pos:     itemPos(p.peek()),
			Message: fmt.Sprintf("unsupported API style or not yet implemented '%T'", api.Style),
		}
	}

	_, err = p.expect(itemRightBrace) // consume '}'
	if err != nil {
		return nil, err
	}

	return api, nil
}

// semanticValidation validates the semantics of the AST, returning a composite error if any issues are found (e.g.
// duplicate declarations, imports defined after models, etc.)
func (p *Parser) semanticValidation(stencil *ast.Stencil) error {
	// TODO: imports must come before any models, apis, or other declarations, but after namespace
	// TODO: warnings on duplicate imported types
	return nil
}

// Parse both lexes and parses the input, returning the func (p *parserState) parseTypeAlias(group *ast.CommentGroup) (*ast.TypeAlias, interface{}) {	 AST is non-nil in the event of an error, allowing the caller to inspect the partially constructed AST.
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

		case itemKeywordType:
			var group *ast.CommentGroup
			if len(comments) > 0 {
				group = &ast.CommentGroup{Comments: comments}
				// trim from file comments
				stencil.Comments = stencil.Comments[:len(stencil.Comments)-len(comments)]
			}

			a, err := pp.parseTypeAlias(group)
			if err != nil {
				return nil, err
			}

			stencil.Specs = append(stencil.Specs, a)

		case itemKeywordModel:
			var group *ast.CommentGroup
			if len(comments) > 0 {
				group = &ast.CommentGroup{Comments: comments}
				// trim from file comments
				stencil.Comments = stencil.Comments[:len(stencil.Comments)-len(comments)]
			}

			model, err := pp.parseModel(group)
			if err != nil {
				return nil, err
			}

			stencil.Specs = append(stencil.Specs, model)

		case itemKeywordInput:
			var group *ast.CommentGroup
			if len(comments) > 0 {
				group = &ast.CommentGroup{Comments: comments}
				// trim from file comments
				stencil.Comments = stencil.Comments[:len(stencil.Comments)-len(comments)]
			}

			input, err := pp.parseInput(group)
			if err != nil {
				return nil, err
			}

			stencil.Specs = append(stencil.Specs, input)

		case itemKeywordAPI:
			var group *ast.CommentGroup
			if len(comments) > 0 {
				group = &ast.CommentGroup{Comments: comments}
				// trim from file comments
				stencil.Comments = stencil.Comments[:len(stencil.Comments)-len(comments)]
			}

			api, err := pp.parseApi(group)
			if err != nil {
				return nil, err
			}

			stencil.Specs = append(stencil.Specs, api)

		default:
			pp.next()
			return nil, &ParsingError{
				Pos:     itemPos(it),
				Message: fmt.Sprintf("unexpected token %q", it.val),
			}
		}
	}
}
