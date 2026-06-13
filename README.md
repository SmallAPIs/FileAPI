# FileAPI

FileAPI is a small local desktop agent written in Go. A browser-based CoWork web app talks to it over **HTTPS on localhost** to perform sandboxed file operations and simple system actions (open app, open URL) without installing a heavy native client.

## Quick start

Requirements: Go 1.22+

```bash
go run ./cmd/fileapi serve
```

On first run the agent:

1. Creates a config directory in the OS-appropriate location
2. Generates a self-signed TLS certificate (4096-bit RSA)
3. Listens on `https://127.0.0.1:8443`

Default config locations:

| OS      | Directory |
|---------|-----------|
| Windows | `%AppData%\FileAPI\` |
| macOS   | `~/Library/Application Support/FileAPI/` |
| Linux   | `~/.config/fileapi/` |

Health check: `GET https://127.0.0.1:8443/health`

API base: `https://127.0.0.1:8443/api/v1`

## Trust the certificate

Browsers will warn about the self-signed certificate until you trust it.

- **Windows**: import `cert.pem` into *Trusted Root Certification Authorities* (certmgr / MMC)
- **macOS**: open `cert.pem` in Keychain Access and set *Always Trust*
- **Linux**: copy to `/usr/local/share/ca-certificates/` and run `update-ca-certificates`, or trust per-browser

For local development you can also use [mkcert](https://github.com/FiloSottile/mkcert) and point `cert_file` / `key_file` in config to the generated files.

## API overview

All responses use a JSON envelope:

```json
{ "ok": true, "data": { } }
{ "ok": false, "error": { "code": "...", "message": "..." } }
```

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/files?path=` | Read UTF-8 text file (max 10 MB) |
| POST | `/api/v1/files` | Create file `{ "path", "content", "create_dirs"? }` |
| PATCH | `/api/v1/files` | Edit file `{ "path", "content", "mode"?: "overwrite"\|"append" }` |
| DELETE | `/api/v1/files?path=` | Delete file |
| POST | `/api/v1/system/open-app` | Open app `{ "name" \| "path" }` |
| POST | `/api/v1/system/open-url` | Open URL `{ "url" }` (http/https only) |

Machine-readable contract: [`api/openapi.yaml`](api/openapi.yaml)

Example config: [`configs/config.example.yaml`](configs/config.example.yaml)

## Browser integration (CORS + Private Network Access)

The agent enables CORS for configured origins (default `*` in dev). When a public HTTPS web app calls `https://127.0.0.1`, Chrome may send a [Private Network Access](https://developer.chrome.com/blog/private-network-access-preflight) preflight; the agent responds with `Access-Control-Allow-Private-Network: true`.

Example fetch from your web app:

```javascript
const res = await fetch('https://127.0.0.1:8443/api/v1/files?path=' + encodeURIComponent('/Users/me/note.txt'), {
  method: 'GET',
});
const body = await res.json();
```

## Security (foundation)

- **Localhost bind** — default `127.0.0.1` only
- **HTTPS** — TLS 1.2+ with local cert
- **Path sandbox** — files must live under `allowed_roots` (default: user home); `..` traversal blocked
- **Auth** — stub middleware only; OAuth/JWT planned for a later release

Tighten `allowed_origins` before any non-local deployment.

## Development

```bash
make test    # go test ./...
make vet     # go vet ./...
make build   # build ./fileapi
make dist    # cross-compile binaries into dist/
```

## CI (Blacksmith)

CI runs on [Blacksmith](https://www.blacksmith.sh/) runners:

- `go test ./...` and `go vet ./...`
- Cross-compile matrix: linux/windows/darwin × amd64/arm64
- Upload build artifacts per platform

**Before the first CI run succeeds**, complete Blacksmith setup:

1. Install the [Blacksmith GitHub App](https://app.blacksmith.sh) on the `SmallAPIs` organization (or the org that owns this repo).
2. Blacksmith targets GitHub **organizations**; personal repos may need to live under an org.
3. If your org uses IP allowlists, allowlist Blacksmith control-plane IPs per [their network docs](https://docs.blacksmith.sh/introduction/quickstart).

Workflow: [`.github/workflows/ci.yml`](.github/workflows/ci.yml)

## Roadmap

- OAuth/JWT authentication
- Binary file support and streaming
- Directory listing and file watching
- OS service install (Windows service, launchd, systemd)
- Signed release artifacts from CI

## License

See [LICENSE](LICENSE).
