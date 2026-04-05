package main

import (
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"strings"

	internalerr "github.com/jimschubert/spray/errors"
	"github.com/jimschubert/spray/internal/formatter"
	"github.com/jimschubert/spray/internal/output"
	"github.com/jimschubert/spray/parser"
)

type FormatCmd struct {
	// Files is a slice of files. type is 'path', allowing expansion such as: spray format ./api/* (works in terminal, not in some IDEs)
	Files               []string `arg:"" help:".stencil files to format" type:"path"`
	AlignMembers        bool     `short:"a" long:"align-members"  default:"true" help:"align members vertically"`
	DecoratorsPerLine   int      `short:"d" long:"decorators-per-line" default:"3" help:"maximum number of decorators per line"`
	LinesBetweenSpecs   int      `short:"s" long:"lines-between-specs" default:"1" help:"number of lines between top-level specs"`
	LinesBetweenMembers int      `short:"m" long:"lines-between-members" default:"0" help:"number of lines between members within a spec"`
	IndentSize          int      `short:"i" long:"indent-size" default:"2" help:"number of spaces to use for indentation"`
	AllowCondensedSpecs bool     `short:"c" long:"allow-condensed-specs" help:"allow condensing specs with one field/member to a single line (e.g. \"input User { id: ID }\")"`
	MultilineImports    bool     `short:"l" long:"multiline-imports" help:"format imports with one spec's name per line"`
	DecoratorsStart     string   `long:"decorators-start" default:"same" enum:"same,next" help:"where to place decorators: same,next ('same'' line as member, 'next' line after member)"`
}

func (f *FormatCmd) Run() error {
	var eee error
	fmt.Println(output.Boldf("Formatting %d file(s)...", len(f.Files)))

	format, err := formatter.New(
		formatter.WithAlignMembers(f.AlignMembers),
		formatter.WithMaxDecoratorsPerLine(f.DecoratorsPerLine),
		formatter.WithLinesBetweenSpecs(f.LinesBetweenSpecs),
		formatter.WithLinesBetweenMembers(f.LinesBetweenMembers),
		formatter.WithIndentSize(f.IndentSize),
		formatter.WithAllowCondensedSpecs(f.AllowCondensedSpecs),
		formatter.WithMultilineImports(f.MultilineImports),
		formatter.WithDecoratorsStartPosition(formatter.DecoratorPositionFromString(f.DecoratorsStart)),
	)
	if err != nil {
		return fmt.Errorf("creating formatter: %w", err)
	}

	for _, path := range f.Files {
		src, err := os.ReadFile(path)
		if err != nil {
			// not a hard failure
			fmt.Println(output.Fail(path + " (could not read file)"))
			eee = errors.Join(eee, fmt.Errorf("reading %s: %w", path, err))
			continue
		}
		hash := f.hashContents(src)

		p, err := parser.New()
		if err != nil {
			// hard failure
			return fmt.Errorf("creating parser: %w", err)
		}

		stencil, err := p.Parse(string(src))
		if err != nil {
			fmt.Println(output.Fail(path))
			if wrapped, ok := errors.AsType[internalerr.JoinUnwrap](err); ok {
				msg := strings.Builder{}
				internalerr.ForEachJoinError(wrapped, func(e error) {
					fmt.Fprintf(&msg, "\t%s\n", e.Error())
				})
				eee = errors.Join(eee, fmt.Errorf("%s:\n%s", path, msg.String()))
			} else {
				eee = errors.Join(eee, fmt.Errorf("%s:\n\t%w", path, err))
			}
			continue
		}

		b, err := format.Format(stencil)
		if err != nil {
			// not a hard failure
			eee = errors.Join(eee, fmt.Errorf("formatting %s: %w", path, err))
			continue
		}

		formatted := string(b)
		if hash != f.hashContents([]byte(formatted)) {
			err = os.WriteFile(path, []byte(formatted), 0o644)
			if err != nil {
				fmt.Println(output.Fail(path + " (could not write file)"))
				eee = errors.Join(eee, fmt.Errorf("writing %s: %w", path, err))
				continue
			}
			fmt.Println(output.Pass(path + " (formatted)"))
		} else {
			fmt.Println(output.Plainf("%s (already formatted)", path))
		}
	}

	return eee
}

func (f *FormatCmd) hashContents(contents []byte) string {
	hasher := fnv.New64a()
	_, _ = hasher.Write(contents)
	return fmt.Sprintf("%x", hasher.Sum64())

}
