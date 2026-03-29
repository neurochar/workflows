package storage

import (
	"go.uber.org/fx"
)

var FxModule = fx.Module(
	"activity_storage",
	fx.Provide(New),
)
