package rpc

import (
	"context"

	"connectrpc.com/connect"
	generated "github.com/wimpheling/heph-v2/gen/heph/test/v1/testv1connect"
	a "github.com/wimpheling/heph-v2/internal/core/authorization"
	bad "github.com/wimpheling/heph-v2/internal/core/domain/bad"
	d "github.com/wimpheling/heph-v2/internal/core/domain/valid"
	"github.com/wimpheling/heph-v2/internal/core/identity"
	op "github.com/wimpheling/heph-v2/internal/core/operation"
	p "github.com/wimpheling/heph-v2/internal/rpc/platform"
)

func input(*int) (a.ListInput, error) { return a.ListInput{}, nil }
func output(int) *int                 { return nil }

type Service struct {
	binding *p.Binding[int, int, a.ListInput, int]
}
type RequestAlias = connect.Request[int]

func (r Service) Good(ctx context.Context, req *connect.Request[int]) (int, error) {
	return r.binding.Call(ctx, req)
}
func (r Service) Missing(ctx context.Context, req *RequestAlias) (int, error) { // want "exactly one typed endpoint binding" "RPC adapters must delegate"
	return 0, nil
}
func (r Service) Repeated(ctx context.Context, req *connect.Request[int]) (int, error) { // want "exactly one typed endpoint binding"
	return r.binding.Call(ctx, req)
}
func setup(reg *p.Registry, service *a.Service) {
	p.Bind(reg, p.Methods{}.ByName("Good"), d.Info(service), input, output)
	p.Bind(reg, p.Methods{}.ByName("Repeated"), d.Info(service), input, output)
	p.Bind(reg, p.Methods{}.ByName("Repeated"), d.Info(service), input, output) // want "duplicate typed endpoint binding"
	generated.NewTestServiceHandler()                                           // want "register RPC handlers through"
	p.Register(nil, nil, nil, generated.NewTestServiceHandler, nil)
	service.Create(context.Background(), aActor(), a.ListInput{}) // want "grant mutations must use guarded"
	_ = bad.Exported                                              // want "raw domain handlers"
	_ = op.Define[int, int](nil, op.Public[int](), nil)           // want "define business operations only"
}
func aActor() identity.ActorRef { return identity.ActorRef{} }
