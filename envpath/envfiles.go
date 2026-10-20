package envpath

import (
	"os"
	"path/filepath"
	"strings"
)

func EnsureFile(path string, content string) error {
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if os.IsNotExist(err) {
		return os.WriteFile(path, []byte(content+"\n"), 0o644)
	}

	keys := make(map[string]struct{})
	for _, line := range strings.Split(string(existing), "\n") {
		if key, _, ok := strings.Cut(line, "="); ok {
			keys[strings.TrimSpace(key)] = struct{}{}
		}
	}

	var missing []string
	for _, line := range strings.Split(content, "\n") {
		key, _, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		key = strings.TrimSpace(key)
		if _, exists := keys[key]; exists {
			continue
		}
		keys[key] = struct{}{}
		missing = append(missing, line)
	}
	if len(missing) == 0 {
		return nil
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()

	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		if _, err := file.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = file.WriteString(strings.Join(missing, "\n") + "\n")
	return err
}
