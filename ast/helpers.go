package ast

// NameOf returns the simple name of any SpecNode.
func NameOf(node SpecNode) string {
	switch n := node.(type) {
	case *Model:
		return n.Name.Value
	case *Input:
		return n.Name.Value
	case *Enum:
		return n.Name.Value
	case *TypeAlias:
		return n.Name.Value
	case *Api:
		return n.Name.Value
	}
	return ""
}
