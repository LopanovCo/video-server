package storage

import (
	"context"
	"io"
)

type ArchiveUnit struct {
	Payload     io.Reader
	Bucket      string
	SegmentName string
}

type ArchiveStorage interface {
	Type() StorageType
	Connect() error
	MakeBucket(string) error
	UploadFile(context.Context, ArchiveUnit) (string, error)
	DownloadFile(context.Context, ArchiveUnit) ([]byte, error)
	RemoveFile(context.Context, ArchiveUnit) (string, error)
}
