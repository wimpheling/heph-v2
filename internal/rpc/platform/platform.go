// Package platform binds documented RPC methods to guarded domain operations.
package platform

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"fmt"
	policyv1 "github.com/wimpheling/heph-v2/gen/heph/platform/v1"
	"github.com/wimpheling/heph-v2/internal/core/authorization"
	"github.com/wimpheling/heph-v2/internal/core/identity"
	"github.com/wimpheling/heph-v2/internal/core/operation"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"net/http"
	"strings"
)

type Registry struct {
	auth       identity.Authenticator
	methods    map[protoreflect.FullName]error
	registered map[protoreflect.FullName]bool
	err        error
}

func NewRegistry(auth identity.Authenticator) *Registry {
	return &Registry{auth: auth, methods: map[protoreflect.FullName]error{}, registered: map[protoreflect.FullName]bool{}}
}
func Authentication(method protoreflect.MethodDescriptor) (policyv1.Authentication, error) {
	if !proto.HasExtension(method.Options(), policyv1.E_Authentication) {
		return 0, fmt.Errorf("%s: missing authentication policy", method.FullName())
	}
	value, ok := proto.GetExtension(method.Options(), policyv1.E_Authentication).(policyv1.Authentication)
	if !ok || (value != policyv1.Authentication_AUTHENTICATION_PUBLIC && value != policyv1.Authentication_AUTHENTICATION_REQUIRED) {
		return 0, fmt.Errorf("%s: unknown authentication policy", method.FullName())
	}
	return value, nil
}

type Binding[PReq, PRes, I, O any] struct {
	registry       *Registry
	authentication policyv1.Authentication
	operation      *operation.Operation[I, O]
	input          func(*PReq) (I, error)
	output         func(O) *PRes
}

func Bind[PReq, PRes, I, O any](r *Registry, method protoreflect.MethodDescriptor, op *operation.Operation[I, O], input func(*PReq) (I, error), output func(O) *PRes) *Binding[PReq, PRes, I, O] {
	b := &Binding[PReq, PRes, I, O]{registry: r, operation: op, input: input, output: output}
	name := method.FullName()
	if _, exists := r.methods[name]; exists {
		r.err = fmt.Errorf("duplicate endpoint binding: %s", name)
	}
	access, err := Authentication(method)
	if err == nil {
		err = op.Validate()
	}
	if err == nil {
		request, okIn := any(new(PReq)).(proto.Message)
		response, okOut := any(new(PRes)).(proto.Message)
		if !okIn || !okOut || request.ProtoReflect().Descriptor().FullName() != method.Input().FullName() || response.ProtoReflect().Descriptor().FullName() != method.Output().FullName() {
			err = fmt.Errorf("%s: protobuf descriptor and binding types disagree", name)
		}
	}
	if err == nil && (input == nil || output == nil) {
		err = fmt.Errorf("%s: missing typed converter", name)
	}
	if err == nil && (access == policyv1.Authentication_AUTHENTICATION_PUBLIC) != op.IsPublic() {
		err = fmt.Errorf("%s: authentication and operation policy disagree", name)
	}
	if err == nil && access == policyv1.Authentication_AUTHENTICATION_REQUIRED && r.auth == nil {
		err = errors.New("protected endpoints require an authenticator")
	}
	b.authentication = access
	r.methods[name] = err
	return b
}
func Register[T any](r *Registry, mux *http.ServeMux, desc protoreflect.ServiceDescriptor, constructor func(T, ...connect.HandlerOption) (string, http.Handler), service T) error {
	if r.err != nil {
		return r.err
	}
	if r.registered[desc.FullName()] {
		return fmt.Errorf("duplicate service registration: %s", desc.FullName())
	}
	for i := 0; i < desc.Methods().Len(); i++ {
		method := desc.Methods().Get(i)
		err, exists := r.methods[method.FullName()]
		if !exists {
			return fmt.Errorf("missing endpoint binding: %s", method.FullName())
		}
		if err != nil {
			return err
		}
	}
	path, handler := constructor(service)
	mux.Handle(path, handler)
	r.registered[desc.FullName()] = true
	return nil
}
func (b *Binding[PReq, PRes, I, O]) Call(ctx context.Context, request *connect.Request[PReq]) (*connect.Response[PRes], error) {
	var actor identity.ActorRef
	if b.authentication == policyv1.Authentication_AUTHENTICATION_REQUIRED {
		headers := request.Header().Values("Authorization")
		if len(headers) != 1 {
			return nil, rpcError(identity.ErrUnauthenticated)
		}
		parts := strings.Fields(headers[0])
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || len(parts[1]) > 4096 {
			return nil, rpcError(identity.ErrUnauthenticated)
		}
		var err error
		actor, err = b.registry.auth.Authenticate(ctx, parts[1])
		if err != nil {
			return nil, rpcError(err)
		}
	}
	input, err := b.input(request.Msg)
	if err != nil {
		return nil, rpcError(err)
	}
	output, err := b.operation.Execute(ctx, actor, input)
	if err != nil {
		return nil, rpcError(err)
	}
	return connect.NewResponse(b.output(output)), nil
}
func rpcError(err error) error {
	code := connect.CodeInternal
	message := "request failed"
	switch {
	case errors.Is(err, identity.ErrUnauthenticated):
		code = connect.CodeUnauthenticated
		message = "authentication required"
	case errors.Is(err, authorization.ErrInvalid):
		code = connect.CodeInvalidArgument
		message = "invalid argument"
	case errors.Is(err, authorization.ErrDenied):
		code = connect.CodePermissionDenied
		message = "permission denied"
	case errors.Is(err, authorization.ErrNotFound):
		code = connect.CodeNotFound
		message = "not found"
	case errors.Is(err, authorization.ErrConflict):
		code = connect.CodeAlreadyExists
		message = "grant already exists"
	case errors.Is(err, authorization.ErrUnavailable):
		code = connect.CodeUnavailable
		message = "permission storage unavailable"
	case errors.Is(err, context.Canceled):
		code = connect.CodeCanceled
		message = "request canceled"
	case errors.Is(err, context.DeadlineExceeded):
		code = connect.CodeDeadlineExceeded
		message = "request timed out"
	}
	return connect.NewError(code, errors.New(message))
}
