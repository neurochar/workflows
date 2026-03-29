package storage

import (
	"context"
	"io"
	"time"
)

type UploadInput struct {
	Key         string
	ContentType string
	Body        io.Reader
	Size        int64
	Metadata    map[string]string
}

type UploadBytesInput struct {
	Key         string
	Hash        string
	ContentType string
	Data        []byte
	Metadata    map[string]string
}

type DownloadOutput struct {
	ContentType string
	ContentLen  int64
	Body        io.ReadCloser
	Metadata    map[string]string
}

type Client interface {
	Upload(ctx context.Context, bucket BucketName, input UploadInput) (hash string, err error)
	UploadWithMultipart(ctx context.Context, bucket BucketName, input UploadInput) (hash string, err error)
	UploadBytes(ctx context.Context, bucket BucketName, input UploadBytesInput) (hash string, err error)
	Download(ctx context.Context, bucket BucketName, key string) (*DownloadOutput, error)
	Delete(ctx context.Context, bucket BucketName, key string) error
	Exists(ctx context.Context, bucket BucketName, key string) (bool, error)

	CreateBucket(ctx context.Context, bucket BucketName, policy string) error
	DeleteBucket(ctx context.Context, bucket BucketName) error

	FileMetaByBytes(ctx context.Context, fileName string, data []byte) (key string, hash string, mimeType string, ext string)

	UploadFileByReader(
		ctx context.Context,
		bucket BucketName,
		fileName string,
		r io.Reader,
		metadata map[string]string,
	) (key string, hash string, mimeType string, err error)

	UploadFileByReaderWithMultipart(
		ctx context.Context,
		bucket BucketName,
		fileName string,
		r io.Reader,
		metadata map[string]string,
	) (key string, hash string, mimeType string, err error)

	UploadFileByBytes(
		ctx context.Context,
		bucket BucketName,
		fileName string,
		data []byte,
		metadata map[string]string,
	) (key string, hash string, mimeType string, err error)

	PresignGetObject(
		ctx context.Context,
		bucket BucketName,
		key string,
		filename string,
		isAttachment bool,
		ttl time.Duration,
	) (string, error)
}
