# Tukey

A high-performance static analysis tool that maps code dependencies, highlights complexity, and uncovers dead code across
large projects. Designed to be **language-agnostic**, the engine can analyze code architecture and usage patterns in any
language.

The initial release focuses on **PHP support**, with additional languages planned for the future.

[![Go Report Card](https://goreportcard.com/badge/github.com/boone-studios/tukey)](https://goreportcard.com/report/github.com/boone-studios/tukey)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- 🔍 Deep Code Analysis — Parses source files to extract structural elements (classes, functions, methods, properties)
- 🕸️ Dependency Mapping — Builds comprehensive graphs showing code relationships
- 📊 Complexity Metrics — Identifies areas of high complexity
- 🎯 Usage Tracking — Finds where functions and classes are used across the project
- 🤖 Native MCP Server — Exposes code dependency search, caller tracing, and dead code detection directly to AI agents (like Cursor, Claude Desktop, and Zed)
- 👻 Dead Code Detection — Flags unused or orphaned code
- ⚡ High Performance — Concurrent processing for fast analysis of large projects

## Installation

### From Source

```bash
git clone https://github.com/boone-studios/tukey.git
cd tukey
make install
```

### Using Go Install

```bash
go install github.com/boone-studios/tukey/cmd/tukey@latest
```

### Download Binary

Download the latest release from the [releases page](https://github.com/boone-studios/tukey/releases).

## Quick Start

```bash
# Basic analysis
tukey /path/to/your/php/project

# Verbose output with function usage report
tukey -v /path/to/your/php/project

# Export results to JSON
tukey -v --output analysis.json /path/to/your/php/project

# Exclude directories
tukey --exclude vendor --exclude tests /path/to/your/php/project

# Start a native Model Context Protocol (MCP) server for AI agents
tukey mcp analysis.json
```

## Configuration

You can configure Tukey by creating a `.tukey.yml` file in the root of your project.

```yaml
language: php
outputFile: analysis.json
verbose: true
excludeDirs:
  - bootstrap
  - public
```

If you prefer JSON, you can use a `.tukey.json` file instead.

```json
{
  "language": "php",
  "outputFile": "analysis.json",
  "verbose": true,
  "excludeDirs": [
    "bootstrap",
    "public"
  ]
}
```

## AI Agent Integration (Tukey Skill & MCP Server)

Tukey is natively designed to act as a **machine-readable codebase map for AI agents** (like Gemini, Claude, Cursor, and other agentic coding assistants). If you are using AI agents to explore or edit your codebase, Tukey offers two integration pathways:

### 1. Native Model Context Protocol (MCP) Server
Tukey features a built-in MCP server that communicates via JSON-RPC 2.0 over standard I/O (stdin/stdout). This allows compatible tools (e.g., Claude Desktop, Cursor, Zed) to invoke Tukey tools directly within the agent's tool-use loop without shell execution permissions or subprocess spawning overhead.

#### Quick Start (MCP)
1. **Analyze your codebase** to generate the graph file:
   ```bash
   tukey -o tukey-results.json /path/to/your/project
   ```
2. **Start the MCP server**:
   ```bash
   tukey mcp tukey-results.json
   ```

To configure the server in your MCP host (like `mcpSettings.json`), add:
```json
{
  "mcpServers": {
    "tukey": {
      "command": "tukey",
      "args": ["mcp", "/absolute/path/to/your/tukey-results.json"]
    }
  }
}
```

Exposed MCP Tools:
- `tukey_find_symbol` — Locate classes, methods, and functions.
- `tukey_get_callers` — Trace what calls/references a symbol.
- `tukey_get_dependents` — Trace what a symbol depends on.
- `tukey_find_orphans` — Identify candidate dead or orphaned code.

### 2. Tukey Command Line Skill
You can also feed Tukey's compact query capabilities directly to LLMs as a CLI skill. For a detailed guide on CLI-based agent configurations, see [docs/tukey_skill.md](docs/tukey_skill.md).

## Use Cases

### Legacy Code Understanding
Perfect for analyzing inherited PHP codebases with little documentation:

```bash
tukey -v ./legacy-project
```

**Output shows:**
- Most critical classes (highly depended upon)
- Dead code candidates (orphaned functions)
- Complex areas needing refactoring
- Helper function usage patterns

### Function Usage Tracking
Find where specific functions like `format_phone()` are used:

```bash
📋 FUNCTION USAGE REPORT
======================================================================
📁 app/Lib/helpers.php
  📋 function format_phone() (line 15) - 8 calls
  🔗 Called from 8 locations:
    📂 app/Http/Controllers/UserController.php:
      → line 45 in store()
      → line 78 in update()
```

### Refactoring Planning
Identify refactoring opportunities:
- **God Classes** - High complexity scores
- **Tight Coupling** - Classes with many dependencies
- **Circular Dependencies** - Problematic architectural patterns

## Output Examples

### Console Summary
```
📊 Graph Statistics:
   • Total Nodes: 1,284
   • Total Dependencies: 2,891
   • Orphaned Elements: 23

🔥 Most Depended Upon Elements:
   1. Database (helpers/Database.php) - 47 dependents
   2. Utils (lib/Utils.php) - 34 dependents

🧠 Most Complex Elements:
   1. OrderController (Http/Controllers/OrderController.php) - Score: 89
   2. UserService (Services/UserService.php) - Score: 67
```

### JSON Export
```json
{
  "nodes": {
    "class:App\\Models\\User:8": {
      "id": "class:App\\Models\\User:8",
      "name": "User",
      "type": "class",
      "file": "/app/Models/User.php",
      "dependencies": {...},
      "dependents": {...}
    }
  },
  "totalNodes": 1284,
  "totalEdges": 2891
}
```

## How It Compares

| Tool                   | Language Focus                   | Primary Purpose                                | Output Style                 | Complexity/Dependency Metrics   | Multi-language     | CI/CD Friendly      | Footprint                     |
| ---------------------- | -------------------------------- | ---------------------------------------------- | ---------------------------- | --------------------------------|--------------------| ------------------- | ----------------------------- |
| **Tukey**              | PHP first (pluggable for others) | **Maps dependencies, complexity, and orphans** | Console summary, JSON export | ✅ Yes (graph, hotspots, orphans) | 🌍 Designed for it | ✅ Simple JSON + CLI | ⚡ Lightweight (single binary) |
| PHPStan                | PHP                              | Type safety, strict type checking              | CLI, IDE integration         | ❌ No                            | ❌ No               | ✅ Yes               | ⚖️ Medium (lots of rules)     |
| Psalm                  | PHP                              | Type checking + code correctness               | CLI, IDE integration         | ❌ No                            | ❌ No               | ✅ Yes               | ⚖️ Medium                     |
| PDepend                | PHP                              | Code metrics, class dependencies               | XML, charts, reports         | ✅ Yes (metrics & graphs)        | ❌ No               | ⚠️ Limited          | 🐘 Heavier (XML reports)      |
| phpmetrics             | PHP                              | High-level project health reports              | HTML dashboards              | ✅ Yes (wrapped from PDepend)    | ❌ No               | ⚠️ Limited          | 🐘 Heavier (GUI focus)        |
| SonarQube              | Many (20+)                       | Enterprise-grade code quality + coverage       | Web dashboards, DB backend   | ✅ Yes (lots, but buried)        | ✅ Yes              | ✅ Deep CI/CD        | 🏢 Heavy (server required)    |
| SourceTrail (archived) | C++, Java, Python                | Interactive code exploration (graph viewer)    | GUI (desktop)                | ✅ Yes (visual graph)            | ❌ Limited          | ❌ No                | 💻 Desktop app only           |

---

### Key Differentiators

* **Tukey is not a linter**: it doesn’t enforce style or types. Instead, it **draws the map** of your system.
* **Output is lightweight**: JSON + console means you can plug it into CI pipelines or explore locally without dashboards.
* **Language-agnostic design**: while starting with PHP, the parser interface makes adding new languages straightforward.
* **Zero infrastructure**: unlike SonarQube, Tukey is just a single binary — no servers, no databases.

For an in-depth, philosophical and technical analysis of how Tukey compares to enterprise quality gates, see the [SonarQube Comparison Guide](docs/sonarqube_comparison.md).

## Architecture

The tool follows clean architecture principles:

- **`cmd/`** - Application entry points
- **`internal/`** - Private application code
- **`pkg/`** - Public library code
- **`testdata/`** - Test fixtures
- **`docs/`** - Documentation

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Development

```bash
# Setup development environment
make deps

# Run tests
make test

# Run with coverage
make test-coverage

# Format code
make fmt

# Run linter
make vet

# Build for development
make dev ARGS="-v ./testdata/sample_project"
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Roadmap

For an in-depth, multi-phase breakdown of Tukey's long-term vision, proposed implementation details, and tracking, see the [Development Roadmap](ROADMAP.md).

Quick summary of upcoming targets:
- [ ] Circular dependency detection & cycles analysis
- [ ] Version Control (Git) Integration (complexity vs. churn and blast radius analysis)
- [ ] Architectural boundary enforcement & layer guardrails (`tukey check`)
- [ ] AI Agent MCP micro-graph context pruning
- [ ] Polyglot analysis (TypeScript/Go) & dynamic execution traces
- [ ] Web dashboard for interactive dependency visualization
- [ ] Integration with popular IDEs

## Acknowledgments

- Inspired by the need to understand complex legacy PHP codebases
- Built with Go for performance and cross-platform compatibility