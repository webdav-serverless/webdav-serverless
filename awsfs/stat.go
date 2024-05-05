package awsfs

import (
	"context"
	"os"
)

func (s *Server) Stat(ctx context.Context, path string) (os.FileInfo, error) {

	path = slashClean(path)

	ref, err := s.MetadataStore.GetReference(ctx, referenceID)
	if err != nil {
		return nil, err
	}

	id, ok := ref.Entries[path]
	if !ok {
		return nil, os.ErrNotExist
	}

	entry, err := s.MetadataStore.GetEntry(ctx, id)
	if err != nil {
		return nil, err
	}

	info := FileInfo{
		name:    entry.Name,
		size:    entry.Size,
		modTime: entry.Modify,
		isDir:   entry.IsDir(),
	}

	return info, nil
}
