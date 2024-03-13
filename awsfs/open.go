package awsfs

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"time"

	"golang.org/x/net/webdav"
)

const referenceID = "root"

var ErrNotSupported = errors.New("not supported")

type FileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
	sys     any
}

func (f FileInfo) Name() string {
	return f.name
}

func (f FileInfo) Size() int64 {
	return f.size
}

func (f FileInfo) Mode() fs.FileMode {
	return f.mode
}

func (f FileInfo) ModTime() time.Time {
	return f.modTime
}

func (f FileInfo) IsDir() bool {
	return f.isDir
}

func (f FileInfo) Sys() any {
	return f.sys
}

func (s Server) OpenFile(ctx context.Context, path string, flag int, perm os.FileMode) (webdav.File, error) {
	if path = slashClean(path); path == "/" {
		return nil, os.ErrExist
	}
	if flag == os.O_RDONLY {
		return s.openFileReader(ctx, path, flag, perm)
	}
	return s.openFileWriter(ctx, path, flag, perm)
}
