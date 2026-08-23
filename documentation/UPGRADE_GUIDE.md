# 🔄 Pogonian Versioning, Compatibility & Upgrade Guide

This guide explains how Pogonian handles versioning, dependency compatibility checking, backward compatibility, and automated project upgrades when new versions of the framework are released (e.g. from v0.1 &rarr; v0.2 &rarr; v1.0 &rarr; v2.0).

---

## 🔍 Checking Current Version

To check your installed Pogonian CLI and runtime version:

```bash
pogn version
# or
pogn --v
# or
pogn -v
# or
pogn --version
```

---

## 🧐 Checking Available Versions & Compatibility (`pogn upgrade --check`)

Before performing an upgrade, you can check which framework versions are available and test whether your system dependencies, Go toolchain, and libraries are compatible:

```bash
pogn upgrade --check
# or
pogn upgrade --list
```

### Output Example:
```text
🔍 Pogonian Version Compatibility & Upgrade Inspector
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Current Project Version: 0.1.0
  System Go Toolchain:     go1.26.6
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Available Upgrade Targets & Compatibility Matrix:

  • v0.1.0      [Stable    ] — 2026-06-01  (Installed Current)
    ↳ Requires Go: 1.21+ | Runtime: Pogonian 0.1.0
    ↳ New: Initial modular-monolith 5-tier architecture; SQLite WAL mode integration

  • v0.2.0      [Latest    ] — 2026-08-23  ✓ Compatible
    ↳ Requires Go: 1.22+ | Runtime: Pogonian 0.2.0
    ↳ New: Radix Trie Router (40x faster); Argon2id auth; Migration engine; BPE Tokenizer; Shadcn module marketplace; pogn test --e2e
    ↳ Breaking Changes: CSRF validation fails closed by default on mutating requests
    ↳ Deprecations: Legacy single-file router regex matching deprecated

  • v1.0.0-LTS  [LTS (Long Term Support)] — 2026-10-15 (Upcoming)  ✓ Compatible
    ↳ Requires Go: 1.22+ | Runtime: Pogonian 1.0.0-LTS
    ↳ New: Long-term support enterprise API guarantees; Redis/Postgres distributed sessions

  • v2.0.0      [Preview   ] — 2027-01-10 (Future)  ✓ Compatible
    ↳ Requires Go: 1.23+ | Runtime: Pogonian 2.0.0
    ↳ New: WebAssembly client streaming; Autonomous AI orchestration
    ↳ Breaking Changes: Requires Go 1.23+ toolchain; App.ai schema v2.0
```

---

## 🚀 Upgrading Your Project

### 1. Upgrade to the Latest Recommended Release
```bash
pogn upgrade
# or
pogn update
```

### 2. Upgrade to a Specific Target Version
```bash
pogn upgrade --target=0.2.0
```

### 3. Preview Upgrade Actions Without Modifying Files
```bash
pogn upgrade --dry-run
```

---

## 🛡️ What `pogn upgrade` Automatically Handles:

1. **Configuration Modernization (`pogonian.yaml`)**:
   - Injects new configuration fields (e.g. `environment`, `strict` rules, connection pool configs) with sensible backward-compatible defaults without overwriting user settings.

2. **AI Semantic Specification (`app.ai`)**:
   - Validates and migrates `.pogonian/architecture/app.ai` to the target schema version.
   - Ensures AI permission boundaries and security policies are active.

3. **Database Migration Readiness**:
   - Ensures the `migrations/` directory exists and matches the active migration engine version.

4. **Cryptographic Key & Secret Verification**:
   - Ensures `.env` contains valid 256-bit `APP_KEY` and `SESSION_SECRET` tokens.

5. **Client Runtime Sync**:
   - Updates `runtime/runtime.js` soft navigation engine and script lifecycles.
