# plain credential helper

`docker-credential-plain` keeps credentials in an easily editable YAML file so it can be consumed safely from scripts and CI tools.

- **Home store:** defaults to `$HOME/.config/docker-credential-plain/credentials.yaml` and is the only location that is modified during `store`/`erase`.
- **System store:** reads `/etc/docker-credential-plain/credentials.yaml` when present. Entries from the home store override system entries when retrieving or listing credentials.
- **File format:** each server URL is a top-level key with two child fields:
  ```yaml
  https://example.com:
    username: myuser
    secret: s3cr3t
  ```

## Commands

- `docker-credential-plain store` accepts JSON _or_ simple YAML payloads that declare `ServerURL`, `Username`, and `Secret`. When no payload is piped in, it falls back to an interactive prompt for the same fields.
- `docker-credential-plain get` and `list` continue to emit JSON wrappers that match the other helpers and rely on the merged view of the system and home stores.
- `docker-credential-plain erase` removes an entry from the writable home store. System credentials remain read-only.

The helper is meant to be used from automation, so it never encrypts credentials and exposes the raw YAML files for inspectors or mounted volumes.
