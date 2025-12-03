package plain

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker-credential-helpers/credentials"
)

// ParseCredentialsPayload decodes JSON or simple YAML payloads into Credentials.
func ParseCredentialsPayload(payload []byte) (*credentials.Credentials, error) {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 {
		return nil, errors.New("no credentials payload provided")
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(trimmed, &raw); err == nil {
		return credentialsFromMap(interfaceToStringMap(raw))
	}

	values := parseSimpleYAMLMap(string(trimmed))
	if len(values) == 0 {
		return nil, errors.New("failed to decode credentials payload")
	}

	return credentialsFromMap(values)
}

// PromptForCredentials asks interactively for the credentials fields.
func PromptForCredentials(in io.Reader, out io.Writer) (*credentials.Credentials, error) {
	reader := bufio.NewReader(in)

	serverURL, err := promptUntilNonEmpty(reader, out, "ServerURL")
	if err != nil {
		return nil, err
	}

	username, err := promptUntilNonEmpty(reader, out, "Username")
	if err != nil {
		return nil, err
	}

	secret, err := promptUntilNonEmpty(reader, out, "Secret")
	if err != nil {
		return nil, err
	}

	return &credentials.Credentials{
		ServerURL: serverURL,
		Username:  username,
		Secret:    secret,
	}, nil
}

func promptUntilNonEmpty(reader *bufio.Reader, writer io.Writer, label string) (string, error) {
	for {
		if _, err := fmt.Fprintf(writer, "%s: ", label); err != nil {
			return "", err
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		value := strings.TrimSpace(line)
		if value != "" {
			return value, nil
		}

		if _, err := fmt.Fprintf(writer, "%s cannot be empty\n", label); err != nil {
			return "", err
		}
	}
}

func credentialsFromMap(values map[string]string) (*credentials.Credentials, error) {
	serverURL, ok := findField(values, "serverurl")
	if !ok || serverURL == "" {
		return nil, credentials.NewErrCredentialsMissingServerURL()
	}

	username, ok := findField(values, "username")
	if !ok || username == "" {
		return nil, credentials.NewErrCredentialsMissingUsername()
	}

	secret, ok := findField(values, "secret")
	if !ok || secret == "" {
		return nil, errors.New("no credentials secret")
	}

	creds := &credentials.Credentials{
		ServerURL: serverURL,
		Username:  username,
		Secret:    secret,
	}

	if err := validateCredentials(creds); err != nil {
		return nil, err
	}

	return creds, nil
}

func findField(values map[string]string, key string) (string, bool) {
	for k, v := range values {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(v), true
		}
	}
	return "", false
}

func interfaceToStringMap(input map[string]interface{}) map[string]string {
	result := make(map[string]string, len(input))
	for k, v := range input {
		result[k] = fmt.Sprint(v)
	}
	return result
}

func parseSimpleYAMLMap(input string) map[string]string {
	values := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(input))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)

		if key != "" {
			values[key] = value
		}
	}

	return values
}
