package worker

import (
	"log/slog"

	tclient "go.temporal.io/sdk/client"
)

type WorkerClient tclient.Client

func NewClient(endpoint string, namespace string, logger *slog.Logger) (WorkerClient, error) {
	c, err := tclient.NewLazyClient(tclient.Options{
		HostPort:  endpoint,
		Logger:    logger.WithGroup("temporal_worker"),
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}
