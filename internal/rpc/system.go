// Package rpc translates typed transport requests to guarded core operations.
package rpc

import (
	"connectrpc.com/connect"
	"context"
	systemv1 "github.com/wimpheling/heph-v2/gen/heph/system/v1"
	"github.com/wimpheling/heph-v2/internal/core"
	"github.com/wimpheling/heph-v2/internal/core/domain/system"
	"github.com/wimpheling/heph-v2/internal/rpc/platform"
)

type SystemService struct {
	info *platform.Binding[systemv1.GetSystemInfoRequest, systemv1.GetSystemInfoResponse, system.Input, core.SystemInfo]
}

func NewSystemService(r *platform.Registry) *SystemService {
	return &SystemService{info: platform.Bind(r, systemv1.File_heph_system_v1_system_proto.Services().ByName("SystemService").Methods().ByName("GetSystemInfo"), system.Info(), systemInput, systemOutput)}
}
func systemInput(_ *systemv1.GetSystemInfoRequest) (system.Input, error) { return system.Input{}, nil }
func systemOutput(info core.SystemInfo) *systemv1.GetSystemInfoResponse {
	return &systemv1.GetSystemInfoResponse{Name: info.Name, Version: info.Version}
}
func (s *SystemService) GetSystemInfo(ctx context.Context, req *connect.Request[systemv1.GetSystemInfoRequest]) (*connect.Response[systemv1.GetSystemInfoResponse], error) {
	return s.info.Call(ctx, req)
}
