package awsfs

import (
	"context"
	"os"
)

func (d Dir) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	if name = resolve(string(d), name); name == "" {
		return nil, os.ErrNotExist
	}
	return os.Stat(name)
}
