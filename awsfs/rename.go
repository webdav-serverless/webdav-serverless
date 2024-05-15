package awsfs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) Rename(ctx context.Context, oldPath, newPath string) error {
	if oldPath = slashClean(oldPath); oldPath == "/" {
		return os.ErrInvalid
	}
	if newPath = slashClean(newPath); newPath == "/" {
		return os.ErrInvalid
	}

	ref, err := s.MetadataStore.GetReference(ctx, referenceID)
	if err != nil {
		return err
	}

	id, ok := ref.Entries[oldPath]
	if !ok {
		return os.ErrNotExist
	}
	parentDirPath := filepath.Dir(newPath)
	parentID, ok := ref.Entries[parentDirPath]
	if !ok {
		return os.ErrNotExist
	}

	entry, err := s.MetadataStore.GetEntry(ctx, id)
	if err != nil {
		return err
	}
	entry.Name = filepath.Base(newPath)
	entry.ParentID = parentID

	ref.Entries[newPath] = id
	delete(ref.Entries, oldPath)

	if entry.Type == EntryTypeDir {
		for k, v := range ref.Entries {
			if strings.HasPrefix(k, oldPath+"/") {
				delete(ref.Entries, k)
				newPath := strings.Replace(k, oldPath, newPath, 1)
				ref.Entries[newPath] = v
			}
		}
	}

	err = s.MetadataStore.UpdateEntryName(ctx, entry, ref)
	if err != nil {
		return err
	}

	return nil
}
