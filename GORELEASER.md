# GoReleaser Configuration

This project uses [GoReleaser](https://goreleaser.com/) for building and releasing binaries across multiple platforms.

## Supported Platforms

The project builds for **13 platform/architecture combinations**:

### Operating Systems
- **macOS (darwin)**: Intel (amd64), Apple Silicon (arm64)
- **Linux**: amd64, arm64, armv5, armv6, armv7, loong64, ppc64le, s390x
- **Windows**: amd64
- **FreeBSD**: amd64
- **OpenBSD**: amd64

## Creating a Release

### Automatic Release (via GitHub Actions)

1. Create and push a version tag:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. GitHub Actions will automatically:
   - Build binaries for all platforms
   - Create archives (tar.gz for Unix, zip for Windows)
   - Generate SHA256 checksums
   - Create GitHub Release with changelog
   - Upload all artifacts

### Manual Release (local testing)

```bash
# Test configuration
goreleaser check

# Test build without release
goreleaser build --snapshot --clean

# Full test with archives
goreleaser release --snapshot --clean --skip=publish

# Clean up
rm -rf dist/
```

## Release Contents

Each release includes:
- Binary for the target platform
- `README.md`
- `LICENSE`
- `checksums.txt` with SHA256 hashes for verification

## Configuration

The configuration is in `.goreleaser.yml` and includes:

- **Build flags**: `-trimpath` for reproducible builds
- **Linker flags**: Version, commit hash, and build time embedded in binary
- **Archives**: Automatic compression with appropriate format per platform
- **Changelog**: Generated from git commits with conventional commit grouping
  - Features (`feat:`)
  - Bug Fixes (`fix:`)
  - Others

## Verification

Users can verify downloaded binaries:

```bash
# Download checksums
curl -LO https://github.com/igusev/glf/releases/download/v0.1.0/checksums.txt

# Download binary
curl -LO https://github.com/igusev/glf/releases/download/v0.1.0/glf-0.1.0-linux_amd64.tar.gz

# Verify checksum
sha256sum -c checksums.txt --ignore-missing
```

## Comparison with Previous Setup

**Before (Makefile)**:
- 5 platforms: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64
- Manual archive creation
- Manual checksum generation
- Manual changelog

**After (GoReleaser)**:
- 13 platforms including ARM variants and server architectures
- Automatic archive creation
- Automatic checksum generation
- Automatic changelog from commits
- Better GitHub integration

## References

- [GoReleaser Documentation](https://goreleaser.com/)
- [fzf's goreleaser config](https://github.com/junegunn/fzf/blob/master/.goreleaser.yml) (used as reference)
