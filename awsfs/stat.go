package awsfs

import (
	"context"
	"os"
)

func (s Server) Stat(ctx context.Context, path string) (os.FileInfo, error) {
	if path = slashClean(path); path == "/" {
		return nil, os.ErrInvalid
	}
	return os.Stat(path)
}
