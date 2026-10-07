package commands

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"reflect"
	"sort"
)

const MaxPayloadBytes = 256 << 10

type Descriptor struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
}
type Catalog struct {
	ids         ports.IDGenerator
	descriptors map[string]Descriptor
	handlers    map[string]func(context.Context, domain.CommandContext, json.RawMessage) (any, error)
}

func newCatalog(ids ports.IDGenerator) *Catalog {
	return &Catalog{ids: ids, descriptors: map[string]Descriptor{}, handlers: map[string]func(context.Context, domain.CommandContext, json.RawMessage) (any, error){}}
}
func register[C any](c *Catalog, name string, run func(context.Context, domain.CommandContext, C) (any, error)) {
	c.descriptors[name] = Descriptor{Name: name, Schema: Schema(reflect.TypeFor[C]())}
	c.handlers[name] = func(ctx context.Context, cc domain.CommandContext, raw json.RawMessage) (any, error) {
		normalized, err := Normalize(raw, reflect.TypeFor[C]())
		if err != nil {
			return nil, domain.WrapError(domain.ErrorCodeInvalidArgument, "invalid command", err)
		}
		var cmd C
		if err = Decode(normalized, &cmd); err != nil {
			return nil, err
		}
		return run(ctx, cc, cmd)
	}
}
func (c *Catalog) Descriptors() []Descriptor {
	v := make([]Descriptor, 0, len(c.descriptors))
	for _, d := range c.descriptors {
		v = append(v, d)
	}
	sort.Slice(v, func(i, j int) bool { return v[i].Name < v[j].Name })
	return v
}
func (c *Catalog) Execute(ctx context.Context, name, key, correlation string, raw json.RawMessage) (any, error) {
	if len(raw) > MaxPayloadBytes {
		return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "command exceeds byte limit")
	}
	if err := domain.ValidateIdempotencyKey(key); err != nil {
		return nil, err
	}
	identity, ok := application.IdentityFromContext(ctx)
	if !ok {
		return nil, domain.NewError(domain.ErrorCodeForbidden, "authenticated identity required")
	}
	fn, ok := c.handlers[name]
	if !ok {
		return nil, domain.NewError(domain.ErrorCodeNotFound, "unknown command")
	}
	id, err := c.ids.NewID()
	if err != nil {
		return nil, err
	}
	return fn(ctx, domain.CommandContext{PrincipalID: identity.PrincipalID, Actor: identity.Actor, CommandID: id, IdempotencyKey: key, CorrelationID: correlation}, raw)
}
