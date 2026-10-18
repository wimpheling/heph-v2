package platform

import (
	policy "github.com/wimpheling/heph-v2/gen/heph/platform/v1"
	pb "github.com/wimpheling/heph-v2/gen/heph/system/v1"
	"github.com/wimpheling/heph-v2/internal/core"
	"github.com/wimpheling/heph-v2/internal/core/domain/system"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"net/http"
	"testing"
)

func TestAuthenticationMetadata(t *testing.T) {
	for _, value := range []policy.Authentication{policy.Authentication_AUTHENTICATION_UNSPECIFIED, policy.Authentication(999), policy.Authentication_AUTHENTICATION_PUBLIC, policy.Authentication_AUTHENTICATION_REQUIRED} {
		file := protodesc.ToFileDescriptorProto(pb.File_heph_system_v1_system_proto)
		options := &descriptorpb.MethodOptions{}
		proto.SetExtension(options, policy.E_Authentication, value)
		file.Service[0].Method[0].Options = options
		descriptor, err := protodesc.NewFile(file, protoregistry.GlobalFiles)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Authentication(descriptor.Services().Get(0).Methods().Get(0))
		valid := value == policy.Authentication_AUTHENTICATION_PUBLIC || value == policy.Authentication_AUTHENTICATION_REQUIRED
		if (err == nil) != valid {
			t.Fatalf("policy %v accepted = %v", value, err == nil)
		}
	}
	file := protodesc.ToFileDescriptorProto(pb.File_heph_system_v1_system_proto)
	file.Service[0].Method[0].Options = nil
	descriptor, err := protodesc.NewFile(file, protoregistry.GlobalFiles)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Authentication(descriptor.Services().Get(0).Methods().Get(0)); err == nil {
		t.Fatal("missing metadata accepted")
	}
}
func TestBindingValidation(t *testing.T) {
	method := pb.File_heph_system_v1_system_proto.Services().Get(0).Methods().Get(0)
	input := func(*pb.GetSystemInfoRequest) (system.Input, error) { return system.Input{}, nil }
	output := func(core.SystemInfo) *pb.GetSystemInfoResponse { return &pb.GetSystemInfoResponse{} }
	r := NewRegistry(nil)
	Bind(r, method, system.Info(), input, output)
	Bind(r, method, system.Info(), input, output)
	if r.err == nil {
		t.Fatal("duplicate binding accepted")
	}
	mismatch := NewRegistry(nil)
	Bind(mismatch, method, system.Info(), func(*pb.GetSystemInfoResponse) (system.Input, error) { return system.Input{}, nil }, output)
	if mismatch.methods[method.FullName()] == nil {
		t.Fatal("descriptor mismatch accepted")
	}
	missing := NewRegistry(nil)
	err := Register[any](missing, http.NewServeMux(), pb.File_heph_system_v1_system_proto.Services().Get(0), nil, nil)
	if err == nil {
		t.Fatal("missing binding accepted")
	}
}
