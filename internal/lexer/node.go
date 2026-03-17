package lexer

type Pos int

func (p Pos) Position() Pos {
	return p
}
