# AGENTS.md

## Cursor Cloud specific instructions

FileAPI is a single, self-contained Go desktop agent (module `github.com/SmallAPIs/FileAPI`). There are no databases or external services. Go 1.22+ is the only prerequisite and is preinstalled in this environment.

### Lint / test / build / run
Standard commands (see `Makefile` and `README.md`):
- Lint: `make vet` (`go vet ./...`)
- Test: `make test` (`go test ./...`)
- Build: `make build` (binary at `./fileapi`)
- Run (dev): `make run` or `go run ./cmd/fileapi serve`

### Running and exercising the agent (non-obvious caveats)
- The server listens on `https://127.0.0.1:8443` (HTTPS only). On first run it auto-generates config and a self-signed 4096-bit RSA cert under `~/.config/fileapi/` (Linux). No setup step is needed for this.
- Because the cert is self-signed, use `curl -k` (or trust the cert) when calling the API; plain `http://` will not work.
- File operations are sandboxed to `allowed_roots` (default: the user's home dir, `~`). Requests targeting paths outside the home dir are rejected, and `..` traversal is blocked. When testing file CRUD, use paths under `$HOME`.
- This is a headless API agent; there is no GUI. Test end-to-end via terminal HTTP clients (`curl -k`) against `/health` and `/api/v1/...`. The `CoWork` web frontend referenced in the README is not part of this repo.
