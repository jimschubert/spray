package ast

type Position struct {
	Line int
	Col  int
}

type Node interface {
	Position() Position
}

type QualifiedIdent struct {
	Pos   Position
	Parts []string
}

func (q *QualifiedIdent) Position() Position {
	return q.Pos
}

type StringLiteral struct {
	Pos   Position
	Value string
}

func (s *StringLiteral) Position() Position {
	return s.Pos
}

type Namespace struct {
	Pos  Position
	Name *QualifiedIdent
}

func (n *Namespace) Position() Position {
	return n.Pos
}

type Stencil struct {
	Pos       Position
	Namespace *Namespace
	// TODO
}
