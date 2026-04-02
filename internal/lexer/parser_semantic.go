package lexer

import (
	"errors"
	"fmt"

	"github.com/jimschubert/spray/ast"
)

// semanticValidation validates the semantics of the AST, returning a composite error if any issues are found (e.g.
// duplicate declarations, imports defined after models, etc.)
func (p *Parser) semanticValidation(stencil *ast.Stencil) error {
	var err error

	// find duplicate spec names
	definedNames := make(map[string]bool)
	for _, spec := range stencil.Specs {
		var name string
		var pos ast.Position
		switch s := spec.(type) {
		case *ast.Model:
			name = s.Name.Value
			pos = s.Name.Position()
		case *ast.Input:
			name = s.Name.Value
			pos = s.Name.Position()
		case *ast.Enum:
			name = s.Name.Value
			pos = s.Name.Position()
		case *ast.TypeAlias:
			name = s.Name.Value
			pos = s.Name.Position()
		case *ast.Api:
			name = s.Name.Value
			pos = s.Name.Position()
		default:
			continue
		}

		if definedNames[name] {
			err = errors.Join(err, fmt.Errorf("duplicate definition: %q [line: %d, col: %d]", name, pos.Line, pos.Col))
		}
		definedNames[name] = true
	}

	for _, spec := range stencil.Specs {
		switch s := spec.(type) {
		case *ast.Model:
			for _, field := range s.Fields {
				for _, dec := range field.Decorators {
					if !isValidModelDecorator(dec.Name) {
						err = errors.Join(err, fmt.Errorf(
							"invalid decorator %q for model field %q [line: %d, col: %d] (model supports: @primary, @unique, @default, @updatedAt, @relation, @deprecated, @raw)",
							dec.Name,
							field.Name.Value,
							dec.Position().Line,
							dec.Position().Col,
						))
					}
				}
			}

		case *ast.Input:
			for _, field := range s.Fields {
				for _, dec := range field.Decorators {
					if !isValidInputDecorator(dec.Name) {
						err = errors.Join(err, fmt.Errorf(
							"invalid decorator %q for input field %q [line: %d, col: %d] (input supports: @default, @raw)",
							dec.Name, field.Name.Value,
							dec.Position().Line,
							dec.Position().Col,
						))
					}
				}
			}

		case *ast.Api:
			for _, dec := range s.ApiDecorators {
				if !isValidApiLevelDecorator(dec.Name) {
					err = errors.Join(err, fmt.Errorf(
						"invalid API-level decorator %q [line: %d, col: %d] (api supports: @version, @style)",
						dec.Name,
						dec.Position().Line,
						dec.Position().Col,
					))
				}
			}

			for _, route := range s.Routes {
				var decorators []ast.Decorator
				switch r := route.(type) {
				case *ast.RestRoute:
					decorators = r.Decorators
				case *ast.RpcRoute:
					decorators = r.Decorators
				case *ast.EventRoute:
					decorators = r.Decorators
				}

				for i := range decorators {
					dec := &decorators[i]
					if !isValidRouteDecorator(dec.Name) {
						err = errors.Join(err, fmt.Errorf(
							"invalid route decorator %q [line: %d, col: %d] (routes support: @body, @query, @errors, @summary, @tag, @version, @deprecated, @raw)",
							dec.Name,
							dec.Position().Line,
							dec.Position().Col,
						))
					}

					// only @errors allows multiple arguments (excluding @raw)
					if dec.Args.Len() > 1 && dec.Name != "errors" && dec.Name != "raw" {
						err = errors.Join(err, fmt.Errorf(
							"decorator %q does not support multiple arguments [line: %d, col: %d] (only @errors supports comma-separated values)",
							dec.Name,
							dec.Position().Line,
							dec.Position().Col,
						))
					}
				}
			}
		}
	}

	return err
}

func isValidModelDecorator(name string) bool {
	validDecorators := map[string]bool{
		"default":    true,
		"deprecated": true,
		"primary":    true,
		"raw":        true,
		"relation":   true,
		"unique":     true,
		"updatedAt":  true,
	}
	return validDecorators[name]
}

func isValidInputDecorator(name string) bool {
	validDecorators := map[string]bool{
		"default": true,
		"raw":     true,
	}
	return validDecorators[name]
}

func isValidApiLevelDecorator(name string) bool {
	validDecorators := map[string]bool{
		"style":   true,
		"version": true,
	}
	return validDecorators[name]
}

func isValidRouteDecorator(name string) bool {
	validDecorators := map[string]bool{
		"body":       true,
		"query":      true,
		"errors":     true,
		"summary":    true,
		"tag":        true,
		"version":    true,
		"deprecated": true,
		"raw":        true,
	}
	return validDecorators[name]
}
