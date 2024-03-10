package awsfs

import (
	"path"
	"path/filepath"
	"strings"
)

type Dir string

// slashClean is equivalent to but slightly more efficient than
// path.Clean("/" + name).
func slashClean(name string) string {
	if name == "" || name[0] != '/' {
		name = "/" + name
	}
	return path.Clean(name)
}

func resolve(root, name string) string {
	// This implementation is based on Dir.Open's code in the standard net/http package.
	if filepath.Separator != '/' && strings.IndexRune(name, filepath.Separator) >= 0 ||
		strings.Contains(name, "\x00") {
		return ""
	}
	if root == "" {
		root = "."
	}
	return filepath.Join(root, filepath.FromSlash(slashClean(name)))
}
