package s3d

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/neurochar/workflows/internal/infra/storage"
)

const (
	maxExtLength     = 31
	minMimeBufLength = 1024 * 8
)

type s3Client struct {
	svc                       *s3.Client
	presignClient             *s3.PresignClient
	miltipartUploadPartSize   int64
	miltipartUploaConcurrency int
}

func New(client *s3.Client) *s3Client {
	return &s3Client{
		svc:                       client,
		presignClient:             s3.NewPresignClient(client),
		miltipartUploadPartSize:   5 * 1024 * 1024,
		miltipartUploaConcurrency: 4,
	}
}

func (c *s3Client) Upload(ctx context.Context, bucket storage.BucketName, input storage.UploadInput) (string, error) {
	h := sha256.New()
	body := io.TeeReader(input.Body, h)
	_, err := c.svc.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(string(bucket)),
		Key:         aws.String(input.Key),
		Body:        body,
		ContentType: aws.String(input.ContentType),
		Metadata:    input.Metadata,
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (c *s3Client) UploadWithMultipart(
	ctx context.Context,
	bucket storage.BucketName,
	input storage.UploadInput,
) (string, error) {
	h := sha256.New()
	tee := io.TeeReader(input.Body, h)

	uploader := manager.NewUploader(c.svc, func(u *manager.Uploader) {
		u.PartSize = c.miltipartUploadPartSize
		u.Concurrency = c.miltipartUploaConcurrency
	})

	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(string(bucket)),
		Key:         aws.String(input.Key),
		Body:        tee,
		ContentType: aws.String(input.ContentType),
		Metadata:    input.Metadata,
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (c *s3Client) UploadBytes(ctx context.Context, bucket storage.BucketName, input storage.UploadBytesInput) (string, error) {
	if input.Hash == "" {
		sum := sha256.Sum256(input.Data)
		input.Hash = hex.EncodeToString(sum[:])
	}
	_, err := c.svc.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(string(bucket)),
		Key:         aws.String(input.Key),
		Body:        bytes.NewReader(input.Data),
		ContentType: aws.String(input.ContentType),
		Metadata:    input.Metadata,
	})
	return input.Hash, err
}

func (c *s3Client) Download(ctx context.Context, bucket storage.BucketName, key string) (*storage.DownloadOutput, error) {
	out, err := c.svc.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(string(bucket)),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return &storage.DownloadOutput{
		ContentType: aws.ToString(out.ContentType),
		ContentLen:  *out.ContentLength,
		Body:        out.Body,
		Metadata:    out.Metadata,
	}, nil
}

func (c *s3Client) Delete(ctx context.Context, bucket storage.BucketName, key string) error {
	_, err := c.svc.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(string(bucket)),
		Key:    aws.String(key),
	})
	return err
}

func (c *s3Client) Exists(ctx context.Context, bucket storage.BucketName, key string) (bool, error) {
	_, err := c.svc.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(string(bucket)),
		Key:    aws.String(key),
	})
	if err != nil {
		var nf *s3types.NotFound
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *s3Client) CreateBucket(ctx context.Context, bucket storage.BucketName, policy string) error {
	_, err := c.svc.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(string(bucket))})
	if err != nil {
		var respErr *awshttp.ResponseError
		if errors.As(err, &respErr) && respErr.HTTPStatusCode() == http.StatusConflict {
			return storage.ErrBucketAlreadyExists
		}

		return err
	}

	if policy != "" {
		_, err = c.svc.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
			Bucket: aws.String(string(bucket)),
			Policy: aws.String(policy),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *s3Client) DeleteBucket(ctx context.Context, bucket storage.BucketName) error {
	_, err := c.svc.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(string(bucket))})
	return err
}

func (c *s3Client) FileMetaByBytes(ctx context.Context, fileName string, data []byte) (string, string, string, string) {
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	prefix := hash[:2] + "/" + hash[2:4] + "/"
	mime, ext := storage.DetectMimeByBytes8KB(data)
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(fileName))
	}
	if len(ext) > maxExtLength {
		ext = ext[:maxExtLength]
	}
	key := prefix + hash[4:] + ext
	return key, hash, mime, ext
}

func (c *s3Client) UploadFileByBytes(
	ctx context.Context,
	bucket storage.BucketName,
	fileName string,
	data []byte,
	metadata map[string]string,
) (string, string, string, error) {
	key, hash, mime, _ := c.FileMetaByBytes(ctx, fileName, data)

	_, err := c.UploadBytes(ctx, bucket, storage.UploadBytesInput{
		Key:         key,
		ContentType: mime,
		Data:        data,
		Metadata:    metadata,
	})
	return key, hash, mime, err
}

func (c *s3Client) UploadFileByReader(
	ctx context.Context,
	bucket storage.BucketName,
	fileName string,
	r io.Reader,
	metadata map[string]string,
) (string, string, string, error) {
	buf := make([]byte, minMimeBufLength)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return "", "", "", err
	}

	mime, ext := storage.DetectMimeByBytes8KB(buf[:n])
	h := sha256.New()
	reader := io.MultiReader(bytes.NewReader(buf[:n]), r)
	tee := io.TeeReader(reader, h)
	all, err := io.ReadAll(tee)
	if err != nil {
		return "", "", "", err
	}
	hash := hex.EncodeToString(h.Sum(nil))
	prefix := hash[:2] + "/" + hash[2:4] + "/"
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(fileName))
	}
	if len(ext) > maxExtLength {
		ext = ext[:maxExtLength]
	}
	key := prefix + hash[4:] + ext
	_, err = c.UploadBytes(ctx, bucket, storage.UploadBytesInput{
		Key:         key,
		ContentType: mime,
		Data:        all,
		Metadata:    metadata,
	})
	return key, hash, mime, err
}

func (c *s3Client) UploadFileByReaderWithMultipart(
	ctx context.Context,
	bucket storage.BucketName,
	fileName string,
	r io.Reader,
	metadata map[string]string,
) (string, string, string, error) {
	buf := make([]byte, minMimeBufLength)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return "", "", "", err
	}

	mime, ext := storage.DetectMimeByBytes8KB(buf[:n])
	h := sha256.New()
	stream := io.MultiReader(bytes.NewReader(buf[:n]), r)
	tee := io.TeeReader(stream, h)

	uploader := manager.NewUploader(c.svc, func(u *manager.Uploader) {
		u.PartSize = c.miltipartUploadPartSize
		u.Concurrency = c.miltipartUploaConcurrency
	})
	tmpKey := "tmp_" + fileName + "_" + time.Now().Format("20060102150405")

	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(string(bucket)),
		Key:         aws.String(tmpKey),
		Body:        tee,
		ContentType: aws.String(mime),
		Metadata:    metadata,
	})
	if err != nil {
		return "", "", "", err
	}

	hash := hex.EncodeToString(h.Sum(nil))
	prefix := hash[:2] + "/" + hash[2:4] + "/"
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(fileName))
	}
	if len(ext) > maxExtLength {
		ext = ext[:maxExtLength]
	}
	finalKey := prefix + hash[4:] + ext

	if finalKey != tmpKey {
		_, err = c.svc.CopyObject(ctx, &s3.CopyObjectInput{
			Bucket:            aws.String(string(bucket)),
			Key:               aws.String(finalKey),
			CopySource:        aws.String(string(bucket) + "/" + tmpKey),
			MetadataDirective: s3types.MetadataDirectiveReplace,
			ContentType:       aws.String(mime),
			Metadata:          metadata,
		})
		if err != nil {
			return "", "", "", err
		}

		_, err = c.svc.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(string(bucket)),
			Key:    aws.String(tmpKey),
		})
		if err != nil {
			return "", "", "", err
		}
	}

	return finalKey, hash, mime, nil
}

func (c *s3Client) PresignGetObject(
	ctx context.Context,
	bucket storage.BucketName,
	key string,
	filename string,
	isAttachment bool,
	ttl time.Duration,
) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(string(bucket)),
		Key:    aws.String(key),
	}

	if filename != "" {
		if isAttachment {
			input.ResponseContentDisposition = aws.String("attachment; filename=\"" + filename + "\"")
		} else {
			input.ResponseContentDisposition = aws.String("inline; filename=\"" + filename + "\"")
		}
	}

	req, err := c.presignClient.PresignGetObject(ctx, input, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}

	return req.URL, nil
}
