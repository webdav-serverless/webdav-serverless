package awsfs

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type SizeReader struct {
	io.Reader
	Size int64
}

func (r *SizeReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.Size += int64(n)
	return
}

func (s Server) Create(ctx context.Context, path string, flag int, perm os.FileMode, r io.Reader) (os.FileInfo, error) {

	ref, err := s.MetadataStore.GetReference(ctx, referenceID)
	if err != nil {
		return nil, err
	}

	//_, ok := ref.Entries[path]
	//if ok {
	//	return nil, os.ErrExist
	//}

	parentDirPath := GetParentDirPath(path)

	parentID, ok := ref.Entries[parentDirPath]
	if !ok {
		return nil, errors.New("no such parent directory")
	}

	sr := &SizeReader{Reader: r}

	entryID := uuid.New().String()

	err = s.PhysicalStore.PutObjectLarge(ctx, entryID, sr)
	if err != nil {
		return nil, err
	}

	newEntry := Entry{
		ID:       entryID,
		ParentID: parentID,
		Name:     filepath.Base(path),
		Type:     EntryTypeFile,
		Size:     sr.Size,
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
