# 📚 Pogonian Documentation Hub

Welcome to the official documentation for **Pogonian** — the architecture-aware, full-stack Go and HTML-first web framework with soft navigation, sub-millisecond responses, and zero global state.

---

## 📑 Documentation Guides

- [**Getting Started & Running Applications (`GETTING_STARTED.md`)**](./GETTING_STARTED.md)  
  *Step-by-step guide on how to install, scaffold, run locally, configure environments (`.env`), and compile standalone release binaries.*

- [**CLI Commands Reference (`CLI_COMMANDS.md`)**](./CLI_COMMANDS.md)  
  *Comprehensive reference of all `pogn` CLI commands, flags, parameters, and usage examples.*

- [**Architecture & Design System (`ARCHITECTURE.md`)**](./ARCHITECTURE.md)  
  *In-depth breakdown of the 5-Tier Modular Monolith, Radix Route Trie, Soft-Navigation Runtime, and Compile-Time Boundary Checker.*

- [**Database, Auth & Migrations Guide (`DATABASE_AND_MIGRATIONS.md`)**](./DATABASE_AND_MIGRATIONS.md)  
  *Guide to local SQLite WAL database, remote PostgreSQL support, Argon2id authentication, persistent sessions, and dynamic entity models.*

---

## ⚡ Quick Cheat Sheet

```bash
# 1. Scaffold a new application
pogn create my-app
cd my-app

# 2. Start the live development server with in-page DevTools
pogn dev

# 3. Add a full-stack entity (Go backend + HTML frontend)
pogn add table cart
pogn add route /cart

# 4. Analyze change blast radius before making edits
pogn impact cart

# 5. Check architecture boundaries and security posture
pogn check
pogn doctor
pogn security audit

# 6. Build a standalone zero-dependency production release
pogn build
./dist/my-app
```
