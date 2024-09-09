package minio

import (
	"context"
	"io"
)

type ImageUnit struct {
	Payload     io.Reader
	Bucket      string
	SegmentName string
}

type ImageStorage interface {
	Type() string
	Connect() error
	MakeBucket(string) error
	UploadFile(context.Context, ImageUnit) (string, error)
	DownloadFile(context.Context, ImageUnit) ([]byte, error)
	RemoveFile(context.Context, ImageUnit) (string, error)
}
