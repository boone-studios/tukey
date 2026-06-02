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

#### Target Symbol Formats Supported:
Tukey's query engine is highly resilient and supports matching a symbol in several ways:
1. **Exact Node ID**: `class:App\Services\PaymentService:5` or `method:App\Services\PaymentService\pay:30` (Fastest $O(1)$ resolution).
2. **Fully-Qualified Member**: `App\Services\PaymentService::pay`
3. **Class-Scoped Member**: `PaymentService::pay`
4. **Namespaced Short Name**: `App\Services\PaymentService`
5. **Short Name**: `PaymentService` or `pay`

---

## 📋 JSON Query Output Schema

Every `tukey query` returns a predictable JSON structure:
```json
{
  "query": "find | callers | dependents | orphans",
  "term": "<search term>",
  "count": 1,
  "results": [
    {
      "id": "class:App\\Services\\PaymentService:5",
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

* `results` is always a JSON array (never `null` or empty on misses; it will be `[]`).
* `refType`, `refLines`, and `refCount` are populated for `--callers` and `--dependents` queries to show exactly *how* and *where* (line numbers) the connection occurs.

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

In addition to symbol search and caller tracing, the MCP server provides the **`tukey_get_localized_context`** tool. This tool performs concurrent BFS traversals up to a specified depth to gather the structural neighborhood around a target symbol. This is ideal for pruning codebase context down to only the relevant interfaces and dependents before editing.

#### Tool Parameters:
* `symbol` (string, required): The target symbol name, namespace, or exact node ID.
* `depth` (integer, optional): The traversal depth boundary (defaults to `1`).

#### JSON Output Format:
```json
{
  "symbol": "PaymentService",
  "targets": [
    {
      "id": "class:App\\Services\\PaymentService:10",
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
      "id": "class:App\\Gateways\\StripeGateway:5",
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
      "id": "class:App\\Http\\Controllers\\CheckoutController:15",
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

---

## 🤖 Agent Best Practices & Token-Saving Tips

* **Step 1: Check for existing graphs**: 
  When you land in a project, check if `tukey-results.json` already exists. If it does, do NOT re-run the full analysis. Go straight to Step 3.
* **Step 2: Run a fast analysis**: 
  If no graph exists, run `tukey -l php,js -o tukey-results.json .` in the background. It takes less than a second even on large codebases.
* **Step 3: Prune your context via MCP Localized Context**:
  If using Tukey's MCP server, invoke `tukey_get_localized_context` for a symbol you plan to edit. This avoids reading unrelated code, returning immediate dependency and dependent boundaries in one step.
* **Step 4: Trace callers before editing**: 
  Before modifying a method or function, run `tukey query --callers "ClassName::methodName"`. This ensures you know every single call site in the project that will be affected by your changes, preventing breaking changes.
* **Step 5: Self-Audit structural changes before staging**:
  After completing edits, generate a temporary graph and run `tukey --compare baseline.json` to verify you haven't introduced circular dependencies, complexity regressions, or orphaned dead code.
* **Step 6: Clean up dead code**: 
  Run `tukey query --orphans` to instantly discover all unused functions, classes, and constants in the project without manually tracing them.
