# Tukey: Agent Skill & Tool Guide

This document defines the **Tukey static analysis skill** for AI agents. If you are an AI agent analyzing, exploring, or debugging a codebase that has Tukey installed, read this file to understand how to leverage Tukey to solve tasks in fractions of a second with **ultra-low token overhead**.

---

## 💡 What is Tukey?

Tukey is a high-performance static analysis tool that scans codebases, extracts structural symbols (classes, functions, methods, enums, constants, imports), and builds a unified **directed dependency graph**. 

Instead of running dozens of expensive file read and search operations in a loop (which consumes high token count and memory), you can run Tukey once to generate a results file and query it in $O(1)$ time.

---

## 🛠️ When to Use Tukey

Use this skill when you need to:
1. **Map a new codebase**: Understand project structure, module relationships, and full-stack flow.
2. **Trace dependency paths**: Find what calls a function, or what a class depends on.
3. **Locate dead code**: Discover completely unused or orphaned classes and functions.
4. **Identify complex elements**: Find files with high structural complexity to pinpoint refactoring targets or hot spots.
5. **Optimize token usage**: Query symbol locations, callers, and relationships without opening files.

---

## 🚀 Commands & Usage

### 1. Build the Codebase Graph (Analysis Phase)
Run the analysis phase to discover code elements, track imports/usage, and export the unified graph:
```bash
# Analyze a codebase (supports polyglot php, js scanning concurrently)
tukey -l php,js -o tukey-results.json ./path/to/project
```

#### CLI Flags:
* `-l, --language <keys>`: Comma-separated list of active languages (e.g., `php`, `js`, or `php,js` for polyglot full-stack projects). Defaults to `php`.
* `-o, --output <file>`: Filepath to export the complete analysis graph JSON (highly recommended).
* `--exclude <dir>`: Directory to ignore (can be used multiple times; `vendor` and `node_modules` are excluded by default).
* `-v, --verbose`: Print verbose summaries and detailed function usage reports directly to stdout.
* `-b, --benchmark`: Quietly run the analysis without terminal spinners, outputting high-precision timing and system resource benchmarking telemetry.

---

### 2. Compare Against a Baseline (Change & Regression Auditing)
Use the `--compare` (or `-c`) flag to diff the current state of the codebase against a previously exported baseline JSON file. This is highly useful for self-auditing your own structural changes:
```bash
# Audits current changes against the baseline report
tukey --compare baseline.json ./path/to/project

# Exit with non-zero status code in CI/CD if architectural violations exist
tukey --compare baseline.json --strict ./path/to/project
```

Tukey will analyze the active graph and report:
* **Codebase Structural Changes:** Additions/deletions of elements and total dependencies.
* **Deleted Elements:** Nodes removed from the project since the baseline was generated.
* **Cleaned Orphans:** Stale orphaned nodes that were successfully integrated or pruned.
* **Complexity Regressions:** Specific classes or methods whose structural complexity scores have increased.

---

### 3. Query the Codebase Graph (Query Phase)
Once you have generated `tukey-results.json`, use the query engine to perform targeted, lightning-fast symbol lookups. **Never parse the entire JSON file into your context.** Always run `tukey query` to retrieve only the relevant subset of results.

```bash
# Query the analysis file
tukey query [FLAG] <term> tukey-results.json
```

#### Query Modes (Mutually Exclusive):
| Flag | What it returns | Supported Query Terms |
|---|---|---|
| `--find <term>` | Nodes matching or containing the term (case-insensitive substring). | Names, namespace segments, folders, or exact IDs. |
| `--callers <term>` | All nodes that call or reference the symbol (incoming edges). | Short names, class-scoped members, FQDNs, or exact IDs. |
| `--dependents <term>` | All nodes that the named symbol directly depends on (outgoing edges). | Short names, class-scoped members, FQDNs, or exact IDs. |
| `--orphans` | All dead-code candidates (zero dependencies and zero dependents). | None (lists all matching elements in the graph). |
| `--diagnostics` | Parser errors and unresolved or ambiguous references, aggregated by file and symbol. | None. |
| `--status` | Freshness status and added, modified, or deleted source files. | None. |

#### Target Symbol Formats Supported:
Tukey's query engine is highly resilient and supports matching a symbol in several ways:
1. **Exact Node ID**: `class:App\Services\PaymentService` or `method:App\Services\PaymentService::pay` (Fastest $O(1)$ resolution).
2. **Fully-Qualified Member**: `App\Services\PaymentService::pay`
3. **Class-Scoped Member**: `PaymentService::pay`
4. **Namespaced Short Name**: `App\Services\PaymentService`
5. **Short Name**: `PaymentService` or `pay`

Caller, dependency, and localized-context responses include a `resolution` object with `matchType` (`id`, `exact`, `substring`, or `not_found`), `ambiguous`, and all matched `targets`. Their `relationships` array preserves each `sourceId`, `targetId`, reference `type`, `lines`, `count`, and traversal `depth`. When `ambiguous` is true, repeat the query with a qualified name or exact ID instead of treating the merged summary as one symbol.

---

## 📋 JSON Query Output Schema

Every `tukey query` returns a predictable JSON envelope. Symbol queries contain node results:
```json
{
  "query": "find | callers | dependents | orphans | diagnostics | status",
  "term": "<search term>",
  "count": 1,
  "results": [
    {
      "id": "class:App\\Services\\PaymentService",
      "name": "PaymentService",
      "type": "class",
      "file": "app/Services/PaymentService.php",
      "namespace": "App\\Services",
      "className": "",
      "line": 5,
      "score": 18,
      "dependencyCount": 3,
      "dependentCount": 5,
      "refType": "instantiation",
      "refLines": [30, 45],
      "refCount": 2
    }
  ]
}
```

* `term` is present only for `find`, `callers`, and `dependents`; `orphans`, `diagnostics`, and `status` omit it.
* `results` is always a JSON array (never `null` or empty on misses; it will be `[]`). For caller/dependency queries it is a unique-node summary.
* `refType`, `refLines`, and `refCount` remain for compatibility, but may summarize only the first target relationship for a node. Use `relationships` when target identity matters.

`--diagnostics` results instead contain `code`, `severity`, `file`, `message`, `count`, optional `symbol` and `referenceType`, and up to 20 sampled `lines`. The envelope's `count` is the number of diagnostic groups; each result's `count` is the number of occurrences in that group. Codes are `parse_error`, `unresolved_source`, `unresolved_reference`, and `ambiguous_reference`.

`--status` returns `status` (`fresh`, `stale`, or `unknown`), snapshot metadata, `checkedFiles`, `changeCount`, and a `changes` array whose entries contain `path` and `change` (`added`, `modified`, or `deleted`). Older exports without source metadata return `unknown`.

---

### 4. Circular Dependency Detection (Cycle Analysis)
Tukey automatically analyzes the dependency graph for circular references (dependency loops) using a DFS-based cycle detection algorithm. This prevents tight coupling and modular design violations.

* **CLI Output:** Detected cycles are printed directly in the analysis summary. They are formatted as readable path sequences:
  ```
  🔄 Circular Dependencies Detected (2 total):
     ⚠️  Looping dependencies can cause tight coupling and testing bottlenecks. Consider refactoring:
     • Database (class) → QueryBuilder (class) → Connection (class) → Database (class)
     • PaymentService (method) → GatewayFactory (class) → PaymentService (method)
  ```
* **JSON Export:** Cycles are serialized in the `cycles` field of the analysis JSON as an array of path ID arrays.

---

### 5. Architectural Boundary Guardrails (Layer Rules)
You can define architectural boundaries and dependency constraints in your `.tukey.yml` (or `.tukey.json`) config to enforce clean code architecture.

#### Configuration Example (`.tukey.yml`):
```yaml
architecture:
  layers:
    - name: Domain
      path: app/Domain
    - name: Application
      path: app/Services
    - name: Infrastructure
      path: app/Infrastructure
  rules:
    - from: Domain
      cannot_depend_on: [Application, Infrastructure]
    - from: Application
      cannot_depend_on: [Infrastructure]
```

* **Violation Detection:** When running analysis, Tukey maps all nodes to their configured layer based on their file path. If a dependency exists from a layer to a forbidden layer, a boundary violation is reported.
* **CLI Output:**
  ```
  🚨 ARCHITECTURAL LAYER BOUNDARY VIOLATIONS (1 detected)
  ======================================================================
     1. [Domain] UserEntity (app/Domain/UserEntity.php:12)
        → depends on [Infrastructure] DatabaseConnection (app/Infrastructure/DatabaseConnection.php:45)
        Reason: layer 'Domain' is forbidden from depending on layer 'Infrastructure'
  ======================================================================
  ```
* **Strict CI Mode (`--strict` / `-s`):** When run with the strict flag, Tukey will exit with status code `2` if any boundary violations are detected. This is ideal for CI pre-commit checks:
  ```bash
  tukey --strict ./my-project
  ```

---

### 6. Model Context Protocol (MCP) & Localized Context Tool
For AI agent environments, Tukey supports the Model Context Protocol (MCP) server over standard input/output.

```bash
tukey mcp tukey-results.json
```

In addition to symbol search and caller tracing, the MCP server provides **`tukey_get_analysis_status`** for source freshness checks, **`tukey_get_analysis_diagnostics`** for parser and resolution warnings, and **`tukey_get_localized_context`** for dependency neighborhoods. Localized context uses breadth-first traversal up to a specified depth to gather the structural neighborhood around a target symbol, helping agents focus on relevant interfaces and dependents before editing.

#### Tool Parameters:
* `symbol` (string, required for symbol tools): The target symbol name, namespace, or exact node ID.
* `depth` (integer, localized context only): Traversal depth (default `1`, maximum `5`).
* `limit` (integer, optional): Maximum items per collection (default `50`, maximum `100`).
* `cursor` (string, optional): Opaque continuation cursor from `page.nextCursor`.

#### JSON Output Format:
```json
{
  "symbol": "PaymentService",
  "targets": [
    {
      "id": "class:App\\Services\\PaymentService",
      "name": "PaymentService",
      "type": "class",
      "file": "app/Services/PaymentService.php",
      "line": 10,
      "score": 25,
      "dependencyCount": 4,
      "dependentCount": 2
    }
  ],
  "dependencies": [
    {
      "id": "class:App\\Gateways\\StripeGateway",
      "name": "StripeGateway",
      "type": "class",
      "file": "app/Gateways/StripeGateway.php",
      "line": 5,
      "score": 12,
      "dependencyCount": 1,
      "dependentCount": 1,
      "refType": "instantiation",
      "refLines": [24],
      "refCount": 1
    }
  ],
  "dependents": [
    {
      "id": "class:App\\Http\\Controllers\\CheckoutController",
      "name": "CheckoutController",
      "type": "class",
      "file": "app/Http/Controllers/CheckoutController.php",
      "line": 15,
      "score": 30,
      "dependencyCount": 2,
      "dependentCount": 0,
      "refType": "method_call",
      "refLines": [40],
      "refCount": 1
    }
  ]
}
```

Paginated MCP responses include `page.limit`, `page.returned`, `page.total`, and optional `page.nextCursor`. Limits apply independently to each returned collection. To continue, repeat the same query and arguments with `cursor` set to `nextCursor`; stop when it is omitted.

---

## 🤖 Agent Best Practices & Token-Saving Tips

* **Step 1: Check graph freshness**:
  When a graph exists, run `tukey query --status` or call `tukey_get_analysis_status`. Continue only when it is `fresh`; `unknown` means the export lacks source metadata.
* **Step 2: Run a fast analysis when needed**:
  If no graph exists or it is stale, run `tukey -l php,js -o tukey-results.json .` in the background.
* **Step 3: Check analysis completeness**:
  Run `tukey query --diagnostics` or call `tukey_get_analysis_diagnostics`. Treat empty graph results cautiously when parser errors or unresolved references are present.
* **Step 4: Prune your context via MCP Localized Context**:
  If using Tukey's MCP server, invoke `tukey_get_localized_context` for a symbol you plan to edit. This avoids reading unrelated code, returning immediate dependency and dependent boundaries in one step.
* **Step 5: Trace callers before editing**:
  Before modifying a method or function, run `tukey query --callers "ClassName::methodName"`. Review the returned call sites alongside diagnostics; static analysis may not resolve dynamic dispatch.
* **Step 6: Self-Audit structural changes before staging**:
  After completing edits, generate a temporary graph and run `tukey --compare baseline.json` to verify you haven't introduced circular dependencies, complexity regressions, or orphaned dead code.
* **Step 7: Review dead-code candidates**:
  Run `tukey query --orphans` to discover isolated elements. Verify entry points, dynamic calls, and framework hooks before deleting anything.
