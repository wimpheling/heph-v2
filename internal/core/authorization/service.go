package authorization

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wimpheling/heph-v2/internal/core/identity"
)

type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }
func (s *Service) Within(ctx context.Context, fn func(*Service) error) error {
	if s == nil || s.store == nil {
		return ErrUnavailable
	}
	return s.store.Transaction(ctx, func(store Store) error { return fn(New(store)) })
}

func validParent(child, parent Scope) bool {
	switch child.Ref.Kind {
	case ScopePersonal, ScopeOrganization:
		return parent.Ref == PlatformScope
	case ScopeProject:
		return parent.Ref.Kind == ScopePersonal || parent.Ref.Kind == ScopeOrganization
	case ScopeResource:
		return parent.Ref.Kind == ScopePersonal || parent.Ref.Kind == ScopeOrganization || parent.Ref.Kind == ScopeProject
	case ScopeGitRef:
		return parent.Ref.Kind == ScopeResource && parent.ResourceKind == "repository" && strings.HasPrefix(child.RefName, "refs/") && len(child.RefName) > 5
	}
	return false
}

func (s *Service) ancestors(ctx context.Context, ref ScopeRef) ([]Scope, error) {
	if !ref.Valid() {
		return nil, ErrInvalid
	}
	seen := map[ScopeRef]bool{}
	chain := []Scope{}
	for {
		if seen[ref] || len(chain) >= 128 {
			return nil, fmt.Errorf("%w: cyclic or excessive scope hierarchy", ErrInvalid)
		}
		seen[ref] = true
		scope, err := s.store.Scope(ctx, ref)
		if err != nil {
			return nil, err
		}
		if scope.Ref != ref {
			return nil, ErrInvalid
		}
		if len(chain) > 0 && !validParent(chain[len(chain)-1], scope) {
			return nil, fmt.Errorf("%w: invalid scope parent", ErrInvalid)
		}
		chain = append(chain, scope)
		if scope.Parent == nil {
			if scope.Ref != PlatformScope {
				return nil, fmt.Errorf("%w: missing scope parent", ErrInvalid)
			}
			return chain, nil
		}
		if !scope.Parent.Valid() {
			return nil, ErrInvalid
		}
		ref = *scope.Parent
	}
}

func (s *Service) Authorize(ctx context.Context, actor ActorRef, action Action, ref ScopeRef) error {
	if !actor.Valid() {
		return identity.ErrUnauthenticated
	}
	if !KnownAction(action) {
		return ErrInvalid
	}
	chain, err := s.ancestors(ctx, ref)
	if err != nil {
		return err
	}
	grants, err := s.store.ActiveGrants(ctx, actor)
	if err != nil {
		return err
	}
	for _, scope := range chain {
		if scope.Ref.Kind == ScopePersonal && scope.Owner != nil && *scope.Owner == actor {
			return nil
		}
		for _, grant := range grants {
			if grant.Grantee == actor && grant.Action == action && grant.Scope == scope.Ref {
				return nil
			}
		}
	}
	return ErrDenied
}

func (s *Service) Create(ctx context.Context, actor ActorRef, input CreateInput) (Grant, error) {
	return s.store.Create(ctx, actor, input)
}
func (s *Service) Get(ctx context.Context, input GrantInput) (GrantRecord, error) {
	return s.store.Get(ctx, input.ID)
}
func (s *Service) List(ctx context.Context, input ListInput) (GrantPage, error) {
	after, err := decodePage(input)
	if err != nil {
		return GrantPage{}, err
	}
	page, err := s.store.List(ctx, input.Scope, input.PageSize, after)
	if err != nil {
		return GrantPage{}, err
	}
	if page.NextPageToken != "" {
		raw, _ := json.Marshal(pageCursor{Scope: input.Scope, After: page.NextPageToken})
		page.NextPageToken = base64.RawURLEncoding.EncodeToString(raw)
	}
	return page, nil
}

type pageCursor struct {
	Scope ScopeRef
	After string
}

func decodePage(input ListInput) (string, error) {
	if input.PageToken == "" {
		return "", nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(input.PageToken)
	if err != nil {
		return "", ErrInvalid
	}
	var cursor pageCursor
	if json.Unmarshal(raw, &cursor) != nil || cursor.Scope != input.Scope || cursor.After == "" || len(cursor.After) > 128 {
		return "", ErrInvalid
	}
	return cursor.After, nil
}
func (s *Service) Revoke(ctx context.Context, actor ActorRef, input GrantInput) (Revocation, error) {
	return s.store.Revoke(ctx, actor, input.ID)
}

// Policy resolvers validate inputs before the guard invokes a private handler.
func CreateScope(_ context.Context, input CreateInput, _ *Service) (ScopeRef, error) {
	if !input.Grantee.Valid() || !KnownAction(input.Action) || !input.Scope.Valid() {
		return ScopeRef{}, ErrInvalid
	}
	return input.Scope, nil
}
func ListScope(_ context.Context, input ListInput, _ *Service) (ScopeRef, error) {
	if !input.Scope.Valid() || input.PageSize < 1 || input.PageSize > 100 || len(input.PageToken) > 2048 {
		return ScopeRef{}, ErrInvalid
	}
	_, err := decodePage(input)
	return input.Scope, err
}
func StoredScope(ctx context.Context, input GrantInput, s *Service) (ScopeRef, error) {
	if input.ID == "" || len(input.ID) > 128 {
		return ScopeRef{}, ErrInvalid
	}
	record, err := s.Get(ctx, input)
	return record.Grant.Scope, err
}
func CanDelegate(ctx context.Context, actor ActorRef, input CreateInput, s *Service) error {
	return s.Authorize(ctx, actor, input.Action, input.Scope)
}
