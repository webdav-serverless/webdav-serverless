package awsfs

import (
	"context"
	"os"
)

func (s Server) RemoveAll(ctx context.Context, path string) error {
	if path = slashClean(path); path == "/" {
		return os.ErrInvalid
	}

	if path = slashClean(path); path == "/" {
		return os.ErrInvalid
	}

	// Referenceの取得
	ref, err := s.MetadataStore.GetReference(ctx, path)
	if err != nil {
		return err
	}

	// pathの項目がなければエラー
	id, ok := ref.Entries[path]
	if ok {
		return os.ErrNotExist
	}
	ids := [id]

	// Referenceからキーの値を削除
	// TODO: pathから始まる項目を全てループで消す
	delete(ref.Entries, path)

	// referenceの更新とentryの削除
	err = s.MetadataStore.DeleteEntries(ctx, ids, ref)
	if err != nil {
		return err
	}

	return nil
}
