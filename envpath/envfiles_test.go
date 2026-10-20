package envpath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureFileAddsMissingVariablesWithoutOverwriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	initial := "REACT_APP_ADMIN_LOGIN=existing\nCUSTOM=value"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	content := "REACT_APP_ADMIN_LOGIN=admin\nREACT_APP_ADMIN_PASSWORD=secret"
	if err := EnsureFile(path, content); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result := string(got)
	if !strings.Contains(result, "REACT_APP_ADMIN_LOGIN=existing") {
		t.Fatalf("existing value was overwritten: %q", result)
	}
	if !strings.Contains(result, "REACT_APP_ADMIN_PASSWORD=secret") {
		t.Fatalf("missing variable was not added: %q", result)
	}
	if strings.Count(result, "REACT_APP_ADMIN_LOGIN=") != 1 {
		t.Fatalf("login variable was duplicated: %q", result)
	}
}
