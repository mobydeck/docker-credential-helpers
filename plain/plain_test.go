package plain

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/docker/docker-credential-helpers/credentials"
)

func TestPlainAddGetListDelete(t *testing.T) {
	dir := t.TempDir()
	homePath := filepath.Join(dir, "home", "credentials.yaml")
	systemPath := filepath.Join(dir, "system.yaml")

	helper := mustNew(t, homePath, systemPath)

	creds := &credentials.Credentials{
		ServerURL: "https://example.com",
		Username:  "ci",
		Secret:    "secret",
	}

	if err := helper.Add(creds); err != nil {
		t.Fatalf("unexpected add error: %v", err)
	}

	username, secret, err := helper.Get(creds.ServerURL)
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}

	if username != creds.Username {
		t.Fatalf("username mismatch: got %s", username)
	}
	if secret != creds.Secret {
		t.Fatalf("secret mismatch: got %s", secret)
	}

	list, err := helper.List()
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}

	if got := list[creds.ServerURL]; got != creds.Username {
		t.Fatalf("list mismatch: got %s", got)
	}

	if err := helper.Delete(creds.ServerURL); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}

	if _, _, err := helper.Get(creds.ServerURL); !credentials.IsErrCredentialsNotFound(err) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestPlainMergeSystemAndHome(t *testing.T) {
	dir := t.TempDir()
	homePath := filepath.Join(dir, "home", "credentials.yaml")
	systemPath := filepath.Join(dir, "system.yaml")

	writeStoreFile(t, systemPath, credentialStore{
		"https://system.example":      {Username: "sys", Secret: "sys-secret"},
		"https://system-only.example": {Username: "sys-only", Secret: "sys-only-secret"},
	})

	helper := mustNew(t, homePath, systemPath)

	if err := helper.Add(&credentials.Credentials{
		ServerURL: "https://home.example",
		Username:  "home",
		Secret:    "home-secret",
	}); err != nil {
		t.Fatalf("unexpected add error: %v", err)
	}

	if err := helper.Add(&credentials.Credentials{
		ServerURL: "https://system.example",
		Username:  "override",
		Secret:    "override-secret",
	}); err != nil {
		t.Fatalf("unexpected override error: %v", err)
	}

	username, secret, err := helper.Get("https://system.example")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if username != "override" || secret != "override-secret" {
		t.Fatalf("override did not take precedence: %s %s", username, secret)
	}

	username, secret, err = helper.Get("https://system-only.example")
	if err != nil {
		t.Fatalf("system get error: %v", err)
	}
	if username != "sys-only" || secret != "sys-only-secret" {
		t.Fatalf("system entry corrupted: %s %s", username, secret)
	}

	list, err := helper.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	expect := map[string]string{
		"https://system.example":      "override",
		"https://system-only.example": "sys-only",
		"https://home.example":        "home",
	}

	if len(list) != len(expect) {
		t.Fatalf("list length mismatch: %d", len(list))
	}

	for server, user := range expect {
		if got := list[server]; got != user {
			t.Fatalf("list entry %s mismatch: got %s", server, got)
		}
	}
}

func writeStoreFile(t *testing.T, path string, store credentialStore) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	servers := make([]string, 0, len(store))
	for server := range store {
		servers = append(servers, server)
	}
	sort.Strings(servers)

	var builder strings.Builder
	for _, server := range servers {
		entry := store[server]
		builder.WriteString(server)
		builder.WriteString(":\n")
		builder.WriteString("  username: ")
		builder.WriteString(entry.Username)
		builder.WriteString("\n")
		builder.WriteString("  secret: ")
		builder.WriteString(entry.Secret)
		builder.WriteString("\n")
	}

	if err := os.WriteFile(path, []byte(builder.String()), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func mustNew(t *testing.T, home, system string) *Plain {
	t.Helper()
	helper, err := New(home, system)
	if err != nil {
		t.Fatalf("failed to create helper: %v", err)
	}
	return helper
}
