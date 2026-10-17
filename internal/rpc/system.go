// Package rpc translates application logic into ConnectRPC services.
package rpc

import (
	"context"

	"connectrpc.com/connect"
	systemv1 "github.com/wimpheling/heph-v2/gen/heph/system/v1"
	"github.com/wimpheling/heph-v2/gen/heph/system/v1/systemv1connect"
	"github.com/wimpheling/heph-v2/internal/core"
)

var _ systemv1connect.SystemServiceHandler = SystemService{}

type SystemService struct{}

func (SystemService) GetSystemInfo(
	_ context.Context,
	_ *connect.Request[systemv1.GetSystemInfoRequest],
) (*connect.Response[systemv1.GetSystemInfoResponse], error) {
	info := core.GetSystemInfo()
	return connect.NewResponse(&systemv1.GetSystemInfoResponse{
		Name:    info.Name,
		Version: info.Version,
	}), nil
}
