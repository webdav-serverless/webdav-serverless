package awsfs

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

func (s Server) Mkdir(ctx context.Context, path string, perm os.FileMode) error {

	if path = slashClean(path); path == "/" {
		return os.ErrExist
	}

	ref, err := s.MetadataStore.GetReference(ctx, referenceID)
	if err != nil {
		return err
	}

	_, ok := ref.Entries[path]
	if ok {
		return os.ErrExist
	}

	parentDirPath := filepath.Dir(path)

	parentID, ok := ref.Entries[parentDirPath]
	if !ok {
		return os.ErrNotExist
	}

	newEntry := Entry{
		ID:       uuid.New().String(),
		ParentID: parentID,
		Name:     filepath.Base(path),
		Type:     EntryTypeDir,
		Size:     0,
		Modify:   time.Now(),
		Version:  1,
	}
	err = s.MetadataStore.AddEntry(ctx, newEntry, path)
	if err != nil {
		return err
	}

	return nil
}
