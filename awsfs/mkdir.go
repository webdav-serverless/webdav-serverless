package awsfs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

func (s Server) Mkdir(ctx context.Context, path string, perm os.FileMode) error {
	fmt.Println("Mkdir: ", path)

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

	parentDirPath := GetParentDirPath(path)

	parentID, ok := ref.Entries[parentDirPath]
	if !ok {
		return errors.New("no such parent directory")
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
	newRef := ref
	newRef.Entries[path] = newEntry.ID
	err = s.MetadataStore.AddEntry(ctx, newEntry, newRef)
	if err != nil {
		return err
	}

	return nil
}

func GetParentDirPath(path string) string {
	return filepath.Dir(path)
}
