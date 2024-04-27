package awsfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/webdav-serverless/webdav-serverless/webdav"
)

const referenceID = "root"

var ErrNotSupported = errors.New("not supported")

type FileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
	sys     any
}

func (f FileInfo) Name() string {
	return f.name
}

func (f FileInfo) Size() int64 {
	return f.size
}

func (f FileInfo) Mode() fs.FileMode {
	return f.mode
}

func (f FileInfo) ModTime() time.Time {
	return f.modTime
}

func (f FileInfo) IsDir() bool {
	return f.isDir
}

func (f FileInfo) Sys() any {
	return f.sys
}

func (s Server) OpenFile(ctx context.Context, path string, flag int, perm os.FileMode) (webdav.File, error) {
	fmt.Println("OpenFile: ", path)
	if path = slashClean(path); path == "" {
		return nil, os.ErrInvalid
	}
	ref, err := s.MetadataStore.GetReference(ctx, referenceID)
	if err != nil {
		return nil, err
	}
	entryID, ok := ref.Entries[path]
	if !ok {
		fmt.Println("OpenFile: ErrNotExist")
		return nil, os.ErrNotExist
	}

	entry, err := s.MetadataStore.GetEntry(ctx, entryID)
	if err != nil {
		return nil, err
	}

	if entry.Type == EntryTypeDir {
		entries, err := s.MetadataStore.GetEntriesByParentID(ctx, entryID)
		if err != nil {
			return nil, err
		}
		var files []fs.FileInfo
		for _, entry := range entries {
			files = append(files, FileInfo{
				name:    entry.Name,
				size:    entry.Size,
				mode:    0,
				modTime: entry.Modify,
				isDir:   entry.Type == EntryTypeDir,
				sys:     nil,
			})
		}
		return &FileReader{
			fileInfo: FileInfo{
				name:    entry.Name,
				size:    entry.Size,
				mode:    perm,
				modTime: entry.Modify,
				isDir:   entry.Type == EntryTypeDir,
				sys:     nil,
			},
			files: files,
		}, nil
	}

	r, err := s.PhysicalStore.GetObject(ctx, entryID)
	if err != nil {
		return nil, err
	}

	return &FileReader{
		ReadCloser: r,
		fileInfo: FileInfo{
			name:    entry.Name,
			size:    entry.Size,
			mode:    perm,
			modTime: entry.Modify,
			isDir:   entry.Type == EntryTypeDir,
			sys:     nil,
		},
	}, nil
}

type FileReader struct {
	io.ReadCloser
	fileInfo FileInfo
	files    []fs.FileInfo
}

func (f FileReader) Close() error {
	if f.ReadCloser == nil {
		return nil
	}
	return f.ReadCloser.Close()
}

func (f FileReader) Read(p []byte) (n int, err error) {
	return f.ReadCloser.Read(p)
}

func (f FileReader) Seek(offset int64, whence int) (int64, error) {
	return f.fileInfo.size, nil
}

func (f FileReader) Readdir(count int) ([]fs.FileInfo, error) {
	for _, v := range f.files {
		fmt.Println("Readdir: ", v.Name())
	}
	return f.files, nil
}

func (f FileReader) Stat() (fs.FileInfo, error) {
	return f.fileInfo, nil
}

func (f FileReader) Write(p []byte) (n int, err error) {
	return 0, ErrNotSupported
}
