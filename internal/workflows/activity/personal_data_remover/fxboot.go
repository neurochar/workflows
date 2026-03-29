package personal_data_remover

import (
	"go.uber.org/fx"
)

var FxModule = fx.Module(
	"activity_personal_data_remover",
	fx.Provide(New),
)
