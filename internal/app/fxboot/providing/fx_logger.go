package providing

import (
	"os"

	"go.uber.org/fx/fxevent"
)

func NewFXLogger(useLogger bool) fxevent.Logger {
	if !useLogger {
		return fxevent.NopLogger
	}
	return &fxevent.ConsoleLogger{
		W: os.Stdout,
	}
}
