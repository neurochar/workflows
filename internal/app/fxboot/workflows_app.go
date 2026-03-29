package fxboot

import (
	"context"
	"log/slog"
	"time"

	backendAdapter "github.com/neurochar/workflows/internal/adapter/backend"
	"github.com/neurochar/workflows/internal/app"
	"github.com/neurochar/workflows/internal/app/config"
	"github.com/neurochar/workflows/internal/app/fxboot/invoking"
	"github.com/neurochar/workflows/internal/app/fxboot/providing"
	grpcClient "github.com/neurochar/workflows/internal/client/grpc"
	temporalWorker "github.com/neurochar/workflows/internal/infra/temporal/worker"
	"github.com/neurochar/workflows/internal/workflows"
	"github.com/neurochar/workflows/internal/workflows/activity"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func WorkflowsAppGetOptionsMap(appID app.ID, cfg config.Config) OptionsMap {
	return OptionsMap{
		Providing: map[ProvidingID]fx.Option{
			ProvidingAppID: fx.Provide(func() app.ID {
				return appID
			}),
			ProvidingIDFXTimeouts: fx.Options(
				fx.StartTimeout(time.Second*time.Duration(cfg.WorkflowsApp.Base.StartTimeoutSec)),
				fx.StopTimeout(time.Second*time.Duration(cfg.WorkflowsApp.Base.StopTimeoutSec)),
			),
			ProvidingIDConfig: fx.Provide(func() config.Config {
				return cfg
			}),
			ProvidingIDLogger: fx.Provide(func(cfg config.Config) *slog.Logger {
				return providing.NewLogger(
					cfg.WorkflowsApp.Name,
					cfg.WorkflowsApp.Version,
					cfg.WorkflowsApp.Base.UseLogger,
					cfg.WorkflowsApp.Base.IsProd,
				)
			}),
			ProvidingIDFXLogger: fx.WithLogger(func(cfg config.Config) fxevent.Logger {
				return providing.NewFXLogger(cfg.WorkflowsApp.Base.UseFxLogger)
			}),
			ProvidingIDTemporalWorker: fx.Provide(
				func(cfg config.Config, logger *slog.Logger) (temporalWorker.WorkerClient, error) {
					return temporalWorker.NewClient(
						cfg.Temporal.Host,
						cfg.Temporal.Namespace,
						logger,
					)
				},
			),
			ProvidingIDWorkflowsController: fx.Provide(
				workflows.NewController,
			),
			ProvidingIDWorkflowsActivies: activity.FxModule,
			ProvidingIDStorageClient:     fx.Provide(providing.NewStorageClient),
			ProvidingIDGRPCBackendPrivateConnection: fx.Provide(
				func(logger *slog.Logger, cfg config.Config) (*backendAdapter.PrivateClientConnection, error) {
					return grpcClient.NewClientConn(
						grpcClient.Config{
							Addr:         cfg.Backend.GRPCPrivateEndpoint,
							RetriesCount: 1,
							Timeout:      time.Minute,
						},
						logger,
					)
				},
			),
			ProvidingIDGRPCBackendClient: fx.Provide(
				func(logger *slog.Logger, privateCC *backendAdapter.PrivateClientConnection) (backendAdapter.Adapter, error) {
					adapter, err := backendAdapter.NewAdapterImpl(
						privateCC,
						logger,
					)
					if err != nil {
						return nil, err
					}

					return adapter, nil
				},
			),
		},
		Invokes: []fx.Option{
			fx.Invoke(WorkflowsAppInitInvoke),
		},
	}
}

type WorkflowsInvokeInput struct {
	fx.In

	LC                   fx.Lifecycle
	Shutdowner           fx.Shutdowner
	Invokes              []invoking.InvokeInit `group:"InvokeInit"`
	Logger               *slog.Logger
	Cfg                  config.Config
	TemporalWorker       temporalWorker.WorkerClient
	WorkflowsController  *workflows.Controller
	BackendGRPCPrivateCC *backendAdapter.PrivateClientConnection
}

// WorkflowsAppInitInvoke - app init
func WorkflowsAppInitInvoke(
	in WorkflowsInvokeInput,
) {
	ctxWork, cancel := context.WithCancel(context.Background())

	in.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Подключаем адаптеры
			err := grpcClient.ConnectToGRPCServer(ctx, in.BackendGRPCPrivateCC)
			if err != nil {
				in.Logger.ErrorContext(ctx, "failed to connect to [backend private] grpc server", slog.Any("error", err))
				return err
			}
			in.Logger.InfoContext(ctx, "connection established to [backend private] grpc server")

			// Регистрируем обработчики
			in.WorkflowsController.RegisterWorkers()
			in.WorkflowsController.StartWorkers()

			// Запускаем invoke функции до открытия
			for _, invokeItem := range in.Invokes {
				if invokeItem.StartBeforeOpen != nil {
					err := invokeItem.StartBeforeOpen(ctxWork)
					if err != nil {
						in.Logger.ErrorContext(ctx, "failed to execute invoke fn start before open", slog.Any("error", err))
						return err
					}
				}
			}

			// Запускаем invoke функции после открытия
			for _, invokeItem := range in.Invokes {
				if invokeItem.StartAfterOpen != nil {
					err := invokeItem.StartAfterOpen(ctxWork)
					if err != nil {
						in.Logger.ErrorContext(ctx, "failed to execute invoke fn start after open", slog.Any("error", err))
						return err
					}
				}
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			cancel()

			in.WorkflowsController.StopWorkers()
			in.TemporalWorker.Close()
			in.Logger.InfoContext(ctx, "workflows and temporal stopped")

			err := in.BackendGRPCPrivateCC.Close()
			if err != nil {
				in.Logger.ErrorContext(ctx, "failed to disconnect from [backend private] grpc server", slog.Any("error", err))
				return err
			}

			return nil
		},
	})
}
