package operation_test

import (
	"context"
	"errors"
	"testing"

	a "github.com/wimpheling/heph-v2/internal/core/authorization"
	"github.com/wimpheling/heph-v2/internal/core/identity"
	"github.com/wimpheling/heph-v2/internal/core/operation"
)

type store struct {
	grantError, scopeError, commitError error
	grants                              []a.Grant
}

func (s *store) Transaction(_ context.Context, fn func(a.Store) error) error {
	if err := fn(s); err != nil {
		return err
	}
	return s.commitError
}
func (s *store) Scope(_ context.Context, ref a.ScopeRef) (a.Scope, error) {
	return a.Scope{Ref: ref}, s.scopeError
}
func (s *store) ActiveGrants(context.Context, identity.ActorRef) ([]a.Grant, error) {
	return s.grants, s.grantError
}
func (s *store) Get(context.Context, a.GrantID) (a.GrantRecord, error) { return a.GrantRecord{}, nil }
func (s *store) List(context.Context, a.ScopeRef, int, string) (a.GrantPage, error) {
	return a.GrantPage{}, nil
}
func (s *store) Create(context.Context, identity.ActorRef, a.CreateInput) (a.Grant, error) {
	return a.Grant{}, nil
}
func (s *store) Revoke(context.Context, identity.ActorRef, a.GrantID) (a.Revocation, error) {
	return a.Revocation{}, nil
}
func TestGuardNeverInvokesHandlerOnFailure(t *testing.T) {
	actor := identity.ActorRef{Source: "test", Subject: "actor"}
	scope := func(context.Context, int, *a.Service) (a.ScopeRef, error) { return a.PlatformScope, nil }
	for _, tc := range []struct {
		name    string
		actor   identity.ActorRef
		storage *store
		extra   bool
		want    error
		called  bool
	}{
		{"missing actor", identity.ActorRef{}, &store{}, false, identity.ErrUnauthenticated, false},
		{"denied", actor, &store{}, false, a.ErrDenied, false},
		{"scope failure", actor, &store{scopeError: a.ErrUnavailable}, false, a.ErrUnavailable, false},
		{"grant lookup failure", actor, &store{grantError: a.ErrUnavailable}, false, a.ErrUnavailable, false},
		{"additional policy denied", actor, &store{grants: []a.Grant{{Grantee: actor, Action: a.GrantCreate, Scope: a.PlatformScope}}}, true, a.ErrDenied, false},
		{"allowed", actor, &store{grants: []a.Grant{{Grantee: actor, Action: a.GrantCreate, Scope: a.PlatformScope}}}, false, nil, true},
		{"commit failure discards result", actor, &store{grants: []a.Grant{{Grantee: actor, Action: a.GrantCreate, Scope: a.PlatformScope}}, commitError: a.ErrUnavailable}, false, a.ErrUnavailable, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			policy := operation.Require(a.GrantCreate, scope)
			if tc.extra {
				policy = operation.Require(a.GrantCreate, scope, func(context.Context, identity.ActorRef, int, *a.Service) error { return a.ErrDenied })
			}
			op := operation.Define(a.New(tc.storage), policy, func(context.Context, identity.ActorRef, int, *a.Service) (int, error) { calls++; return 42, nil })
			result, err := op.Execute(context.Background(), tc.actor, 0)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if (calls > 0) != tc.called {
				t.Fatalf("handler calls = %d", calls)
			}
			if err != nil && result != 0 {
				t.Fatal("failed transaction leaked its result")
			}
		})
	}
}
func TestOperationRequiresExplicitPolicy(t *testing.T) {
	handler := func(context.Context, identity.ActorRef, int, *a.Service) (int, error) { return 1, nil }
	if operation.Define(nil, operation.Policy[int]{}, handler).Validate() == nil {
		t.Fatal("missing policy accepted")
	}
	if operation.Define[int, int](nil, operation.Public[int](), nil).Validate() == nil {
		t.Fatal("missing handler accepted")
	}
	public := operation.Define(nil, operation.Public[int](), handler)
	if result, err := public.Execute(context.Background(), identity.ActorRef{}, 0); err != nil || result != 1 {
		t.Fatalf("public operation = %d,%v", result, err)
	}
}
