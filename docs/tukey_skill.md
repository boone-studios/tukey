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

---

### 2. Query the Codebase Graph (Query Phase)
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

## 🤖 Agent Best Practices & Token-Saving Tips

* **Step 1: Check for existing graphs**: 
  When you land in a project, check if `tukey-results.json` already exists. If it does, do NOT re-run the full analysis. Go straight to Step 3.
* **Step 2: Run a fast analysis**: 
  If no graph exists, run `tukey -l php,js -o tukey-results.json .` in the background. It takes less than a second even on large codebases.
* **Step 3: Query target symbols**: 
  To find where a class or function is defined, run `tukey query --find "MySymbol"`. The JSON response will give you the exact file path and line number.
* **Step 4: Trace callers before editing**: 
  Before modifying a method or function, run `tukey query --callers "ClassName::methodName"`. This ensures you know every single call site in the project that will be affected by your changes, preventing breaking changes.
* **Step 5: Clean up dead code**: 
  Run `tukey query --orphans` to instantly discover all unused functions, classes, and constants in the project without manually tracing them.
