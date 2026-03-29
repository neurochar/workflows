package storage

import (
	"strings"

	"github.com/neurochar/workflows/internal/app/config"
)

type BucketName string

const (
	BucketCommonFiles BucketName = "neurochar"
)

func GetBucketURL(bucket BucketName, cfg *config.Config) string {
	var builder strings.Builder
	builder.WriteString(cfg.Storage.S3URL)
	builder.WriteString("/")
	builder.WriteString(string(bucket))

	return builder.String()
}
