package valid

import (
	"context"
	"errors"
	a "github.com/wimpheling/heph-v2/internal/core/authorization"
	"github.com/wimpheling/heph-v2/internal/core/identity"
	op "github.com/wimpheling/heph-v2/internal/core/operation"
	"io/fs"
)

type Input = a.ListInput

func Info(service *a.Service) *op.Operation[Input, int] {
	return op.Define(service, op.Require(a.GrantRead, a.StoredScope), handle)
}
func Public() *op.Operation[Input, int] { return op.Define(nil, op.Public[Input](), handle) }
func handle(ctx context.Context, _ identity.ActorRef, input Input, service *a.Service) (int, error) {
	if input.Blocked || !input.Allowed {
		return 0, a.ErrDenied
	}
	value, err := service.Get(ctx, input)
	if err != nil {
		return 0, err
	}
	if nil == err && input.Allowed {
		value = 1
	}
	var target *fs.PathError
	if errors.Is(err, a.ErrDenied) || errors.As(err, &target) {
		return 0, err
	}
	return value, nil
}
func port(ctx context.Context, actor identity.ActorRef, reader a.GrantReader) (bool, error) {
	return reader.ActiveGrants(ctx, actor)
}
