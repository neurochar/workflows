package providing

import (
	"io"
	"log/slog"
	"os"

	"github.com/neurochar/workflows/internal/infra/loghandler"
	"github.com/neurochar/workflows/pkg/prettylog"
)

func NewLogger(appName string, appVersion string, useLogger bool, isProd bool) *slog.Logger {
	var appLogger *slog.Logger

	switch {
	case !useLogger:
		appLogger = slog.New(loghandler.NewHandlerMiddleware(slog.NewTextHandler(io.Discard, nil)))
	case isProd:
		appLogger = slog.New(loghandler.NewHandlerMiddleware(slog.NewJSONHandler(os.Stdout, nil)))
	default:
		appLogger = slog.New(loghandler.NewHandlerMiddleware(prettylog.NewHandler(nil)))
	}

	appLogger = appLogger.With("app_name", appName).With("app_version", appVersion)

	return appLogger
}
