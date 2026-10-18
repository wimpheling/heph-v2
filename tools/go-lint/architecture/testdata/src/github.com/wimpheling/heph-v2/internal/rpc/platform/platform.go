package platform

import (
	"connectrpc.com/connect"
	"context"
	a "github.com/wimpheling/heph-v2/internal/core/authorization"
	op "github.com/wimpheling/heph-v2/internal/core/operation"
)

type Registry struct{}
type Method struct{}
type Methods struct{}

func (Methods) ByName(string) Method { return Method{} }

type Binding[P, R, I, O any] struct{}

func (*Binding[P, R, I, O]) Call(context.Context, *connect.Request[P]) (O, error) {
	var zero O
	return zero, nil
}
func Bind[P, R, I, O any](*Registry, Method, *op.Operation[I, O], func(*P) (I, error), func(O) *R) *Binding[P, R, I, O] {
	return nil
}
func Register(any, any, any, any, any) error { return nil }

var _ a.Action
