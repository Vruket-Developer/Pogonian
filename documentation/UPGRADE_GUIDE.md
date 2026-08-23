# 🔄 Pogonian Versioning & Upgrade Guide

This guide explains how Pogonian handles versioning, backward compatibility, and automated project upgrades when new versions of the framework are released (e.g. from v0.1 &rarr; v1.0 &rarr; v2.0).

---

## 🔍 Checking Current Version

To check your installed Pogonian CLI and runtime version:

```bash
pogn version
# or
pogn -v
# or
pogn --version
# or
pogn --v
```

---

## 🚀 Upgrading Existing Projects (`pogn upgrade` / `pogn update`)

When a developer installs a new version of the Pogonian CLI binary and opens an existing project, Pogonian provides a single command to adapt and modernize the codebase:

```bash
pogn upgrade
# or
pogn update
```

### What `pogn upgrade` Automatically Does:

1. **Configuration Modernization (`pogonian.yaml`)**:
   - Updates project version tag.
   - Injects new configuration fields (e.g. `environment`, `strict` rules, connection pool configs) with sensible backward-compatible defaults without overwriting user settings.

2. **AI Semantic Specification (`app.ai`)**:
   - Validates and migrates `.pogonian/architecture/app.ai` to the latest schema version (e.g. `1.0.0` &rarr; `2.0.0`).
   - Ensures AI permission boundaries and security policies are active.

3. **Database Migration Readiness**:
   - Ensures the `migrations/` directory exists and matches the active migration engine version.

4. **Cryptographic Key & Secret Verification**:
   - Ensures `.env` contains valid 256-bit `APP_KEY` and `SESSION_SECRET` tokens.

5. **Client Runtime Sync**:
   - Updates `runtime/runtime.js` soft navigation engine and script lifecycles.

---

## 🛡️ Backward Compatibility Guarantees

1. **HTML-First Stability**: Because Pogonian uses standard semantic HTML5 and native Go `html/template`, your UI code will never break due to JavaScript framework churn.
2. **Deterministic Go Modules**: Your Go dependencies in `go.mod` remain locked and reproducible.
3. **Database Migration History**: Your applied migrations in `_pogn_migrations` are preserved across upgrades.
