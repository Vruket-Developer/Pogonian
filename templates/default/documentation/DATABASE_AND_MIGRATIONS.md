# 💾 Database, Auth & Migrations Guide

Pogonian is database-driver agnostic, with built-in zero-config support for local **SQLite (WAL mode)** and remote **PostgreSQL** databases.

---

## 🗄️ Database Drivers

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

## 👤 Built-in Authentication & Sessions

Pogonian includes production-grade authentication out of the box.

### Users Table Schema
```sql
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT,
    role TEXT DEFAULT 'user',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Sessions Table Schema
Sessions are persisted in SQLite to ensure active logins survive server restarts:
```sql
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    user_email TEXT NOT NULL,
    user_name TEXT,
    user_role TEXT,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL
);
```

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
