# 🏠 Gorphanage

> **A home for finding Go's lost code.** Discover orphaned symbols with surgical precision using advanced reachability analysis.

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.19-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

**Gorphanage** uses reachability analysis to identify truly dead code in Go projects. Unlike simple grep-based tools, it traces execution paths from `main()` functions to find symbols that are genuinely unreachable.

## ✨ Features

- **🎯 Precise Detection** - Uses Go's type system and AST for analysis
- **🚀 Reachability Tracing** - BFS algorithm starting from main package entry points
- **📊 Smart Analysis** - Handles complex dependency chains and indirect references
- **🛠️ Professional CLI** - Built with Cobra & Viper for excellent UX
- **⚙️ Flexible Configuration** - YAML config files, environment variables, and CLI flags
- **🎨 Multiple Output Formats** - Human-readable terminal output and JSON for tooling
- **📋 Smart Exclusions** - Exclude vendor code, generated files, and custom patterns
- **⚡ Fast & Reliable** - Leverages the same engine that powers `gopls`

## 🚀 Installation

### Option 1: Install from Source (Recommended)

```bash
go install github.com/mirrir0/gorphanage@latest
```

### Option 2: Install Script

```bash
curl -sSL https://raw.githubusercontent.com/mirrir0/gorphanage/main/install.sh | bash
```

### Option 3: Clone and Build

```bash
git clone https://github.com/mirrir0/gorphanage.git
cd gorphanage
make install
```

### Option 4: Download Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/yourusername/gorphanage/releases).

## 🔧 Quick Start

### Basic Usage

```bash
# Analyze current directory
gorphanage .

# Analyze specific project
gorphanage /path/to/your/go/project

# Verbose output with progress
gorphanage --verbose .

# JSON output for tooling integration
gorphanage --json . > orphans.json
```

### Advanced Usage

```bash
# Exclude specific packages
gorphanage --exclude "vendor/*,*.pb.go" .

# Include test files in analysis
gorphanage --include-tests .

# Multiple exclusion patterns
gorphanage -e vendor -e generated -e "*.pb.go" .

# Use custom config file
gorphanage --config ./custom-config.yaml .
```

## 📊 Example Output

### Clean Project
```bash
$ gorphanage .
🔍 Analyzing project at: /home/user/myproject
📦 Loaded 8 packages
🔍 Found 147 symbols
📦 Found 2 main package(s)
🔍 Tracing reachability from main packages...
📊 Reachability analysis: 132/147 symbols reachable from main packages

✅ No orphaned code found!
All symbols are reachable from main package entry points.
```

### Issues Found
```bash
$ gorphanage --verbose .
🔍 Analyzing project at: /home/user/myproject
📦 Loaded 8 packages
🔍 Found 147 symbols
📦 Found 2 main package(s)
    github.com/user/myproject/cmd/server
    github.com/user/myproject/cmd/client
🔍 Tracing reachability from main packages...
🎯 Starting with 6 entry points
📊 Reachability analysis: 132/147 symbols reachable from main packages

🗑️  ORPHANED CODE ANALYSIS
Found 15 symbols that are NOT reachable from any main package:

=== Functions ===
  📍 processLegacyData (private) - internal/legacy.go:67:1
  📍 ExportedButUnused (exported) - pkg/api.go:34:1
  📍 helperFunc (private) - utils/string.go:123:1

=== Types ===
  📍 OldConfig (exported) - config/deprecated.go:18:1
  📍 internalState (private) - state/manager.go:45:1

=== Variables ===
  📍 debugFlag (private) - main.go:15:1

💡 These symbols are not reachable from any main() or init() function.
💡 Test functions are excluded as they have separate entry points.
💡 Analysis based on 2 main package(s) found in the project.

📊 Analysis Summary:
  • Total symbols: 147
  • Reachable symbols: 132
  • Orphaned symbols: 15
  • Orphan rate: 10.2%
```

### JSON Output
```bash
$ gorphanage --json .
{
  "project_path": "/home/user/myproject",
  "total_symbols": 147,
  "reachable_symbols": 132,
  "main_packages": 2,
  "excluded_packages": ["vendor/*", "*.pb.go"],
  "included_tests": false,
  "orphaned_symbols": [
    {
      "name": "processLegacyData",
      "kind": "function",
      "file": "/home/user/myproject/internal/legacy.go",
      "start": { "line": 67, "column": 1 },
      "end": { "line": 74, "column": 2 },
      "exported": false,
      "package": "github.com/user/myproject/internal"
    }
  ]
}
```

## ⚙️ Configuration

### Configuration File

Create a configuration file for persistent settings:

```bash
# Initialize default config
gorphanage config init

# Show current configuration
gorphanage config show
```

Example `~/.gorphanage.yaml`:

```yaml
# Output settings
json: false
verbose: false

# Analysis options
include-tests: false

# Exclude patterns (glob patterns for package paths)
exclude:
  - "vendor/*"              # Vendor dependencies
  - "*.pb.go"               # Protocol buffer generated files
  - "*_generated.go"        # Generated Go files
  - "third_party/*"         # Third-party code
  - "mocks/*"               # Mock implementations
  - "testdata/*"            # Test data directories
```

### Environment Variables

```bash
export GORPHANAGE_VERBOSE=true
export GORPHANAGE_EXCLUDE="vendor/*,generated/*"
export GORPHANAGE_JSON=true
gorphanage .  # Uses environment settings
```

### Command Line Flags

```bash
Usage: gorphanage [flags] <project-path>

Flags:
  -e, --exclude strings      exclude packages matching these patterns
  -h, --help                help for gorphanage
      --include-tests       include test files in analysis
      --json                output results in JSON format
  -v, --verbose             verbose output
      --version             version for gorphanage

Global Flags:
      --config string       config file (default is $HOME/.gorphanage.yaml)
```

## 🎯 How It Works

Gorphanage uses a sophisticated **reachability analysis** algorithm:

1. **📦 Package Discovery** - Loads all Go packages with full type information
2. **🔍 Symbol Mapping** - Identifies all functions, types, variables, and constants
3. **🎯 Entry Point Detection** - Finds `main()` and `init()` functions as starting points
4. **🌊 BFS Traversal** - Traces all possible execution paths from entry points
5. **💀 Orphan Detection** - Reports symbols not reached during traversal

### Why This Approach?

Traditional tools count references, but **Gorphanage** understands execution flow:

```go
// ❌ Simple tools miss this
func main() {
    if false {
        deadFunction() // This is actually dead!
    }
}

// ✅ Gorphanage catches it
// Uses control flow analysis, not just reference counting
```

## 🛡️ Safety Features

- **🧪 Test-Aware** - Automatically excludes `Test*`, `Benchmark*`, and `Example*` functions
- **📚 Library-Safe** - Adapts behavior for library vs application projects
- **🔒 Conservative** - When in doubt, preserves code rather than flagging it
- **📍 Precise Locations** - Shows exact file and line numbers for easy cleanup
- **🎨 Smart Filtering** - Configurable exclusion patterns for generated code

## 🔧 CI/CD Integration

### GitHub Actions

```yaml
name: Dead Code Check
on: [push, pull_request]

jobs:
  orphan-check:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install Gorphanage
      run: go install github.com/yourusername/gorphanage@latest
    
    - name: Check for orphaned code
      run: |
        gorphanage --json . > orphans.json
        if [ "$(jq '.orphaned_symbols | length' orphans.json)" -gt 0 ]; then
          echo "❌ Orphaned code detected!"
          jq '.orphaned_symbols[] | "\(.file):\(.start.line): \(.name) (\(.kind))"' orphans.json
          exit 1
        fi
        echo "✅ No orphaned code found"
```

### Pre-commit Hook

```bash
#!/bin/sh
# .git/hooks/pre-commit
echo "🔍 Checking for orphaned code..."
gorphanage . || exit 1
```

### Makefile Integration

```makefile
.PHONY: check-orphans
check-orphans:
	@echo "🔍 Checking for orphaned code..."
	@gorphanage --json . | jq -e '.orphaned_symbols | length == 0' > /dev/null || \
		(echo "❌ Orphaned code found. Run 'gorphanage .' for details" && exit 1)
	@echo "✅ No orphaned code found"
```

## 🆚 Comparison

| Tool | Method | Accuracy[^1] | Go-Aware | Config | JSON Output |
|------|--------|----------|----------|--------|-------------|
| **Gorphanage** | Reachability Analysis | 99.9% | ✅ | ✅ | ✅ |
| `deadcode` | Reference Counting | 85% | ✅ | ❌ | ❌ |
| `ineffassign` | Assignment Analysis | 70% | ✅ | ❌ | ❌ |
| `grep -r "funcName"` | Text Search | 60% | ❌ | ❌ | ❌ |

[^1]: I **Completely** made this up.
## 📚 Advanced Features

### Custom Entry Points

For complex applications with non-standard entry points:

```yaml
# Future feature
entry-points:
  - "github.com/myorg/myproject/pkg/plugin.Init"
  - "github.com/myorg/myproject/pkg/server.Start"
```

### Performance Tuning

```yaml
# Future features
max-depth: 100
timeout: "10m"
parallel: true
cache: true
```

## 🤝 Contributing

We love contributions! Here's how to get started:

1. **🍴 Fork** the repository
2. **🌿 Branch** from `main`: `git checkout -b feature/amazing-feature`
3. **📝 Commit** your changes: `git commit -m 'Add amazing feature'`
4. **🚀 Push** to your fork: `git push origin feature/amazing-feature`
5. **📬 Submit** a Pull Request

### Development Setup

```bash
git clone https://github.com/yourusername/gorphanage.git
cd gorphanage
make dev        # Build with race detection
make test       # Run tests
make lint       # Run linter
```

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

- **Go Team** - For the amazing `go/packages` and AST libraries
- **gopls** - Inspiration for the analysis approach
- **Cobra & Viper** - For excellent CLI framework
- **Community** - All the contributors and users

---

<div align="center">

**Made with ❤️ for the Go community**

[Report Bug](https://github.com/yourusername/gorphanage/issues) · [Request Feature](https://github.com/yourusername/gorphanage/issues) · [Discussions](https://github.com/yourusername/gorphanage/discussions)

</div>
