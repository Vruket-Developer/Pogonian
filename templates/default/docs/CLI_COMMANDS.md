# 🛠️ Pogonian CLI Command Reference

This document provides a complete reference for all available commands in the `pogn` CLI tool.

---

## 📋 Table of Contents

- [Core Development](#1-core-development)
  - [`pogn create`](#pogn-create)
  - [`pogn dev`](#pogn-dev)
  - [`pogn check`](#pogn-check)
  - [`pogn build`](#pogn-build)
- [Scaffolding & Architecture](#2-scaffolding--architecture)
  - [`pogn add table`](#pogn-add-table)
  - [`pogn add route`](#pogn-add-route)
  - [`pogn remove table`](#pogn-remove-table)
  - [`pogn remove route`](#pogn-remove-route)
  - [`pogn inspect`](#pogn-inspect)
  - [`pogn analyze`](#pogn-analyze)
- [Diagnostics & Blast Radius Analysis](#3-diagnostics--blast-radius-analysis)
  - [`pogn impact`](#pogn-impact)
  - [`pogn doctor`](#pogn-doctor)
  - [`pogn explain`](#pogn-explain)
  - [`pogn blueprint`](#pogn-blueprint)
  - [`pogn docs`](#pogn-docs)
- [Secrets, Security & Maintenance](#4-secrets-security--maintenance)
  - [`pogn secret` / `pogn key:generate`](#pogn-secret--pogn-keygenerate)
  - [`pogn clean`](#pogn-clean)
  - [`pogn reset`](#pogn-reset)
  - [`pogn security audit`](#pogn-security-audit)
  - [`pogn benchmark`](#pogn-benchmark)
- [AI & Legacy Migration](#5-ai--legacy-migration)
  - [`pogn ai`](#pogn-ai)
  - [`pogn import`](#pogn-import)
  - [`pogn migrate`](#pogn-migrate)

---

## 1. Core Development

### `pogn create`
Scaffolds a new Pogonian application complete with 5-tier architecture, `.env` file with 256-bit cryptographic `APP_KEY`, `.env.example`, `pogonian.yaml`, and `pogonian.lock`.

```bash
# Syntax
pogn create <app-name> [target-dir] [flags]

# Examples
pogn create shop-app
pogn create my-portal ./custom-dir
pogn create demo --dry-run
pogn create existing-folder --force
```

**Flags:**
- `--force`, `-f`: Overwrite existing non-empty directory.
- `--dry-run`: Preview files to be created without writing to disk.
- `--template=<name>`: Choose scaffolding template (default: `default`).

---

### `pogn dev`
Launches the high-speed local development server with Radix Route Trie matching, soft-navigation client runtime, hot reload, and in-page DevTools HUD.

```bash
# Syntax
pogn dev [project-dir] [flags]

# Examples
pogn dev
pogn dev ./my-app
pogn dev --port=8080
```

**Features Active in Dev Mode:**
- In-Page DevTools Dashboard at `http://localhost:3000/__pogn/devtools`
- Live Architecture DevBar HUD at bottom right of all pages.
- Automatic route re-scanning on filesystem updates.
- SQLite WAL mode database at `./data.db`.

---

### `pogn check`
Performs static AST architecture boundary analysis, verifies route mappings, detects circular package dependencies, and verifies layer encapsulation rules.

```bash
# Syntax
pogn check [project-dir] [flags]

# Examples
pogn check
pogn check ./my-app
pogn check --strict
```

**Rules Enforced:**
1. Pages in `app/` cannot import database drivers directly.
2. HTTP Handlers in `server/` cannot execute raw SQL; they must delegate to Services.
3. Repositories must encapsulate all database queries.
4. No circular dependencies across domain packages.

---

### `pogn build`
Synthesizes a standalone application server entrypoint (`.pogonian/build/main.go`), embeds all HTML templates, client runtime (`runtime.js`), CSS, and assets, and compiles a single optimized binary.

```bash
# Syntax
pogn build [project-dir] [flags]

# Examples
pogn build
pogn build ./my-app
pogn build --standalone
```

**Output:**  
Creates `./dist/<app-name>` (Single executable binary with zero external dependencies).

---

## 2. Scaffolding & Architecture

### `pogn add table`
Generates the complete 5-tier backend (`entity.go`, `repository.go`, `service.go`, `handler.go`) and frontend route (`app/<name>/page.html`) for a new database entity.

```bash
# Syntax
pogn add table <table-name> [flags]

# Examples
pogn add table cart
pogn add table orders
pogn add table feedback
```

**Created Artifacts:**
- `server/<name>/entity.go` (Go struct)
- `server/<name>/repository.go` (Parameterized SQL repository)
- `server/<name>/service.go` (Domain business logic)
- `server/<name>/handler.go` (HTTP action handler)
- `app/<name>/page.html` (Frontend view and form)

---

### `pogn add route`
Scaffolds a standalone frontend route page in `app/` without backend CRUD boilerplate.

```bash
# Syntax
pogn add route <path> [flags]

# Examples
pogn add route /about
pogn add route /faq
pogn add route /dashboard/analytics
```

---

### `pogn remove table`
Deletes all backend tiers (`server/<name>/`) and the matching frontend view (`app/<name>/`).

```bash
# Examples
pogn remove table cart
pogn rm table orders
```

---

### `pogn remove route`
Deletes a standalone route directory from `app/`.

```bash
# Examples
pogn remove route /faq
pogn rm route /about
```

---

### `pogn inspect`
Opens an interactive terminal UI to inspect discovered routes, layer bindings, middleware chains, and SQLite schemas.

```bash
pogn inspect
```

---

### `pogn analyze`
Parses the Go AST and HTML templates to emit the Pogonian Intermediate Representation (IR) and dependency graph in JSON or YAML.

```bash
pogn analyze
pogn analyze --format=json
pogn analyze --format=yaml
```

---

## 3. Diagnostics & Blast Radius Analysis

### `pogn impact`
Calculates the blast radius of modifying a specific file, struct, or feature, identifying all affected routes, handlers, services, repositories, database schemas, and AI prompt context.

```bash
# Syntax
pogn impact <target> [flags]

# Examples
pogn impact server/products/service.go
pogn impact cart
pogn impact products --ai
```

**Output Includes:**
- Impact Level (`LOW`, `MEDIUM`, `HIGH`, `CRITICAL`)
- Affected Routes
- Affected Handlers & Services
- Affected Database Entities
- Potential Breaking Risks (e.g. signature changes)
- AI Subgraph Token Optimization Scope

---

### `pogn doctor`
Performs comprehensive system diagnostics, verifying the Go toolchain, Node runtime, project configuration, SQLite/Postgres reachability, port availability, and security posture.

```bash
pogn doctor
```

---

### `pogn explain`
Explains the full-stack architecture flow (Browser &rarr; Trie Router &rarr; Handler &rarr; Service &rarr; Repository &rarr; Database) for the whole application or for a specific feature.

```bash
# Whole app explanation
pogn explain

# Specific feature explanation
pogn explain products
pogn explain cart
```

---

### `pogn blueprint`
Generates a structured architecture blueprint as a Mermaid graph diagram, JSON schema, or YAML export.

```bash
# Mermaid graph (default)
pogn blueprint

# JSON export
pogn blueprint --format=json

# YAML export
pogn blueprint --format=yaml
```

---

### `pogn docs`
Generates an automatic API and route catalog table showing endpoints, HTTP methods, authorization requirements, and descriptions.

```bash
pogn docs
```

---

## 4. Secrets, Security & Maintenance

### `pogn secret` / `pogn key:generate`
Generates a new cryptographically secure 256-bit `APP_KEY` and updates your local `.env` file.

```bash
pogn secret
# or
pogn key:generate
```

---

### `pogn clean`
Deletes build caches, `./dist/` binaries, `.pogonian/build/`, and temporary test artifacts.

```bash
pogn clean
```

---

### `pogn reset`
Cleans build artifacts and safely resets the local development SQLite database (`data.db`, `data.db-wal`, `data.db-shm`) to a clean slate.

```bash
pogn reset
```

---

### `pogn security audit`
Performs static AST security auditing against SQL injections, hardcoded secret keys, insecure password hashes, CSRF vulnerabilities, and path traversals.

```bash
pogn security audit
```

---

### `pogn benchmark`
Runs the MSc empirical research evaluation benchmark comparing Pogonian metrics (TTFB, JS bundle size, AI context tokens) against traditional stacks.

```bash
pogn benchmark
```

---

## 5. AI & Legacy Migration

### `pogn ai`
Generates an architecture-aware implementation plan for a new feature and produces minimal sub-graph token context.

```bash
# Examples
pogn ai "Add shopping cart with stripe checkout"
pogn ai "Add user profiles and avatar uploads"
```

---

### `pogn import`
Analyzes legacy React, Next.js, Vue, or Express codebases and extracts route hierarchies and components.

```bash
pogn import /path/to/legacy-react-app
```

---

### `pogn migrate`
Converts legacy frontend applications into native Pogonian Go backend handlers and HTML-first templates.

```bash
pogn migrate /path/to/legacy-nextjs-app
```
