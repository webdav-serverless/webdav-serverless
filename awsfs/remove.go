package awsfs

import (
	"context"
	"os"
	"strings"
)

func (s *Server) RemoveAll(ctx context.Context, path string) error {
	if path = slashClean(path); path == "/" {
		return os.ErrInvalid
	}

	ref, err := s.MetadataStore.GetReference(ctx, referenceID)
	if err != nil {
		return err
	}

	_, ok := ref.Entries[path]
	if !ok {
		return os.ErrNotExist
	}
	var ids []string
	for k, v := range ref.Entries {

		if k == path || strings.HasPrefix(k, path+"/") {
			delete(ref.Entries, k)
			ids = append(ids, v)
		}
	}

	err = s.MetadataStore.DeleteEntries(ctx, ids, ref)
	if err != nil {
		return err
	}

	return nil
}
