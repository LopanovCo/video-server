package storage

import (
	"context"
)

type ArchiveUnit struct {
	Payload     []byte
	Bucket      string
	SegmentName string
}

type ArchiveStorage interface {
	Type() string
	Connect() error
	MakeBucket(string) error
	UploadFile(context.Context, ArchiveUnit) (string, error)
	DownloadFile(context.Context, ArchiveUnit) ([]byte, error)
	RemoveFile(context.Context, ArchiveUnit) (string, error)
}
