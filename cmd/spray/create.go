package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jimschubert/spray/ast"
	"github.com/jimschubert/spray/emitter"
	"github.com/jimschubert/spray/emitter/jsonschema"
	"github.com/jimschubert/spray/emitter/markdown"
	"github.com/jimschubert/spray/emitter/mermaid"
	errs "github.com/jimschubert/spray/errors"
	"github.com/jimschubert/spray/internal/output"
	"github.com/jimschubert/spray/internal/plug"
	"github.com/jimschubert/spray/parser"
	"github.com/jimschubert/spray/resolver"
)

// createBase holds flags and logic shared by all create subcommands.
type createBase struct {
	Out string `short:"o" help:"Output directory" default:"./out"`

	// Files is a slice of files. type is 'path', allowing expansion such as: spray create markdown ./api/*
	Files []string `arg:"" help:".stencil files to compile" type:"path"`
}

// createOutputDir ensures the output directory exists.
func (b *createBase) createOutputDir() error {
	return os.MkdirAll(b.Out, 0o755)
}

// writeOutputFiles writes all output files to the specified directory.
func (b *createBase) writeOutputFiles(outputs []emitter.Output) error {
	for _, out := range outputs {
		dest := filepath.Join(b.Out, out.Filename())
		if err := os.WriteFile(dest, out.Contents(), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", dest, err)
		}

		fmt.Println(output.Pass("wrote: " + b.relativePath(dest)))
	}
	return nil
}

// invoke runs the emitter and handles writing output files and errors.
func (b *createBase) invoke(emitterName string, targetEmitter emitter.Emitter) error {
	outputs, err := targetEmitter.EmitAll()
	if err != nil {
		return fmt.Errorf("emitting %s: %w", emitterName, err)
	}

	if err := b.createOutputDir(); err != nil {
		return fmt.Errorf("creating output directory %s: %w", b.Out, err)
	}

	if err := b.writeOutputFiles(outputs); err != nil {
		return err
	}

	return nil
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
		fmt.Println(output.Pass("loaded: " + display))
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
	Markdown   CreateMarkdownCmd   `cmd:"" help:"Markdown documentation"`
	JsonSchema CreateJsonSchemaCmd `cmd:"" help:"JSON Schema"`
	Mermaid    CreateMermaidCmd    `cmd:"" help:"Mermaid ERD diagram"`
	Ext        CreateExtCmd        `cmd:"" help:"External emitter plugin"`
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

	targetEmitter, err := markdown.New(resolved)
	if err != nil {
		return fmt.Errorf("creating markdown emitter: %w", err)
	}

	return c.invoke("markdown", targetEmitter)
}

// CreateJsonSchemaCmd generates JSON Schema output from .stencil files.
type CreateJsonSchemaCmd struct {
	Draft         string `help:"JSON Schema draft version to target" enum:"2020-12,2019-09" default:"2020-12"`
	IDPrefix      string `help:"Prefix for all $id values in emitted schemas"`
	RefProcessing string `help:"Processing strategy for emitting $ref (e.g. inline, file, or id)" enum:"inline,file,id" default:"file"`
	createBase
}

func (c *CreateJsonSchemaCmd) Run() error {
	fmt.Println(output.Boldf("Generating JSON Schema from %d file(s)...", len(c.Files)))

	resolved, err := c.resolve()
	if err != nil {
		return err
	}

	targetEmitter, err := jsonschema.New(resolved,
		jsonschema.WithDraft(c.Draft),
		jsonschema.WithIDPrefix(c.IDPrefix),
		jsonschema.WithRefProcessing(c.RefProcessing),
	)

	if err != nil {
		return fmt.Errorf("creating JSON Schema emitter: %w", err)
	}

	return c.invoke("json schema", targetEmitter)
}

// CreateMermaidCmd generates Mermaid ERD diagram output from .stencil files.
type CreateMermaidCmd struct {
	createBase
}

func (c *CreateMermaidCmd) Run() error {
	fmt.Println(output.Boldf("Generating Mermaid ERD from %d file(s)...", len(c.Files)))

	resolved, err := c.resolve()
	if err != nil {
		return err
	}

	targetEmitter, err := mermaid.New(resolved)
	if err != nil {
		return fmt.Errorf("creating Mermaid emitter: %w", err)
	}

	return c.invoke("mermaid", targetEmitter)
}

// CreateExtCmd invokes an external emitter plugin by name. Name must be in the format `spray-emitter-<name>`.
// Looks first in ~/.spray/plugins/, then in $PATH.
//
// The plugin system emits a schema as JSON on the plugin's stdin and expects a JSON response to stdout.
type CreateExtCmd struct {
	Ext struct {
		Name string `arg:"" help:"External emitter name (matches spray-emitter-<name> on PATH)"`
	} `embed:""`
	createBase
}

func (c *CreateExtCmd) Run() error {
	fmt.Println(output.Boldf("Generating %q output from %d file(s)...", c.Ext.Name, len(c.Files)))
	bin, err := plug.Find(c.Ext.Name)
	if err != nil {
		fmt.Println(output.Errorf("Plugin %q not found.", c.Ext.Name))
		fmt.Println(output.Errorf("Spray searches for plugins named spray-emitter-%s in:", c.Ext.Name))
		for _, dir := range plug.LookupDirs() {
			fmt.Println(output.Errorf(" -  %s", dir))
		}
		return err
	}

	fmt.Println(output.Pass("plugin: " + bin))

	//nolint:staticcheck
	resolved, err := c.createBase.resolve()
	if err != nil {
		return err
	}

	targetEmitter, err := plug.New(c.Ext.Name, resolved)
	if err != nil {
		return fmt.Errorf("initializing plugin %q: %w", c.Ext.Name, err)
	}

	//nolint:staticcheck
	return c.createBase.invoke(c.Ext.Name, targetEmitter)
}
