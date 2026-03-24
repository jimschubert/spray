package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/jimschubert/spray/ast"
	errs "github.com/jimschubert/spray/errors"
	"github.com/jimschubert/spray/internal/output"
	"github.com/jimschubert/spray/parser"
	"github.com/jimschubert/spray/resolver"
)

type GenerateCmd struct {
	Targets []string `short:"t" help:"Output targets" enum:"markdown" required:""`
	Out     string   `short:"o" help:"Output directory" default:"./out"`

	// Files is a slice of files. type is 'path', allowing expansion such as: validate ./api/* (works in terminal, not in some IDEs)
	Files []string `arg:"" help:".stencil files to validate" type:"path"`
}

func (g *GenerateCmd) Run() error {
	fmt.Println(output.Boldf("Generating %d...", len(g.Targets)))

	var parseErr error
	stencils := make([]*ast.Stencil, len(g.Files))
	for i, file := range g.Files {
		src, err := os.ReadFile(file)
		if err != nil {
			fmt.Println(output.Fail(file))
			return fmt.Errorf("reading %s: %w", file, err)
		}
		p, err := parser.New()
		if err != nil {
			return fmt.Errorf("creating parser: %w", err)
		}
		fmt.Println(output.Pass(file))

		s, err := p.Parse(string(src))
		if err != nil {
			parseErr = errors.Join(parseErr, fmt.Errorf("parsing %s: %w", file, err))
		}
		stencils[i] = s
	}

	if parseErr != nil {
		fmt.Println(output.Errorf("\nOne or more files failed to parse."))
		return parseErr
	}

	res := resolver.New(stencils...)
	resolved, err := res.Resolve()
	if err != nil {
		if joined, ok := errors.AsType[errs.JoinUnwrap](res.Error()); ok {
			fmt.Println(output.Errorf("\nOne or more files failed to resolve:"))
			errs.ForEachJoinError(joined, func(e error) {
				fmt.Println(output.Errorf(" - %s", e))
			})
		}
		return fmt.Errorf("resolving stencils: %w", err)
	}

	// TODO: Emitter
	println(resolved)

	return nil
}
