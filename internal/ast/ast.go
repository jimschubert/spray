package ast

// Position represents the line and column of a node in the source file.
type Position struct {
	Line int
	Col  int
}

// Node is the base interface for all AST nodes.
type Node interface {
	Position() Position
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
	result := ""
	for i, part := range q.Parts {
		if i > 0 {
			result += "."
		}
		result += part
	}
	return result
}

// StringLiteral represents a string literal in the source file.
type StringLiteral struct {
	Pos   Position
	Value string
}

func (s *StringLiteral) Position() Position {
	return s.Pos
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

// Stencil represents the entire parsed file. This will be used for any code generation.
type Stencil struct {
	Comments  []*Comment
	Namespace *Namespace
	// TODO
}
