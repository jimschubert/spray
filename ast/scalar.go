package ast

// IsBuiltinScalar returns true if the given name is a built-in scalar type.
func IsBuiltinScalar(name string) bool {
	switch name {
	case "string", "int", "float", "boolean", "uuid", "timestamp", "date", "any", "void":
		return true
	}
	return false
}
