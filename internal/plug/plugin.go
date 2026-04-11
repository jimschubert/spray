package plug

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/jimschubert/spray/emitter"
	"github.com/jimschubert/spray/resolver"
)

type pluginOutput struct {
	Filename_    string      `json:"filename"`
	ContentType_ ContentType `json:"content_type"`
	Encoding_    Encoding    `json:"encoding"`
	Content_     string      `json:"content"`
}

func (p pluginOutput) Filename() string {
	return p.Filename_
}

func (p pluginOutput) Contents() []byte {
	if p.Encoding_ == EncodingBase64 {
		dec := make([]byte, base64.StdEncoding.DecodedLen(len(p.Content_)))
		n, err := base64.StdEncoding.Decode(dec, []byte(p.Content_))
		if err != nil {
			return []byte(p.Content_)
		}
		return dec[:n]
	}
	return []byte(p.Content_)
}

func (p pluginOutput) ContentType() emitter.ContentType {
	switch p.ContentType_ {
	case ContentTypeText:
		return emitter.ContentText
	case ContentTypeBinary:
		return emitter.ContentBinary
	default:
		return emitter.ContentText
	}
}

// PluginResponse is the standard output from a plugin process.
type PluginResponse struct {
	Outputs []pluginOutput `json:"outputs"`
	Error   string         `json:"error,omitempty"`
}

// PluginRequest is sent to a plugin process via stdin.
type PluginRequest struct {
	Command Command `json:"command"`
	// Schema is the fully resolved intermediate representation.
	Schema PluginSchema `json:"schema"`
	// SpecType is required when Command is "emit_one".
	SpecType *SpecType `json:"spec_type,omitempty"`
	// SpecName is required when Command is "emit_one".
	SpecName string `json:"spec_name,omitempty"`
}

// PluginSchema is the root schema representation sent to a plugin.
type PluginSchema struct {
	Namespace  string            `json:"namespace,omitempty"`
	Models     []PluginModel     `json:"models,omitempty"`
	Inputs     []PluginInput     `json:"inputs,omitempty"`
	Enums      []PluginEnum      `json:"enums,omitempty"`
	Apis       []PluginApi       `json:"apis,omitempty"`
	Monomorphs []PluginMonomorph `json:"monomorphs,omitempty"`
	// Aliases maps namespace to type aliases declared within that namespace.
	Aliases map[string][]PluginTypeAlias `json:"aliases,omitempty"`
	// Extensions maps FQN (e.g. "acme.v1.User") to @raw blocks attached to that node.
	Extensions map[string][]PluginExtension `json:"extensions,omitempty"`
}

// PluginTypeAlias represents a type alias, e.g. "type Email = string".
type PluginTypeAlias struct {
	Name string        `json:"name"`
	Type PluginTypeRef `json:"type"`
	// HeadComment is the comment block preceding the alias declaration.
	HeadComment string `json:"head_comment,omitempty"`
	// LineComment is the trailing inline comment.
	LineComment string `json:"line_comment,omitempty"`
}

// PluginExtension represents a @raw block attached to a model or API.
type PluginExtension struct {
	Target string                `json:"target"`
	Pairs  []PluginExtensionPair `json:"pairs"`
}

// PluginExtensionPair is a single key/value entry within a @raw block.
type PluginExtensionPair struct {
	Key string `json:"key"`
	// Value may be string, number, bool, or nil for null.
	Value any `json:"value"`
}

// PluginModel represents a data model definition.
type PluginModel struct {
	Name        string        `json:"name"`
	Namespace   string        `json:"namespace,omitempty"`
	HeadComment string        `json:"head_comment,omitempty"`
	Fields      []PluginField `json:"fields"`
}

// PluginTypeRef is a reference to a type.
type PluginTypeRef struct {
	// FQN is the fully qualified name, e.g. "acme.v1.User", or a scalar like "string".
	FQN        string `json:"fqn"`
	IsArray    bool   `json:"array"`
	IsOptional bool   `json:"optional"`
	IsScalar   bool   `json:"scalar"`
	// Args holds the concrete type arguments for parameterized types, e.g. "Page<User>" has Args: ["User"].
	Args []PluginTypeRef `json:"args,omitempty"`
}

// PluginField is a single field within a model or input.
type PluginField struct {
	Name string        `json:"name"`
	Type PluginTypeRef `json:"type"`
	// HeadComment is the comment block preceding the field declaration.
	HeadComment string `json:"head_comment,omitempty"`
	// LineComment is the trailing inline comment.
	LineComment string            `json:"line_comment,omitempty"`
	Decorators  []PluginDecorator `json:"decorators,omitempty"`
}

// PluginDecoratorArg is a single argument within a decorator.
type PluginDecoratorArg struct {
	// Name is set for key=value arguments, nil for positional arguments.
	Name  *string `json:"name,omitempty"`
	Value *string `json:"value"`
}

// PluginDecorator represents a decorator applied to a node.
type PluginDecorator struct {
	// Name is the decorator identifier without the "@" prefix.
	Name string               `json:"name"`
	Args []PluginDecoratorArg `json:"args,omitempty"`
}

// PluginInput represents a request payload shape.
type PluginInput struct {
	Name        string        `json:"name"`
	Namespace   string        `json:"namespace,omitempty"`
	HeadComment string        `json:"head_comment,omitempty"`
	Fields      []PluginField `json:"fields"`
}

// PluginEnum represents an enumerated type.
type PluginEnum struct {
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace,omitempty"`
	HeadComment string   `json:"head_comment,omitempty"`
	Values      []string `json:"values"`
}

// PluginApi represents a route or procedure definition.
// The Style field determines which other fields are populated:
//   - rest:   Method, Path, Return are set. Input may be set via @body or @query decorators.
//   - rpc:    Name, Input, Return, Streaming are set. Method and Path are empty.
//   - events: Name, Direction, Return are set. Method, Path, Input are empty.
type PluginApi struct {
	Style       RouteStyle        `json:"style"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	HeadComment string            `json:"head_comment,omitempty"`
	Method      Method            `json:"method,omitempty"`
	Path        string            `json:"path,omitempty"`
	Input       *PluginTypeRef    `json:"input,omitempty"`
	Return      PluginTypeRef     `json:"return"`
	Streaming   bool              `json:"streaming"`
	Direction   EventDirection    `json:"direction,omitempty"`
	Decorators  []PluginDecorator `json:"decorators,omitempty"`
}

// PluginMonomorph represents a concrete instantiation of a generic type.
type PluginMonomorph struct {
	// Name is the generated concrete type name, e.g. "PageUser".
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	// HeadComment from the original generic definition.
	HeadComment string `json:"head_comment,omitempty"`
	// Original is the FQN of the generic definition, e.g. "acme.v1.Page".
	Original string `json:"original"`
	// Args holds the concrete type arguments substituted into this instantiation.
	Args []PluginTypeRef `json:"args"`
}

type pluginEmitter struct {
	bin    string
	schema *resolver.ResolvedSchema
}

func (p *pluginEmitter) EmitAll() ([]emitter.Output, error) {
	return p.invoke(PluginRequest{
		Schema:  NewMapper(p.schema).Map(),
		Command: CommandEmitAll,
	})
}

func (p *pluginEmitter) EmitOne(typ emitter.SpecType, name string) (emitter.Output, error) {
	req := PluginRequest{
		Schema:   NewMapper(p.schema).Map(),
		Command:  CommandEmitOne,
		SpecName: name,
	}

	var specType SpecType
	switch typ {
	case emitter.SpecModel:
		specType = SpecModel
	case emitter.SpecInput:
		specType = SpecInput
	case emitter.SpecEnum:
		specType = SpecEnum
	case emitter.SpecApi:
		specType = SpecApi
	default:
		return nil, fmt.Errorf("unsupported SpecType %v", typ)
	}
	req.SpecType = &specType

	outputs, err := p.invoke(req)
	if err != nil {
		return nil, err
	}

	if len(outputs) == 0 {
		return nil, fmt.Errorf("plugin returned no output for %v %q", typ, name)
	}
	return outputs[0], nil
}

func (p *pluginEmitter) invoke(req PluginRequest) ([]emitter.Output, error) {
	proc := exec.Command(p.bin)
	proc.Stderr = os.Stderr

	stdin, err := proc.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("plugin stdin: %w", err)
	}
	stdout, err := proc.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("plugin stdout: %w", err)
	}

	if err := proc.Start(); err != nil {
		return nil, fmt.Errorf("starting plugin %q: %w", p.bin, err)
	}

	defer func() {
		if proc.Process != nil {
			_ = proc.Process.Kill()
		}
	}()

	enc := json.NewEncoder(stdin)
	if err := enc.Encode(req); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("encoding request to plugin: %w", err)
	}
	// signal EOF so the plugin knows the request is complete
	if err := stdin.Close(); err != nil {
		return nil, fmt.Errorf("closing plugin stdin: %w", err)
	}

	var resp PluginResponse
	dec := json.NewDecoder(stdout)
	if err := dec.Decode(&resp); err != nil {
		return nil, fmt.Errorf("decoding response from plugin: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("plugin error: %s", resp.Error)
	}

	if err := proc.Wait(); err != nil {
		return nil, fmt.Errorf("plugin exited with error: %w", err)
	}

	outputs := make([]emitter.Output, len(resp.Outputs))
	for i := range resp.Outputs {
		outputs[i] = resp.Outputs[i]
	}

	return outputs, nil
}

// New creates a new plugin-backed emitter for the given emitter name.
func New(name string, resolved *resolver.ResolvedSchema) (emitter.Emitter, error) {
	bin, err := Find(name)
	if err != nil {
		return nil, err
	}

	return &pluginEmitter{
		bin:    bin,
		schema: resolved,
	}, nil
}
