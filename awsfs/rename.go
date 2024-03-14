package awsfs

import (
	"context"
	"fmt"
	"os"
)

func (s Server) Rename(ctx context.Context, oldPath, newPath string) error {
	fmt.Println("Rename: ", oldPath, " ->", newPath)

	if oldPath = slashClean(oldPath); oldPath == "/" {
		return os.ErrInvalid
	}
	if newPath = slashClean(newPath); newPath == "/" {
		return os.ErrInvalid
	}

	ref, err := s.MetadataStore.GetReference(ctx, referenceID)
	if err != nil {
		fmt.Println("Rename1: ", err)
		return err
	}

	id, ok := ref.Entries[oldPath]
	if !ok {
		fmt.Println("Rename2: ErrNotExist")
		return os.ErrNotExist
	}

	entry, err := s.MetadataStore.GetEntry(ctx, id)
	if err != nil {
		fmt.Println("Rename3: ", err)
		return err
	}

	ref.Entries[newPath] = id
	delete(ref.Entries, oldPath)
	entry.Name = newPath

	err = s.MetadataStore.UpdateEntryName(ctx, entry, ref)
	if err != nil {
		fmt.Println("Rename3: UpdateEntryName", err)
		return err
	}

	return nil
}
