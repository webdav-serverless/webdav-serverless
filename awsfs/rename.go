package awsfs

import (
	"context"
	"os"
	"path/filepath"
)

func (d Dir) Rename(ctx context.Context, oldName, newName string) error {
	if oldName = resolve(string(d), oldName); oldName == "" {
		return os.ErrNotExist
	}
	if newName = resolve(string(d), newName); newName == "" {
		return os.ErrNotExist
	}
	if root := filepath.Clean(string(d)); root == oldName || root == newName {
		// Prohibit renaming from or to the virtual root directory.
		return os.ErrInvalid
	}
	return os.Rename(oldName, newName)
}
