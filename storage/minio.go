package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

type MinioProvider struct {
	client *minio.Client

	DefaultBucket string
	Path          string
}

func NewMinioProvider(client *minio.Client, bucket, path string) (ArchiveStorage, error) {
	return &MinioProvider{
		client:        client,
		DefaultBucket: bucket,
		Path:          path,
	}, nil
}

func (m *MinioProvider) Type() StorageType {
	return STORAGE_MINIO
}

func (m *MinioProvider) MakeBucket(bucket string) error {
	_ = m.client.MakeBucket(context.Background(),
		bucket,
		minio.MakeBucketOptions{
			ObjectLocking: true,
		})
	config := lifecycle.NewConfiguration()
	config.Rules = []lifecycle.Rule{
		{
			ID:     "expire-bucket",
			Status: "Enabled",
			Expiration: lifecycle.Expiration{
				Days: 2,
			},
		},
	}

	_ = m.client.SetBucketLifecycle(context.Background(), bucket, config)
	return nil
}

func (m *MinioProvider) UploadFile(ctx context.Context, object ArchiveUnit) (string, error) {
	fname := fmt.Sprintf("%s/%s", m.Path, object.SegmentName)

	buf := &bytes.Buffer{}

	size, err := io.Copy(buf, object.Payload)
	if err != nil {
		return "", err
	}
	bucket := m.DefaultBucket
	if object.Bucket != "" {
		bucket = object.Bucket
	}
	_, err = m.client.PutObject(
		ctx,
		bucket,
		fname,
		buf,
		size,
		minio.PutObjectOptions{
			ContentType: "application/octet-stream",
		},
	)
	return object.SegmentName, err
}

func (m *MinioProvider) DownloadFile(ctx context.Context, object ArchiveUnit) ([]byte, error) {
	fname := fmt.Sprintf("%s/%s", m.Path, object.SegmentName)
	bucket := m.DefaultBucket
	if object.Bucket != "" {
		bucket = object.Bucket
	}
	reader, err := m.client.GetObject(
		ctx,
		bucket,
		fname,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	imgBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return imgBytes, nil
}

func (m *MinioProvider) RemoveFile(ctx context.Context, object ArchiveUnit) (string, error) {
	fname := fmt.Sprintf("%s/%s", m.Path, object.SegmentName)
	bucket := m.DefaultBucket
	if object.Bucket != "" {
		bucket = object.Bucket
	}
	err := m.client.RemoveObject(
		ctx,
		bucket,
		fname,
		minio.RemoveObjectOptions{},
	)
	return object.SegmentName, err
}
