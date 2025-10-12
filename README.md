# GLF - GitLab Fuzzy Finder

⚡ Fast CLI tool for instant fuzzy search across self-hosted GitLab projects using local cache.

![Phase Status](https://img.shields.io/badge/Phase%204-Complete-success)
![Go Version](https://img.shields.io/badge/Go-1.25+-blue)
![License](https://img.shields.io/badge/license-TBD-lightgrey)

## ✨ Features

- ⚡ **Lightning-fast fuzzy search** with local caching
- 🎨 **Interactive TUI** with Monokai Pro color scheme and fzf-like UI
- 🔍 **Multi-token search** - Search with spaces: `"api storage"` finds projects with both terms
- 🧠 **Smart ranking** - Frequently selected projects automatically appear first
- 🔄 **Parallel pagination** - 5-8x faster sync with concurrent API requests
- 🔁 **Auto-sync on startup** - Projects refresh in background while you search
- 🔁 **Live sync** - Press `Ctrl+R` to manually refresh anytime (non-blocking)
- 📍 **Clean activity indicator** - Single circle (○/●) shows sync and history loading status
- ⚙️ **Easy configuration** via interactive wizard or YAML
- 🌍 **Cross-platform** builds for macOS, Linux, and Windows
- 📝 **Verbose logging** with progress indicators for troubleshooting

## 🚀 Quick Start

### Installation

#### From Source

```bash
# Clone the repository
git clone https://github.com/igusev/glf.git
cd glf

# Build and install
make install
```

#### Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/igusev/glf/releases).

### Configuration

Run the interactive configuration wizard:

```bash
glf config
```

This will prompt you for:
- GitLab instance URL (e.g., `https://gitlab.example.com`)
- Personal Access Token (with `read_api` scope)
- API timeout (default: 30 seconds)

Configuration is saved to `~/.config/glf/config.yaml`.

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
glf config            Configure GitLab connection
glf sync              Sync projects from GitLab to local cache
glf find <query>      Search projects (alias for default)
glf --help            Show help
```

### Flags

```
-v, --verbose         Enable verbose logging
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

# Verbose mode for debugging
glf sync --verbose

# Configure GitLab connection
glf config
```

### Smart Ranking

GLF learns your selection patterns and automatically boosts frequently used projects:

- **First time**: Search `"api"` → Select `myorg/api/storage`
- **Next time**: Search `"api"` → `myorg/api/storage` appears **first**!
- The more you select a project, the higher it ranks
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

# Format code
make fmt

# Run linters
make lint
```

### Project Structure

```
glf/
├── cmd/glf/              # CLI entry point
│   ├── main.go           # Main command and search logic
│   ├── sync.go           # Sync command
│   ├── config_cmd.go     # Config command
│   └── find.go           # Find command (alias)
├── internal/
│   ├── cache/            # Cache management
│   ├── config/           # Configuration handling
│   ├── gitlab/           # GitLab API client
│   ├── history/          # Selection frequency tracking
│   ├── logger/           # Logging utilities
│   ├── search/           # Fuzzy search with multi-token support
│   └── tui/              # Terminal UI (Bubbletea)
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

### Creating a Personal Access Token

1. Go to your GitLab instance
2. Navigate to **User Settings** → **Access Tokens**
3. Create a new token with `read_api` scope
4. Copy the token and use it in `glf config`

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
glf config

# Check current configuration
cat ~/.config/glf/config.yaml
```

## 🎨 Color Scheme

GLF uses the **Monokai Pro Filter Spectrum** color scheme for a modern and comfortable visual experience:

- Cyan: Titles and highlights
- Orange/Peach: Prompts
- Pink/Red: Selection indicator (fzf-style)
- Green: Selected items
- Yellow: Filtered results counter
- White/Light Gray: Normal text

## ⚡ Performance

### Parallel Pagination

GLF uses intelligent parallel pagination to dramatically improve sync performance:

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| **648 projects** | ~26 seconds | ~3-5 seconds | **5-8x faster** |
| **2000 projects** | ~74 seconds | ~7-8 seconds | **10x faster** |

**How it works:**
- First request discovers total page count
- Launches up to 10 concurrent requests for remaining pages
- Uses goroutines with semaphore-based rate limiting
- Results are collected via channels and reassembled in order

**Verbose mode** shows real-time progress:
```bash
$ glf sync --verbose
[DEBUG] Total pages: 7, Total projects: 648
[DEBUG] Starting parallel fetch: 7 pages with max 10 concurrent requests
[DEBUG] Fetched page 2/7 (28%)
[DEBUG] Fetched page 7/7 (100%)
[DEBUG] Parallel fetch completed in 3.2s: fetched 648 projects from 7 pages
✓ Successfully fetched 648 projects
```

See [PERFORMANCE_OPTIMIZATION.md](PERFORMANCE_OPTIMIZATION.md) for detailed technical analysis.

## 📚 Technical Documentation

For in-depth implementation details:

- **[MULTI_TOKEN_SEARCH.md](MULTI_TOKEN_SEARCH.md)** - Multi-token search algorithm and performance analysis
- **[FREQUENCY_MEMORY_FEATURE.md](FREQUENCY_MEMORY_FEATURE.md)** - Selection frequency tracking and smart ranking implementation
- **[PERFORMANCE_OPTIMIZATION.md](PERFORMANCE_OPTIMIZATION.md)** - Parallel pagination and sync optimization

## 📝 License

TBD

## 🤝 Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## 📊 Project Phases

- ✅ **Phase 0**: Foundation (Project setup, dependencies)
- ✅ **Phase 1**: MVP (Basic sync, search, cache)
- ✅ **Phase 2**: Interactive TUI (Fuzzy finder interface)
- ✅ **Phase 3**: Polish (Config wizard, logging, cross-platform builds)
- ✅ **Phase 4**: Performance (Parallel pagination, non-blocking sync, fzf-like UI)

## 🙏 Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- UI powered by [Bubbletea](https://github.com/charmbracelet/bubbletea)
- Styling with [Lipgloss](https://github.com/charmbracelet/lipgloss)
- GitLab API via [go-gitlab](https://github.com/xanzy/go-gitlab)
