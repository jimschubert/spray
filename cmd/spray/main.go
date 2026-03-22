package main

import (
	"errors"
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/jimschubert/spray/internal/output"
)

var (
	programName = "spray"
	version     = "dev"
	commit      = "unknown SHA"
)

type kongExitCoderError interface {
	kong.ExitCoder
	error
}

var CLI struct {
	Validate ValidateCmd      `cmd:"" help:"Validate .stencil files without emitting"`
	Version  kong.VersionFlag `short:"v" help:"Print version information"`
}

func main() {
	formattedVersion := fmt.Sprintf("%s (%s)", version, commit)

	ctx := kong.Parse(&CLI,
		kong.Name(programName),
		kong.Description("A DSL for documenting APIs and data models. Write .stencil, emit as your next work of art."),
		kong.UsageOnError(),
		kong.Vars{
			"version": formattedVersion,
		},
	)

	err := ctx.Run()
	if e, ok := errors.AsType[kongExitCoderError](err); ok {
		ctx.FatalIfErrorf(e)
	} else if err != nil {
		// assume non-Kong errors are our errors, so optionally colorize as output.Errorf. The "%s" format string is required here.
		_, _ = ctx.Stderr.Write([]byte(output.Errorf("%s", err.Error())))
		ctx.Exit(1)
	}
}
