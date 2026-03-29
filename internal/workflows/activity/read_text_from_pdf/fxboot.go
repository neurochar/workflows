package read_text_from_pdf

import (
	"go.uber.org/fx"
)

var FxModule = fx.Module(
	"activity_read_text_from_pdf",
	fx.Provide(New),
)
