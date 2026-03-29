package word2pdf

import (
	"go.uber.org/fx"
)

var FxModule = fx.Module(
	"activity_word2pdf",
	fx.Provide(New),
)
