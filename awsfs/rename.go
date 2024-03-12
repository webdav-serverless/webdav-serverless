package awsfs

import (
	"context"
	"os"
)

func (s Server) Rename(ctx context.Context, oldPath, newPath string) error {
	if oldPath = slashClean(oldPath); oldPath == "/" {
		return os.ErrInvalid
	}
	if newPath = slashClean(newPath); newPath == "/" {
		return os.ErrInvalid
	}

	// Referenceの取得
	ref, err := s.MetadataStore.GetReference(ctx, oldPath)
	if err != nil {
		return err
	}

	// oldpathの項目がなければエラー
	id, ok := ref.Entries[oldPath]
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
