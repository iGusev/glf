# GitHub Actions Workflows

Comprehensive CI/CD automation for the GLF project.

## 📋 Overview

This project uses GitHub Actions for continuous integration, security scanning, and automated releases. All workflows are located in `.github/workflows/`.

## 🔄 Workflows

### 1. CI Workflow (`ci.yml`)

**Purpose**: Continuous integration testing across multiple Go versions and operating systems.

**Triggers**:
- Push to `main` branch
- Pull requests to `main` branch
- Manual dispatch

**Jobs**:

#### Test Job
- **Matrix**: Go 1.25.x, 1.24.x × Ubuntu, macOS, Windows
- **Actions**:
  - Run tests with race detector: `go test -v -race -coverprofile=coverage.out ./...`
  - Upload coverage to Codecov (Ubuntu + Go 1.25.x only)
- **Performance**: Parallel execution across all matrix combinations

#### Lint Job
- **Platform**: Ubuntu with Go 1.25.x
- **Tool**: golangci-lint (latest version)
- **Config**: Uses `.golangci.yml` with 30+ enabled linters
- **Timeout**: 5 minutes

#### Format Check Job
- **Platform**: Ubuntu with Go 1.25.x
- **Tool**: gofmt with simplify flag (`-s`)
- **Action**: Fails if any files need formatting

#### Build Job
- **Platform**: Ubuntu with Go 1.25.x
- **Action**:
  - Build binary using `make build`
  - Verify binary runs: `./glf --help`

**Caching**: Go modules are cached for faster builds

---

### 2. Release Workflow (`release.yml`)

**Purpose**: Automated releases with cross-platform binaries.

**Triggers**:
- Git tags matching `v*` (e.g., `v1.0.0`, `v2.3.1-beta`)
- Manual dispatch

**Process**:
1. **Build**: Compiles for all platforms using `make build-all`
   - Linux: amd64, arm64
   - macOS: amd64 (Intel), arm64 (Apple Silicon)
   - Windows: amd64
2. **Package**: Creates release archives using `make release`
   - Linux/macOS: `.tar.gz` files
   - Windows: `.zip` files
3. **Checksums**: Generates SHA256 checksums for verification
4. **Release**: Creates GitHub release with:
   - All binary archives
   - Checksum file
   - Auto-generated release notes
   - Custom installation instructions

**Versioning**: Version extracted from git tag

---

### 3. Security Workflow (`security.yml`)

**Purpose**: Automated security scanning for vulnerabilities and issues.

**Triggers**:
- Push to `main` branch
- Pull requests to `main` branch
- Weekly schedule (Mondays at 00:00 UTC)
- Manual dispatch

**Jobs**:

#### gosec (Security Scanner)
- **Tool**: gosec - Go security checker
- **Output**: SARIF format uploaded to GitHub Security tab
- **Checks**:
  - SQL injection risks
  - Command injection
  - File path traversal
  - Weak cryptography
  - And 50+ other security issues

#### govulncheck (Vulnerability Scanner)
- **Tool**: govulncheck - Official Go vulnerability scanner
- **Database**: Go vulnerability database
- **Action**: Scans dependencies for known CVEs

#### Dependency Review
- **Trigger**: Pull requests only
- **Tool**: GitHub's dependency review action
- **Threshold**: Fails on moderate+ severity vulnerabilities
- **Checks**: Reviews new/updated dependencies for security issues

**Permissions**: Requires `security-events: write` for SARIF upload

---

### 4. Dependabot (`dependabot.yml`)

**Purpose**: Automated dependency updates.

**Configuration**:

#### Go Modules
- **Schedule**: Weekly on Mondays
- **Limit**: Max 5 open PRs
- **Labels**: `dependencies`, `go`
- **Commit prefix**: `chore:`

#### GitHub Actions
- **Schedule**: Weekly on Mondays
- **Limit**: Max 5 open PRs
- **Labels**: `dependencies`, `github-actions`
- **Commit prefix**: `chore:`

---

## 🔐 Required Secrets

Configure these secrets in **Settings → Secrets and variables → Actions**:

### CODECOV_TOKEN
- **Required for**: Coverage reporting in CI workflow
- **How to get**:
  1. Sign up at [codecov.io](https://codecov.io)
  2. Add your GitHub repository
  3. Copy the upload token
  4. Add as repository secret
- **Alternative**: Can use GitHub token if repository is public

### GITHUB_TOKEN
- **Automatically provided**: GitHub creates this for each workflow run
- **Used for**: Creating releases, uploading SARIF files
- **No configuration needed**

---

## 📊 Badges

Add these badges to your README to display CI status:

```markdown
[![CI](https://github.com/igusev/glf/workflows/CI/badge.svg)](https://github.com/igusev/glf/actions/workflows/ci.yml)
[![Security](https://github.com/igusev/glf/workflows/Security/badge.svg)](https://github.com/igusev/glf/actions/workflows/security.yml)
[![codecov](https://codecov.io/gh/igusev/glf/branch/main/graph/badge.svg)](https://codecov.io/gh/igusev/glf)
[![Go Report Card](https://goreportcard.com/badge/github.com/igusev/glf)](https://goreportcard.com/report/github.com/igusev/glf)
```

---

## 🚀 Usage

### Running Tests Locally

Match CI environment:
```bash
# Run all tests with race detector
go test -v -race -coverprofile=coverage.out ./...

# View coverage
go tool cover -html=coverage.out
```

### Running Linters Locally

```bash
# Using golangci-lint (recommended)
golangci-lint run ./...

# Or via Makefile
make lint
```

### Creating a Release

1. **Tag the release**:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

2. **Automatic process**:
   - Release workflow triggers automatically
   - Builds all platform binaries
   - Creates GitHub release with artifacts
   - Generates checksums

3. **Manual release** (if needed):
   - Go to Actions → Release workflow
   - Click "Run workflow"
   - Enter tag name

---

## 🔧 Configuration Files

### `.golangci.yml`
Comprehensive linting configuration with:
- 30+ enabled linters
- Customized rules for error checking, formatting, security
- Test file exceptions for certain linters

### `.github/dependabot.yml`
Automated dependency updates for:
- Go modules (weekly)
- GitHub Actions (weekly)

---

## 📈 Performance Optimizations

1. **Caching**: Go modules cached across workflow runs
2. **Matrix Strategy**: Parallel execution across platforms
3. **Selective Coverage**: Upload only once (Ubuntu + Go 1.25.x)
4. **Fail-Fast Disabled**: See all test failures, not just first

---

## 🐛 Troubleshooting

### CI Failing on Format Check
```bash
# Fix locally
go fmt ./...
git commit -am "Fix formatting"
git push
```

### Linter Failures
```bash
# Run locally to see issues
golangci-lint run ./...

# Fix automatically where possible
golangci-lint run --fix ./...
```

### Release Not Creating
- **Check**: Tag format must match `v*` (e.g., `v1.0.0`)
- **Verify**: Tag pushed to GitHub: `git push origin --tags`
- **Permissions**: Ensure `contents: write` permission in workflow

### Security Scan False Positives
- Review in GitHub Security tab
- If false positive, add exclusion in `.golangci.yml` or gosec config
- Document reason for exclusion

---

## 📚 Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [golangci-lint Linters](https://golangci-lint.run/usage/linters/)
- [gosec Rules](https://github.com/securego/gosec#available-rules)
- [Go Vulnerability Database](https://vuln.go.dev/)
- [Codecov Documentation](https://docs.codecov.com/)

---

## 🎯 Best Practices

1. **Always run tests locally** before pushing
2. **Keep dependencies updated** - review Dependabot PRs promptly
3. **Monitor security alerts** - check Security tab regularly
4. **Use semantic versioning** for releases
5. **Test on multiple platforms** if making OS-specific changes
6. **Review coverage reports** - aim for >80% coverage
7. **Address linter warnings** - don't disable without good reason

---

## 🔄 Workflow Maintenance

### Updating Go Version
When new Go version released:

1. Update `go.mod`:
   ```bash
   go mod edit -go=1.26
   ```

2. Update workflows:
   - `.github/workflows/ci.yml`: Add new version to matrix
   - `.github/workflows/release.yml`: Update to latest stable
   - `.github/workflows/security.yml`: Update to latest stable

3. Update badges in README.md

### Adding New Linters
1. Add to `.golangci.yml` under `linters.enable`
2. Configure settings under `linters-settings` if needed
3. Test locally: `golangci-lint run ./...`
4. Commit and push - CI will use new config

---

## 📞 Support

If workflows fail unexpectedly:
1. Check workflow logs in Actions tab
2. Verify required secrets are configured
3. Review recent changes to workflow files
4. Open an issue with workflow run link
