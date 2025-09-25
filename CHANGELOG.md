# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - Unreleased

### Added
- **Parser Registry**
    - Introduced `LanguageParser` interface (`ProcessFiles`, `Language`, `FileExtensions`).
    - Added `registry.go` for pluggable parser registration (`Register`, `Get`, `SupportedLanguages`).
    - Added `progress.go` with generic `ProcessFilesWithProgress` wrapper for progress bars.
    - Integration test ensures at least one parser is always registered.
- **JavaScript Stub Parser**
    - Implemented `JSParser` as a minimal parser for `.js`, `.mjs`, `.cjs` files to prove pluggability.

### Changed
- **PHP Parser**
    - Renamed `Parser` → `PHPParser`.
    - Now implements `LanguageParser` and self-registers via `init()`.
- **Scanner**
    - Refactored to accept parser-defined file extensions via `SetExtensions`.
    - Golden test updated to set extensions dynamically from the parser.
- **CLI**
    - Added `--lang` flag to choose parser (default: `php`).
    - Main now looks up parsers via the registry instead of hard-wired PHP.

### Fixed
- **Tests**
    - Updated `php_test.go` to use `NewPHPParser()`.
    - Fixed golden test mismatch by applying parser extensions.
    - Added unit tests for registry (`Register`, `Get`, `SupportedLanguages`, duplicate panic).
    - Added smoke test for progress wrapper.
    - Added integration test to ensure parsers are always registered.
- **CLI**
    - Fixed the project name being the incorrect case.

## [0.1.0] - 2025-09-24

### Added
- **File Scanner**
    - Recursive discovery of PHP files with extension list (`.php`, `.phtml`, `.php3`, `.php4`, `.php5`).
    - Exclude common directories (`vendor`, `node_modules`, `.git`, `storage`, etc.).
    - Records file path, relative path, and size as `FileInfo`.
- **Parser (PHP)**
    - Supports namespaces, `use` imports, classes, methods, functions, and constants.
    - Extracts metadata: visibility, static/abstract flags, parameters, return types, line numbers.
    - Usage detection for instantiations, method calls, and function calls (excluding builtins).
    - Concurrent file parsing via `ProcessFiles`.
- **Analyzer**
    - Dependency graph construction with nodes, edges, orphans, “highly depended” elements, and “complex” nodes.
    - Complexity scoring for classes, functions, and properties.
    - Exports graph to JSON.
- **Output**
    - **ConsoleFormatter**: summary and dependency report, with verbose mode showing additional details.
    - **JSONExporter**: exports full analysis results and dependency graph to JSON.
- **Progress Indicators**
    - A simple progress bar and spinner utilities for CLI feedback.
- **CLI**
    - `cmd/tukey`: entrypoint with flags for verbosity, output file, and directory excludes.
    - `--help` and `--version` support with build metadata (`main.version`, `main.commit`, `main.date`).
- **Configuration**
    - Example configs under `configs/` for customizing excludes, includes, concurrency, and reporting.
- **CI/CD**
    - GitHub Actions workflow for CI (tests, race detector, coverage, smoke build).
    - GitHub Actions workflow for tagged releases: cross-platform binaries, `.tar.gz`/`.zip` packaging, `SHA256SUMS`, and GitHub Release upload.
- **Testing**
    - Unit tests for scanner, parser, analyzer, CLI arg parsing, and console/json output.
    - Golden test pattern with `testdata/` fixtures for deterministic scanner/parser expectations.