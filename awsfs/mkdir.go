package awsfs

import (
	"context"
	"os"
)

func (d Dir) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	if name = resolve(string(d), name); name == "" {
		return os.ErrNotExist
	}
	return os.Mkdir(name, perm)
}
