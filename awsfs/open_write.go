package awsfs

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/webdav"
)

func (s Server) openFileWriter(ctx context.Context, path string, flag int, perm os.FileMode) (webdav.File, error) {
	ref, err := s.MetadataStore.GetReference(ctx, referenceID)
	if err != nil {
		return nil, err
	}

	_, ok := ref.Entries[path]
	if ok {
		return nil, os.ErrExist
	}

	parentDirPath := GetParentDirPath(path)

	parentID, ok := ref.Entries[parentDirPath]
	if !ok {
		return nil, errors.New("no such parent directory")
	}

	entryID := uuid.New().String()
	buf := bytes.Buffer{}
	errChan := make(chan error)
	w := &FileWriter{
		Buffer: buf,
		close: func() (fs.FileInfo, error) {
			newEntry := Entry{
				ID:       entryID,
				ParentID: parentID,
				Name:     filepath.Base(path),
				Type:     EntryTypeFile,
				Size:     int64(buf.Len()),
				Modify:   time.Now(),
				Version:  1,
			}
			newRef := ref
			newRef.Entries[path] = newEntry.ID
			err := s.MetadataStore.AddEntry(ctx, newEntry, newRef)
			if err != nil {
				return nil, err
			}
			return FileInfo{
				name:    newEntry.Name,
				size:    newEntry.Size,
				mode:    perm,
				modTime: newEntry.Modify,
				isDir:   false,
				sys:     nil,
			}, nil
		},
		errChan: errChan,
	}
	go func() {
		err = s.PhysicalStore.PutObjectLarge(ctx, entryID, w)
		if err != nil {
			errChan <- err
		}
	}()
	return w, nil
}

type FileWriter struct {
	bytes.Buffer
	close   func() (fs.FileInfo, error)
	errChan chan error
	stat    fs.FileInfo
}

func (f FileWriter) Close() error {
	stat, err := f.close()
	if err != nil {
		return err
	}
	f.stat = stat
	return nil
}

func (f FileWriter) Read(p []byte) (n int, err error) {
	select {
	case e := <-f.errChan:
		return 0, e
	default:
	}
	return f.Read(p)
}

func (f FileWriter) Seek(offset int64, whence int) (int64, error) {
	return 0, ErrNotSupported
}

func (f FileWriter) Readdir(count int) ([]fs.FileInfo, error) {
	return nil, ErrNotSupported
}

func (f FileWriter) Stat() (fs.FileInfo, error) {
	return f.stat, nil
}

func (f FileWriter) Write(p []byte) (n int, err error) {
	return f.Buffer.Write(p)
}
