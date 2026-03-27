package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jimschubert/spray/ast"
	"github.com/jimschubert/spray/emitter/markdown"
	errs "github.com/jimschubert/spray/errors"
	"github.com/jimschubert/spray/internal/output"
	"github.com/jimschubert/spray/parser"
	"github.com/jimschubert/spray/resolver"
)

// createBase holds flags and logic shared by all create subcommands.
type createBase struct {
	Out string `short:"o" help:"Output directory" default:"./out"`

	// Files is a slice of files. type is 'path', allowing expansion such as: spray create markdown ./api/*
	Files []string `arg:"" help:".stencil files to compile" type:"path"`
}

// relativePath returns path relative to the working directory only if it is nested, otherwise returns original.
func (b *createBase) relativePath(path string) string {
	wd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(wd, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return rel
}

// resolve parses and resolves all input files, returning the resolved schema.
func (b *createBase) resolve() (*resolver.ResolvedSchema, error) {
	var parseErr error
	stencils := make([]*ast.Stencil, len(b.Files))
	for i, file := range b.Files {
		display := b.relativePath(file)
		src, err := os.ReadFile(file)
		if err != nil {
			fmt.Println(output.Fail(display))
			return nil, fmt.Errorf("reading %s: %w", file, err)
		}
		p, err := parser.New()
		if err != nil {
			return nil, fmt.Errorf("creating parser: %w", err)
		}
		s, err := p.Parse(string(src))
		if err != nil {
			parseErr = errors.Join(parseErr, fmt.Errorf("parsing %s: %w", file, err))
			fmt.Println(output.Fail(display))
			continue
		}
		fmt.Println(output.Pass(display))
		stencils[i] = s
	}

	if parseErr != nil {
		fmt.Println(output.Errorf("\nOne or more files failed to parse."))
		return nil, parseErr
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
		return nil, fmt.Errorf("resolving stencils: %w", err)
	}

	return resolved, nil
}

// CreateCmd is the top-level `create` command; each subcommand targets a specific output format.
type CreateCmd struct {
	Markdown CreateMarkdownCmd `cmd:"" help:"Markdown documentation"`
}

// CreateMarkdownCmd generates Markdown output from .stencil files.
type CreateMarkdownCmd struct {
	createBase
	// shared functionality exists in createBase, and is accessible to future commands, allowing each subcommand to have
	// its own set of options.
	// some ideas for markdown-based options:
	// CommonMark bool `help:"Use CommonMark extensions for tables, etc."`
	// FrontMatter bool `help:"Emit front matter in each file"`
}

func (c *CreateMarkdownCmd) Run() error {
	fmt.Println(output.Boldf("Generating Markdown from %d file(s)...", len(c.Files)))

	resolved, err := c.resolve()
	if err != nil {
		return err
	}

	emitter, err := markdown.New(resolved)
	if err != nil {
		return fmt.Errorf("creating markdown emitter: %w", err)
	}

	outputs, err := emitter.EmitAll()
	if err != nil {
		return fmt.Errorf("emitting markdown: %w", err)
	}

	if err := os.MkdirAll(c.Out, 0o755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", c.Out, err)
	}

	for _, out := range outputs {
		dest := filepath.Join(c.Out, out.Filename())
		if err := os.WriteFile(dest, out.Contents(), 0o644); err != nil {
			fmt.Println(output.Fail(c.relativePath(dest)))
			return fmt.Errorf("writing %s: %w", dest, err)
		}
		fmt.Println(output.Pass(c.relativePath(dest)))
	}

	return nil
}
