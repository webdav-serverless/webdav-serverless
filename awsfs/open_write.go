package awsfs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
		fmt.Println("[debug] openFileWriter: ", path, " already exists")
		return nil, os.ErrExist
	}

	parentDirPath := GetParentDirPath(path)

	parentID, ok := ref.Entries[parentDirPath]
	if !ok {
		return nil, errors.New("no such parent directory")
	}

	entryID := uuid.New().String()
	buf := &bytes.Buffer{}

	newEntry := Entry{
		ID:       entryID,
		ParentID: parentID,
		Name:     filepath.Base(path),
		Type:     EntryTypeFile,
		//Size:     int64(buf.Len()),
		Modify:  time.Now(),
		Version: 1,
	}

	w := &FileWriter{
		Buffer: buf,
		entry:  newEntry,
		close: func(buf *bytes.Buffer) (fs.FileInfo, error) {
			fmt.Println(" buf.Len()!!!: ", buf.Len())
			err = s.PhysicalStore.PutObject(ctx, entryID, buf)
			if err != nil {
				fmt.Println("Error: ", err)
			}

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
	}
	return w, nil
}

type FileWriter struct {
	Buffer *bytes.Buffer
	close  func(buf *bytes.Buffer) (fs.FileInfo, error)
	entry  Entry
	pos    int
}

func (f *FileWriter) Close() error {
	_, err := f.close(f.Buffer)
	if err != nil {
		return err
	}
	return nil
}

func (f *FileWriter) Read(p []byte) (n int, err error) {
	fmt.Println("FileWriter: Read: ", len(p))
	return f.Buffer.Read(p)
}

func (f *FileWriter) Seek(offset int64, whence int) (int64, error) {
	fmt.Println("FileWriter: Seek: ", offset, whence)
	npos := f.pos
	// TODO: How to handle offsets greater than the size of system int?
	switch whence {
	case io.SeekStart:
		npos = int(offset)
	case io.SeekCurrent:
		npos += int(offset)
	case io.SeekEnd:
		//npos = len(f.n.data) + int(offset)
	default:
		npos = -1
	}
	if npos < 0 {
		return 0, os.ErrInvalid
	}
	f.pos = npos
	return int64(f.pos), nil
}

func (f *FileWriter) Readdir(count int) ([]fs.FileInfo, error) {
	fmt.Println("FileWriter: Readdir: ", count)
	return nil, ErrNotSupported
}

func (f *FileWriter) Stat() (fs.FileInfo, error) {
	file := FileInfo{
		name:    f.entry.Name,
		size:    int64(f.Buffer.Len()),
		mode:    0,
		modTime: f.entry.Modify,
		isDir:   f.entry.Type == EntryTypeDir,
		sys:     nil,
	}
	return file, nil
}

func (f *FileWriter) Write(p []byte) (n int, err error) {
	fmt.Println("FileWriter: Write: ", len(p))
	return f.Buffer.Write(p)
}
