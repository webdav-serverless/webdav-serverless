package awsfs

import (
	"context"
	"os"

	"golang.org/x/net/webdav"
)

func (s Server) OpenFile(ctx context.Context, path string, flag int, perm os.FileMode) (webdav.File, error) {
	if path = slashClean(path); path == "/" {
		return nil, os.ErrExist
	}
	f, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return nil, err
	}
	return f, nil
}
