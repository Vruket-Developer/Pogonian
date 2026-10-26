# 🛠️ Pogonian CLI Command Reference

This document provides a complete reference for all available commands in the `pogn` CLI tool.

---

## 📋 Table of Contents

- [Core Development & Testing](#1-core-development--testing)
  - [`pogn create`](#pogn-create)
  - [`pogn dev`](#pogn-dev)
  - [`pogn swagger` / `pogn openapi`](#pogn-swagger--pogn-openapi)
  - [`pogn test`](#pogn-test)
  - [`pogn check`](#pogn-check)
  - [`pogn tree`](#pogn-tree)
  - [`pogn build`](#pogn-build)
  - [`pogn upgrade`](#pogn-upgrade)
- [Database & Migrations](#2-database--migrations)
  - [`pogn db:migrate`](#pogn-dbmigrate)
  - [`pogn db:rollback`](#pogn-dbrollback)
  - [`pogn db:status`](#pogn-dbstatus)
  - [`pogn db:new`](#pogn-dbnew)
- [Scaffolding & Architecture](#3-scaffolding--architecture)
  - [`pogn modules` / `pogn marketplace`](#pogn-modules--pogn-marketplace)
  - [`pogn add module` / `pogn add component`](#pogn-add-module--pogn-add-component)
  - [`pogn add table`](#pogn-add-table)
  - [`pogn add route`](#pogn-add-route)
  - [`pogn remove module`](#pogn-remove-module)
  - [`pogn remove table`](#pogn-remove-table)
  - [`pogn remove route`](#pogn-remove-route)
  - [`pogn inspect`](#pogn-inspect)
  - [`pogn analyze`](#pogn-analyze)
- [Diagnostics & Blast Radius Analysis](#4-diagnostics--blast-radius-analysis)
  - [`pogn impact`](#pogn-impact)
  - [`pogn doctor`](#pogn-doctor)
  - [`pogn explain`](#pogn-explain)
  - [`pogn blueprint`](#pogn-blueprint)
  - [`pogn docs`](#pogn-docs)
- [Secrets, Security & Benchmarks](#5-secrets-security--benchmarks)
  - [`pogn secret` / `pogn key:generate`](#pogn-secret--pogn-keygenerate)
  - [`pogn clean`](#pogn-clean)
  - [`pogn reset`](#pogn-reset)
  - [`pogn security audit`](#pogn-security-audit)
  - [`pogn benchmark` / `pogn benchmark --stress`](#pogn-benchmark)
- [AI & Legacy Migration](#6-ai--legacy-migration)
  - [`pogn ai` & `pogn ai rollback`](#pogn-ai)
  - [`pogn import`](#pogn-import)
  - [`pogn migrate`](#pogn-migrate)

---

## 1. Core Development & Testing

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

---

### `pogn dev`
Launches the high-speed local development server with Radix Route Trie matching, soft-navigation client runtime, hot reload, and in-page DevTools HUD.

```bash
pogn dev [project-dir] [flags]
```

**Features Active in Dev Mode:**
- In-Page DevTools Dashboard at `http://localhost:3000/__pogn/devtools`
- Live Architecture DevBar HUD at bottom right of all pages.
- Automatic route re-scanning on filesystem updates.
- SQLite WAL mode database at `./data.db`.

---

### `pogn swagger` / `pogn openapi`
Generates an OpenAPI 3.0 specification from discovered routes, handlers, and models, and optionally launches an interactive Swagger UI server.

```bash
# Launch interactive Swagger UI server in browser
pogn swagger
pogn swagger --port=3000 --open

# Export OpenAPI 3.0 specification in JSON format
pogn swagger --json
pogn swagger --export=./openapi.json
```

---

### `pogn test`
Unified test suite orchestrator running Architecture Contract Verification, Security & Vulnerability Scans, Go Unit & Integration Tests, and automated Fresh Disposable Project E2E runs.

```bash
# Run all standard tests
pogn test

# Run tests with automated Disposable Fresh Project E2E run
pogn test --e2e

# Run unit tests only
pogn test --unit-only -v
```

---

### `pogn check`
Performs static AST architecture boundary analysis, verifies route mappings, detects circular package dependencies, and verifies layer encapsulation contracts.

```bash
pogn check [project-dir] [--strict]
```

**Rules Enforced:**
1. Pages in `app/` cannot import database drivers directly.
2. HTTP Handlers in `server/` cannot execute raw SQL; they must delegate to Services.
3. Repositories must encapsulate all database queries.
4. No circular dependencies across domain packages.

---

### `pogn tree`
Generates a colorized application, routing, component, and backend directory tree. Shows all filesystem routes mapped to their HTTP endpoints (e.g. `page.html  ➔  GET /products/:id`).

```bash
# View complete application tree
pogn tree

# View filesystem routing & UI components only
pogn tree app
# or
pogn tree --app-only

# Output machine-readable JSON format
pogn tree --json

# Limit recursion depth
pogn tree --depth=3
```

---

### `pogn build`
Compiles your application using Pogonian's multi-stage compiler. Supports targeting Web (HTML5/HTMX/runtime.js) or genuinely native applications for Android (Jetpack Compose), iOS (SwiftUI), Linux Desktop (GTK4), Windows (WinUI), macOS (AppKit), and Android TV (10-Foot spatial UI).

```bash
# Standard optimized web standalone build
pogn build

# Build specifically for native Android (Jetpack Compose)
pogn build --target=android

# Build specifically for native iOS (SwiftUI)
pogn build --target=ios

# Build native desktop binaries
pogn build --target=linux
pogn build --target=windows
pogn build --target=macos

# Build for Android TV / Google TV (10-foot D-pad UI)
pogn build --target=tv

# Compile all native targets simultaneously into ./dist/
pogn build --target=all
```

---

### `pogn upgrade`
Checks project configuration, schema version of `app.ai`, missing migration directories, and upgrades project assets to the latest Pogonian version.

```bash
pogn upgrade [project-dir]
```

---

## 2. Database & Migrations

### `pogn db:migrate`
Applies all pending schema migrations inside the `migrations/` directory using transactional execution and checksum verification.

```bash
pogn db:migrate
```

---

### `pogn db:rollback`
Reverts the last applied batch of migrations.

```bash
# Rollback 1 step
pogn db:rollback

# Rollback 2 steps
pogn db:rollback 2
```

---

### `pogn db:status`
Displays the migration status table, applied timestamps, and batch numbers.

```bash
pogn db:status
```

---

### `pogn db:new`
Creates a timestamped and versioned `.up.sql` and `.down.sql` migration pair.

```bash
pogn db:new add_user_avatar
```

---

## 3. Scaffolding & Architecture

### `pogn modules` / `pogn marketplace`
Lists all available UI components and full-stack modules from the Shadcn-style marketplace registry.

```bash
pogn modules
pogn marketplace
```

---

### `pogn add module` / `pogn add component`
Installs clean, native HTML templates, scoped CSS, Go backend tiers, and database migrations directly into your repository with zero npm dependencies.

```bash
# Install UI-only components (footer, header, sidebar, card)
pogn add component footer
pogn add component header

# Install full-stack feature modules (auth/login, etc.)
pogn add module login
```

---

### `pogn add table`
Generates the complete 5-tier backend (`entity.go`, `repository.go`, `service.go`, `handler.go`) and frontend route (`app/<name>/page.html`) for a new database entity.

```bash
pogn add table cart
pogn add table orders
```

---

### `pogn add route`
Scaffolds a standalone frontend route page in `app/` without backend CRUD boilerplate.

```bash
pogn add route /about
pogn add route /faq
```

---

### `pogn remove module`
Removes an installed marketplace module or component from the project and deregisters it from `app.ai`.

```bash
pogn remove module footer
pogn remove module login
```

---

### `pogn remove table`
Deletes all backend tiers (`server/<name>/`) and the matching frontend view (`app/<name>/`).

```bash
pogn remove table cart
```

---

### `pogn remove route`
Deletes a standalone route directory from `app/`.

```bash
pogn remove route /faq
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
pogn analyze --format=json
```

---

## 4. Diagnostics & Blast Radius Analysis

### `pogn impact`
Calculates the blast radius of modifying a specific file, struct, or feature, identifying all affected routes, handlers, services, repositories, database schemas, and AI prompt context.

```bash
pogn impact server/products/service.go
pogn impact cart
```

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
pogn explain products
```

---

### `pogn blueprint`
Generates a structured architecture blueprint as a Mermaid graph diagram, JSON schema, or YAML export.

```bash
pogn blueprint
```

---

### `pogn docs`
Generates an automatic API and route catalog table showing endpoints, HTTP methods, authorization requirements, and descriptions.

```bash
pogn docs
```

---

### `pogn ir`
Emits or inspects the Universal Pogonian Intermediate Representation (IR v2.0) containing the normalized UI element tree, portable application state, event/action models, and capability requirements.

```bash
# Export as JSON
pogn ir

# Export as YAML
pogn ir --yaml

# Compile IR specifically targeting a platform
pogn ir --target=android
```

---

### `pogn capabilities`
Audits the complete cross-platform hardware & OS capability matrix across all target environments (Web, Android, iOS, Linux, Windows, macOS, Android TV), reporting supported, permission-conditional, and unsupported capabilities with actionable recommendations.

```bash
pogn capabilities
```

---

## 5. Secrets, Security & Benchmarks

### `pogn secret` / `pogn key:generate`
Generates a new cryptographically secure 256-bit `APP_KEY` and updates your local `.env` file.

```bash
pogn secret
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
Performs static AST security auditing and automated dynamic probes against SQL injections, hardcoded secret keys, CSRF vulnerabilities, and path traversals.

```bash
pogn security audit
```

---

### `pogn benchmark`
Runs research evaluation benchmarks and high-throughput scale stress tests.

```bash
# Standard evaluation benchmark
pogn benchmark

# 10,000+ route stress benchmark (Trie vs Regex performance)
pogn benchmark --stress --routes=10000 --lookups=100000
```

---

## 6. AI & Legacy Migration

### `pogn ai` & `pogn ai rollback`
Generates an architecture-aware implementation plan using exact BPE tokenizer measurements, sandbox validation, and automated snapshotting.

```bash
# Plan feature with token estimation
pogn ai "Add shopping cart with stripe checkout"

# Apply feature with permission check and sandbox validation
pogn ai "Add shopping cart with stripe checkout" --apply

# Rollback last applied AI modifications
pogn ai rollback
```

---

### `pogn import`
Analyzes legacy React, Next.js, Vue, or Express codebases and extracts route hierarchies and behavioral interaction patterns.

```bash
pogn import /path/to/legacy-react-app
```

---

### `pogn migrate`
Converts legacy frontend applications into native Pogonian Go backend handlers and HTML-first templates.

```bash
pogn migrate /path/to/legacy-nextjs-app
```
