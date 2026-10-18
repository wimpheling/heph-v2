package authorization

import (
	"context"
	"errors"
	"testing"
)

type scopeStore struct {
	Store
	scopes map[ScopeRef]Scope
	grants []Grant
}

func (s scopeStore) Scope(_ context.Context, ref ScopeRef) (Scope, error) {
	if value, ok := s.scopes[ref]; ok {
		return value, nil
	}
	return Scope{}, ErrNotFound
}
func (s scopeStore) ActiveGrants(context.Context, ActorRef) ([]Grant, error) { return s.grants, nil }
func TestOwnershipAndInvalidScopeGraphs(t *testing.T) {
	actor := ActorRef{Source: "source", Subject: "owner"}
	personal := ScopeRef{Kind: ScopePersonal, ID: "personal"}
	project := ScopeRef{Kind: ScopeProject, ID: "project"}
	root := Scope{Ref: PlatformScope}
	cases := []struct {
		name   string
		scopes map[ScopeRef]Scope
		target ScopeRef
		want   error
	}{
		{"personal ownership", map[ScopeRef]Scope{PlatformScope: root, personal: {Ref: personal, Parent: &PlatformScope, Owner: &actor}, project: {Ref: project, Parent: &personal}}, project, nil},
		{"missing parent", map[ScopeRef]Scope{project: {Ref: project}}, project, ErrInvalid},
		{"missing parent record", map[ScopeRef]Scope{project: {Ref: project, Parent: &personal}}, project, ErrNotFound},
		{"wrong parent kind", map[ScopeRef]Scope{PlatformScope: root, project: {Ref: project, Parent: &PlatformScope}}, project, ErrInvalid},
		{"cycle", map[ScopeRef]Scope{project: {Ref: project, Parent: &project}}, project, ErrInvalid},
		{"unknown kind", nil, ScopeRef{Kind: "invented", ID: "x"}, ErrInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := New(scopeStore{scopes: tc.scopes}).Authorize(context.Background(), actor, GrantRead, tc.target)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}
