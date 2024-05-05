package awsfs

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/webdav-serverless/webdav-serverless/webdav"
)

const referenceID = "root"

var ErrNotSupported = errors.New("not supported")

type FileInfo struct {
	name      string
	size      int64
	modTime   time.Time
	isDir     bool
	sys       any
	deadProps map[xml.Name]webdav.Property
}

func (f FileInfo) Name() string {
	return f.name
}

func (f FileInfo) Size() int64 {
	return f.size
}

func (f FileInfo) Mode() fs.FileMode {
	if f.isDir {
		return os.ModeDir | 0o777
	}
	return 0o777
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

func (s *Server) OpenFile(ctx context.Context, path string, flag int, perm os.FileMode) (webdav.File, error) {

	if path = slashClean(path); path == "" {
		return nil, os.ErrInvalid
	}
	ref, err := s.MetadataStore.GetReference(ctx, referenceID)
	if err != nil {
		return nil, err
	}
	entryID, ok := ref.Entries[path]
	if !ok {
		return nil, os.ErrNotExist
	}

	entry, err := s.MetadataStore.GetEntry(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if entry.IsDir() {
		return &FileReader{
			tempFile:      nil,
			entry:         entry,
			metadataStore: s.MetadataStore,
			ctx:           ctx,
		}, nil
	}

	r, err := s.PhysicalStore.GetObject(ctx, entryID)
	if err != nil {
		return nil, err
	}

	temp, err := os.CreateTemp(s.TempDir, "webdav-temp-")
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(temp, r)
	if err != nil {
		return nil, err
	}

	return &FileReader{
		tempFile:      temp,
		entry:         entry,
		metadataStore: s.MetadataStore,
		ctx:           ctx,
	}, nil
}

type FileReader struct {
	tempFile      *os.File
	entry         Entry
	metadataStore MetadataStore
	ctx           context.Context
}

func (f FileReader) Close() error {
	if f.tempFile == nil {
		return nil
	}
	err := f.tempFile.Close()
	_ = os.Remove(f.tempFile.Name())
	return err
}

func (f FileReader) Read(p []byte) (n int, err error) {
	return f.tempFile.Read(p)
}

func (f FileReader) Seek(offset int64, whence int) (int64, error) {
	return f.tempFile.Seek(offset, whence)
}

func (f FileReader) Readdir(count int) ([]fs.FileInfo, error) {
	if f.entry.Type == EntryTypeDir {
		entries, err := f.metadataStore.GetEntriesByParentID(f.ctx, f.entry.ID)
		if err != nil {
			return nil, err
		}
		var files []fs.FileInfo
		for _, entry := range entries {
			files = append(files, FileInfo{
				name:    entry.Name,
				size:    entry.Size,
				modTime: entry.Modify,
				isDir:   entry.IsDir(),
				sys:     nil,
			})
		}
		return files, nil
	}
	return nil, nil
}

func (f FileReader) Stat() (fs.FileInfo, error) {
	return FileInfo{
		name:    f.entry.Name,
		size:    f.entry.Size,
		modTime: f.entry.Modify,
		isDir:   f.entry.IsDir(),
		sys:     nil,
	}, nil
}

func (f FileReader) Write(p []byte) (n int, err error) {
	return 0, ErrNotSupported
}

func (f FileReader) DeadProps() (map[xml.Name]webdav.Property, error) {
	props := make(map[xml.Name]webdav.Property, len(f.entry.DeadProps))
	for _, v := range f.entry.DeadProps {
		var prop webdav.Property
		err := xml.Unmarshal([]byte(v), &prop)
		if err != nil {
			return nil, err
		}
		props[prop.XMLName] = prop
	}
	return props, nil
}

func (f FileReader) Patch(patches []webdav.Proppatch) ([]webdav.Propstat, error) {
	pstat := webdav.Propstat{Status: http.StatusOK}
	for _, patch := range patches {
		for _, p := range patch.Props {
			pstat.Props = append(pstat.Props, webdav.Property{
				XMLName:  p.XMLName,
				Lang:     p.Lang,
				InnerXML: p.InnerXML,
			})
			propKey := p.XMLName.Space + ":" + p.XMLName.Local
			if patch.Remove {
				delete(f.entry.DeadProps, propKey)
				continue
			}
			marshaled, err := xml.Marshal(p)
			if err != nil {
				return nil, err
			}
			f.entry.DeadProps[propKey] = string(marshaled)
		}
	}
	err := f.metadataStore.UpdateEntry(f.ctx, f.entry)
	if err != nil {
		return nil, err
	}
	return []webdav.Propstat{pstat}, nil
}
