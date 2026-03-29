package personal_data_remover

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/neurochar/workflows/internal/app/config"
	"go.temporal.io/sdk/activity"
)

type Activity struct {
	logger      *slog.Logger
	mlWorkerCfg MlWorkersConfig
	httpClient  *http.Client
}

type MlWorkersConfig struct {
	Service   string
	Readiness string
}

type AnonymizeInput struct {
	Text     string
	Language string
}

type AnonymizeOutput struct {
	AnonymizedText string
}

func New(cfg config.Config, logger *slog.Logger) *Activity {
	return &Activity{
		logger: logger,
		mlWorkerCfg: MlWorkersConfig{
			Service:   cfg.Workers.PDRemover.Service,
			Readiness: cfg.Workers.PDRemover.Readiness,
		},
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

type anonymizeRequest struct {
	Text        string `json:"text"`
	Language    string `json:"language"`
	ReturnStats bool   `json:"return_stats"`
}

type anonymizeResponse struct {
	AnonymizedText string `json:"anonymized_text"`
}

func (d *Activity) isServiceReady(ctx context.Context) bool {
	if d.mlWorkerCfg.Readiness == "" {
		return true
	}

	reqCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, d.mlWorkerCfg.Readiness, nil)
	if err != nil {
		if d.logger != nil {
			d.logger.Warn("pd-remover readiness request build failed",
				slog.String("url", d.mlWorkerCfg.Readiness),
				slog.Any("err", err),
			)
		}
		return false
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		if d.logger != nil {
			d.logger.Warn("pd-remover readiness check failed",
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
		d.logger.Warn("pd-remover readiness bad status",
			slog.String("url", d.mlWorkerCfg.Readiness),
			slog.Int("status_code", resp.StatusCode),
		)
	}

	return false
}

func (d *Activity) AnonymizeText(ctx context.Context, payload *AnonymizeInput) (*AnonymizeOutput, error) {
	activity.RecordHeartbeat(ctx, "start")

	lang := payload.Language
	if lang == "" {
		lang = "ru"
	}

	const maxAttempts = 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		activity.RecordHeartbeat(ctx, fmt.Sprintf("attempt=%d", attempt))

		if !d.isServiceReady(ctx) {
			lastErr = fmt.Errorf("pd-remover not ready at %s", d.mlWorkerCfg.Readiness)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
			continue
		}

		activity.RecordHeartbeat(ctx, "service_ready")

		result, err := d.callAnonymize(ctx, payload.Text, lang)
		if err == nil {
			activity.RecordHeartbeat(ctx, "done")
			return result, nil
		}

		lastErr = err

		if isRetryable(err) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
			continue
		}

		return nil, err
	}

	return nil, lastErr
}

func (d *Activity) callAnonymize(ctx context.Context, text string, language string) (*AnonymizeOutput, error) {
	reqBody := anonymizeRequest{
		Text:        text,
		Language:    language,
		ReturnStats: false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("json marshal request: %w", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, d.mlWorkerCfg.Service+"/anonymize", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("http request build: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http call pd-remover: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("pd-remover returned status=%d body=%s", resp.StatusCode, string(respBytes))
	}

	var respData anonymizeResponse
	if err := json.Unmarshal(respBytes, &respData); err != nil {
		return nil, fmt.Errorf("json unmarshal response: %w", err)
	}

	return &AnonymizeOutput{
		AnonymizedText: respData.AnonymizedText,
	}, nil
}

func isRetryable(err error) bool {
	return true
}
