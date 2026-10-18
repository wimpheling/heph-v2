// Package operation guards every domain entry point, including internal calls.
package operation

import (
	"context"
	"errors"

	"github.com/wimpheling/heph-v2/internal/core/authorization"
	"github.com/wimpheling/heph-v2/internal/core/identity"
)

type Policy[I any] struct {
	public     bool
	action     authorization.Action
	resolve    func(context.Context, I, *authorization.Service) (authorization.ScopeRef, error)
	additional func(context.Context, identity.ActorRef, I, *authorization.Service) error
}

func Public[I any]() Policy[I] { return Policy[I]{public: true} }
func Require[I any](action authorization.Action, resolve func(context.Context, I, *authorization.Service) (authorization.ScopeRef, error), additional ...func(context.Context, identity.ActorRef, I, *authorization.Service) error) Policy[I] {
	p := Policy[I]{action: action, resolve: resolve}
	if len(additional) > 1 {
		panic("at most one additional authorization policy")
	}
	if len(additional) == 1 {
		p.additional = additional[0]
	}
	return p
}

type Operation[I, O any] struct {
	service *authorization.Service
	policy  Policy[I]
	handler func(context.Context, identity.ActorRef, I, *authorization.Service) (O, error)
}

func Define[I, O any](service *authorization.Service, policy Policy[I], handler func(context.Context, identity.ActorRef, I, *authorization.Service) (O, error)) *Operation[I, O] {
	return &Operation[I, O]{service: service, policy: policy, handler: handler}
}
func (o *Operation[I, O]) Validate() error {
	if o == nil || o.handler == nil {
		return errors.New("operation requires a private handler")
	}
	if o.policy.public {
		if o.policy.action != "" || o.policy.resolve != nil || o.policy.additional != nil {
			return errors.New("public policy cannot declare protected requirements")
		}
		return nil
	}
	if !authorization.KnownAction(o.policy.action) || o.policy.resolve == nil || o.service == nil {
		return errors.New("operation requires an explicit policy and authorization service")
	}
	return nil
}
func (o *Operation[I, O]) IsPublic() bool { return o.policy.public }
func (o *Operation[I, O]) Execute(ctx context.Context, actor identity.ActorRef, input I) (O, error) {
	var result O
	if err := o.Validate(); err != nil {
		return result, err
	}
	if o.policy.public {
		return o.handler(ctx, actor, input, nil)
	}
	if !actor.Valid() {
		return result, identity.ErrUnauthenticated
	}
	err := o.service.Within(ctx, func(service *authorization.Service) error {
		scope, err := o.policy.resolve(ctx, input, service)
		if err != nil {
			return err
		}
		if err = service.Authorize(ctx, actor, o.policy.action, scope); err != nil {
			return err
		}
		if o.policy.additional != nil {
			if err = o.policy.additional(ctx, actor, input, service); err != nil {
				return err
			}
		}
		result, err = o.handler(ctx, actor, input, service)
		return err
	})
	if err != nil {
		var zero O
		return zero, err
	}
	return result, nil
}
