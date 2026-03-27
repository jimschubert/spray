package markdown

import (
	"fmt"
	"strings"

	"github.com/jimschubert/spray/ast"
	"github.com/jimschubert/spray/emitter"
	"github.com/jimschubert/spray/resolver"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type markdownFile struct {
	filename string
	sb       strings.Builder
}

func (r markdownFile) Filename() string {
	return r.filename
}

func (r markdownFile) ContentType() emitter.ContentType {
	return emitter.ContentText
}

func (r markdownFile) Contents() []byte {
	return []byte(r.sb.String())
}

type emitMarkdown struct {
	schema resolver.ResolvedSchema
}

//nolint:unused
func (e *emitMarkdown) specNamespace(spec ast.SpecNode) string {
	ns, ok := e.schema.NamespaceOf(spec)
	if !ok {
		return "global"
	}
	return ns
}

func (e *emitMarkdown) EmitAll() ([]emitter.Output, error) {
	out := make([]emitter.Output, 0)
	generated := 0

	for _, stencil := range e.schema.Stencils {
		collected := emitter.CollectAll(stencil)
		sb := &strings.Builder{}

		if specs := collected[emitter.SpecApi]; len(specs) > 0 {
			sb.WriteString("## APIs\n\n")
			for i := range specs {
				api, ok := specs[i].(*ast.Api)
				if !ok {
					continue
				}
				e.emitApi(sb, api)
			}
			sb.WriteString("\n\n")
		}

		if specs := collected[emitter.SpecEnum]; len(specs) > 0 {
			sb.WriteString("## Enums\n\n")
			for i := range specs {
				enum, ok := specs[i].(*ast.Enum)
				if !ok {
					continue
				}
				e.emitEnum(sb, enum)
			}
		}

		if specs := collected[emitter.SpecModel]; len(specs) > 0 {
			sb.WriteString("## Models\n\n")
			for i := range specs {
				model, ok := specs[i].(*ast.Model)
				if !ok {
					continue
				}
				e.emitModel(sb, model)
			}
			sb.WriteString("\n\n")
		}

		if specs := collected[emitter.SpecInput]; len(specs) > 0 {
			sb.WriteString("## Inputs\n\n")
			for i := range specs {
				input, ok := specs[i].(*ast.Input)
				if !ok {
					continue
				}
				e.emitInput(sb, input)
			}

			sb.WriteString("\n\n")
		}

		var filename string
		if stencil.Namespace != nil {
			filename = e.cleanedFilename(stencil.Namespace.FullName())
		} else {
			generated += 1
			filename = fmt.Sprintf("output_%d.md", generated)
		}

		out = append(out, markdownFile{
			filename: filename,
			sb:       *sb,
		})
	}

	return out, nil
}

func (e *emitMarkdown) EmitOne(typ emitter.SpecType, name string) (emitter.Output, error) {
	collected := emitter.CollectAll(e.schema.Stencils...)
	specs, ok := collected[typ]
	if !ok {
		return nil, fmt.Errorf("no specs of type %d found", typ)
	}

	sb := &strings.Builder{}
	for _, spec := range specs {
		switch typ {
		case emitter.SpecApi:
			api, ok := spec.(*ast.Api)
			if !ok || api.Name.Value != name {
				continue
			}
			e.emitApi(sb, api)
		case emitter.SpecEnum:
			enum, ok := spec.(*ast.Enum)
			if !ok || enum.Name.Value != name {
				continue
			}
			e.emitEnum(sb, enum)
		case emitter.SpecModel:
			model, ok := spec.(*ast.Model)
			if !ok || model.Name.Value != name {
				continue
			}
			e.emitModel(sb, model)
		case emitter.SpecInput:
			input, ok := spec.(*ast.Input)
			if !ok || input.Name.Value != name {
				continue
			}
			e.emitInput(sb, input)
		}

		if sb.Len() > 0 {
			return markdownFile{
				filename: e.cleanedFilename(name),
				sb:       *sb,
			}, nil
		}
	}

	return nil, fmt.Errorf("%q not found", name)
}

func (e *emitMarkdown) emitInput(sb *strings.Builder, input *ast.Input) {
	fmt.Fprintf(sb, "### `%s`\n\n", input.Name.Value)
	if len(input.Fields) > 0 {
		sb.WriteString("| Field | Type | Constraints |\n")
		sb.WriteString("|---|---|---|\n")
		for i := range input.Fields {
			e.emitFieldRow(sb, &input.Fields[i])
		}
	}
	sb.WriteString("\n")
}

func (e *emitMarkdown) emitModel(sb *strings.Builder, model *ast.Model) {
	name := model.Name.Value
	if len(model.GenericParams) > 0 {
		params := make([]string, len(model.GenericParams))
		for i, p := range model.GenericParams {
			params[i] = p.Value
		}
		name += "<" + strings.Join(params, ", ") + ">"
	}
	fmt.Fprintf(sb, "### `%s`\n\n", name)
	if len(model.Fields) > 0 {
		sb.WriteString("| Field | Type | Constraints |\n")
		sb.WriteString("|---|---|---|\n")
		for i := range model.Fields {
			e.emitFieldRow(sb, &model.Fields[i])
		}
	}
	if len(model.Extensions) > 0 {
		sb.WriteString("\n**Extensions:**\n\n")
		for _, ext := range model.Extensions {
			fmt.Fprintf(sb, "* `@raw(%s)`\n", ext.Target.Value)
			for _, pair := range ext.Pairs {
				fmt.Fprintf(sb, "  * `%s`", pair.Key.Value)
				if pair.Value != nil {
					sb.WriteString(": ")
					e.emitTypeNode(sb, pair.Value, "`", "`")
				}
				sb.WriteString("\n")
			}
		}
	}
	sb.WriteString("\n")
}

func (e *emitMarkdown) emitApi(sb *strings.Builder, input *ast.Api) {
	title := input.Name.Value
	var version string
	for i := range input.ApiDecorators {
		directive := input.ApiDecorators[i]
		if directive.Name == "version" {
			for k := range directive.Args.Keys() {
				if strings.HasPrefix(k, "v") {
					version = k
				} else {
					version = "v" + k
				}
				break
			}
		}
	}

	fmt.Fprintf(sb, "### `%s`\n\n", title)

	var style string
	switch input.Style {
	case ast.StyleREST:
		style = "REST"
	case ast.StyleRPC:
		style = "RPC"
	case ast.StyleEvents:
		style = "EVENTS"
	default:
		style = "REST"
	}

	fmt.Fprintf(sb, "- **style**: %s\n", style)
	if version != "" {
		fmt.Fprintf(sb, "- **version**: %s\n", version)
	}
	sb.WriteString("\n\n")

	// Directives
	for i := range input.ApiDirectives {
		directive := input.ApiDirectives[i]
		first := true
		count := directive.Args.Len()
		fmt.Fprintf(sb, "- **%s**", directive.Name)
		directive.Args.All()(func(key string, value ast.TypeNode) bool {
			if first {
				sb.WriteString(":")
			}

			if count > 1 {
				if first {
					sb.WriteString(" (")
				} else {
					sb.WriteString(", ")
				}
			}

			first = false

			sb.WriteString(" ")
			sb.WriteString(key)
			if value == nil {
				sb.WriteString("\n")
				return true
			}

			e.emitTypeNode(sb, value, " `", "`\n")
			return true
		})
	}

	if len(input.ApiDirectives) > 0 {
		sb.WriteString("\n\n")
	}

	switch input.Style {
	case ast.StyleREST:
		e.emitRestTableHeader(sb)
		for i := range input.Routes {
			route := input.Routes[i].(*ast.RestRoute)
			e.emitwriteRestTableRow(sb, route)
		}
	case ast.StyleRPC:
		e.emitRpcTableHeader(sb)
		for i := range input.Routes {
			route := input.Routes[i].(*ast.RpcRoute)
			e.emitRpcTableRow(sb, route)
		}
	case ast.StyleEvents:
		e.emitEventsTableHeader(sb)
		for i := range input.Routes {
			route := input.Routes[i].(*ast.EventRoute)
			e.emitEventsTableRow(sb, route)
		}
	default:
		e.emitRestTableHeader(sb)
		for i := range input.Routes {
			route := input.Routes[i].(*ast.RestRoute)
			e.emitwriteRestTableRow(sb, route)
		}
	}

	sb.WriteString("\n")
}

func (e *emitMarkdown) emitFieldRow(sb *strings.Builder, field *ast.Field) {
	constraints := make([]string, 0, len(field.Decorators))
	for _, d := range field.Decorators {
		constraints = append(constraints, d.String())
	}
	fmt.Fprintf(sb, "| %s | `%s` | %s |\n",
		field.Name.Value,
		field.Type.String(),
		strings.Join(constraints, ", "),
	)
}

func (e *emitMarkdown) emitEnum(sb *strings.Builder, input *ast.Enum) {
	fmt.Fprintf(sb, "### `%s`\n\n", input.Name.Value)
	sb.WriteString("**Values:**\n\n")
	for i := range input.Elements {
		fmt.Fprintf(sb, "* %s\n", input.Elements[i].Value)
	}
	sb.WriteString("\n")
}

func (e *emitMarkdown) cleanedFilename(target string) string {
	sb := strings.Builder{}
	for i := range target {
		if target[i] == '.' {
			sb.WriteRune('_')
			continue
		}

		if target[i] >= 'A' && target[i] <= 'Z' {
			if i > 0 {
				sb.WriteString("_")
			}
			sb.WriteString(strings.ToLower(string(target[i])))
		} else if target[i] >= 'a' && target[i] <= 'z' || target[i] >= '0' && target[i] <= '9' {
			sb.WriteByte(target[i])
		} else {
			sb.WriteRune('_')
		}
	}
	return sb.String() + ".md"
}

func (e *emitMarkdown) emitRestTableHeader(sb *strings.Builder) {
	sb.WriteString("| Method | Path | Request | Response | Notes |\n|---|---|---|---|---|\n")
}

func (e *emitMarkdown) emitwriteRestTableRow(sb *strings.Builder, route *ast.RestRoute) {
	body := e.findBodyDecorator(route.Decorators)
	var bodyName string
	if body != nil {
		for s := range body.Args.Keys() {
			bodyName = s
			break
		}
	}

	notes := e.buildNotes(route.Decorators, func(d ast.Decorator) bool { return d.Name == "body" })

	resp := route.Return.String()
	fmt.Fprintf(sb, "| %s | %s | %s | %s | %s |\n",
		route.Method,
		emitter.JoinPathSegments(route.Path),
		bodyName,
		resp,
		notes,
	)
}

func (e *emitMarkdown) emitRpcTableHeader(sb *strings.Builder) {
	sb.WriteString("| Streaming | Request | Response | Notes |\n|---|---|---|---|\n")
}

func (e *emitMarkdown) emitRpcTableRow(sb *strings.Builder, route *ast.RpcRoute) {
	body := e.findBodyDecorator(route.Decorators)
	var bodyName string
	if body != nil {
		for s := range body.Args.Keys() {
			bodyName = s
			break
		}
	}

	notes := e.buildNotes(route.Decorators, func(d ast.Decorator) bool { return d.Name == "body" })

	resp := route.Return.String()
	fmt.Fprintf(sb, "| %v | `%s` | `%s` | `%s` |\n",
		route.Streaming,
		bodyName,
		resp,
		notes,
	)
}

func (e *emitMarkdown) emitEventsTableHeader(sb *strings.Builder) {
	sb.WriteString("| Event | Payload | Notes |\n|---|---|---|\n|---|---|---|\n")
}

func (e *emitMarkdown) emitEventsTableRow(sb *strings.Builder, route *ast.EventRoute) {
	body := e.findBodyDecorator(route.Decorators)
	var bodyName string
	if body != nil {
		for s := range body.Args.Keys() {
			bodyName = s
			break
		}
	}

	notes := e.buildNotes(route.Decorators, func(d ast.Decorator) bool { return d.Name == "body" })

	resp := route.Event.String()
	fmt.Fprintf(sb, "| `%s` | `%s` | `%s` |\n",
		bodyName,
		resp,
		notes,
	)
}

func (e *emitMarkdown) emitTypeNode(sb *strings.Builder, node ast.TypeNode, prefix string, suffix string) {
	if node == nil {
		return
	}

	if suffix != "" {
		sb.WriteString(prefix)
	}

	switch v := node.(type) {
	case *ast.StringLiteral:
		// parser will handle quoted and unquoted strings
		sb.WriteString(v.Value)
	case *ast.IntLiteral:
		fmt.Fprintf(sb, "%d", v.Value)
	case *ast.FloatLiteral:
		fmt.Fprintf(sb, "%g", v.Value)
	case *ast.TypeExpression:
		sb.WriteString(v.String())
	case fmt.Stringer:
		sb.WriteString(v.String())
	default:
		sb.WriteString("unknown")
	}
	if suffix != "" {
		sb.WriteString(suffix)
	}
}

func (e *emitMarkdown) buildNotes(decorators []ast.Decorator, exclude func(d ast.Decorator) bool) string {
	notes := strings.Builder{}

	titleCaser := cases.Title(language.Und, cases.NoLower)

	// build notes from decorators
	for i, decorator := range decorators {
		if i > 0 {
			notes.WriteString(" · ")
		}

		if exclude != nil && exclude(decorator) {
			continue
		}

		// pascal case the key, csv the values
		notes.WriteString(fmt.Sprintf("%s: ", titleCaser.String(decorator.Name)))
		count := decorator.Args.Len()
		first := true
		decorator.Args.All()(func(key string, value ast.TypeNode) bool {
			if count > 1 {
				if first {
					notes.WriteString("(")
					first = false
				} else {
					notes.WriteString(", ")
				}
			}

			notes.WriteString(key)
			e.emitTypeNode(&notes, value, " ", "")

			return true
		})
		if count > 1 {
			notes.WriteString(")")
		}
	}

	return notes.String()
}

func (e *emitMarkdown) findBodyDecorator(decorators []ast.Decorator) *ast.Decorator {
	for _, d := range decorators {
		if d.Name == "body" {
			return &d
		}
	}
	return nil
}

// New creates a new Markdown emitter with the given resolved schema.
func New(schema *resolver.ResolvedSchema) (emitter.Emitter, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema cannot be nil")
	}

	return &emitMarkdown{
		schema: *schema,
	}, nil
}
