package storage

import (
	"context"
)

type ImageUnit struct {
	Payload     []byte
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
