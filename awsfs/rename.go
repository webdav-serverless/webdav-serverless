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

	ref, err := s.MetadataStore.GetReference(ctx, referenceID)
	if err != nil {
		return err
	}

	id, ok := ref.Entries[oldPath]
	if ok {
		return os.ErrNotExist
	}

	entry, err := s.MetadataStore.GetEntry(ctx, id)
	if err != nil {
		return err
	}

	ref.Entries[newPath] = id
	delete(ref.Entries, oldPath)
	entry.Name = newPath

	err = s.MetadataStore.UpdateEntryName(ctx, entry, ref)
	if err != nil {
		return err
	}

	return nil
}
