# GitHub Copilot Instructions for Kuberaptor

## Project Overview
Kuberaptor is a Kubernetes cluster management tool written in Go that automates cluster creation, management, and operations on Hetzner Cloud infrastructure. This project uses K3s (lightweight Kubernetes) and provides CLI commands for cluster lifecycle management.

**Project Type:** Go CLI application with embedded web marketing site (React/TypeScript)
**Go Version:** 1.24.0+ (toolchain 1.24.11)
**Primary Language:** Go
**Architecture:** Modular design with clear separation between CLI, business logic, and infrastructure

## Project Structure

```
kuberaptor/
├── cmd/kuberaptor/              # CLI application entry point
│   ├── main.go                # Application initialization
│   └── commands/              # Cobra CLI commands (create, delete, upgrade, run, releases)
├── internal/                  # Private application code
│   ├── cluster/               # Core cluster operations (create, delete, upgrade, run)
│   ├── config/                # Configuration management (YAML-based)
│   ├── cloudinit/             # Cloud-init template generation
│   ├── addons/                # Kubernetes addon management
│   └── util/                  # Utility functions (SSH, shell, file operations)
├── pkg/                       # Public reusable libraries
│   ├── hetzner/               # Hetzner Cloud API wrapper
│   ├── k3s/                   # K3s operations
│   └── templates/             # Template rendering
├── page/                      # React website (IGNORE THIS FOLDER)
├── Makefile                   # Build automation
├── go.mod                     # Go module dependencies
└── README.md                  # Comprehensive documentation
```

## Folders to Ignore

**IMPORTANT:** Ignore the `page/` directory - it contains the marketing website built with React, Vite, and TypeScript. Focus only on Go code in `cmd/`, `internal/`, and `pkg/` directories.

## Building the Project

### Quick Build
```bash
make build
# Binary output: dist/kuberaptor
```

### All Build Commands
- `make build` - Build for current platform
- `make build-linux` - Linux AMD64
- `make build-linux-arm` - Linux ARM64
- `make build-darwin-arm` - macOS ARM64
- `make build-all` - All platforms
- `make clean` - Clean build artifacts
- `make deps` - Download and tidy dependencies
- `make fmt` - Format code
- `make install` - Install to /usr/local/bin

### Build Details
- **Build time:** ~20-30 seconds
- **Binary size:** ~12MB
- **Output directory:** `dist/`
- **Ignored artifacts:** `dist/`, `coverage.out`, test binaries (`*.test`)

## Testing

### Run Tests
```bash
make test
# Runs: go test -v -race -coverprofile=coverage.out ./...
```

### Test Coverage
```bash
make coverage
# Generates HTML coverage report
```

### Test Files
- Test files follow `*_test.go` naming convention
- Located alongside source files in respective packages
- Use Go's built-in testing framework
- Race detector enabled by default
- Examples: `internal/cluster/create_enhanced_test.go`, `internal/cloudinit/generator_test.go`

## Security Audit Guidelines

### Security Considerations
1. **SSH Key Management:** Private keys in `internal/util/ssh.go` - handle with care
2. **API Tokens:** Hetzner Cloud API token in config - never log or expose
3. **Network Security:** Firewall rules and allowed networks in `internal/cluster/network_resources.go`
4. **Secrets in Code:** Avoid hardcoding tokens, passwords, or sensitive data
5. **Input Validation:** All config values validated in `internal/config/validator.go`

### Key Security Areas
- SSH client implementation: `internal/util/ssh.go`
- Cloud-init templates: `internal/cloudinit/` (user-data generation)
- Configuration loading: `internal/config/loader.go` (path expansion, validation)
- Network resources: `internal/cluster/network_resources.go` (firewall, load balancer)

### Security Best Practices
- Use CIDR validation for allowed networks
- Validate all user inputs before processing
- Use proper error handling (don't expose sensitive info in errors)
- Follow principle of least privilege for SSH access
- Keep dependencies updated (check `go.mod`)

## Code Style and Conventions

### Go Style
- **Formatting:** Use `go fmt` (enforced via `make fmt`)
- **Imports:** Group standard library, external, and internal imports
- **Error Handling:** Always return and check errors explicitly
- **Comments:** Document exported functions and types (godoc style)
- **Naming:** Use camelCase for private, PascalCase for exported identifiers
- **Constants:** Use UPPERCASE for constants at package level

### Architecture Patterns
- **Concurrency:** Use goroutines with sync.WaitGroup for parallel operations
- **Error Handling:** Return errors with context using `fmt.Errorf("msg: %w", err)`
- **Resource Cleanup:** Use `defer` for cleanup operations
- **Interfaces:** Define interfaces in consumer packages, not producer packages
- **Configuration:** YAML-based with validation and defaults

### Example Code Style
```go
// CreatorEnhanced handles cluster creation with full implementation
type CreatorEnhanced struct {
    Config        *config.Main
    HetznerClient *hetzner.Client
    SSHClient     *util.SSH
    ctx           context.Context
}

// NewCreatorEnhanced creates a new enhanced cluster creator
func NewCreatorEnhanced(cfg *config.Main, hetznerClient *hetzner.Client) (*CreatorEnhanced, error) {
    privKeyPath, err := cfg.Networking.SSH.ExpandedPrivateKeyPath()
    if err != nil {
        return nil, fmt.Errorf("failed to expand private key path: %w", err)
    }
    // ... implementation
}
```

## Dependencies

### Core Dependencies
- `github.com/hetznercloud/hcloud-go/v2` v2.34.0 - Official Hetzner Cloud SDK
- `github.com/spf13/cobra` v1.10.2 - CLI framework
- `golang.org/x/crypto` v0.47.0 - SSH client implementation
- `gopkg.in/yaml.v3` v3.0.1 - YAML parsing

### Dependency Management
- All dependencies statically linked into binary
- Use `make deps` to download and tidy dependencies
- Check `go.mod` for full dependency tree
- Keep dependencies minimal and up-to-date

## Validation Process

### Before Committing Changes
1. **Format code:** `make fmt`
2. **Run tests:** `make test`
3. **Build binary:** `make build`
4. **Test manually:** Run the binary with sample configs
5. **Run linter (if available):** `make lint` (requires golangci-lint)

### CI/CD Pipeline
- **Go Format:** Automatic formatting on PRs (`go_format.yml`)
- **PR Title Validation:** Enforces conventional commit format
- **Build for Release:** Multi-platform builds
- **Deploy Pages:** Automatic deployment of website (ignore this)

### Manual Testing Commands
```bash
# Build and test create command
./dist/kuberaptor create --config test-cluster.yaml --dry-run

# Test run command
./dist/kuberaptor run --config test-cluster.yaml --command "uptime"

# List K3s releases
./dist/kuberaptor releases
```

## Common Operations

### Adding a New CLI Command
1. Create command file in `cmd/kuberaptor/commands/`
2. Use Cobra framework pattern (see existing commands)
3. Add command to root command in `commands/root.go`
4. Implement business logic in `internal/cluster/` or appropriate package
5. Add tests alongside implementation

### Adding Configuration Options
1. Update struct in `internal/config/main.go`
2. Add YAML tag for field mapping
3. Add validation in `internal/config/validator.go`
4. Update SetDefaults if needed
5. Document in README.md

### Adding New Cloud Resources
1. Implement in `internal/cluster/` or `pkg/hetzner/`
2. Use existing patterns (context, error handling, logging)
3. Add resource cleanup in delete operation
4. Label resources with cluster name for tracking

## Key Technical Concepts

- **Embedded etcd:** High availability multi-master setup
- **K3s:** Lightweight Kubernetes distribution
- **Cloud-init:** Server initialization automation
- **Hetzner Cloud:** Infrastructure provider APIs
- **SSH Client Pooling:** Efficient connection reuse
- **Goroutine Concurrency:** Parallel operations for speed
- **YAML Configuration:** Human-readable cluster definitions

## Useful Commands

```bash
# Full development workflow
make fmt && make test && make build

# Check dependencies
go mod tidy && go mod verify

# View specific package tests
go test -v ./internal/cluster/

# Build with specific version
make build VERSION=1.0.0

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 make build
```

## Additional Notes

- **No runtime dependencies:** Binary is statically compiled
- **Fast iteration:** Quick build times enable rapid development
- **Type safety:** Compile-time type checking throughout
- **Comprehensive docs:** See README.md for detailed features and usage
- **Status:** Work in progress (WIP) - active development
- **License:** MIT

## When Writing Code

1. **Focus on Go code** - ignore the `page/` directory entirely
2. **Maintain modularity** - respect package boundaries
3. **Add error context** - use `fmt.Errorf` with `%w` for wrapping
4. **Test concurrent code** - race detector is enabled
5. **Validate configs** - all user input must be validated
6. **Document exports** - add godoc comments for public APIs
7. **Keep it simple** - prefer clarity over cleverness
8. **Static binary first** - avoid dynamic dependencies
