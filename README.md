# FileAPI

FileAPI is a small local desktop agent written in Go. A browser-based CoWork web app talks to it over **HTTPS on localhost** to perform sandboxed file operations and simple system actions (open app, open URL) without installing a heavy native client.

## Quick start

### Install (recommended)

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/SmallAPIs/FileAPI/main/scripts/install.ps1 | iex
```

**Linux / macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/SmallAPIs/FileAPI/main/scripts/install.sh | sh
```

This downloads the latest release binary, installs it, and adds `fileapi` to your PATH.
Then manage the agent from your terminal:

```bash
fileapi serve    # start the HTTPS API server
fileapi status   # check whether it is running
fileapi version  # print the installed version
```

Pin a specific release with `FILEAPI_VERSION=v1.0.0` (shell) or `$env:FILEAPI_VERSION="v1.0.0"` (PowerShell).

### Run from source

Requirements: Go 1.22+

```bash
go run ./cmd/fileapi serve
```

On first run the agent:

1. Creates a config directory in the OS-appropriate location
2. Generates a self-signed TLS certificate (ECDSA P-256)
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
| GET | `/api/v1/files/raw?path=` | Stream UTF-8 text file (no JSON; supports gzip) |
| POST | `/api/v1/files` | Create file `{ "path", "content", "create_dirs"?, "include_content"? }` |
| PATCH | `/api/v1/files` | Edit file `{ "path", "content", "mode"?: "overwrite"\|"append", "include_content"? }` |
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

Build outputs (local `make dist` or CI artifacts):

| Artifact | Platform |
|----------|----------|
| `fileapi-linux-amd64` | Linux x64 |
| `fileapi-linux-arm64` | Linux ARM64 |
| `fileapi-windows-amd64.exe` | Windows x64 |
| `fileapi-darwin-amd64` | macOS Intel |
| `fileapi-darwin-arm64` | macOS Apple Silicon |

## CI (Blacksmith)

CI runs on [Blacksmith](https://www.blacksmith.sh/) runners — not GitHub-hosted `ubuntu-latest`.

Workflow: [`.github/workflows/ci.yml`](.github/workflows/ci.yml)

| Job | What it does |
|-----|----------------|
| `test` | `go test`, `go vet`, and a smoke `go build` |
| `build` | Cross-compile 5 platform binaries and upload artifacts |

### Blacksmith setup (skip the Migration Wizard)

This repo **already uses Blacksmith** (`runs-on: blacksmith-2vcpu-ubuntu-2404`). The Migration Wizard only converts GitHub runners like `ubuntu-latest` → Blacksmith, so it will show **“No GitHub runners detected”** for `ci.yml`. That is correct — **close or skip the wizard**; no migration PR is needed.

**One-time checklist:**

1. Install the [Blacksmith GitHub App](https://app.blacksmith.sh) on the **SmallAPIs** org.
2. In the app settings, grant access to the **FileAPI** repository (all repos that use `runs-on: blacksmith-*` must be included).
3. If your org uses GitHub IP allowlists, allowlist Blacksmith control-plane IPs per [network docs](https://docs.blacksmith.sh/introduction/quickstart).

**Trigger a build:**

- Push to `main` or open a pull request, **or**
- GitHub → **Actions** → **CI** → **Run workflow** (manual `workflow_dispatch`)

**Verify it worked:**

- GitHub Actions: jobs show runner `blacksmith-2vcpu-ubuntu-2404` (not `ubuntu-latest`).
- Blacksmith console: [app.blacksmith.sh](https://app.blacksmith.sh) lists the workflow run.
- After `build` completes, download artifacts from the Actions run page (one zip per platform).

**If jobs stay on “Waiting for a runner”:** the Blacksmith app is not installed on this repo, or the org/repo is not linked in the app.

### Production release

Workflow: [`.github/workflows/release.yml`](.github/workflows/release.yml)

Production binaries are built on Blacksmith runners when you push a version tag (`v*`) or manually run the **Release** workflow from the Actions tab (select an existing tag).

| Job | What it does |
|-----|----------------|
| `test` | `go test` and `go vet` quality gates |
| `build` | Cross-compile 5 platform binaries with release ldflags |
| `release` | Publish a GitHub Release with binaries and `SHA256SUMS` |

**Cut a release:**

```bash
git tag -a v1.0.0 -m "FileAPI v1.0.0"
git push origin v1.0.0
```

After the workflow completes, download artifacts from the [GitHub Releases](https://github.com/SmallAPIs/FileAPI/releases) page. Verify a binary with `fileapi version`.

## Roadmap

- OAuth/JWT authentication
- Binary file support and streaming
- Directory listing and file watching
- OS service install (Windows service, launchd, systemd)
- Signed release artifacts from CI

## License

See [LICENSE](LICENSE).
