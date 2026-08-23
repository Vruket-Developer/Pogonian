# 💾 Database, Auth & First-Class Migrations Guide

Pogonian provides an enterprise-grade database abstraction layer (`DBDriver`), connection pooling, Argon2id password security, persistent session stores, and a first-class SQL migration engine.

---

## 🗄️ Database Drivers & Abstraction

Pogonian is database-driver agnostic with built-in zero-config support for local **SQLite (WAL mode)** and remote **PostgreSQL / MySQL** databases.

### 1. SQLite WAL Mode (Default Local Database)

By default, every Pogonian application creates a local SQLite database named `data.db`.

**Configured Pragmas for High Concurrency:**
- `PRAGMA journal_mode = WAL;` (Write-Ahead Logging allows concurrent readers and writers without locks)
- `PRAGMA busy_timeout = 5000;` (Waits up to 5 seconds before reporting busy errors)
- `PRAGMA synchronous = NORMAL;` (Maximum I/O performance with data durability)
- `PRAGMA foreign_keys = ON;` (Enforces referential integrity)

---

### 2. Remote PostgreSQL Configuration

To connect Pogonian to an external PostgreSQL database:

1. **Option A: Via `.env`**:
   ```env
   DATABASE_URL=postgres://app_user:secret_pass@192.168.1.100:5432/myapp_db?sslmode=disable
   ```

2. **Option B: Via `pogonian.yaml`**:
   ```yaml
   database:
     driver: postgres
     url: "postgres://app_user:secret_pass@192.168.1.100:5432/myapp_db?sslmode=disable"
   ```

When Pogonian detects a `postgres://` or `postgresql://` connection string, it routes queries to the PostgreSQL connection pool.

---

## 🔄 First-Class Database Migration Engine

Pogonian includes an integrated schema migration engine tracking versioned `.up.sql` and `.down.sql` files with transaction safety and checksum validation.

### Creating Migrations
```bash
# Create a new versioned migration pair
pogn db:new add_user_profiles
```
Generates:
- `migrations/001_add_user_profiles.up.sql`
- `migrations/001_add_user_profiles.down.sql`

### Applying Migrations
```bash
# Apply all pending schema migrations
pogn db:migrate
```

### Rolling Back Migrations
```bash
# Revert the last applied batch of migrations
pogn db:rollback

# Revert specific number of steps
pogn db:rollback 2
```

### Checking Migration Status
```bash
# View all discovered migrations, applied timestamps, and batch IDs
pogn db:status
```

---

## 👤 Authentication, Argon2id & Session Storage

Pogonian includes production-grade authentication out of the box.

### Argon2id Password Hashing
Passwords are cryptographically hashed using **Argon2id** (`time=1`, `memory=64MB`, `threads=4`, `salt=16 bytes`), resisting GPU and ASIC attacks.

### Pluggable Session Storage
Sessions support three interchangeable storage drivers:
- **SQLite Session Store**: High-performance persistent storage surviving server restarts.
- **Disk File Store**: JSON-serialized session files in `.pogonian/sessions/`.
- **Memory Session Store**: In-memory ephemeral store for testing and microservices.

### Session Fixation & Cookie Hardening
- Automatic session ID regeneration on login (`RegenerateSession`).
- `HttpOnly`, `SameSite=Lax`, and `Secure` cookie attributes automatically configured based on `APP_ENV`.

---

## 🛠️ Dynamic Schema-Driven Entities

When you run `pogn add table <name>`, Pogonian scaffolds:
- The SQL table in `data.db`
- The Go struct model in `server/<name>/entity.go`
- Parameterized SQL CRUD operations in `server/<name>/repository.go`
- HTML input forms matching the struct fields

### Resetting the Database

To wipe and reinitialize the local development database:

```bash
pogn reset
```
This safely removes `data.db`, `data.db-wal`, `data.db-shm` and initializes fresh tables on the next request.
