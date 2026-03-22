package ast

import (
	"fmt"
	"strings"
)

type ApiStyle int

const (
	REST ApiStyle = iota
	RPC
	EVENTS
)

type EventDirection int

const (
	PUBLISH EventDirection = iota
	SUBSCRIBE
)

// Node is the base interface for all AST nodes.
type Node interface {
	Position() Position
}

// SpecNode is an interface which is used for type specifications (api, model, input, enum, aliases) and is used to distinguish them from other nodes (e.g. comments).
type SpecNode interface {
	Node
	specNode() // disallows using other Node values where SpecNode is required; strategy taken from Go's own ast package
}

// TypeNode is an interface which is used for type expressions (e.g. field types, type alias definitions) and is used to
// distinguish them from other nodes. This allows us to enforce that only type expressions can be used in certain contexts
// (e.g. field definitions, type aliases).
type TypeNode interface {
	Node
	typeNode() // disallows other Node values where TypeNode is required
}

// Route defines the shared interface for rest, rpc, event style routes
type Route interface {
	Style() ApiStyle
}

// Position represents the line and column of a node in the source file.
type Position struct {
	Line int
	Col  int
}

// QualifiedIdent represents a qualified identifier, a dot-delimited set of identifiers (e.g., foo.bar.baz).
type QualifiedIdent struct {
	Pos   Position
	Parts []string
}

func (q *QualifiedIdent) Position() Position {
	return q.Pos
}

func (q *QualifiedIdent) String() string {
	var result strings.Builder
	for i, part := range q.Parts {
		if i > 0 {
			result.WriteString(".")
		}
		result.WriteString(part)
	}
	return result.String()
}

// StringLiteral represents a string literal in the source file. (i.e. IDENT)
type StringLiteral struct {
	Pos   Position
	Value string
}

func (s *StringLiteral) Position() Position {
	return s.Pos
}

type IntLiteral struct {
	Pos   Position
	Value int
}

func (i *IntLiteral) Position() Position {
	return i.Pos
}

type FloatLiteral struct {
	Pos   Position
	Value float64
}

func (f *FloatLiteral) Position() Position {
	return f.Pos
}

type Comment struct {
	Pos  Position
	Text string
}

func (c *Comment) Position() Position {
	return c.Pos
}

func (c *Comment) String() string {
	if c == nil {
		return ""
	}
	return c.Text
}

// CommentGroup represents a comment block to be associated with some other definition.
// This strategy for grouping comments is taken from go's AST package.
type CommentGroup struct {
	Comments []*Comment
}

func (cg *CommentGroup) Position() Position {
	if len(cg.Comments) == 0 {
		return Position{}
	}
	return cg.Comments[0].Position()
}

func (cg *CommentGroup) String() string {
	if cg == nil || len(cg.Comments) == 0 {
		return ""
	}

	sb := strings.Builder{}
	for idx, comment := range cg.Comments {
		if comment != nil {
			sb.WriteString(comment.String())
			if idx < len(cg.Comments)-1 {
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

func (cg *CommentGroup) IsEmpty() bool {
	return len(cg.Comments) == 0
}

type RawPair struct {
	Pos   Position
	Key   StringLiteral
	Value TypeNode // StringLiteral, IntLiteral, FloatLiteral, or nil for null
}

func (r *RawPair) Position() Position {
	return r.Pos
}

type RawBlock struct {
	Pos    Position
	Target StringLiteral
	Pairs  []RawPair
}

func (r *RawBlock) Position() Position {
	return r.Pos
}

// Namespace represents a namespace declaration, which has a qualified identifier and comments.
type Namespace struct {
	Pos         Position
	Name        QualifiedIdent
	HeadComment *Comment
	LineComment *Comment
	// Implicit determines whether the namespace was implicitly created by the parser (e.g. "default") or user-defined.
	Implicit bool
}

func (n *Namespace) Position() Position {
	return n.Pos
}

func (n *Namespace) FullName() string {
	return n.Name.String()
}

// Import represents an imported type declaration.
type Import struct {
	Pos         Position
	Path        QualifiedIdent
	Names       []StringLiteral
	HeadComment *CommentGroup
	LineComment *Comment
}

func (i *Import) Position() Position {
	return i.Pos
}

// FQNs returns the fully qualified names of the imported symbols, e.g. ["acme.common.v1.Page", "acme.common.v1.PaginationInput"].
func (i *Import) FQNs() []string {
	var fqns []string
	for _, n := range i.Names {
		fqns = append(fqns, i.Path.String()+"."+n.Value)
	}
	return fqns
}

func (i *Import) String() string {
	var names []string
	for _, n := range i.Names {
		names = append(names, n.Value)
	}
	return "import " + i.Path.String() + " { " + strings.Join(names, ", ") + " }"
}

type Enum struct {
	Pos         Position
	Name        StringLiteral
	Elements    []StringLiteral
	HeadComment *CommentGroup
}

func (e *Enum) Position() Position {
	return e.Pos
}

type TypeAlias struct {
	Pos         Position
	Name        StringLiteral
	Type        TypeExpression
	HeadComment *CommentGroup
	LineComment *Comment
}

func (a *TypeAlias) Position() Position {
	return a.Pos
}

type TypeExpression struct {
	Pos Position
	// Base is either a qualified identifier or scalar type name
	Base QualifiedIdent
	// GenericArgs (e.g., <User, ApiError>)
	GenericArgs []TypeExpression
	IsArray     bool
	IsOptional  bool
}

func (t *TypeExpression) Position() Position {
	return t.Pos
}

func (t *TypeExpression) IsScalar() bool {
	if len(t.Base.Parts) != 1 {
		return false
	}
	scalar := t.Base.Parts[0]
	switch scalar {
	case "string", "int", "float", "boolean", "uuid", "timestamp", "date", "any":
		return true
	default:
		return false
	}
}

func (t *TypeExpression) String() string {
	sb := strings.Builder{}
	sb.WriteString(t.Base.String())

	if len(t.GenericArgs) > 0 {
		sb.WriteString("<")
		for i, arg := range t.GenericArgs {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(arg.String())
		}
		sb.WriteString(">")
	}

	if t.IsArray {
		sb.WriteString("[]")
	}

	if t.IsOptional {
		sb.WriteString("?")
	}

	return sb.String()
}

type Decorator struct {
	Pos  Position
	Name string
	// Args is an ordered map of argument name to value with position tracking.
	Args OrderedTypeMap
}

func (d *Decorator) Position() Position {
	return d.Pos
}

func (d *Decorator) String() string {
	sb := strings.Builder{}
	sb.WriteString("@")
	sb.WriteString(d.Name)
	first := true
	d.Args.All()(func(key string, value TypeNode) bool {
		if first {
			sb.WriteString("(")
			first = false
		} else {
			sb.WriteString(", ")
		}

		if value == nil {
			// e.g. @default(now)
			sb.WriteString(key)
			return true
		}

		// e.g. @relation(key: value)
		sb.WriteString(key)
		sb.WriteString(": ")
		switch v := value.(type) {
		case *StringLiteral:
			// parser will handle quoted and unquoted strings
			sb.WriteString(v.Value)
		case *IntLiteral:
			sb.WriteString(fmt.Sprintf("%d", v.Value))
		case *FloatLiteral:
			sb.WriteString(fmt.Sprintf("%g", v.Value))
		case *TypeExpression:
			sb.WriteString(v.String())
		default:
			sb.WriteString("unknown")
		}
		return true
	})

	if !first {
		sb.WriteString(")")
	}
	return sb.String()
}

type Field struct {
	Pos         Position
	Name        StringLiteral
	Type        TypeExpression
	Decorators  []Decorator
	HeadComment *CommentGroup
	LineComment *Comment
}

func (f *Field) Position() Position {
	return f.Pos
}

type Model struct {
	Pos           Position
	Name          StringLiteral
	GenericParams []StringLiteral
	Fields        []Field
	Extensions    []RawBlock
	HeadComment   *CommentGroup
}

func (m *Model) Position() Position {
	return m.Pos
}

type Input struct {
	Pos         Position
	Name        StringLiteral
	Fields      []Field
	HeadComment *CommentGroup
}

func (i *Input) Position() Position {
	return i.Pos
}

type Api struct {
	Pos           Position
	Name          StringLiteral
	Style         ApiStyle
	ApiDecorators []Decorator // before '{', control features within api block
	ApiDirectives []Decorator
	Routes        []Route
	Extensions    []RawBlock
	HeadComment   *CommentGroup
}

func (a *Api) Position() Position {
	return a.Pos
}

type RestRoute struct {
	Pos         Position
	Method      string
	Path        []PathSegment
	Return      TypeExpression
	Decorators  []Decorator
	HeadComment *CommentGroup
}

func (r *RestRoute) Position() Position {
	return r.Pos
}

func (r *RestRoute) Style() ApiStyle {
	return REST
}

type PathSegment struct {
	Pos Position
	// Value is the literal value of the segment (e.g. "users" for /users or ":id" for /:id)
	Value   string
	IsParam bool
}

func (s *PathSegment) Position() Position {
	return s.Pos
}

type RpcRoute struct {
	Pos         Position
	Streaming   bool
	Name        StringLiteral
	Input       TypeExpression
	Return      TypeExpression
	Decorators  []Decorator
	HeadComment *CommentGroup
}

func (r *RpcRoute) Position() Position {
	return r.Pos
}

func (r *RpcRoute) Style() ApiStyle {
	return RPC
}

type EventRoute struct {
	Pos         Position
	Direction   EventDirection
	Name        StringLiteral
	Event       TypeExpression
	Decorators  []Decorator
	HeadComment *CommentGroup
}

func (e *EventRoute) Position() Position {
	return e.Pos
}

func (e *EventRoute) Style() ApiStyle {
	return EVENTS
}

func (e *Enum) specNode()      {}
func (a *TypeAlias) specNode() {}
func (m *Model) specNode()     {}
func (i *Input) specNode()     {}
func (a *Api) specNode()       {}

func (s *StringLiteral) typeNode()  {}
func (i *IntLiteral) typeNode()     {}
func (f *FloatLiteral) typeNode()   {}
func (t *TypeExpression) typeNode() {}
func (q *QualifiedIdent) typeNode() {}

// Stencil represents the entire parsed file. This will be used for any code generation.
type Stencil struct {
	Comments  []*Comment
	Namespace *Namespace
	Imports   []Import

	// Specs is imports, models, inputs, type aliases, enums, etc. in the order they were defined in the source file.
	Specs []SpecNode
}
