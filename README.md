# spray

[![Build Status](https://github.com/jimschubert/spray/actions/workflows/build.yml/badge.svg)](https://github.com/jimschubert/spray/actions/workflows/build.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/jimschubert/spray)](https://github.com/jimschubert/spray/blob/main/go.mod)
[![License](https://img.shields.io/github/license/jimschubert/spray?a=b&color=blue)](https://github.com/jimschubert/spray/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/jimschubert/spray)](https://goreportcard.com/report/github.com/jimschubert/spray)
[![GitHub Release](https://img.shields.io/github/v/release/jimschubert/spray)](https://github.com/jimschubert/spray/releases/latest)

`spray` is a tool for creating your next API-based work of art from `.stencil` files.
It validates source, formats files in place, and generates outputs like Markdown, JSON Schema, or Mermaid ER diagrams.
Spray also supports custom generator _plugins_, so you can create your own output formats.

## Install

```shell
go install github.com/jimschubert/spray/cmd/spray@latest
```

Or, check the [releases page](https://github.com/jimschubert/spray/releases) for binaries.

## Usage

The CLI commands are self-documenting, start at `spray --help` for more details on usage and options.

For example, the `format` command has the following options:

```shell
$ spray format --help
Usage: spray format <files> ... [flags]

Format .stencil files in place

Arguments:
  <files> ...    .stencil files to format

Flags:
  -h, --help                       Show context-sensitive help.
  -v, --version                    Print version information

  -a, --align-members              align members vertically
  -d, --decorators-per-line=3      maximum number of decorators per line
  -s, --lines-between-specs=1      number of lines between top-level specs
  -m, --lines-between-members=0    number of lines between members within a spec
  -i, --indent-size=2              number of spaces to use for indentation
  -c, --allow-condensed-specs      allow condensing specs with one field/member
                                   to a single line (e.g. "input User { id:
                                   ID }")
  -l, --multiline-imports          format imports with one spec's name per line
      --decorators-start="same"    where to place decorators: same,next ('same”
                                   line as member, 'next' line after member)
```

## Quick start

Create a file such as `api.stencil`:

```stencil
namespace acme.v1

model User {
  id: uuid
  name: string
}

input CreateUser {
  name: string
}

api Users @style(rest) {
  GET /users -> User
}
```

Then run some commands, such as:

```sh
spray validate ./api/*.stencil
spray format ./api/*.stencil --decorators-per-line=2 --lines-between-specs=2
spray create markdown ./api/*.stencil -o ./out/markdown
spray create jsonschema ./api/*.stencil -o ./out/jsonschema
spray create mermaid ./api/*.stencil -o ./out/mermaid
```

For the full language reference, see [`specification.md`](specification.md).

## License

Apache 2.0, see [LICENSE](LICENSE)
