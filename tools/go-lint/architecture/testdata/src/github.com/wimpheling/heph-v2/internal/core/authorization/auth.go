package authorization

import (
	"context"
	"errors"
	"github.com/wimpheling/heph-v2/internal/core/identity"
)

type Action string

const GrantRead Action = "authorization.grants.read"

var ErrDenied = errors.New("denied")

type ScopeRef struct{}
type ListInput struct {
	Allowed, Blocked bool
	Err              error
	Role             string
	Count            int
	Name             string
}
type Service struct{}

func (*Service) Get(context.Context, ListInput) (int, error)                       { return 1, nil }
func (*Service) Create(context.Context, identity.ActorRef, ListInput) (int, error) { return 1, nil }

type GrantReader interface {
	ActiveGrants(context.Context, identity.ActorRef) (bool, error)
}

func StoredScope(context.Context, ListInput, *Service) (ScopeRef, error) { return ScopeRef{}, nil }
