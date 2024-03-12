package awsfs

import (
	"context"
	"os"
)

func (s Server) Rename(ctx context.Context, oldPath, newPath string) error {
	if oldPath = slashClean(oldPath); oldPath == "/" {
		return os.ErrInvalid
	}
	if newPath = slashClean(newPath); newPath == "/" {
		return os.ErrInvalid
	}
	return os.Rename(oldPath, newPath)
}
