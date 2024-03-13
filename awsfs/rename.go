package awsfs

import (
	"context"
	"fmt"
	"net/url"
	"os"
)

func (s Server) Rename(ctx context.Context, oldPath, newPath string) error {
	fmt.Println("Rename: ", oldPath, newPath)
	if oldPath = slashClean(oldPath); oldPath == "/" {
		return os.ErrInvalid
	}
	if newPath = slashClean(newPath); newPath == "/" {
		return os.ErrInvalid
	}

	// Referenceの取得
	ref, err := s.MetadataStore.GetReference(ctx, referenceID)
	if err != nil {
		return err
	}

	// oldpathの項目がなければエラー
	unescaped, _ := url.QueryUnescape(oldPath)
	id, ok := ref.Entries[unescaped]
	if ok {
		return os.ErrNotExist
	}

	// enrtyの取得
	entry, err := s.MetadataStore.GetEntry(ctx, id)
	if err != nil {
		return err
	}

	// Referenceに新しいキーの値を追加
	ref.Entries[newPath] = id

	// Referenceから古いキーの値を削除
	delete(ref.Entries, oldPath)

	// entryのnameを書き換え
	entry.Name = newPath

	// referenceとentryの更新
	err = s.MetadataStore.UpdateEntryName(ctx, entry, ref)
	if err != nil {
		return err
	}

	return nil
}
