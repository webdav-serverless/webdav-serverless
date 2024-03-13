package awsfs

import (
	"context"
	"fmt"
	"net/url"
	"os"
)

func (s Server) Stat(ctx context.Context, path string) (os.FileInfo, error) {
	fmt.Println("Stat: ", path)
	path = slashClean(path)

	ref, err := s.MetadataStore.GetReference(ctx, referenceID)
	if err != nil {
		return nil, err
	}

	unescaped, _ := url.QueryUnescape(path)
	id, ok := ref.Entries[unescaped]
	if !ok {
		fmt.Println("Stat: ", path, "- not found")
		return nil, os.ErrNotExist
	}

	entry, err := s.MetadataStore.GetEntry(ctx, id)
	if err != nil {
		return nil, err
	}

	info := FileInfo{
		name:    entry.Name,
		size:    entry.Size,
		mode:    os.FileMode(0), // FIXME
		modTime: entry.Modify,
		isDir:   entry.Type == EntryTypeDir,
	}

	fmt.Println("info: ", info)

	return info, nil
}
