// Package system declares the explicitly public system operation.
package system

import (
	"context"

	"github.com/wimpheling/heph-v2/internal/core"
	"github.com/wimpheling/heph-v2/internal/core/authorization"
	"github.com/wimpheling/heph-v2/internal/core/identity"
	"github.com/wimpheling/heph-v2/internal/core/operation"
)

type Input struct{}

func Info() *operation.Operation[Input, core.SystemInfo] {
	return operation.Define(nil, operation.Public[Input](), info)
}
func info(_ context.Context, _ identity.ActorRef, _ Input, _ *authorization.Service) (core.SystemInfo, error) {
	return core.GetSystemInfo(), nil
}
