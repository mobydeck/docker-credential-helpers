package plain

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/docker/docker-credential-helpers/credentials"
)

const defaultSystemPath = "/etc/docker-credential-plain/credentials.yaml"

// Plain stores Docker credentials in a YAML file that is easy to inspect.
type Plain struct {
	homePath   string
	systemPath string
	mu         sync.RWMutex
}

// credentialEntry mirrors how credentials are stored in the YAML file.
type credentialEntry struct {
	Username string
	Secret   string
}

// credentialStore maps a server URL to the stored credentials.
type credentialStore map[string]credentialEntry

// New returns a Plain helper that stores credentials at homePath and reads the
// system-wide credentials from systemPath. Whenever homePath is empty we fall
// back to "$HOME/.config/docker-credential-plain/credentials.yaml" and when
// systemPath is empty we use "/etc/docker-credential-plain/credentials.yaml".
func New(homePath, systemPath string) (*Plain, error) {
	if homePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		homePath = filepath.Join(homeDir, ".config", "docker-credential-plain", "credentials.yaml")
	}

	if systemPath == "" {
		systemPath = defaultSystemPath
	}

	return &Plain{
		homePath:   homePath,
		systemPath: systemPath,
	}, nil
}

// Add stores the credentials in the home directory YAML file.
func (p *Plain) Add(creds *credentials.Credentials) error {
	if err := validateCredentials(creds); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	store, err := p.loadStore(p.homePath)
	if err != nil {
		return err
	}

	store[strings.TrimSpace(creds.ServerURL)] = credentialEntry{
		Username: creds.Username,
		Secret:   creds.Secret,
	}

	return p.writeHomeStore(store)
}

// Delete removes credentials stored in the home YAML file.
func (p *Plain) Delete(serverURL string) error {
	serverURL = strings.TrimSpace(serverURL)
	if serverURL == "" {
		return credentials.NewErrCredentialsMissingServerURL()
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	store, err := p.loadStore(p.homePath)
	if err != nil {
		return err
	}

	if _, ok := store[serverURL]; !ok {
		return credentials.NewErrCredentialsNotFound()
	}

	delete(store, serverURL)
	return p.writeHomeStore(store)
}

// Get returns the username and secret for a registry server.
func (p *Plain) Get(serverURL string) (string, string, error) {
	serverURL = strings.TrimSpace(serverURL)
	if serverURL == "" {
		return "", "", credentials.NewErrCredentialsMissingServerURL()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	merged, err := p.loadMergedStore()
	if err != nil {
		return "", "", err
	}

	entry, ok := merged[serverURL]
	if !ok {
		return "", "", credentials.NewErrCredentialsNotFound()
	}

	return entry.Username, entry.Secret, nil
}

// List returns every stored server URL with the associated username.
func (p *Plain) List() (map[string]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	merged, err := p.loadMergedStore()
	if err != nil {
		return nil, err
	}

	resp := make(map[string]string, len(merged))
	for server, entry := range merged {
		resp[server] = entry.Username
	}

	return resp, nil
}

func (p *Plain) loadMergedStore() (credentialStore, error) {
	merged := credentialStore{}

	system, err := p.loadStore(p.systemPath)
	if err != nil {
		return nil, err
	}

	for server, entry := range system {
		merged[server] = entry
	}

	home, err := p.loadStore(p.homePath)
	if err != nil {
		return nil, err
	}

	for server, entry := range home {
		merged[server] = entry
	}

	return merged, nil
}

func (p *Plain) loadStore(path string) (credentialStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return credentialStore{}, nil
		}
		return nil, err
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return credentialStore{}, nil
	}

	store := credentialStore{}
	scanner := bufio.NewScanner(bytes.NewReader(trimmed))
	var currentServer string

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if len(line) == 0 {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		if indent == 0 {
			if server, ok := parseServerLine(line); ok {
				currentServer = server
			} else {
				currentServer = ""
			}
			continue
		}

		if currentServer == "" {
			continue
		}

		if key, value, ok := parseKeyValueLine(line); ok {
			entry := store[currentServer]
			switch strings.ToLower(key) {
			case "username":
				entry.Username = value
			case "secret":
				entry.Secret = value
			default:
				continue
			}
			store[currentServer] = entry
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return store, nil
}

func (p *Plain) writeHomeStore(store credentialStore) error {
	if len(store) == 0 {
		if err := os.Remove(p.homePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
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

	if err := os.MkdirAll(filepath.Dir(p.homePath), 0o700); err != nil {
		return err
	}

	return os.WriteFile(p.homePath, []byte(builder.String()), 0o600)
}

func parseServerLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", false
	}

	if !strings.HasSuffix(trimmed, ":") {
		return "", false
	}

	server := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
	server = strings.Trim(server, `"'`)
	if server == "" {
		return "", false
	}

	return server, true
}

func parseKeyValueLine(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}

	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	value = strings.Trim(value, `"'`)
	return key, value, key != ""
}

func validateCredentials(creds *credentials.Credentials) error {
	if creds == nil {
		return errors.New("missing credentials")
	}

	if strings.TrimSpace(creds.ServerURL) == "" {
		return credentials.NewErrCredentialsMissingServerURL()
	}

	if strings.TrimSpace(creds.Username) == "" {
		return credentials.NewErrCredentialsMissingUsername()
	}

	return nil
}
