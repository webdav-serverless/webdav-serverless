package awsfs

import (
	"context"
	"os"
	"path/filepath"
)

func (d Dir) RemoveAll(ctx context.Context, name string) error {
	if name = resolve(string(d), name); name == "" {
		return os.ErrNotExist
	}
	if name == filepath.Clean(string(d)) {
		// Prohibit removing the virtual root directory.
		return os.ErrInvalid
	}
	return os.RemoveAll(name)
}
