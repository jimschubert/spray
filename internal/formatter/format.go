package formatter

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/jimschubert/spray/ast"
)

type DecoratorPosition int

const (
	DecoratorPositionSameLine DecoratorPosition = iota
	DecoratorPositionNextLine
)

func DecoratorPositionFromString(s string) DecoratorPosition {
	switch s {
	case "same":
		return DecoratorPositionSameLine
	case "next":
		return DecoratorPositionNextLine
	default:
		return DecoratorPositionSameLine
	}
}

type Formatter struct {
	maxDecoratorPerLine     int
	linesBetweenSpecs       int
	linesBetweenMembers     int
	indentSize              int
	alignMembers            bool
	allowCondensedSpecs     bool
	multilineImports        bool
	decoratorsStartPosition DecoratorPosition
}

func (f *Formatter) Format(stencil *ast.Stencil) ([]byte, error) {
	buf := &bytes.Buffer{}

	if stencil.Namespace != nil {
		ns := *stencil.Namespace
		// write any comments preceding the namespace declaration
		if stencil.Comments != nil {
			for _, comment := range stencil.Comments {
				if comment.Pos.Line < ns.Pos.Line {
					fmt.Fprintf(buf, "%s\n", comment.Text)
				}
			}
		}
		fmt.Fprintf(buf, "namespace %s\n", ns.Name.String())
		f.writeLinesBetweenSpecs(buf)
	}

	if stencil.Imports != nil {
		for _, imp := range stencil.Imports {
			f.formatImport(buf, imp)
			buf.WriteByte('\n')
		}

		f.writeLinesBetweenSpecs(buf)
	}

	for i, spec := range stencil.Specs {
		if i > 0 {
			f.writeLinesBetweenSpecs(buf)
		}
		switch s := spec.(type) {
		case *ast.Enum:
			f.formatEnum(buf, s)
		case *ast.TypeAlias:
			f.formatTypeAlias(buf, s)
		case *ast.Model:
			f.formatModel(buf, s)
		case *ast.Input:
			f.formatInput(buf, s)
		case *ast.Api:
			f.formatApi(buf, s)
		default:
			return nil, fmt.Errorf("unknown spec type: %T", spec)
		}
		buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}

func (f *Formatter) formatImport(buf *bytes.Buffer, imp ast.Import) {
	if imp.HeadComment != nil {
		buf.WriteString(imp.HeadComment.String())
	}
	if !f.multilineImports {
		buf.WriteString(imp.String())
		if imp.LineComment != nil {
			buf.WriteString(imp.LineComment.String())
		}
		return
	}

	fmt.Fprintf(buf, "import %s {\n", imp.Path.String())
	for idx, name := range imp.Names {
		f.indent(buf, 1)
		buf.WriteString(name.Value)
		if idx < len(imp.Names)-1 {
			buf.WriteString(",\n")
		}
	}
	fmt.Fprintf(buf, "}%s", imp.LineComment.String())
}

func (f *Formatter) formatEnum(buf *bytes.Buffer, s *ast.Enum) {
	if s.HeadComment != nil {
		buf.WriteString(s.HeadComment.String())
	}
	fmt.Fprintf(buf, "enum %s {", s.Name.Value)
	if f.allowCondensedSpecs && len(s.Elements) == 1 {
		member := s.Elements[0]
		fmt.Fprintf(buf, " %s ", member.Value)
	} else {
		buf.WriteByte('\n')
		for i, member := range s.Elements {
			if i > 0 {
				f.writeLinesBetweenMembers(buf)
			}
			f.indent(buf, 1)
			fmt.Fprintf(buf, "%s\n", member.Value)
		}
		f.ensureSingleTrailingNewline(buf)
	}
	buf.WriteString("}")
}

func (f *Formatter) formatApi(buf *bytes.Buffer, s *ast.Api) {
	fmt.Fprintf(buf, "api %s", s.Name.Value)
	if s.ApiDecorators != nil {
		for _, decorator := range s.ApiDecorators {
			buf.WriteByte(' ')
			buf.WriteString(decorator.String())
		}
	}

	buf.WriteString(" {\n")

	if s.ApiDirectives != nil {
		for i, directive := range s.ApiDirectives {
			f.indent(buf, 1)
			buf.WriteString(directive.String())
			buf.WriteByte('\n')

			if i == len(s.ApiDirectives)-1 && f.linesBetweenMembers == 0 {
				buf.WriteByte('\n')
			}
		}
		f.writeLinesBetweenMembers(buf)
	}

	maxOperationLen := 0
	maxNameLen := 0
	maxReturn := 0
	if f.alignMembers {
		switch s.Style {
		case ast.StyleREST:
			for _, route := range s.Routes {
				rest := route.(*ast.RestRoute)
				maxOperationLen = max(len(rest.Method), maxOperationLen)
				maxNameLen = max(len(rest.Path.String()), maxNameLen)
				maxReturn = max(len(rest.Return.String()), maxReturn)
			}
		case ast.StyleRPC:
			for _, route := range s.Routes {
				rpc := route.(*ast.RpcRoute)
				if rpc.Streaming {
					// rpc stream
					maxOperationLen = max(10, maxOperationLen) // length of "rpc stream"
				} else {
					maxOperationLen = max(3, maxOperationLen) // length of "rpc"
				}
				maxNameLen = max(len(rpc.Name.Value)+2+len(rpc.Input.String()), maxNameLen)
				maxReturn = max(len(rpc.Return.String()), maxReturn)
			}
		case ast.StyleEvents:
			for _, route := range s.Routes {
				evt := route.(*ast.EventRoute)
				if evt.Direction == ast.EventSubscribe {
					maxOperationLen = max(9, maxOperationLen) // length of "subscribe"
				} else {
					maxOperationLen = max(7, maxOperationLen) // length of "publish"
				}
				maxNameLen = max(len(evt.Name.Value), maxNameLen)
				maxReturn = max(len(evt.Event.String()), maxReturn)
			}
		}
	}

	for i, route := range s.Routes {
		if i > 0 && i < len(s.Routes)-1 {
			f.writeLinesBetweenMembers(buf)
		}

		switch r := route.(type) {
		case *ast.RestRoute:
			f.indent(buf, 1)
			f.emitPadded(buf, r.Method, maxOperationLen)
			buf.WriteByte(' ')
			f.emitPadded(buf, r.Path.String(), maxNameLen)
			buf.WriteString(" -> ")
			f.emitPadded(buf, r.Return.String(), maxReturn)
			if len(r.Decorators) > 0 {
				buf.WriteByte(' ')
			}
			f.writeDecorators(buf, r.Decorators, 2, 0)
			f.ensureNewline(buf)
		case *ast.RpcRoute:
			operation := "rpc"
			if r.Streaming {
				operation += " stream"
			}
			f.indent(buf, 1)
			f.emitPadded(buf, operation, maxOperationLen)
			buf.WriteByte(' ')
			f.emitPadded(buf, r.Name.Value+"("+r.Input.String()+")", maxNameLen)
			buf.WriteString(" -> ")
			f.emitPadded(buf, r.Return.String(), maxReturn)
			if len(r.Decorators) > 0 {
				buf.WriteByte(' ')
			}
			f.writeDecorators(buf, r.Decorators, 2, 0)
			f.ensureNewline(buf)
		case *ast.EventRoute:
			direction := "publish"
			if r.Direction == ast.EventSubscribe {
				direction = "subscribe"
			}
			f.indent(buf, 1)
			f.emitPadded(buf, direction, maxOperationLen)
			buf.WriteByte(' ')
			f.emitPadded(buf, r.Name.Value, maxNameLen)
			buf.WriteString(" -> ")
			f.emitPadded(buf, r.Event.String(), maxReturn)
			if len(r.Decorators) > 0 {
				buf.WriteByte(' ')
			}
			f.writeDecorators(buf, r.Decorators, 2, 0)
			f.ensureNewline(buf)
		default:
			panic(fmt.Sprintf("unknown route type: %T", route))
		}
	}

	f.ensureSingleTrailingNewline(buf)
	buf.WriteByte('}')
}

//goland:noinspection DuplicatedCode
func (f *Formatter) formatInput(buf *bytes.Buffer, s *ast.Input) {
	fmt.Fprintf(buf, "input %s {", s.Name.Value)

	if f.allowCondensedSpecs && len(s.Fields) == 1 {
		member := s.Fields[0]
		fmt.Fprintf(buf, " %s: %s ", member.Name.Value, member.Type.String())
		f.writeDecorators(buf, member.Decorators, 2, 0)
		buf.WriteByte('}')
		return
	}

	buf.WriteByte('\n')

	maxNameLen := 0
	maxTypeLen := 0
	if f.alignMembers {
		for _, member := range s.Fields {
			maxNameLen = max(maxNameLen, len(member.Name.Value))
			maxTypeLen = max(maxTypeLen, len(member.Type.String()))
		}
	}

	for i, member := range s.Fields {
		if i > 0 {
			f.writeLinesBetweenMembers(buf)
		}

		f.indent(buf, 1)
		hangingIndent := 0
		if f.alignMembers {
			hangingIndent = maxNameLen + 2
		}

		f.emitPadded(buf, member.Name.Value, maxNameLen)
		buf.WriteString(": ")
		f.emitPadded(buf, member.Type.String(), maxTypeLen)

		if member.LineComment != nil {
			fmt.Fprintf(buf, " %s", member.LineComment.String())
			buf.WriteByte('\n')
		}

		if len(member.Decorators) > 0 {
			f.writeDecorators(buf, member.Decorators, 2, hangingIndent)
			f.ensureNewline(buf)
		}

		if member.LineComment == nil {
			// decorators will "follow" if there's a line comment and start on same line if there's not.
			// if the last op was a field with no comment or decorator, we need a newline if it doesn't exist
			f.ensureNewline(buf)
		}
	}
	f.ensureSingleTrailingNewline(buf)
	buf.WriteByte('}')
}

//goland:noinspection DuplicatedCode
func (f *Formatter) formatModel(buf *bytes.Buffer, s *ast.Model) {
	fmt.Fprintf(buf, "model %s", s.Name.Value)
	if s.GenericParams != nil {
		parts := make([]string, len(s.GenericParams))
		for i, param := range s.GenericParams {
			parts[i] = param.Value
		}
		fmt.Fprintf(buf, "<%s> ", strings.Join(parts, ", "))
	} else {
		buf.WriteByte(' ')
	}
	buf.WriteByte('{')

	if f.allowCondensedSpecs && len(s.Fields) == 1 {
		member := s.Fields[0]
		fmt.Fprintf(buf, " %s: %s ", member.Name.Value, member.Type.String())
		f.writeDecorators(buf, member.Decorators, 2, 0)
		// if buf ends in a newline, remove it since we're condensing to a single line
		if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] == '\n' {
			buf.Truncate(buf.Len() - 1)
		}
		buf.WriteByte('}')
		return
	}

	buf.WriteByte('\n')

	maxNameLen := 0
	maxTypeLen := 0
	if f.alignMembers {
		for _, member := range s.Fields {
			maxNameLen = max(maxNameLen, len(member.Name.Value))
			maxTypeLen = max(maxTypeLen, len(member.Type.String()))
		}
	}

	for i, member := range s.Fields {
		if i > 0 {
			f.writeLinesBetweenMembers(buf)
		}

		f.indent(buf, 1)
		hangingIndent := 0
		if f.alignMembers {
			hangingIndent = maxNameLen + 2
		}

		f.emitPadded(buf, member.Name.Value, maxNameLen)
		buf.WriteString(": ")
		f.emitPadded(buf, member.Type.String(), maxTypeLen)

		if member.LineComment != nil {
			fmt.Fprintf(buf, " %s", member.LineComment.String())
			buf.WriteByte('\n')
		}

		if len(member.Decorators) > 0 {
			f.writeDecorators(buf, member.Decorators, 2, hangingIndent)
		}

		if member.LineComment == nil {
			// decorators will "follow" if there's a line comment and start on same line if there's not.
			// if the last op was a field with no comment or decorator, we need a newline if it doesn't exist
			f.ensureNewline(buf)
		}
	}
	f.ensureSingleTrailingNewline(buf)
	buf.WriteByte('}')
}

func (f *Formatter) formatTypeAlias(buf *bytes.Buffer, s *ast.TypeAlias) {
	if s.HeadComment != nil {
		buf.WriteString(s.HeadComment.String())
	}
	fmt.Fprintf(buf, "type %s = %s%s", s.Name.Value, s.Type.String(), s.LineComment.String())
}

func (f *Formatter) spaces(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("%*s", n, "")
}

func (f *Formatter) emitPadded(b *bytes.Buffer, val string, padding int) {
	if padding <= 0 {
		b.WriteString(val)
		return
	}
	fmt.Fprintf(b, "%-*s", padding, val)
}

func (f *Formatter) writeLinesBetweenSpecs(buf *bytes.Buffer) {
	for i := 0; i < f.linesBetweenSpecs; i++ {
		buf.WriteByte('\n')
	}
}

func (f *Formatter) writeLinesBetweenMembers(buf *bytes.Buffer) {
	for i := 0; i < f.linesBetweenMembers; i++ {
		buf.WriteByte('\n')
	}
}

func (f *Formatter) indent(buf *bytes.Buffer, level int) {
	for i := 0; i < level*f.indentSize; i++ {
		buf.WriteByte(' ')
	}
}

func (f *Formatter) writeDecorators(buf *bytes.Buffer, decorators []ast.Decorator, level int, padding int) {
	wrapped := false
	for i, decorator := range decorators {
		// level indicates indentation; level=0 means we're not indenting (e.g. api-level decorators).
		if (i > 0 && level > 0 && i%f.maxDecoratorPerLine == 0) || (i == 0 && f.decoratorsStartPosition == DecoratorPositionNextLine) {
			buf.WriteByte('\n')
			f.indent(buf, level)
			if padding > 0 {
				// for multi-line decorators with aligned members, add extra padding to align with the type
				buf.WriteString(f.spaces(padding))
			}

			buf.WriteString(decorator.String())

			wrapped = true
		} else {
			buf.WriteByte(' ')
			buf.WriteString(decorator.String())
		}

		if wrapped && i == len(decorators)-1 {
			// if we've wrapped decorators to multiple lines, add an extra newline after the last one to separate from the field/member
			buf.WriteByte('\n')
			buf.WriteByte('\n')
		}
	}

	if len(decorators) > 0 {
		f.ensureNewline(buf)
	}
}

// ensureNewline makes sure the buf has a newline at the end, adding if it doesn't.
func (f *Formatter) ensureNewline(buf *bytes.Buffer) {
	if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] != '\n' {
		buf.WriteByte('\n')
	}
}

// ensureSingleTrailingNewline is a special case where we want to avoid lots of space before a closing brace.
func (f *Formatter) ensureSingleTrailingNewline(buf *bytes.Buffer) {
	if buf.Len() == 0 {
		buf.WriteByte('\n')
		return
	}

	b := buf.Bytes()
	// trim trailing newlines
	end := len(b)
	for end > 0 && b[end-1] == '\n' {
		end--
	}
	buf.Truncate(end)
	buf.WriteByte('\n')
}

func New(opts ...Options) (*Formatter, error) {
	f := &Formatter{
		maxDecoratorPerLine: 3,
		linesBetweenSpecs:   1,
		linesBetweenMembers: 0,
		indentSize:          2,
		alignMembers:        true,
		allowCondensedSpecs: true,
		multilineImports:    false,
	}

	for _, opt := range opts {
		opt(f)
	}

	return f, nil
}
