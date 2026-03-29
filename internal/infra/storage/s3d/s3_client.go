package s3d

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewS3Client(endpoint, region, accessKey, secretKey string, usePathStyle bool) *s3.Client {
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey, secretKey, "",
		)),
	)
	if err != nil {
		panic(err)
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = usePathStyle
		o.Retryer = retry.AddWithMaxAttempts(retry.NewStandard(), 2)
	})

	return s3Client
}

func PingS3Client(ctx context.Context, s3Client *s3.Client) error {
	_, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	return err
}
