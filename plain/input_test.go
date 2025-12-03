package plain

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseCredentialsPayloadJSON(t *testing.T) {
	payload := []byte(`{"ServerURL":"https://example.com","Username":"user","Secret":"secret"}`)
	creds, err := ParseCredentialsPayload(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if creds.ServerURL != "https://example.com" || creds.Username != "user" || creds.Secret != "secret" {
		t.Fatalf("credentials mismatch: %+v", creds)
	}
}

func TestParseCredentialsPayloadYAML(t *testing.T) {
	payload := []byte(`
ServerURL: https://example.org
Username: automation
Secret: s3cret
`)
	creds, err := ParseCredentialsPayload(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if creds.ServerURL != "https://example.org" || creds.Username != "automation" || creds.Secret != "s3cret" {
		t.Fatalf("credentials mismatch: %+v", creds)
	}
}

func TestParseCredentialsPayloadMissingField(t *testing.T) {
	payload := []byte(`{"ServerURL":"https://example.org","Username":"automation"}`)
	if _, err := ParseCredentialsPayload(payload); err == nil {
		t.Fatalf("expected error when field missing")
	}
}

func TestPromptForCredentials(t *testing.T) {
	input := strings.NewReader("https://example.net\nci-user\nsecret\n")
	var output bytes.Buffer

	creds, err := PromptForCredentials(input, &output)
	if err != nil {
		t.Fatalf("unexpected prompt error: %v", err)
	}

	if creds.ServerURL != "https://example.net" || creds.Username != "ci-user" || creds.Secret != "secret" {
		t.Fatalf("prompt caused wrong values: %+v", creds)
	}

	if !strings.Contains(output.String(), "ServerURL:") {
		t.Fatalf("prompt missing server label: %s", output.String())
	}
}
