package emitter

import (
	"strings"

	"github.com/jimschubert/spray/ast"
)

// JoinPathSegments takes a slice of PathSegment and joins them into a single path string, prefixing parameter segments with a colon.
func JoinPathSegments(segments []ast.PathSegment) string {
	if len(segments) == 0 {
		return "/"
	}

	var parts []string
	for i, segment := range segments {
		if i == 0 {
			parts = append(parts, "") // Ensure the path starts with a slash
		}
		v := segment.Value
		if segment.IsParam {
			v = ":" + v
		}
		parts = append(parts, v)
	}
	return strings.Join(parts, "/")
}
