package blob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/ncecere/open_model_gateway/backend/internal/config"
)

type s3Store struct {
	client *s3.Client
	bucket string
	prefix string
}

func newS3Store(cfg config.FilesConfig, awsCfg aws.Config) (*s3Store, error) {
	if cfg.S3.Bucket == "" {
		return nil, fmt.Errorf("files.s3.bucket must be provided for s3 storage")
	}
	client := s3.NewFromConfig(applyEndpointOverrides(awsCfg, cfg.S3))
	prefix := strings.Trim(cfg.S3.Prefix, "/")
	return &s3Store{client: client, bucket: cfg.S3.Bucket, prefix: prefix}, nil
}

func applyEndpointOverrides(base aws.Config, s3cfg config.FilesS3Config) aws.Config {
	if s3cfg.Endpoint == "" && s3cfg.Region != "" {
		base.Region = s3cfg.Region
		return base
	}
	if s3cfg.Region != "" {
		base.Region = s3cfg.Region
	}
	if s3cfg.Endpoint == "" {
		return base
	}
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID && region == base.Region {
			return aws.Endpoint{
				PartitionID:       "aws",
				URL:               s3cfg.Endpoint,
				Source:            aws.EndpointSourceCustom,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
	})
	base.EndpointResolverWithOptions = resolver
	return base
}

func loadS3Config(ctx context.Context, cfg config.FilesConfig) (aws.Config, error) {
	opts := []func(*awscfg.LoadOptions) error{}
	if cfg.S3.Region != "" {
		opts = append(opts, awscfg.WithRegion(cfg.S3.Region))
	}
	if cfg.S3.Endpoint != "" {
		resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == s3.ServiceID {
				return aws.Endpoint{URL: cfg.S3.Endpoint, Source: aws.EndpointSourceCustom, HostnameImmutable: true}, nil
			}
			return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
		})
		opts = append(opts, awscfg.WithEndpointResolverWithOptions(resolver))
	}
	return awscfg.LoadDefaultConfig(ctx, opts...)
}

func (s *s3Store) Put(ctx context.Context, key string, body io.Reader, opts PutOptions) (ObjectInfo, error) {
	objectKey := s.objectKey(key)
	input := &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &objectKey,
		Body:        body,
		ContentType: aws.String(opts.ContentType),
		Metadata:    opts.Metadata,
	}
	if _, err := s.client.PutObject(ctx, input); err != nil {
		return ObjectInfo{}, err
	}
	return ObjectInfo{Key: key, ContentType: opts.ContentType, Metadata: opts.Metadata}, nil
}

func (s *s3Store) Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	objectKey := s.objectKey(key)
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &objectKey,
	})
	if err != nil {
		var nf *s3types.NoSuchKey
		if strings.Contains(err.Error(), "NoSuchKey") || errors.As(err, &nf) {
			return nil, ObjectInfo{}, ErrNotFound
		}
		return nil, ObjectInfo{}, err
	}
	info := ObjectInfo{Key: key, ContentType: aws.ToString(out.ContentType), Metadata: out.Metadata}
	return out.Body, info, nil
}

func (s *s3Store) Delete(ctx context.Context, key string) error {
	objectKey := s.objectKey(key)
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &objectKey,
	})
	return err
}

func (s *s3Store) objectKey(key string) string {
	if s.prefix == "" {
		return key
	}
	return s.prefix + "/" + strings.TrimPrefix(key, "/")
}
