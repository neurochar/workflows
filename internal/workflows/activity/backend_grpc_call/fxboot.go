package backend_grpc_call

import (
	"go.uber.org/fx"
)

var FxModule = fx.Module(
	"activity_backend_grpc_call",
	fx.Provide(New),
)
