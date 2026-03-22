package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	internalerr "github.com/jimschubert/spray/errors"
	"github.com/jimschubert/spray/internal/output"
	"github.com/jimschubert/spray/parser"
)

type ValidateCmd struct {
	// Files is a slice of files. type is 'path', allowing expansion such as: validate ./api/* (works in terminal, not in some IDEs)
	Files []string `arg:"" help:".stencil files to validate" type:"path"`
}

func (v *ValidateCmd) Run() error {
	var eee error
	fmt.Println(output.Boldf("Validating %d file(s)...", len(v.Files)))
	for _, path := range v.Files {
		src, err := os.ReadFile(path)
		if err != nil {
			// not a hard failure
			fmt.Println(output.Fail(path + " (could not read file)"))
			eee = errors.Join(eee, fmt.Errorf("reading %s: %w", path, err))
			continue
		}

		p, err := parser.New()
		if err != nil {
			// hard failure
			return fmt.Errorf("creating parser: %w", err)
		}

		_, err = p.Parse(string(src))
		if err != nil {
			fmt.Println(output.Fail(path))
			if wrapped, ok := errors.AsType[internalerr.JoinUnwrap](err); ok {
				msg := strings.Builder{}
				for _, e := range wrapped.Unwrap() {
					msg.WriteString(fmt.Sprintf("\t%s\n", e.Error()))
				}
				eee = errors.Join(eee, fmt.Errorf("%s:\n%s", path, msg.String()))
			} else {
				eee = errors.Join(eee, fmt.Errorf("%s:\n\t%w", path, err))
			}
			continue
		}

		fmt.Println(output.Pass(path))
	}

	if eee != nil {
		fmt.Println(output.Errorf("\nValidation completed with errors."))
	} else {
		fmt.Println(output.Successf("\nValidation completed successfully!"))
	}

	return eee
}
