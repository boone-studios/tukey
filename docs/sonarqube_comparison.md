# Tukey vs. SonarQube: Architectural Comparison

This document details the differences in design philosophy, execution footprint, target audience, and feature sets between Tukey and SonarQube.

---

## 🗺️ High-Level Summary

While both Tukey and SonarQube perform static analysis on source code, they are built for fundamentally different use cases:

* **Tukey** is a **lightweight codebase navigator and architectural intelligence engine** designed for developers and AI agents. It maps code relationships, exposes structural dead code (orphans), and measures complexity hotspots locally in milliseconds with zero infrastructure.
* **SonarQube** is an **enterprise-grade code quality gate and security platform** designed for management, compliance, and centralized reporting. It aggregates linters, security scanners, and test coverage metrics into centralized web dashboards.

---

## 📊 Feature Comparison Table

| Metric | Tukey | SonarQube |
| :--- | :--- | :--- |
| **Primary Focus** | Structural dependencies, complexity mapping, dead code | SAST Security, bug hunting, code smells, test coverage |
| **Infrastructure** | **Zero** (Single Go binary) | **Heavy** (Web server, DB backend, Elasticsearch) |
| **Execution Speed** | ⚡ Extremely fast (milliseconds) | ⏳ Slow (seconds to minutes) |
| **Developer Loop** | Local CLI, JSON export, and instant diffs | Asynchronous CI runs, server processing, web dashboards |
| **AI Agent Friendly** | ✅ Natively integrates via Model Context Protocol (MCP) | ❌ Human-centric web dashboard only |
| **Local Regressions** | ✅ Instant via `--compare` flag | ❌ Requires central server analysis run |
| **Deployment Model** | Local binary or lightweight runner | Hosted server (Docker, VM, SaaS) |

---

## 🔍 Detailed Differences

### 1. Philosophy & Analysis Target
* **SonarQube (The Quality Inspector):** Focuses heavily on code correctness, security compliance (OWASP Top 10), style linting, and aggregating metrics (e.g., test coverage percentages). It reports "issues" that need fixing.
* **Tukey (The Cartographer):** Focuses on code relationships. Tukey is not a linter and does not enforce styling or test coverage rules. Instead, it draws the dependency map of your codebase, highlighting coupling (which components depend on each other) and spotting unused entry points (orphans) that traditional linters fail to track.

### 2. Infrastructure & Local Usability
* **SonarQube:** Requires a dedicated server (running on a VM or Docker), an external SQL database (PostgreSQL, MSSQL, or Oracle), and an Elasticsearch index. Running SonarQube locally is resource-intensive and slow, meaning developers rarely run it during their local development loops.
* **Tukey:** Built in Go, Tukey compiles to a single, lightweight executable. It runs instantly on your local machine, allowing you to trace dependencies, find callers of a symbol, or detect structural regressions with zero setup.

### 3. Change Impact & Regression Audits
* **SonarQube:** Regression reporting is handled server-side. You push a branch, wait for the CI scanner to run, upload the report to the SonarQube server, wait for processing, and review the results in a browser.
* **Tukey:** Includes native local regression auditing. By passing a baseline analysis JSON file (e.g., `tukey --compare baseline.json ./src`), you can run instant structural diffs to see newly added code elements, deleted elements, resolved orphans, or complexity regressions locally in milliseconds before even staging your commit.

### 4. Machine & AI Agent Readability (MCP Server)
* **SonarQube:** Built for human developers, QA leads, and engineering management to consume reports via web browser interfaces.
* **Tukey:** Built to serve as a machine-readable codebase brain for **AI coding agents** (such as Gemini, Claude, Cursor, and Zed). Tukey features a built-in **Model Context Protocol (MCP)** server. compatible agents can invoke Tukey tools over standard I/O (stdin/stdout) to programmatically map classes, trace callers, and detect orphans dynamically during autonomous code edits.

---

## 💡 When to Use Which?

> [!TIP]
> **Use Tukey when you want to:**
> * Understand a complex legacy codebase or map relationships between modules.
> * Identify circular dependencies, tightly coupled classes, and candidate dead code (orphans).
> * Perform instant local regression tests on your code structure.
> * Supercharge your AI coding assistant with a fast, machine-readable codebase map via MCP.
> * Maintain a fast developer feedback loop without waiting for CI server queues.

> [!NOTE]
> **Use SonarQube when you want to:**
> * Enforce enterprise security compliance policies (e.g., scanning for SQL injections or hardcoded keys).
> * Aggregate and block PRs based on code coverage metrics.
> * Provide management with centralized code quality health dashboards across dozens of repositories.
