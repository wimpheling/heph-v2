package operation

import (
	"context"
	a "github.com/wimpheling/heph-v2/internal/core/authorization"
	"github.com/wimpheling/heph-v2/internal/core/identity"
)

type Policy[I any] struct{}
type Operation[I, O any] struct{}

func Public[I any]() Policy[I] { return Policy[I]{} }
func Require[I any](a.Action, func(context.Context, I, *a.Service) (a.ScopeRef, error)) Policy[I] {
	return Policy[I]{}
}
func Define[I, O any](_ *a.Service, _ Policy[I], _ func(context.Context, identity.ActorRef, I, *a.Service) (O, error)) *Operation[I, O] {
	return &Operation[I, O]{}
}
func (*Operation[I, O]) Execute(context.Context, identity.ActorRef, I) (O, error) {
	var zero O
	return zero, nil
}
