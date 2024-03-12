package awsfs

import (
	"context"
	"os"
)

func (s Server) RemoveAll(ctx context.Context, path string) error {
	if path = slashClean(path); path == "/" {
		return os.ErrInvalid
	}
	return os.RemoveAll(path)
}
