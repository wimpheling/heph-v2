package rpc

import (
	"context"

	"connectrpc.com/connect"
	pb "github.com/wimpheling/heph-v2/gen/heph/authorization/v1"
	a "github.com/wimpheling/heph-v2/internal/core/authorization"
	"github.com/wimpheling/heph-v2/internal/core/domain/grants"
	"github.com/wimpheling/heph-v2/internal/core/identity"
	"github.com/wimpheling/heph-v2/internal/rpc/platform"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthorizationService struct {
	create *platform.Binding[pb.CreateGrantRequest, pb.CreateGrantResponse, a.CreateInput, a.Grant]
	get    *platform.Binding[pb.GetGrantRequest, pb.GetGrantResponse, a.GrantInput, a.GrantRecord]
	list   *platform.Binding[pb.ListGrantsRequest, pb.ListGrantsResponse, a.ListInput, a.GrantPage]
	revoke *platform.Binding[pb.RevokeGrantRequest, pb.RevokeGrantResponse, a.GrantInput, a.Revocation]
}

func NewAuthorizationService(r *platform.Registry, service *a.Service) *AuthorizationService {
	methods := pb.File_heph_authorization_v1_authorization_proto.Services().ByName("AuthorizationService").Methods()
	return &AuthorizationService{
		create: platform.Bind(r, methods.ByName("CreateGrant"), grants.Create(service), createInput, createOutput),
		get:    platform.Bind(r, methods.ByName("GetGrant"), grants.Get(service), getInput, getOutput),
		list:   platform.Bind(r, methods.ByName("ListGrants"), grants.List(service), listInput, listOutput),
		revoke: platform.Bind(r, methods.ByName("RevokeGrant"), grants.Revoke(service), revokeInput, revokeOutput),
	}
}
func (s *AuthorizationService) CreateGrant(ctx context.Context, r *connect.Request[pb.CreateGrantRequest]) (*connect.Response[pb.CreateGrantResponse], error) {
	return s.create.Call(ctx, r)
}
func (s *AuthorizationService) GetGrant(ctx context.Context, r *connect.Request[pb.GetGrantRequest]) (*connect.Response[pb.GetGrantResponse], error) {
	return s.get.Call(ctx, r)
}
func (s *AuthorizationService) ListGrants(ctx context.Context, r *connect.Request[pb.ListGrantsRequest]) (*connect.Response[pb.ListGrantsResponse], error) {
	return s.list.Call(ctx, r)
}
func (s *AuthorizationService) RevokeGrant(ctx context.Context, r *connect.Request[pb.RevokeGrantRequest]) (*connect.Response[pb.RevokeGrantResponse], error) {
	return s.revoke.Call(ctx, r)
}

var scopeKinds = map[pb.ScopeKind]a.ScopeKind{pb.ScopeKind_SCOPE_KIND_PLATFORM: a.ScopePlatform, pb.ScopeKind_SCOPE_KIND_PERSONAL: a.ScopePersonal, pb.ScopeKind_SCOPE_KIND_ORGANIZATION: a.ScopeOrganization, pb.ScopeKind_SCOPE_KIND_PROJECT: a.ScopeProject, pb.ScopeKind_SCOPE_KIND_RESOURCE: a.ScopeResource, pb.ScopeKind_SCOPE_KIND_GIT_REF: a.ScopeGitRef}

func scopeInput(scope *pb.ScopeRef) (a.ScopeRef, error) {
	if scope == nil {
		return a.ScopeRef{}, a.ErrInvalid
	}
	result := a.ScopeRef{Kind: scopeKinds[scope.Kind], ID: a.ScopeID(scope.Id)}
	if !result.Valid() {
		return result, a.ErrInvalid
	}
	return result, nil
}
func actorInput(actor *pb.ActorRef) identity.ActorRef {
	if actor == nil {
		return identity.ActorRef{}
	}
	return identity.ActorRef{Source: actor.Source, Subject: actor.Subject}
}
func createInput(r *pb.CreateGrantRequest) (a.CreateInput, error) {
	scope, err := scopeInput(r.Scope)
	return a.CreateInput{Grantee: actorInput(r.Grantee), Action: a.Action(r.Action), Scope: scope}, err
}
func getInput(r *pb.GetGrantRequest) (a.GrantInput, error) {
	return a.GrantInput{ID: a.GrantID(r.Id)}, nil
}
func revokeInput(r *pb.RevokeGrantRequest) (a.GrantInput, error) {
	return a.GrantInput{ID: a.GrantID(r.Id)}, nil
}
func listInput(r *pb.ListGrantsRequest) (a.ListInput, error) {
	scope, err := scopeInput(r.Scope)
	size := int(r.PageSize)
	if size == 0 {
		size = 50
	}
	return a.ListInput{Scope: scope, PageSize: size, PageToken: r.PageToken}, err
}
func actorOutput(actor identity.ActorRef) *pb.ActorRef {
	return &pb.ActorRef{Source: actor.Source, Subject: actor.Subject}
}
func scopeOutput(scope a.ScopeRef) *pb.ScopeRef {
	for kind, value := range scopeKinds {
		if value == scope.Kind {
			return &pb.ScopeRef{Kind: kind, Id: string(scope.ID)}
		}
	}
	return nil
}
func grantOutput(g a.Grant) *pb.Grant {
	return &pb.Grant{Id: string(g.ID), Grantee: actorOutput(g.Grantee), Action: string(g.Action), Scope: scopeOutput(g.Scope), GrantedBy: actorOutput(g.GrantedBy), CreatedAt: timestamppb.New(g.CreatedAt)}
}
func revocationOutput(r a.Revocation) *pb.Revocation {
	return &pb.Revocation{GrantId: string(r.GrantID), RevokedBy: actorOutput(r.RevokedBy), RevokedAt: timestamppb.New(r.RevokedAt)}
}
func createOutput(g a.Grant) *pb.CreateGrantResponse {
	return &pb.CreateGrantResponse{Grant: grantOutput(g)}
}
func getOutput(r a.GrantRecord) *pb.GetGrantResponse {
	result := &pb.GetGrantResponse{Grant: grantOutput(r.Grant)}
	if r.Revocation != nil {
		result.Revocation = revocationOutput(*r.Revocation)
	}
	return result
}
func listOutput(page a.GrantPage) *pb.ListGrantsResponse {
	result := &pb.ListGrantsResponse{NextPageToken: page.NextPageToken}
	for _, g := range page.Grants {
		result.Grants = append(result.Grants, grantOutput(g))
	}
	return result
}
func revokeOutput(r a.Revocation) *pb.RevokeGrantResponse {
	return &pb.RevokeGrantResponse{Revocation: revocationOutput(r)}
}
