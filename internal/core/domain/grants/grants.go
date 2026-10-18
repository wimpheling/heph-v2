// Package grants declares guarded grant operations and their private handlers.
package grants

import (
	"context"

	"github.com/wimpheling/heph-v2/internal/core/authorization"
	"github.com/wimpheling/heph-v2/internal/core/identity"
	"github.com/wimpheling/heph-v2/internal/core/operation"
)

func Create(service *authorization.Service) *operation.Operation[authorization.CreateInput, authorization.Grant] {
	return operation.Define(service, operation.Require(authorization.GrantCreate, authorization.CreateScope, authorization.CanDelegate), create)
}
func Get(service *authorization.Service) *operation.Operation[authorization.GrantInput, authorization.GrantRecord] {
	return operation.Define(service, operation.Require(authorization.GrantRead, authorization.StoredScope), get)
}
func List(service *authorization.Service) *operation.Operation[authorization.ListInput, authorization.GrantPage] {
	return operation.Define(service, operation.Require(authorization.GrantRead, authorization.ListScope), list)
}
func Revoke(service *authorization.Service) *operation.Operation[authorization.GrantInput, authorization.Revocation] {
	return operation.Define(service, operation.Require(authorization.GrantRevoke, authorization.StoredScope), revoke)
}
func create(ctx context.Context, actor identity.ActorRef, input authorization.CreateInput, service *authorization.Service) (authorization.Grant, error) {
	return service.Create(ctx, actor, input)
}
func get(ctx context.Context, _ identity.ActorRef, input authorization.GrantInput, service *authorization.Service) (authorization.GrantRecord, error) {
	return service.Get(ctx, input)
}
func list(ctx context.Context, _ identity.ActorRef, input authorization.ListInput, service *authorization.Service) (authorization.GrantPage, error) {
	return service.List(ctx, input)
}
func revoke(ctx context.Context, actor identity.ActorRef, input authorization.GrantInput, service *authorization.Service) (authorization.Revocation, error) {
	return service.Revoke(ctx, actor, input)
}
