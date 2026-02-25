package envpath

import (
	"os"
	"path/filepath"
)

func EnsureFile(path string, content string) error {
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// если файл уже существует — НЕ перезаписываем
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	return os.WriteFile(path, []byte(content+"\n"), 0o644)
}
