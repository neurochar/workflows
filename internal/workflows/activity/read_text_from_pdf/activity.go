package read_text_from_pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neurochar/workflows/internal/app/config"
	"github.com/neurochar/workflows/internal/infra/storage"
	"go.temporal.io/sdk/activity"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	ocrpb "github.com/neurochar/workflows/pkg/proto_pb/workers/ocr"
)

type Activity struct {
	logger       *slog.Logger
	mlWorkerCfg  MlWorkersConfig
	primaryConn  mlWorkerConn
	fallbackConn mlWorkerConn
}

func New(cfg config.Config, s3Client storage.Client, logger *slog.Logger) *Activity {
	return &Activity{
		logger: logger,
		mlWorkerCfg: MlWorkersConfig{
			Service:         cfg.Workers.Ocr.Service,
			Readiness:       cfg.Workers.Ocr.Readiness,
			FallbackService: cfg.Workers.Ocr.FallbackService,
		},
	}
}

type mlWorkerConn struct {
	mu          sync.RWMutex
	conn        *grpc.ClientConn
	ocr         ocrpb.OcrServiceClient
	connGen     uint64
	reconnectMu sync.Mutex
}

type MlWorkersConfig struct {
	Service         string
	Readiness       string
	FallbackService string
}

type ReadTextFromPDFInput struct {
	Filename string
	FileData []byte
}

type ReadTextFromPDFOutput struct {
	Text            []string
	UsedFallback    bool
	ProcessDuration time.Duration
}

var errNotConnected = errors.New("ml-worker grpc: not connected")

const (
	pdfChunkSize = 256 * 1024

	pingEvery    = 3 * time.Second
	pongDeadline = 8 * time.Second

	heartbeatEvery = 10 * time.Second
)

func (d *Activity) ConnectToMLWorker(ctx context.Context) error {
	_, _, _, err := d.getActiveClient(ctx)
	return err
}

func (d *Activity) CloseConnectionToMLWorker(ctx context.Context) error {
	_ = ctx

	var errs []error

	if err := d.closeConn(&d.primaryConn); err != nil {
		errs = append(errs, fmt.Errorf("close primary connection: %w", err))
	}
	if err := d.closeConn(&d.fallbackConn); err != nil {
		errs = append(errs, fmt.Errorf("close fallback connection: %w", err))
	}

	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}

	return fmt.Errorf("multiple close errors: %v", errs)
}

func (d *Activity) closeConn(c *mlWorkerConn) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	c.ocr = nil
	c.connGen++

	return err
}

func (d *Activity) hasActiveConn(c *mlWorkerConn) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.conn != nil && c.ocr != nil
}

func (d *Activity) reconnectPrimary(ctx context.Context) error {
	return d.reconnect(ctx, &d.primaryConn, d.mlWorkerCfg.Service)
}

func (d *Activity) reconnectFallback(ctx context.Context) error {
	if d.mlWorkerCfg.FallbackService == "" {
		return fmt.Errorf("fallback service is not configured")
	}
	return d.reconnect(ctx, &d.fallbackConn, d.mlWorkerCfg.FallbackService)
}

func (d *Activity) reconnect(ctx context.Context, c *mlWorkerConn, workerAddr string) error {
	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()

	c.mu.RLock()
	if c.conn != nil {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	old := c.conn
	c.conn = nil
	c.ocr = nil
	c.connGen++
	c.mu.Unlock()

	if old != nil {
		_ = old.Close()
	}

	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		workerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(64*1024*1024),
			grpc.MaxCallSendMsgSize(64*1024*1024),
		),
	)
	if err != nil {
		return fmt.Errorf("ml-worker grpc dial failed addr=%s: %w", workerAddr, err)
	}

	client := ocrpb.NewOcrServiceClient(conn)

	c.mu.Lock()
	c.conn = conn
	c.ocr = client
	c.mu.Unlock()

	return nil
}

func (d *Activity) getConnClient(c *mlWorkerConn) (ocrpb.OcrServiceClient, *grpc.ClientConn, uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ocr, c.conn, c.connGen
}

func (d *Activity) isPrimaryReady(ctx context.Context) bool {
	if d.mlWorkerCfg.Readiness == "" {
		return true
	}

	reqCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, d.mlWorkerCfg.Readiness, nil)
	if err != nil {
		if d.logger != nil {
			d.logger.Warn("ml-worker readiness request build failed",
				slog.String("url", d.mlWorkerCfg.Readiness),
				slog.Any("err", err),
			)
		}
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if d.logger != nil {
			d.logger.Warn("ml-worker readiness check failed",
				slog.String("url", d.mlWorkerCfg.Readiness),
				slog.Any("err", err),
			)
		}
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true
	}

	if d.logger != nil {
		d.logger.Warn("ml-worker readiness bad status",
			slog.String("url", d.mlWorkerCfg.Readiness),
			slog.Int("status_code", resp.StatusCode),
		)
	}

	return false
}

func (d *Activity) getActiveClient(ctx context.Context) (*mlWorkerConn, string, bool, error) {
	primaryReady := d.isPrimaryReady(ctx)

	if d.logger != nil {
		d.logger.Info("ml-worker select start",
			slog.Bool("primary_ready", primaryReady),
			slog.String("primary_addr", d.mlWorkerCfg.Service),
			slog.String("fallback_addr", d.mlWorkerCfg.FallbackService),
		)
	}

	if primaryReady {
		if d.hasActiveConn(&d.primaryConn) {
			if d.logger != nil {
				d.logger.Info("ml-worker selected primary existing connection",
					slog.String("addr", d.mlWorkerCfg.Service),
				)
			}
			return &d.primaryConn, d.mlWorkerCfg.Service, false, nil
		}

		if err := d.reconnectPrimary(ctx); err == nil {
			if d.logger != nil {
				d.logger.Info("ml-worker selected primary new connection",
					slog.String("addr", d.mlWorkerCfg.Service),
				)
			}
			return &d.primaryConn, d.mlWorkerCfg.Service, false, nil
		} else {
			if d.logger != nil {
				d.logger.Warn("ml-worker primary grpc connect failed, switching to fallback",
					slog.String("primary_addr", d.mlWorkerCfg.Service),
					slog.Any("err", err),
				)
			}
		}
	} else {
		if d.logger != nil {
			d.logger.Warn("ml-worker primary not ready, switching to fallback",
				slog.String("readiness_url", d.mlWorkerCfg.Readiness),
			)
		}
	}

	if d.mlWorkerCfg.FallbackService == "" {
		return nil, "", false, fmt.Errorf("primary unavailable and fallback is not configured")
	}

	if d.hasActiveConn(&d.fallbackConn) {
		if d.logger != nil {
			d.logger.Info("ml-worker selected fallback existing connection",
				slog.String("addr", d.mlWorkerCfg.FallbackService),
			)
		}
		return &d.fallbackConn, d.mlWorkerCfg.FallbackService, true, nil
	}

	if err := d.reconnectFallback(ctx); err == nil {
		if d.logger != nil {
			d.logger.Info("ml-worker selected fallback new connection",
				slog.String("addr", d.mlWorkerCfg.FallbackService),
			)
		}
		return &d.fallbackConn, d.mlWorkerCfg.FallbackService, true, nil
	} else {
		if d.logger != nil {
			d.logger.Error("ml-worker fallback grpc connect failed",
				slog.String("addr", d.mlWorkerCfg.FallbackService),
				slog.Any("err", err),
			)
		}
		return nil, "", false, fmt.Errorf("primary unavailable and fallback connect failed: %w", err)
	}
}

func shouldReconnect(err error) bool {
	if err == nil {
		return false
	}

	s, ok := status.FromError(err)
	if !ok {
		return errors.Is(err, io.EOF) || errors.Is(err, errNotConnected)
	}

	switch s.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Internal, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

func (d *Activity) ReadTextFromPDF(ctx context.Context, payload *ReadTextFromPDFInput) (*ReadTextFromPDFOutput, error) {
	activity.RecordHeartbeat(ctx, "start")

	const maxAttempts = 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		activity.RecordHeartbeat(ctx, fmt.Sprintf("attempt=%d", attempt))

		connState, workerAddr, usedFallback, err := d.getActiveClient(ctx)
		if err != nil {
			lastErr = err
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
			continue
		}

		activity.RecordHeartbeat(ctx, fmt.Sprintf("worker_addr=%s", workerAddr))

		result, err := d.extractTextBidiOnce(ctx, connState, workerAddr, payload.FileData, payload.Filename)
		if err == nil {
			activity.RecordHeartbeat(ctx, "done")
			result.UsedFallback = usedFallback
			return result, nil
		}

		lastErr = err

		if shouldReconnect(err) {
			_ = d.closeConn(connState)
			continue
		}

		return nil, err
	}

	return nil, lastErr
}

func (d *Activity) extractTextBidiOnce(
	ctx context.Context,
	connState *mlWorkerConn,
	workerAddr string,
	pdf []byte,
	filename string,
) (*ReadTextFromPDFOutput, error) {
	client, _, gen := d.getConnClient(connState)
	if client == nil {
		return nil, errNotConnected
	}

	callCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	stream, err := client.ExtractTextBidi(callCtx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stream.CloseSend() }()

	stopCh := make(chan struct{})
	stop := func() {
		select {
		case <-stopCh:
		default:
			close(stopCh)
		}
	}
	defer stop()

	var lastPong atomic.Int64
	lastPong.Store(0)

	var lastPingSent atomic.Int64
	lastPingSent.Store(0)

	errCh := make(chan error, 1)
	resCh := make(chan *ReadTextFromPDFOutput, 1)

	var once sync.Once
	signalErr := func(e error) {
		if e == nil {
			return
		}
		once.Do(func() {
			stop()
			cancel()
			select {
			case errCh <- e:
			default:
			}
		})
	}
	signalRes := func(r *ReadTextFromPDFOutput) {
		once.Do(func() {
			stop()
			select {
			case resCh <- r:
			default:
			}
		})
	}

	go func(expectedGen uint64) {
		for {
			select {
			case <-stopCh:
				return
			default:
			}

			msg, rerr := stream.Recv()
			if rerr == io.EOF {
				signalErr(fmt.Errorf("ml-worker closed stream (EOF) before result"))
				return
			}
			if rerr != nil {
				signalErr(rerr)
				return
			}

			switch x := msg.Payload.(type) {
			case *ocrpb.ServerMsg_Ready:

			case *ocrpb.ServerMsg_Pong:
				lastPong.Store(time.Now().UnixNano())

			case *ocrpb.ServerMsg_Progress:
				activity.RecordHeartbeat(ctx, fmt.Sprintf("progress=%v worker=%s", x.Progress, workerAddr))

			case *ocrpb.ServerMsg_Result:
				signalRes(&ReadTextFromPDFOutput{
					Text:            x.Result.PagesText,
					ProcessDuration: time.Duration(x.Result.ProcessDurationMs) * time.Millisecond,
				})
				return

			case *ocrpb.ServerMsg_Error:
				signalErr(fmt.Errorf("ml-worker error addr=%s: %s", workerAddr, x.Error.Message))
				return
			}

			_, _, curGen := d.getConnClient(connState)
			if curGen != expectedGen {
				signalErr(fmt.Errorf("connection changed during stream addr=%s", workerAddr))
				return
			}
		}
	}(gen)

	hbTicker := time.NewTicker(heartbeatEvery)
	defer hbTicker.Stop()
	go func() {
		for {
			select {
			case <-stopCh:
				return
			case <-callCtx.Done():
				return
			case <-hbTicker.C:
				activity.RecordHeartbeat(ctx, fmt.Sprintf("running worker=%s", workerAddr))
			}
		}
	}()

	reqID := fmt.Sprintf("tworker-%d", time.Now().UnixNano())
	if err := stream.Send(&ocrpb.ClientMsg{
		Payload: &ocrpb.ClientMsg_Start{
			Start: &ocrpb.Start{
				RequestId:   reqID,
				Filename:    filename,
				ContentType: "application/pdf",
			},
		},
	}); err != nil {
		return nil, err
	}

	activity.RecordHeartbeat(ctx, fmt.Sprintf("sent_start worker=%s", workerAddr))

	pingTicker := time.NewTicker(pingEvery)
	defer pingTicker.Stop()
	go func() {
		for {
			select {
			case <-stopCh:
				return
			case <-callCtx.Done():
				return
			case <-pingTicker.C:
				lastPingSent.Store(time.Now().UnixNano())
				if err := stream.Send(&ocrpb.ClientMsg{
					Payload: &ocrpb.ClientMsg_Ping{
						Ping: &ocrpb.Ping{
							Id:   fmt.Sprintf("ping-%d", time.Now().UnixNano()),
							TsMs: time.Now().UnixMilli(),
						},
					},
				}); err != nil {
					signalErr(err)
					return
				}
			}
		}
	}()

	go func() {
		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-callCtx.Done():
				return
			case <-t.C:
				lp := lastPingSent.Load()
				if lp == 0 {
					continue
				}

				lpong := lastPong.Load()
				if lpong == 0 {
					if time.Since(time.Unix(0, lp)) > pongDeadline {
						signalErr(fmt.Errorf("pong timeout addr=%s: no first pong for %s", workerAddr, pongDeadline))
						return
					}
					continue
				}

				last := time.Unix(0, lpong)
				if time.Since(last) > pongDeadline {
					signalErr(fmt.Errorf("pong timeout addr=%s: no pong for %s", workerAddr, pongDeadline))
					return
				}
			}
		}
	}()

	r := bytes.NewReader(pdf)
	buf := make([]byte, pdfChunkSize)

	var sent int64

	for {
		select {
		case <-stopCh:
			goto WAIT_RESULT
		case <-callCtx.Done():
			goto WAIT_RESULT
		default:
		}

		n, rerr := r.Read(buf)
		if n > 0 {
			if err := stream.Send(&ocrpb.ClientMsg{
				Payload: &ocrpb.ClientMsg_Chunk{
					Chunk: &ocrpb.PdfChunk{Data: buf[:n]},
				},
			}); err != nil {
				return nil, err
			}
			sent += int64(n)
			activity.RecordHeartbeat(ctx, fmt.Sprintf("sent_bytes=%d worker=%s", sent, workerAddr))
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return nil, rerr
		}
	}

	select {
	case <-stopCh:
	case <-callCtx.Done():
	default:
		if err := stream.Send(&ocrpb.ClientMsg{
			Payload: &ocrpb.ClientMsg_End{End: &ocrpb.End{}},
		}); err != nil {
			return nil, err
		}
		activity.RecordHeartbeat(ctx, fmt.Sprintf("sent_end worker=%s", workerAddr))
	}

WAIT_RESULT:
	for {
		select {
		case result := <-resCh:
			activity.RecordHeartbeat(ctx, fmt.Sprintf("got_result worker=%s process_ms=%d", workerAddr, result.ProcessDuration.Milliseconds()))
			return result, nil

		case e := <-errCh:
			return nil, e

		case <-callCtx.Done():
			return nil, callCtx.Err()
		}
	}
}
