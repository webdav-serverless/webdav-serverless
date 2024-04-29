package awsfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

func (s Server) Create(ctx context.Context, path string, flag int, perm os.FileMode, r io.Reader) (os.FileInfo, error) {

	fmt.Println("Create:", path, flag, perm)

	if path = slashClean(path); path == "" {
		return nil, os.ErrInvalid
	}

	ref, err := s.MetadataStore.GetReference(ctx, referenceID)
	if err != nil {
		return nil, err
	}

	entryID, shouldUpdate := ref.Entries[path]
	if !shouldUpdate {
		entryID = uuid.New().String()
	}

	sr := &sizingReader{Reader: r}

	err = s.PhysicalStore.PutObjectLarge(ctx, entryID, sr)
	if err != nil {
		return nil, err
	}

	if shouldUpdate {
		entry, err := s.MetadataStore.GetEntry(ctx, entryID)
		if err != nil {
			return nil, err
		}
		entry.Size = sr.size
		entry.Modify = time.Now()
		err = s.MetadataStore.UpdateEntry(ctx, entry)
		if err != nil {
			return nil, err
		}
		return &FileInfo{
			name:    entry.Name,
			size:    entry.Size,
			mode:    perm,
			modTime: entry.Modify,
			isDir:   false,
			sys:     nil,
		}, nil
	} else {
		parentDirPath := filepath.Dir(path)
		parentID, ok := ref.Entries[parentDirPath]
		if !ok {
			return nil, os.ErrNotExist
		}
		newEntry := Entry{
			ID:       entryID,
			ParentID: parentID,
			Name:     filepath.Base(path),
			Type:     EntryTypeFile,
			Size:     sr.size,
			Modify:   time.Now(),
			Version:  1,
		}
		err = s.MetadataStore.AddEntry(ctx, newEntry, path)
		if err != nil {
			return nil, err
		}
		return &FileInfo{
			name:    newEntry.Name,
			size:    newEntry.Size,
			mode:    perm,
			modTime: newEntry.Modify,
			isDir:   false,
			sys:     nil,
		}, nil
	}
}

type sizingReader struct {
	io.Reader
	size int64
}

func (r *sizingReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.size += int64(n)
	return
}
