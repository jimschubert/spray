package formatter

type Options func(*Formatter)

func limit(n, min, max int) int {
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

// WithMaxDecoratorsPerLine sets the maximum number of decorators per line. Minimum is 1. Maximum is 10.
func WithMaxDecoratorsPerLine(n int) Options {
	return func(f *Formatter) {
		f.maxDecoratorPerLine = limit(n, 1, 10)
	}
}

// WithLinesBetweenSpecs sets the number of lines between specs. Minimum is 0. Maximum is 2.
func WithLinesBetweenSpecs(n int) Options {
	return func(f *Formatter) {
		f.linesBetweenSpecs = limit(n, 0, 2)
	}
}

// WithLinesBetweenMembers sets the number of lines between members. Minimum is 0. Maximum is 2.
func WithLinesBetweenMembers(n int) Options {
	return func(f *Formatter) {
		f.linesBetweenMembers = limit(n, 0, 2)
	}
}

// WithIndentSize sets the number of spaces to use for indentation. Minimum is 0 (no indentation), maximum is 8.
func WithIndentSize(n int) Options {
	return func(f *Formatter) {
		f.indentSize = limit(n, 0, 8)
	}
}

// WithAlignMembers sets whether to align members vertically.
func WithAlignMembers(align bool) Options {
	return func(f *Formatter) {
		f.alignMembers = align
	}
}

// WithAllowCondensedSpecs sets whether to allow condensed specs (e.g. "type User { id: ID }" instead of "type User {\n  id: ID\n}").
func WithAllowCondensedSpecs(allow bool) Options {
	return func(f *Formatter) {
		f.allowCondensedSpecs = allow
	}
}

// WithMultilineImports sets whether to format imports with one spec per line.
func WithMultilineImports(multiline bool) Options {
	return func(f *Formatter) {
		f.multilineImports = multiline
	}
}

// WithDecoratorsStartPosition sets the position of the first decorator when formatting.
// The default is DecoratorPositionSameLine, which means the first decorator will be on same line as the field or definition.
// If set to DecoratorPositionNextLine, the first decorator will be on the following line, and members will have an additional newline after decorators.
func WithDecoratorsStartPosition(pos DecoratorPosition) Options {
	return func(f *Formatter) {
		f.decoratorsStartPosition = pos
	}
}
