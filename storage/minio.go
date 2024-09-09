package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

type MinioProvider struct {
	minioAuthData
	client *minio.Client

	DefaultBucket string
	Path          string
}

type minioAuthData struct {
	url      string
	user     string
	password string
	// token    string
	ssl bool
}

func NewMinioProvider(minioURL string, minioUser string, minioPassword string, ssl bool, bucket, path string) (ArchiveStorage, error) {
	return &MinioProvider{
		DefaultBucket: bucket,
		Path:          path,
		minioAuthData: minioAuthData{
			password: minioPassword,
			url:      minioURL,
			user:     minioUser,
			ssl:      ssl,
		},
	}, nil
}

func (m *MinioProvider) Type() string {
	return "minio"
}

func (m *MinioProvider) Connect() error {
	var err error
	m.client, err = minio.New(m.url, &minio.Options{
		Creds:  credentials.NewStaticV4(m.user, m.password, ""),
		Secure: m.ssl,
	})
	if err != nil {
		return err
	}

	_ = m.client.MakeBucket(context.Background(),
		m.DefaultBucket,
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

	err = m.client.SetBucketLifecycle(context.Background(), m.DefaultBucket, config)
	if err != nil {
		fmt.Println(err)
	}
	return nil
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
	imageReader := bytes.NewReader(object.Payload)
	imageReaderSize := bytes.NewReader(object.Payload)

	bucket := m.DefaultBucket
	if object.Bucket != "" {
		bucket = object.Bucket
	}
	_, err := m.client.PutObject(
		ctx,
		bucket,
		fname,
		imageReader,
		imageReaderSize.Size(),
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
