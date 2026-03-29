package storage

import (
	"context"
	"io"
	"log/slog"

	"github.com/neurochar/workflows/internal/infra/storage"
	"go.temporal.io/sdk/activity"
)

type Activity struct {
	s3Client storage.Client
	logger   *slog.Logger
}

func New(s3Client storage.Client, logger *slog.Logger) *Activity {
	return &Activity{
		s3Client: s3Client,
		logger:   logger,
	}
}

type GetFileInput struct {
	Bucket storage.BucketName
	Key    string
}

type GetFileOutput struct {
	Data []byte
}

func (d *Activity) GetFile(ctx context.Context, payload *GetFileInput) (*GetFileOutput, error) {
	activity.RecordHeartbeat(ctx, "start")

	file, err := d.s3Client.Download(ctx, payload.Bucket, payload.Key)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Body.Close()
	}()

	activity.RecordHeartbeat(ctx, "downloaded")

	fileData, err := io.ReadAll(file.Body)
	if err != nil {
		return nil, err
	}

	return &GetFileOutput{
		Data: fileData,
	}, nil
}
