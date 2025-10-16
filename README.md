# GLF - GitLab Fuzzy Finder

⚡ Fast CLI tool for instant fuzzy search across self-hosted GitLab projects using local cache.

[![CI](https://github.com/igusev/glf/workflows/CI/badge.svg)](https://github.com/igusev/glf/actions/workflows/ci.yml)
[![Security](https://github.com/igusev/glf/workflows/Security/badge.svg)](https://github.com/igusev/glf/actions/workflows/security.yml)
[![codecov](https://codecov.io/gh/igusev/glf/branch/main/graph/badge.svg)](https://codecov.io/gh/igusev/glf)
[![Go Report Card](https://goreportcard.com/badge/github.com/igusev/glf)](https://goreportcard.com/report/github.com/igusev/glf)
[![Go Version](https://img.shields.io/badge/Go-1.25+-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## ✨ Features

- ⚡ **Lightning-fast fuzzy search** with local caching
- 🔍 **Multi-token search** - Search with spaces: `"api storage"` finds projects with both terms
- 🧠 **Smart ranking** - Frequently selected projects automatically appear first
- 🔁 **Auto-sync on startup** - Projects refresh in background while you search
- 🔌 **JSON API mode** - Machine-readable output for Raycast, Alfred, and custom integrations
- 🌍 **Cross-platform** builds for macOS, Linux, and Windows

## 🚀 Quick Start

### Installation

#### Homebrew (macOS/Linux)

The easiest way to install GLF on macOS or Linux:

```bash
# Add the tap
brew tap igusev/tap

# Install GLF
brew install glf

# Update to latest version
brew upgrade glf
```

#### MacPorts (macOS)

Alternative installation method for macOS users:

```bash
# Clone the ports repository
git clone https://github.com/igusev/macports-ports.git
cd macports-ports

# Add as a local port source (requires sudo)
sudo bash -c "echo 'file://$(pwd)' >> /opt/local/etc/macports/sources.conf"

# Update and install
sudo port sync
sudo port install glf

# Update to latest version
sudo port selfupdate
sudo port upgrade glf
```

#### Scoop (Windows)

The easiest way to install GLF on Windows:

```powershell
# Add the bucket
scoop bucket add igusev https://github.com/igusev/scoop-bucket

# Install GLF
scoop install igusev/glf

# Update to latest version
scoop update glf
```

#### From Source

```bash
# Clone the repository
git clone https://github.com/igusev/glf.git
cd glf

# Build and install
make install
```

#### Binary Releases

You can download the official GLF binaries from the [releases page](https://github.com/igusev/glf/releases).

Available for: **macOS** (Intel & Apple Silicon), **Linux** (x64, ARM, ARM64, and more), **Windows** (x64), **FreeBSD**, **OpenBSD**.

### Configuration

Run the interactive configuration wizard:

```bash
glf --init
```

This will prompt you for:
- GitLab instance URL (e.g., `https://gitlab.example.com`)
- Personal Access Token (with `read_api` scope)
- API timeout (default: 30 seconds)

Configuration is saved to `~/.config/glf/config.yaml`.

To reset and reconfigure:

```bash
glf --init --reset
```

#### Manual Configuration

Create `~/.config/glf/config.yaml`:

```yaml
gitlab:
  url: "https://gitlab.example.com"
  token: "your-personal-access-token"
  timeout: 30  # optional, defaults to 30 seconds

cache:
  dir: "~/.cache/glf"  # optional
```

#### Environment Variables

You can also use environment variables:

```bash
export GLF_GITLAB_URL="https://gitlab.example.com"
export GLF_GITLAB_TOKEN="your-token-here"
export GLF_GITLAB_TIMEOUT=30  # optional
```

### Creating a Personal Access Token

1. Go to your GitLab instance
2. Navigate to **User Settings** → **Access Tokens**
3. Create a new token with `read_api` scope
4. Copy the token and use it in `glf --init`

### Sync Projects

Fetch projects from GitLab and build local cache:

```bash
glf sync
```

### Search Projects

#### Interactive Mode (Default)

```bash
# Launch interactive fuzzy finder
glf

# Start with initial query
glf backend
```

**Navigation:**
- `↑/↓` - Navigate through results
- `Enter` - Select project
- `Ctrl+R` - Manually refresh/sync projects from GitLab
- `Ctrl+X` - Exclude/un-exclude project from search results
- `Ctrl+H` - Toggle showing excluded projects
- `?` - Toggle help text
- `Esc`/`Ctrl+C` - Quit
- Type to filter projects in real-time

**Activity Indicator:**
- `○` - Idle (nothing happening)
- `●` (green) - Active: syncing projects or loading selection history
- `●` (red) - Error: sync failed
- Auto-sync runs on startup, manual sync available with `Ctrl+R`

## 📖 Usage

### Commands

```
glf [query]           Search projects (default: interactive TUI)
glf --init            Configure GitLab connection
glf --init --reset    Reset and reconfigure GitLab connection
glf --sync            Sync projects from GitLab to local cache
glf --help            Show help
```

### Flags

```
--init                Run interactive configuration wizard
--reset               Reset configuration and start from scratch (use with --init)
-g, --open            Alias for --go (for compatibility)
--go                  Auto-select first result and open in browser
-s, --sync            Synchronize projects cache
--full                Force full sync (use with --sync)
-v, --verbose         Enable verbose logging
--scores              Show score breakdown for debugging ranking
--json                Output results in JSON format (for API integrations)
--limit N             Limit number of results in JSON mode (default: 20)
```

### Examples

```bash
# Interactive search
glf

# Search with pre-filled query
glf microservice

# Multi-token search (matches projects with all terms)
glf api storage        # Finds projects containing both "api" AND "storage"
glf user auth service  # Finds projects with all three terms

# Auto-select first result and open in browser
glf ingress -g         # Opens first "ingress" match
glf api --go           # Same as -g (alias for compatibility)

# Open current Git repository in browser
glf .

# Sync projects from GitLab
glf --sync             # Incremental sync
glf --sync --full      # Full sync (removes deleted projects)

# Verbose mode for debugging
glf sync --verbose

# Show ranking scores for debugging
glf --scores

# Configure GitLab connection
glf --init

# Reset and reconfigure
glf --init --reset
```

### JSON Output Mode (API Integration)

GLF supports JSON output for integration with tools like Raycast, Alfred, or custom scripts:

```bash
# Output search results as JSON
glf --json api

# Limit number of results
glf --json --limit 5 backend

# Include relevance scores (optional)
glf --json --scores microservice

# Get all projects (no query)
glf --json --limit 100
```

**JSON Output Format (without --scores):**

```json
{
  "query": "api",
  "results": [
    {
      "path": "backend/api-server",
      "name": "API Server",
      "description": "REST API for authentication",
      "url": "https://gitlab.example.com/backend/api-server"
    }
  ],
  "total": 5,
  "limit": 20
}
```

**JSON Output Format (with --scores):**

```json
{
  "query": "api",
  "results": [
    {
      "path": "backend/api-server",
      "name": "API Server",
      "description": "REST API for authentication",
      "url": "https://gitlab.example.com/backend/api-server",
      "score": 123.45
    }
  ],
  "total": 5,
  "limit": 20
}
```

**Score Breakdown:**

When using `--scores` flag, each project includes a relevance score that combines:
- **Search Relevance**: Fuzzy matching + full-text search score
- **Usage History**: Frequency of previous selections (with exponential decay)
- **Query-Specific Boost**: 3x multiplier for projects selected with this exact query

Higher scores indicate better matches. Projects are automatically sorted by score (descending).

**Use Cases:**
- **Raycast Extension**: Quick project navigation from Raycast
- **Alfred Workflow**: GitLab project search in Alfred
- **CI/CD Scripts**: Automated project discovery and URL generation
- **Custom Tools**: Build your own integrations on top of GLF's search
- **Analytics**: Use `--scores` to understand ranking and optimize search queries

**Error Handling:**

When errors occur, GLF outputs JSON error format and exits with code 1:

```json
{
  "error": "no projects in cache"
}
```

### Smart Ranking

GLF learns your selection patterns and automatically boosts frequently used projects:

- **First time**: Search `"api"` → Select `myorg/api/storage`
- **Next time**: Search `"api"` → `myorg/api/storage` appears **first**!
- The more you select a project, the higher it ranks
- Query-specific boost: projects selected for specific search terms rank higher for those terms
- Recent selections get extra boost (last 7 days)

History is stored in `~/.cache/glf/history.gob` and persists across sessions.

## 🔧 Development

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build for specific platform
make build-linux
make build-macos
make build-windows

# Create release archives
make release
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run linters
make lint
```

### Releasing

GLF uses automated CI/CD for releases via GitHub Actions and [GoReleaser](https://goreleaser.com/).

#### Automatic Release Process

When a new version tag is pushed, the release workflow automatically:

1. ✅ Builds binaries for all supported platforms (macOS, Linux, Windows, FreeBSD, OpenBSD)
2. ✅ Creates GitHub Release with artifacts and changelog
3. ✅ Updates [Homebrew tap](https://github.com/igusev/homebrew-tap) for macOS/Linux users
4. ✅ Updates [MacPorts Portfile](https://github.com/igusev/macports-ports) for macOS users
5. ✅ Updates [Scoop bucket](https://github.com/igusev/scoop-bucket) for Windows users

#### Creating a New Release

```bash
# Create and push a version tag
git tag v0.3.0
git push origin v0.3.0

# GitHub Actions will automatically:
# - Run GoReleaser
# - Build cross-platform binaries
# - Create GitHub release
# - Update package managers (Homebrew, MacPorts, Scoop)
```

#### Manual Release (optional)

You can also trigger releases manually from GitHub Actions UI:
- Go to **Actions** → **Release** → **Run workflow**

### Project Structure

```
glf/
├── cmd/glf/              # CLI entry point
│   └── main.go           # Main command and search logic
├── internal/
│   ├── config/           # Configuration handling
│   ├── gitlab/           # GitLab API client
│   ├── history/          # Selection frequency tracking
│   ├── index/            # Description indexing (Bleve)
│   ├── logger/           # Logging utilities
│   ├── search/           # Combined fuzzy + full-text search
│   ├── sync/             # Sync logic
│   ├── tui/              # Terminal UI (Bubbletea)
│   └── types/            # Shared types
├── Makefile              # Build automation
└── README.md
```

## ⚙️ Configuration Options

### GitLab Settings

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `gitlab.url` | GitLab instance URL | - | Yes |
| `gitlab.token` | Personal Access Token | - | Yes |
| `gitlab.timeout` | API timeout in seconds | 30 | No |

### Cache Settings

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `cache.dir` | Cache directory path | `~/.cache/glf` | No |

### Exclusions

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `exclusions` | List of project paths to exclude | `[]` | No |

Example with exclusions:

```yaml
gitlab:
  url: "https://gitlab.example.com"
  token: "your-token"

exclusions:
  - "archived/old-project"
  - "deprecated/legacy-api"
```

Excluded projects can be toggled with `Ctrl+X` in the TUI or hidden/shown with `Ctrl+H`.

## 🐛 Troubleshooting

### Connection Issues

```bash
# Use verbose mode to see detailed logs
glf sync --verbose
```

**Common issues:**
- Invalid GitLab URL: Verify URL in config
- Token expired: Regenerate token in GitLab
- Network timeout: Increase timeout in config
- Insufficient permissions: Ensure token has `read_api` scope

### Cache Issues

```bash
# Check cache location
ls -la ~/.cache/glf/

# Clear cache and re-sync
rm -rf ~/.cache/glf/
glf sync
```

### Configuration Issues

```bash
# Reconfigure GitLab connection
glf --init

# Reset and reconfigure from scratch
glf --init --reset

# Check current configuration
cat ~/.config/glf/config.yaml
```

## 📝 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## 🙏 Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- UI powered by [Bubbletea](https://github.com/charmbracelet/bubbletea)
- Styling with [Lipgloss](https://github.com/charmbracelet/lipgloss)
- Search indexing with [Bleve](https://github.com/blevesearch/bleve)
- GitLab API via [go-gitlab](https://github.com/xanzy/go-gitlab)
