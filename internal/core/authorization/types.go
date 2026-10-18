// Package authorization owns Heph's grant, ownership, and scope policies.
package authorization

import (
	"context"
	"errors"
	"time"
	"unicode"

	"github.com/wimpheling/heph-v2/internal/core/identity"
)

type ActorRef = identity.ActorRef
type GrantID string
type Action string
type ScopeID string
type ScopeKind string

const (
	ScopePlatform            ScopeKind = "platform"
	ScopePersonal            ScopeKind = "personal"
	ScopeOrganization        ScopeKind = "organization"
	ScopeProject             ScopeKind = "project"
	ScopeResource            ScopeKind = "resource"
	ScopeGitRef              ScopeKind = "git_ref"
	GrantRead                Action    = "authorization.grants.read"
	GrantCreate              Action    = "authorization.grants.create"
	GrantRevoke              Action    = "authorization.grants.revoke"
	RefUpdate                Action    = "git.ref.update"
	RefCreate                Action    = "git.ref.create"
	RefDelete                Action    = "git.ref.delete"
	RefForceUpdate           Action    = "git.ref.force_update"
	RepositorySettingsUpdate Action    = "repository.settings.update"
)

var (
	ErrInvalid     = errors.New("invalid argument")
	ErrDenied      = errors.New("permission denied")
	ErrNotFound    = errors.New("not found")
	ErrConflict    = errors.New("grant already exists")
	ErrUnavailable = errors.New("permission storage unavailable")
)

func Actions() []Action {
	return []Action{GrantRead, GrantCreate, GrantRevoke, RefUpdate, RefCreate, RefDelete, RefForceUpdate, RepositorySettingsUpdate}
}

func KnownAction(action Action) bool {
	for _, known := range Actions() {
		if action == known {
			return true
		}
	}
	return false
}

type ScopeRef struct {
	Kind ScopeKind
	ID   ScopeID
}

func (s ScopeRef) Valid() bool {
	if !validID(string(s.ID)) {
		return false
	}
	switch s.Kind {
	case ScopePlatform, ScopePersonal, ScopeOrganization, ScopeProject, ScopeResource, ScopeGitRef:
		return true
	}
	return false
}

func validID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if unicode.IsSpace(character) || unicode.IsControl(character) {
			return false
		}
	}
	return true
}

var PlatformScope = ScopeRef{Kind: ScopePlatform, ID: "platform"}

type Scope struct {
	Ref    ScopeRef
	Parent *ScopeRef
	Owner  *ActorRef
	// ResourceKind and RefName describe trusted resource bindings, not request claims.
	ResourceKind string
	RefName      string
}

type Grant struct {
	ID        GrantID
	Grantee   ActorRef
	Action    Action
	Scope     ScopeRef
	GrantedBy ActorRef
	CreatedAt time.Time
}
type Revocation struct {
	GrantID   GrantID
	RevokedBy ActorRef
	RevokedAt time.Time
}
type GrantRecord struct {
	Grant      Grant
	Revocation *Revocation
}
type CreateInput struct {
	Grantee ActorRef
	Action  Action
	Scope   ScopeRef
}
type GrantInput struct{ ID GrantID }
type ListInput struct {
	Scope     ScopeRef
	PageSize  int
	PageToken string
}
type GrantPage struct {
	Grants        []Grant
	NextPageToken string
}

type GrantReader interface {
	ActiveGrants(context.Context, ActorRef) ([]Grant, error)
}
type Store interface {
	GrantReader
	Scope(context.Context, ScopeRef) (Scope, error)
	Get(context.Context, GrantID) (GrantRecord, error)
	List(context.Context, ScopeRef, int, string) (GrantPage, error)
	Create(context.Context, ActorRef, CreateInput) (Grant, error)
	Revoke(context.Context, ActorRef, GrantID) (Revocation, error)
	Transaction(context.Context, func(Store) error) error
}
