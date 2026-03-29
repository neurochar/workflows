package grpc

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpcretry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	appErrors "github.com/neurochar/workflows/internal/app/errors"
)

var ErrGRPCServerNotCreated = appErrors.ErrInternal.Extend("cant create connect to grpc server")

type Config struct {
	Addr         string
	RetriesCount int
	Timeout      time.Duration
}

func NewClientConn(cfg Config, logger *slog.Logger) (*grpc.ClientConn, error) {
	retryOpts := []grpcretry.CallOption{
		grpcretry.WithCodes(codes.Internal, codes.Unavailable),
		grpcretry.WithMax(uint(cfg.RetriesCount)),
		grpcretry.WithPerRetryTimeout(cfg.Timeout),
	}

	logOpts := []logging.Option{
		logging.WithLogOnEvents(logging.PayloadReceived, logging.PayloadSent),
	}

	intercepts := []grpc.UnaryClientInterceptor{
		HeaderUnaryClientInterceptor(nil),
		grpcretry.UnaryClientInterceptor(retryOpts...),
	}

	if logger != nil {
		intercepts = append(intercepts, logging.UnaryClientInterceptor(InterceptorLogger(logger), logOpts...))
	}

	cc, err := grpc.NewClient(cfg.Addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(intercepts...),
	)
	if err != nil {
		return nil, err
	}

	return cc, nil
}

func InterceptorLogger(logger *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		logger.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

func ConnectToGRPCServer(ctx context.Context, cc *grpc.ClientConn) error {
	cc.Connect()
	if !cc.WaitForStateChange(ctx, connectivity.Idle) {
		_ = cc.Close()
		return ErrGRPCServerNotCreated
	}

	return nil
}

func HeaderUnaryClientInterceptor(headers map[string]string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}

		ctx = metadata.NewOutgoingContext(ctx, md)

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
