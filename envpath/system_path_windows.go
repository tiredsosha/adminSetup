package envpath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func AddMongoBinToSystemPath() error {
	mongoBin, err := findMongoBin()
	if err != nil {
		return err
	}

	key, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\Session Manager\Environment`,
		registry.QUERY_VALUE|registry.SET_VALUE,
	)
	if err != nil {
		return err
	}
	defer key.Close()

	currentPath, _, err := key.GetStringValue("Path")
	if err != nil {
		return err
	}

	if containsPath(currentPath, mongoBin) {
		return nil
	}

	newPath := currentPath
	if newPath != "" && !strings.HasSuffix(newPath, ";") {
		newPath += ";"
	}
	newPath += mongoBin

	if err := key.SetStringValue("Path", newPath); err != nil {
		return err
	}

	BroadcastEnvChange()
	fmt.Println("Added to PATH:", mongoBin)
	return nil
}

func findMongoBin() (string, error) {
	base := `C:\Program Files\MongoDB\Server`
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", base, err)
	}
	var found []string
	for _, e := range entries {
		if e.IsDir() {
			found = append(found, e.Name())
		}
	}
	if len(found) == 0 {
		return "", fmt.Errorf("MongoDB not found in %s", base)
	}

	// Пытаемся выбрать “самую новую” по имени папки (8.2, 8.2.5, ...)
	best := found[0]
	for _, v := range found[1:] {
		if versionGreater(v, best) {
			best = v
		}
	}

	return filepath.Join(base, best, "bin"), nil
}

func containsPath(pathValue, toFind string) bool {
	p := strings.ToLower(pathValue)
	f := strings.ToLower(toFind)
	parts := strings.Split(p, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == f {
			return true
		}
	}
	return false
}

// очень простое сравнение версий вида "8.2" / "8.2.5"
func versionGreater(a, b string) bool {
	parse := func(s string) [3]int {
		var out [3]int
		parts := strings.Split(s, ".")
		for i := 0; i < len(parts) && i < 3; i++ {
			n := 0
			for _, ch := range parts[i] {
				if ch >= '0' && ch <= '9' {
					n = n*10 + int(ch-'0')
				} else {
					break
				}
			}
			out[i] = n
		}
		return out
	}
	av := parse(a)
	bv := parse(b)
	for i := 0; i < 3; i++ {
		if av[i] > bv[i] {
			return true
		}
		if av[i] < bv[i] {
			return false
		}
	}
	return false
}
