# 🏛️ Pogonian Architecture Specification

Pogonian implements a strict **5-Tier Modular Monolith** architecture designed to eliminate architectural drift, minimize AI context overhead, and provide sub-millisecond request execution with zero global state.

---

## 🏗️ 5-Tier Architectural Pipeline

Every request follows a deterministic unidirectional flow:

```text
┌────────────────────────────────────────────────────────┐
│ 1. Browser Client (runtime.js SPA Soft Navigation)      │
└───────────────────────────┬────────────────────────────┘
                            │ HTTP Request (X-Pogonian-Request)
                            ▼
┌────────────────────────────────────────────────────────┐
│ 2. Radix Route Trie (internal/router/trie.go - O(k))   │
└───────────────────────────┬────────────────────────────┘
                            │ Route Match & Extracted Params (:id)
                            ▼
┌────────────────────────────────────────────────────────┐
│ 3. HTTP Handler (server/<module>/handler.go)           │
│    - CSRF Token Validation                             │
│    - Session / Auth Cookie Extraction                  │
│    - Form Payload Decoding                             │
└───────────────────────────┬────────────────────────────┘
                            │ Typed Request DTO
                            ▼
┌────────────────────────────────────────────────────────┐
│ 4. Domain Service (server/<module>/service.go)         │
│    - Business Validation & Business Logic              │
│    - Domain Rule Enforcement                           │
└───────────────────────────┬────────────────────────────┘
                            │ Service Method Call
                            ▼
┌────────────────────────────────────────────────────────┐
│ 5. Repository (server/<module>/repository.go)          │
│    - Parameterized SQL Execution                       │
│    - Row Scanning & Model Hydration                    │
└───────────────────────────┬────────────────────────────┘
                            │ SQL Queries
                            ▼
┌────────────────────────────────────────────────────────┐
│ 6. Database (SQLite WAL Mode / PostgreSQL Connection)  │
└────────────────────────────────────────────────────────┘
```

---

## 🛡️ Compile-Time Boundary Enforcements

The `pogn check` engine enforces strict isolation rules on every build:

| Layer | Can Call | CANNOT Call |
| :--- | :--- | :--- |
| **`app/` (Pages & Layouts)** | Client runtime, HTML components | Direct SQL, Database drivers, Repositories |
| **`server/<mod>/handler.go`** | Services, Response Writers | Raw SQL queries, Database drivers directly |
| **`server/<mod>/service.go`** | Repositories, Domain Entities | HTTP Request/Response objects, Direct SQL |
| **`server/<mod>/repository.go`**| Database Connection Pool | HTTP objects, Services, Handlers |

---

## ⚡ Zero-Drift Radix Route Trie

Pogonian replaces linear route arrays with a Radix Tree (Trie) router in [`internal/router/trie.go`](file:///home/vraj/Desktop/Vruket/Pogonian/internal/router/trie.go):
- **Complexity**: $O(k)$ lookup time, where $k$ is the URL segment depth.
- **Precedence**: Static routes (e.g. `/products/featured`) take precedence over parameterized routes (`/products/:id`).
- **Type Constraints**: Supports typed route constraints like `[id:int]` or `[slug:string]`.

---

## 🌐 Client-Side Soft Navigation & Lifecycle

Pogonian uses a 3.2 KB zero-dependency client script (`runtime.js`) that intercepts link clicks and form submissions:
1. **`beforeNavigate` Hook**: Allows listeners to prepare or cancel transitions.
2. **`fetch`**: Requests partial HTML with `X-Pogonian-Request: true`.
3. **`beforeSwap` Hook**: Allows custom transition effects.
4. **DOM Swap**: Replaces the inner root container (`#pogn-root`) without destroying `<header>` or layout state.
5. **Head Reconciliation**: Updates `<title>`, OpenGraph meta tags, and canonical URLs.
6. **Script Execution**: Re-executes embedded `<script>` tags dynamically.
7. **`afterNavigate` Hook**: Dispatches `pogonian:after-navigate` for UI components.

---

## 🔒 Security Posture

1. **Argon2id Hashing**: Password storage adheres to modern RFC cryptographic standards (`$argon2id$v=19$m=65536,t=1,p=4$`).
2. **Fail-Closed CSRF**: Every mutable HTTP request (`POST`, `PUT`, `DELETE`, `PATCH`) must carry a valid cryptographic CSRF token or session match.
3. **Session Persistence**: Sessions are stored in the SQLite database (`sessions` table) to survive server reboots.
4. **Path Traversal Protection**: Static asset serving strictly checks folder boundaries preventing `../` directory escapes.
